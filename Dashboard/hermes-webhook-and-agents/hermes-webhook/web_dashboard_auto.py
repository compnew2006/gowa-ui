#!/usr/bin/env python3
"""
Facebook Comments Dashboard with Real-time Updates
Supports auto-refresh and SSE for instant updates
"""
from flask import Flask, render_template_string, jsonify, request
import sqlite3
from pathlib import Path
from datetime import datetime
import time
import threading

app = Flask(__name__)

# Background thread to watch for new comments
class CommentWatcher:
    def __init__(self):
        self.last_counts = {}
        self.lock = threading.Lock()
    
    def check_for_updates(self, page_id):
        """Check if there are new comments"""
        try:
            db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
            if not Path(db_path).exists():
                return {"has_new": False, "count": 0}
            
            conn = sqlite3.connect(db_path)
            conn.row_factory = sqlite3.Row
            cursor = conn.cursor()
            
            cursor.execute('SELECT COUNT(*) as count FROM comments')
            result = cursor.fetchone()
            current_count = result['count']
            
            with self.lock:
                last_count = self.last_counts.get(page_id, 0)
                has_new = current_count > last_count
                
                if has_new:
                    self.last_counts[page_id] = current_count
                
                conn.close()
                return {"has_new": has_new, "count": current_count}
                
        except Exception as e:
            print(f"Error checking for updates: {e}")
            return {"has_new": False, "count": 0}

