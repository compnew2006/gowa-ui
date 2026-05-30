#!/usr/bin/env python3
"""
Facebook Comments SQLite Database Manager
Each Facebook page gets its own SQLite database
"""
import sqlite3
import json
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional

class FacebookPageDB:
    """Manage comments for a single Facebook page"""
    
    def __init__(self, page_id: str, base_dir: str = "/opt/hermes-webhook/databases"):
        self.page_id = page_id
        self.base_dir = Path(base_dir)
        self.base_dir.mkdir(parents=True, exist_ok=True)
        
        # Each page gets its own database file
        self.db_path = self.base_dir / f"page_{page_id}.db"
        self._init_db()
    
    def _init_db(self):
        """Initialize database tables"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        # Comments table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS comments (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                comment_id TEXT UNIQUE NOT NULL,
                post_id TEXT NOT NULL,
                message TEXT NOT NULL,
                from_name TEXT NOT NULL,
                from_id TEXT,
                parent_id TEXT,
                verb TEXT,
                created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                facebook_timestamp INTEGER,
                processed BOOLEAN DEFAULT FALSE,
                replied BOOLEAN DEFAULT FALSE,
                reply_id TEXT,
                reply_text TEXT,
                raw_data TEXT
            )
        ''')
        
        # Create indexes
        cursor.execute('CREATE INDEX IF NOT EXISTS idx_comment_id ON comments(comment_id)')
        cursor.execute('CREATE INDEX IF NOT EXISTS idx_post_id ON comments(post_id)')
        cursor.execute('CREATE INDEX IF NOT EXISTS idx_created_time ON comments(created_time)')
        
        # Replies table (track our auto-replies)
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS replies (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                comment_id TEXT NOT NULL,
                reply_id TEXT NOT NULL,
                reply_text TEXT NOT NULL,
                created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                ai_model TEXT,
                reply_type TEXT
            )
        ''')
        
        cursor.execute('CREATE INDEX IF NOT EXISTS idx_replies_comment_id ON replies(comment_id)')
        
        # Interactions table (for learning)
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS interactions (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                comment_id TEXT NOT NULL,
                commenter_name TEXT,
                question_type TEXT,
                language TEXT,
                created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        ''')
        
        cursor.execute('CREATE INDEX IF NOT EXISTS idx_interactions_comment_id ON interactions(comment_id)')
        
        # Stats table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS stats (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                date DATE DEFAULT CURRENT_DATE,
                total_comments INTEGER DEFAULT 0,
                total_replies INTEGER DEFAULT 0,
                unique_commenters INTEGER DEFAULT 0,
                page_id TEXT,
                UNIQUE(date, page_id)
            )
        ''')
        
        conn.commit()
        conn.close()
    
    def add_comment(self, comment_data: Dict) -> int:
        """Add a new comment to the database"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        try:
            cursor.execute('''
                INSERT OR REPLACE INTO comments 
                (comment_id, post_id, message, from_name, from_id, parent_id, 
                 verb, facebook_timestamp, raw_data)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            ''', (
                comment_data.get('comment_id'),
                comment_data.get('post_id'),
                comment_data.get('message', ''),
                comment_data.get('from', 'Unknown'),
                comment_data.get('sender_id', ''),
                comment_data.get('parent_id', ''),
                comment_data.get('verb', 'add'),
                comment_data.get('created_time', int(datetime.now().timestamp())),
                json.dumps(comment_data.get('raw', {}), ensure_ascii=False)
            ))
            
            comment_db_id = cursor.lastrowid
            conn.commit()
            
            # Update stats
            self._update_stats(conn)
            
            return comment_db_id
        except sqlite3.IntegrityError:
            # Comment already exists
            return 0
        finally:
            conn.close()
    
    def mark_as_replied(self, comment_id: str, reply_id: str, reply_text: str, 
                       ai_model: str = "unknown", reply_type: str = "auto"):
        """Mark a comment as replied"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        cursor.execute('''
            UPDATE comments 
            SET replied = TRUE, reply_id = ?, reply_text = ?
            WHERE comment_id = ?
        ''', (reply_id, reply_text, comment_id))
        
        # Add to replies table
        cursor.execute('''
            INSERT INTO replies (comment_id, reply_id, reply_text, ai_model, reply_type)
            VALUES (?, ?, ?, ?, ?)
        ''', (comment_id, reply_id, reply_text, ai_model, reply_type))
        
        conn.commit()
        conn.close()
    
    def get_unreplied_comments(self, limit: int = 100) -> List[Dict]:
        """Get comments that haven't been replied to"""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()
        
        cursor.execute('''
            SELECT * FROM comments 
            WHERE replied = FALSE 
            ORDER BY created_time DESC 
            LIMIT ?
        ''', (limit,))
        
        comments = [dict(row) for row in cursor.fetchall()]
        conn.close()
        
        return comments
    
    def get_recent_comments(self, hours: int = 24, limit: int = 100) -> List[Dict]:
        """Get recent comments from last N hours"""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()
        
        cursor.execute('''
            SELECT * FROM comments 
            WHERE datetime(created_time) >= datetime('now', '-' || ? || ' hours')
            ORDER BY created_time DESC 
            LIMIT ?
        ''', (hours, limit))
        
        comments = [dict(row) for row in cursor.fetchall()]
        conn.close()
        
        return comments
    
    def search_comments(self, query: str, limit: int = 50) -> List[Dict]:
        """Search comments by text"""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()
        
        cursor.execute('''
            SELECT * FROM comments 
            WHERE message LIKE ? OR from_name LIKE ?
            ORDER BY created_time DESC 
            LIMIT ?
        ''', (f'%{query}%', f'%{query}%', limit))
        
        comments = [dict(row) for row in cursor.fetchall()]
        conn.close()
        
        return comments
    
    def get_stats(self) -> Dict:
        """Get statistics for this page"""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()
        
        # Total comments
        cursor.execute('SELECT COUNT(*) as total FROM comments')
        total_comments = cursor.fetchone()['total']
        
        # Comments today
        cursor.execute('''
            SELECT COUNT(*) as today_count 
            FROM comments 
            WHERE DATE(created_time) = DATE('now')
        ''')
        today_comments = cursor.fetchone()['today_count']
        
        # Replied comments
        cursor.execute('SELECT COUNT(*) as replied FROM comments WHERE replied = TRUE')
        replied_comments = cursor.fetchone()['replied']
        
        # Unique commenters
        cursor.execute('SELECT COUNT(DISTINCT from_id) as unique_count FROM comments')
        unique_commenters = cursor.fetchone()['unique_count']
        
        conn.close()
        
        return {
            'total_comments': total_comments,
            'today_comments': today_comments,
            'replied_comments': replied_comments,
            'unique_commenters': unique_commenters,
            'reply_rate': round((replied_comments / total_comments * 100) if total_comments > 0 else 0, 2)
        }
    
    def _update_stats(self, conn):
        """Update daily statistics"""
        cursor = conn.cursor()
        today = datetime.now().strftime('%Y-%m-%d')
        
        cursor.execute('''
            INSERT OR REPLACE INTO stats (date, total_comments, page_id)
            VALUES (
                COALESCE((SELECT date FROM stats WHERE date = ? AND page_id = ?), ?),
                COALESCE((SELECT total_comments + 1 FROM stats WHERE date = ? AND page_id = ?), 1),
                ?
            )
        ''', (today, self.page_id, today, today, self.page_id, self.page_id))
        
        conn.commit()

# Global database manager
_db_cache = {}

def get_page_db(page_id: str) -> FacebookPageDB:
    """Get or create database for a page"""
    global _db_cache
    if page_id not in _db_cache:
        _db_cache[page_id] = FacebookPageDB(page_id)
    return _db_cache[page_id]

if __name__ == "__main__":
    # Test the database
    import sys
    
    if len(sys.argv) > 1:
        page_id = sys.argv[1]
        db = FacebookPageDB(page_id)
        
        # Test adding a comment
        test_comment = {
            'comment_id': 'test_123',
            'post_id': 'post_456',
            'message': 'Test message',
            'from': 'Test User',
            'sender_id': 'user_789',
            'verb': 'add',
            'created_time': int(datetime.now().timestamp()),
            'raw': {}
        }
        
        comment_id = db.add_comment(test_comment)
        print(f"✅ Added comment with DB ID: {comment_id}")
        
        # Get stats
        stats = db.get_stats()
        print(f"📊 Stats: {stats}")
    else:
        print("Usage: python facebook_db.py <page_id>")
