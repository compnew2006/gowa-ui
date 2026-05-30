#!/usr/bin/env python3
"""
Facebook Comments Viewer - View comments from SQLite databases
"""
import sqlite3
from pathlib import Path
from datetime import datetime
import json

def list_all_pages():
    """List all Facebook pages with databases"""
    db_dir = Path("/opt/hermes-webhook/databases")
    if not db_dir.exists():
        print("❌ No databases directory found")
        return []
    
    pages = []
    for db_file in db_dir.glob("page_*.db"):
        page_id = db_file.stem.replace("page_", "")
        pages.append(page_id)
    
    return pages

def view_page_comments(page_id, limit=20):
    """View recent comments for a page"""
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    
    if not Path(db_path).exists():
        print(f"❌ Database not found for page {page_id}")
        return
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    # Get stats
    cursor.execute('''
        SELECT 
            COUNT(*) as total,
            SUM(CASE WHEN replied = 1 THEN 1 ELSE 0 END) as replied,
            COUNT(DISTINCT from_id) as unique_commenters
        FROM comments
    ''')
    stats = cursor.fetchone()
    
    print(f"\n{'='*60}")
    print(f"📊 Page: {page_id}")
    print(f"{'='*60}")
    print(f"Total Comments: {stats['total']}")
    print(f"Replied: {stats['replied']}")
    print(f"Pending: {stats['total'] - stats['replied']}")
    print(f"Unique Commenters: {stats['unique_commenters']}")
    print(f"{'='*60}\n")
    
    # Get recent comments
    cursor.execute('''
        SELECT * FROM comments 
        ORDER BY created_time DESC 
        LIMIT ?
    ''', (limit,))
    
    comments = cursor.fetchall()
    
    if not comments:
        print("No comments found")
        return
    
    for i, comment in enumerate(comments, 1):
        status = "✅" if comment['replied'] else "⏳"
        timestamp = datetime.fromisoformat(comment['created_time']).strftime('%Y-%m-%d %H:%M')
        
        print(f"{i}. {status} {timestamp}")
        print(f"   👤 {comment['from_name']}")
        print(f"   💬 {comment['message'][:100]}")
        
        if comment['replied']:
            print(f"   ↩️  Reply: {comment['reply_text'][:80]}")
        
        print()
    
    conn.close()

def search_comments(page_id, query, limit=10):
    """Search comments by text"""
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    
    if not Path(db_path).exists():
        print(f"❌ Database not found for page {page_id}")
        return
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    cursor.execute('''
        SELECT * FROM comments 
        WHERE message LIKE ? OR from_name LIKE ?
        ORDER BY created_time DESC 
        LIMIT ?
    ''', (f'%{query}%', f'%{query}%', limit))
    
    comments = cursor.fetchall()
    
    print(f"\n🔍 Search results for '{query}' in page {page_id}:")
    print(f"{'='*60}\n")
    
    for i, comment in enumerate(comments, 1):
        status = "✅" if comment['replied'] else "⏳"
        timestamp = datetime.fromisoformat(comment['created_time']).strftime('%Y-%m-%d %H:%M')
        
        print(f"{i}. {status} {timestamp}")
        print(f"   👤 {comment['from_name']}")
        print(f"   💬 {comment['message'][:100]}")
        print()
    
    conn.close()

def show_recent_replies(page_id, limit=10):
    """Show recent auto-replies"""
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    
    if not Path(db_path).exists():
        print(f"❌ Database not found for page {page_id}")
        return
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    cursor.execute('''
        SELECT r.*, c.from_name, c.message as original_message
        FROM replies r
        JOIN comments c ON r.comment_id = c.comment_id
        ORDER BY r.created_time DESC
        LIMIT ?
    ''', (limit,))
    
    replies = cursor.fetchall()
    
    print(f"\n💬 Recent Auto-Replies for page {page_id}:")
    print(f"{'='*60}\n")
    
    for i, reply in enumerate(replies, 1):
        timestamp = datetime.fromisoformat(reply['created_time']).strftime('%Y-%m-%d %H:%M')
        
        print(f"{i}. {timestamp}")
        print(f"   👤 From: {reply['from_name']}")
        print(f"   ❓ Question: {reply['original_message'][:80]}")
        print(f"   ✍️  Reply: {reply['reply_text'][:100]}")
        print(f"   🤖 Model: {reply['ai_model']}")
        print()
    
    conn.close()

def main():
    import sys
    
    if len(sys.argv) < 2:
        print("Facebook Comments Viewer")
        print("=" * 30)
        print("\nUsage:")
        print("  python view_comments.py list                    # List all pages")
        print("  python view_comments.py view <page_id>          # View page comments")
        print("  python view_comments.py search <page_id> <query>  # Search comments")
        print("  python view_comments.py replies <page_id>       # View recent replies")
        print("\nExamples:")
        print("  python view_comments.py list")
        print("  python view_comments.py view 895247390337022")
        print("  python view_comments.py search 895247390337022 'يوتيوب'")
        print("  python view_comments.py replies 895247390337022")
        return
    
    command = sys.argv[1]
    
    if command == "list":
        pages = list_all_pages()
        if pages:
            print("\n📋 Facebook Pages with Databases:")
            print("=" * 40)
            for page_id in pages:
                print(f"  • {page_id}")
        else:
            print("No pages found")
    
    elif command == "view" and len(sys.argv) >= 3:
        page_id = sys.argv[2]
        limit = int(sys.argv[3]) if len(sys.argv) > 3 else 20
        view_page_comments(page_id, limit)
    
    elif command == "search" and len(sys.argv) >= 4:
        page_id = sys.argv[2]
        query = sys.argv[3]
        search_comments(page_id, query)
    
    elif command == "replies" and len(sys.argv) >= 3:
        page_id = sys.argv[2]
        show_recent_replies(page_id)
    
    else:
        print("❌ Invalid command. Use: list, view, search, or replies")

if __name__ == "__main__":
    main()
