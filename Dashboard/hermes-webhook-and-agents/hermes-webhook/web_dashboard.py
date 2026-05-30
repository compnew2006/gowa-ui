#!/usr/bin/env python3
"""
Simple web dashboard to view Facebook comments
"""
from flask import Flask, render_template_string, jsonify, request
import sqlite3
from pathlib import Path
from datetime import datetime

app = Flask(__name__)

HTML_TEMPLATE = """
<!DOCTYPE html>
<html dir="rtl" lang="ar">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Facebook Comments Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            background: white;
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .header h1 {
            color: #667eea;
            font-size: 28px;
            margin-bottom: 10px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }
        .stat-card {
            background: white;
            border-radius: 8px;
            padding: 15px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }
        .stat-card h3 {
            font-size: 14px;
            color: #666;
            margin-bottom: 5px;
        }
        .stat-card .value {
            font-size: 28px;
            font-weight: bold;
            color: #667eea;
        }
        .comments-section {
            background: white;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .comments-section h2 {
            color: #667eea;
            margin-bottom: 15px;
            font-size: 20px;
        }
        .comment-card {
            border: 1px solid #e0e0e0;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 15px;
            background: #fafafa;
        }
        .comment-card.pending {
            border-left: 4px solid #ffa726;
        }
        .comment-card.replied {
            border-left: 4px solid #66bb6a;
        }
        .comment-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 10px;
        }
        .commenter {
            font-weight: bold;
            color: #333;
        }
        .timestamp {
            font-size: 12px;
            color: #999;
        }
        .comment-message {
            color: #555;
            line-height: 1.5;
            margin-bottom: 10px;
        }
        .reply-section {
            background: white;
            border-radius: 6px;
            padding: 10px;
            margin-top: 10px;
            border-right: 3px solid #667eea;
        }
        .reply-label {
            font-size: 12px;
            color: #667eea;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .reply-text {
            color: #555;
            font-size: 14px;
        }
        .search-box {
            margin-bottom: 20px;
        }
        .search-box input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 16px;
            font-family: inherit;
        }
        .search-box input:focus {
            outline: none;
            border-color: #667eea;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
        }
        .status-badge.pending {
            background: #fff3e0;
            color: #ffa726;
        }
        .status-badge.replied {
            background: #e8f5e9;
            color: #66bb6a;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💬 Facebook Comments Dashboard</h1>
            <p>Page ID: {{ page_id }}</p>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <h3>Total Comments</h3>
                <div class="value">{{ stats.total_comments }}</div>
            </div>
            <div class="stat-card">
                <h3>Today</h3>
                <div class="value">{{ stats.today_comments }}</div>
            </div>
            <div class="stat-card">
                <h3>Replied</h3>
                <div class="value">{{ stats.replied_comments }}</div>
            </div>
            <div class="stat-card">
                <h3>Pending</h3>
                <div class="value">{{ stats.total_comments - stats.replied_comments }}</div>
            </div>
        </div>
        
        <div class="comments-section">
            <h2>Recent Comments</h2>
            <div class="search-box">
                <input type="text" id="searchInput" placeholder="🔍 Search comments..." 
                       onkeyup="if(event.key === 'Enter') searchComments()">
            </div>
            <div id="commentsContainer">
                {% for comment in comments %}
                <div class="comment-card {{ 'replied' if comment.replied else 'pending' }}">
                    <div class="comment-header">
                        <span class="commenter">{{ comment.from_name }}</span>
                        <span class="timestamp">{{ comment.created_time }}</span>
                    </div>
                    <div class="comment-message">{{ comment.message }}</div>
                    {% if comment.replied %}
                    <div class="reply-section">
                        <div class="reply-label">✅ Auto-Reply</div>
                        <div class="reply-text">{{ comment.reply_text }}</div>
                    </div>
                    {% else %}
                    <span class="status-badge pending">⏳ Pending Reply</span>
                    {% endif %}
                </div>
                {% endfor %}
            </div>
        </div>
    </div>
    
    <script>
        function searchComments() {
            const query = document.getElementById('searchInput').value;
            if (query.length < 2) return;
            
            fetch(`/search?q=${encodeURIComponent(query)}`)
                .then(response => response.json())
                .then(data => {
                    displayComments(data.comments);
                });
        }
        
        function displayComments(comments) {
            const container = document.getElementById('commentsContainer');
            container.innerHTML = comments.map(comment => `
                <div class="comment-card ${comment.replied ? 'replied' : 'pending'}">
                    <div class="comment-header">
                        <span class="commenter">${comment.from_name}</span>
                        <span class="timestamp">${comment.created_time}</span>
                    </div>
                    <div class="comment-message">${comment.message}</div>
                    ${comment.replied ? `
                        <div class="reply-section">
                            <div class="reply-label">✅ Auto-Reply</div>
                            <div class="reply-text">${comment.reply_text}</div>
                        </div>
                    ` : '<span class="status-badge pending">⏳ Pending Reply</span>'}
                </div>
            `).join('');
        }
        
        // Auto-refresh every 30 seconds
        setInterval(() => {
            location.reload();
        }, 30000);
    </script>
</body>
</html>
"""

def get_page_stats(page_id):
    """Get page statistics"""
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    if not Path(db_path).exists():
        return {
            'total_comments': 0,
            'today_comments': 0,
            'replied_comments': 0,
            'unique_commenters': 0
        }
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    cursor.execute('SELECT COUNT(*) as total FROM comments')
    total = cursor.fetchone()['total']
    
    cursor.execute("SELECT COUNT(*) as today_count FROM comments WHERE DATE(created_time) = DATE('now')")
    today = cursor.fetchone()['today_count']
    
    cursor.execute('SELECT COUNT(*) as replied FROM comments WHERE replied = TRUE')
    replied = cursor.fetchone()['replied']
    
    conn.close()
    
    return {
        'total_comments': total,
        'today_comments': today,
        'replied_comments': replied,
        'unique_commenters': 0
    }

def get_recent_comments(page_id, limit=20):
    """Get recent comments"""
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    if not Path(db_path).exists():
        return []
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    cursor.execute('''
        SELECT * FROM comments 
        ORDER BY created_time DESC 
        LIMIT ?
    ''', (limit,))
    
    comments = []
    for row in cursor.fetchall():
        comment = dict(row)
        # Format timestamp
        dt = datetime.fromisoformat(comment['created_time'])
        comment['created_time'] = dt.strftime('%Y-%m-%d %H:%M')
        comments.append(comment)
    
    conn.close()
    return comments

@app.route('/')
def index():
    page_id = request.args.get('page', '895247390337022')
    
    stats = get_page_stats(page_id)
    comments = get_recent_comments(page_id)
    
    return render_template_string(HTML_TEMPLATE, 
                                  page_id=page_id,
                                  stats=stats,
                                  comments=comments)

@app.route('/search')
def search():
    page_id = request.args.get('page', '895247390337022')
    query = request.args.get('q', '')
    
    if not query:
        return jsonify({'comments': []})
    
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    if not Path(db_path).exists():
        return jsonify({'comments': []})
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    cursor.execute('''
        SELECT * FROM comments 
        WHERE message LIKE ? OR from_name LIKE ?
        ORDER BY created_time DESC 
        LIMIT 50
    ''', (f'%{query}%', f'%{query}%'))
    
    comments = []
    for row in cursor.fetchall():
        comment = dict(row)
        dt = datetime.fromisoformat(comment['created_time'])
        comment['created_time'] = dt.strftime('%Y-%m-%d %H:%M')
        comments.append(comment)
    
    conn.close()
    
    return jsonify({'comments': comments})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001, debug=False)