watcher = CommentWatcher()

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
        .refresh-info {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-top: 15px;
            padding: 10px;
            background: #f0f0f0;
            border-radius: 6px;
        }
        .refresh-btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            transition: background 0.3s;
        }
        .refresh-btn:hover {
            background: #5568d3;
        }
        .refresh-btn:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .last-updated {
            font-size: 14px;
            color: #666;
        }
        .new-comments-badge {
            background: #ff4444;
            color: white;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            display: none;
        }
        .new-comments-badge.show {
            display: inline-block;
            animation: pulse 1s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
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
            transition: transform 0.3s;
        }
        .stat-card:hover {
            transform: translateY(-2px);
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
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .new-badge {
            background: #ff4444;
            color: white;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            animation: pulse 1s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        .comments-list {
            max-height: 600px;
            overflow-y: auto;
        }
        .comment-card {
            border: 1px solid #e0e0e0;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 15px;
            background: #fafafa;
            transition: all 0.3s;
        }
        .comment-card.new {
            border-left: 4px solid #ff4444;
            animation: slideIn 0.5s ease-out;
        }
        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateX(20px);
            }
            to {
                opacity: 1;
                transform: translateX(0);
            }
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
            transition: border-color 0.3s;
        }
        .search-box input:focus {
            outline: none;
            border-color: #667eea;
        }
        .loading {
            text-align: center;
            padding: 20px;
            display: none;
        }
        .loading.show {
            display: block;
        }
        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .no-comments {
            text-align: center;
            padding: 40px;
            color: #999;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💬 Facebook Comments Dashboard</h1>
            <p>Page ID: {{ page_id }}</p>
            
            <div class="refresh-info">
                <button class="refresh-btn" onclick="loadComments(true)">
                    🔄 تحديث الآن
                </button>
                <span class="last-updated">
                    آخر تحديث: <span id="lastUpdated">جاري التحميل...</span>
                </span>
                <span class="new-comments-badge" id="newBadge">
                    🔔 تعليقات جديدة!
                </span>
            </div>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <h3>💬 إجمالي التعليقات</h3>
                <div class="value" id="totalCount">{{ stats.total_comments }}</div>
            </div>
            <div class="stat-card">
                <h3>📆 اليوم</h3>
                <div class="value" id="todayCount">{{ stats.today_comments }}</div>
            </div>
            <div class="stat-card">
                <h3>✅ تم الرد</h3>
                <div class="value" id="repliedCount">{{ stats.replied_comments }}</div>
            </div>
            <div class="stat-card">
                <h3>⏳ في الانتظار</h3>
                <div class="value" id="pendingCount">{{ stats.total_comments - stats.replied_comments }}</div>
            </div>
        </div>
        
        <div class="comments-section">
            <h2>
                <span>التعليقات الأخيرة</span>
                <span class="new-badge" id="commentBadge" style="display:none;">🆕 جديد</span>
            </h2>
            <div class="search-box">
                <input type="text" id="searchInput" placeholder="🔍 البحث في التعليقات..." 
                       onkeyup="if(event.key === 'Enter') searchComments()">
            </div>
            
            <div class="loading" id="loading">
                <div class="spinner"></div>
                <p>جاري التحميل...</p>
            </div>
            
            <div class="comments-list" id="commentsContainer">
                {% if comments %}
                    {% for comment in comments %}
                    <div class="comment-card {{ 'new' if comment.is_new else '' }} {{ 'replied' if comment.replied else 'pending' }}">
                        <div class="comment-header">
                            <span class="commenter">{{ comment.from_name }}</span>
                            <span class="timestamp">{{ comment.created_time }}</span>
                        </div>
                        <div class="comment-message">{{ comment.message }}</div>
                        {% if comment.replied %}
                        <div class="reply-section">
                            <div class="reply-label">✅ الرد التلقائي</div>
                            <div class="reply-text">{{ comment.reply_text }}</div>
                        </div>
                        {% else %}
                        <span style="color: #ffa726; font-size: 12px; font-weight: bold;">⏳ في الانتظار</span>
                        {% endif %}
                    </div>
                    {% endfor %}
                    {% else %}
                    <div class="no-comments">
                        <p>لا توجد تعليقات حتى الآن</p>
                    </div>
                {% endif %}
            </div>
        </div>
    </div>
    
    <script>
        let currentCommentCount = 0;
        let autoRefreshInterval;
        let isNewComment = false;
        
        // Load comments on page load
        window.onload = function() {
            loadComments(false);
            startAutoRefresh();
            checkForNewComments();
        };
        
        function loadComments(showLoading = false) {
            const pageId = '{{ page_id }}';
            
            if (showLoading) {
                document.getElementById('loading').classList.add('show');
                document.querySelector('.comments-list').style.opacity = '0.5';
            }
            
            fetch('/comments?page=' + pageId + '&_t=' + Date.now())
                .then(response => response.json())
                .then(data => {
                    updateComments(data.comments || []);
                    updateStats(data.stats || {});
                    updateLastUpdated();
                    
                    // Check if there are new comments
                    if (data.comments && data.comments.length > currentCommentCount) {
                        if (!isNewComment) {
                            showNewCommentsBadge();
                        }
                        currentCommentCount = data.comments.length;
                        isNewComment = false;
                    }
                })
                .catch(error => {
                    console.error('Error loading comments:', error);
                })
                .finally(() => {
                    if (showLoading) {
                        document.getElementById('loading').classList.remove('show');
                        document.querySelector('.comments-list').style.opacity = '1';
                    }
                });
        }
        
        function updateStats(stats) {
            document.getElementById('totalCount').textContent = stats.total_comments || 0;
            document.getElementById('todayCount').textContent = stats.today_comments || 0;
            document.getElementById('repliedCount').textContent = stats.replied_comments || 0;
            document.getElementById('pendingCount').textContent = (stats.total_comments || 0) - (stats.replied_comments || 0);
        }
        
        function updateComments(comments) {
            const container = document.getElementById('commentsContainer');
            
            if (!comments || comments.length === 0) {
                container.innerHTML = '<div class="no-comments"><p>لا توجد تعليقات حتى الآن</p></div>';
                return;
            }
            
            container.innerHTML = comments.map((comment, index) => {
                const isNew = comment.is_new || index >= currentCommentCount;
                const statusClass = comment.replied ? 'replied' : 'pending';
                const replySection = comment.replied ? `
                    <div class="reply-section">
                        <div class="reply-label">✅ الرد التلقائي</div>
                        <div class="reply-text">${escapeHtml(comment.reply_text)}</div>
                    </div>
                ` : `<span style="color: #ffa726; font-size: 12px; font-weight: bold;">⏳ في الانتظار</span>`;
                
                return `
                    <div class="comment-card ${isNew ? 'new' : ''} ${statusClass}">
                        <div class="comment-header">
                            <span class="commenter">${escapeHtml(comment.from_name)}</span>
                            <span class="timestamp">${comment.created_time}</span>
                        </div>
                        <div class="comment-message">${escapeHtml(comment.message)}</div>
                        ${replySection}
                    </div>
                `;
            }).join('');
            
            // Scroll to top if there are new comments
            if (comments.some(c => c.is_new)) {
                container.scrollTop = 0;
            }
        }
        
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        function updateLastUpdated() {
            const now = new Date();
            const time = now.toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' });
            document.getElementById('lastUpdated').textContent = time + ' - ' + now.toLocaleDateString('ar-EG');
        }
        
        function showNewCommentsBadge() {
            const badge = document.getElementById('commentBadge');
            const newCommentsBadge = document.getElementById('newBadge');
            
            if (badge) badge.style.display = 'inline-block';
            if (newCommentsBadge) {
                newCommentsBadge.classList.add('show');
                
                // Hide after 5 seconds
                setTimeout(() => {
                    newCommentsBadge.classList.remove('show');
                }, 5000);
            }
        }
        
        function searchComments() {
            const query = document.getElementById('searchInput').value;
            if (query.length < 2) return;
            
            const pageId = '{{ page_id }}';
            fetch('/search?q=' + encodeURIComponent(query) + '&page=' + pageId + '&_t=' + Date.now())
                .then(response => response.json())
                .then(data => {
                    updateComments(data.comments || []);
                    hideNewCommentsBadge();
                });
        }
        
        function startAutoRefresh() {
            // Auto-refresh every 15 seconds
            autoRefreshInterval = setInterval(() => {
                checkForNewComments();
            }, 15000);
        }
        
        function checkForNewComments() {
            fetch('/check-updates?page={{ page_id }}&_t=' + Date.now())
                .then(response => response.json())
                .then(data => {
                    if (data.has_new) {
                        loadComments(false);
                        showNewCommentsBadge();
                        // Play notification sound if available
                        playNotificationSound();
                    }
                });
        }
        
        function playNotificationSound() {
            // Optional: Play a subtle notification sound
            try {
                const audio = new Audio('/static/notification.mp3');
                audio.volume = 0.3;
                audio.play().catch(() => {}); // Ignore errors
            } catch (e) {
                // Audio not supported, ignore
            }
        }
        
        function hideNewCommentsBadge() {
            const badge = document.getElementById('commentBadge');
            if (badge) badge.style.display = 'none';
        }
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

def get_recent_comments(page_id, limit=20, mark_new=False):
    """Get recent comments and optionally mark new ones"""
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
    for i, row in enumerate(cursor.fetchall()):
        comment = dict(row)
        dt = datetime.fromisoformat(comment['created_time'])
        comment['created_time'] = dt.strftime('%Y-%m-%d %H:%M')
        comment['is_new'] = mark_new and i == 0  # Mark first comment as new if requested
        comments.append(comment)
    
    conn.close()
    return comments

@app.route('/')
def index():
    page_id = request.args.get('page', '895247390337022')
    
    stats = get_page_stats(page_id)
    comments = get_recent_comments(page_id)
    
    # Update current count
    global currentCommentCount
    currentCommentCount = len(comments)
    
    return render_template_string(HTML_TEMPLATE, 
                                  page_id=page_id,
                                  stats=stats,
                                  comments=comments)

@app.route('/comments')
def comments():
    page_id = request.args.get('page', '895247390337022')
    limit = int(request.args.get('limit', 20))
    
    stats = get_page_stats(page_id)
    comments = get_recent_comments(page_id, limit, mark_new=True)
    
    return jsonify({
        'stats': stats,
        'comments': comments
    })

@app.route('/check-updates')
def check_updates():
    """Check for new comments via AJAX"""
    page_id = request.args.get('page', '895247390337022')
    
    # Check if there are new comments
    db_path = f"/opt/hermes-webhook/databases/page_{page_id}.db"
    if not Path(db_path).exists():
        return jsonify({'has_new': False, 'count': 0})
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    cursor.execute('SELECT COUNT(*) as count FROM comments')
    result = cursor.fetchone()

    conn.close()

    current_count = result[0]
    has_new = current_count > currentCommentCount
    
    return jsonify({
        'has_new': has_new,
        'count': current_count
    })

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
        comment['is_new'] = False
        comments.append(comment)
    
    conn.close()
    
    return jsonify({'comments': comments})

# Global variable
currentCommentCount = 0

if __name__ == '__main__':
    app.run(host='127.0.0.1', port=5001, debug=False, threaded=True)
