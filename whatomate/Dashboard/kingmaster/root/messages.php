<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

$page_title = "الرسائل | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<style>
.messages-container {
    display: grid;
    grid-template-columns: 350px 1fr;
    height: calc(100vh - 120px);
    margin-top: 120px;
    gap: 0;
    background: var(--card-bg);
    border-radius: 20px;
    overflow: hidden;
    max-width: 1400px;
    margin-left: auto;
    margin-right: auto;
    border: 2px solid var(--border-color);
}

.conversations-sidebar {
    background: var(--card-bg);
    border-left: 2px solid var(--border-color);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    height: 100%;
}

.conversations-sidebar::-webkit-scrollbar {
    width: 8px;
}

.conversations-sidebar::-webkit-scrollbar-track {
    background: var(--card-bg);
}

.conversations-sidebar::-webkit-scrollbar-thumb {
    background: linear-gradient(135deg, #667eea, #764ba2);
    border-radius: 10px;
}

.conversations-sidebar::-webkit-scrollbar-thumb:hover {
    background: linear-gradient(135deg, #764ba2, #667eea);
}

#conversations-list {
    flex: 1;
    overflow-y: auto;
}

.conversations-header {
    padding: 20px;
    border-bottom: 2px solid var(--border-color);
    background: linear-gradient(135deg, #667eea, #764ba2);
    color: white;
}

.conversations-header h2 {
    margin: 0;
    font-size: 24px;
    font-family: 'Cairo', sans-serif;
    font-weight: 800;
}

.conversation-item {
    padding: 15px 20px;
    border-bottom: 1px solid var(--border-color);
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    gap: 15px;
    align-items: center;
}

.conversation-item:hover {
    background: rgba(102, 126, 234, 0.1);
}

.conversation-item.active {
    background: linear-gradient(135deg, rgba(102, 126, 234, 0.2), rgba(118, 75, 162, 0.2));
    border-left: 4px solid #667eea;
}

.conversation-avatar {
    width: 50px;
    height: 50px;
    border-radius: 50%;
    object-fit: cover;
}

.conversation-info {
    flex: 1;
}

.conversation-name {
    font-weight: 700;
    color: var(--text-primary);
    font-family: 'Cairo', sans-serif;
    margin-bottom: 5px;
}

.conversation-last-message {
    font-size: 13px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.conversation-unread {
    background: #667eea;
    color: white;
    border-radius: 50%;
    width: 25px;
    height: 25px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 700;
}

.chat-area {
    display: flex;
    flex-direction: column;
    background: var(--bg-primary);
    height: 100%;
    overflow: hidden;
}

.chat-header {
    padding: 20px;
    border-bottom: 2px solid var(--border-color);
    background: var(--card-bg);
    display: flex;
    align-items: center;
    gap: 15px;
}

.chat-header-avatar {
    width: 45px;
    height: 45px;
    border-radius: 50%;
}

.chat-header-info h3 {
    margin: 0;
    font-size: 20px;
    font-weight: 800;
    color: var(--text-primary);
    font-family: 'Cairo', sans-serif;
}

.chat-messages {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 15px;
    min-height: 0;
}

.chat-messages::-webkit-scrollbar {
    width: 8px;
}

.chat-messages::-webkit-scrollbar-track {
    background: var(--bg-primary);
}

.chat-messages::-webkit-scrollbar-thumb {
    background: linear-gradient(135deg, #667eea, #764ba2);
    border-radius: 10px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
    background: linear-gradient(135deg, #764ba2, #667eea);
}

.message {
    display: flex;
    gap: 10px;
    max-width: 70%;
}

.message.sent {
    align-self: flex-end;
    flex-direction: row-reverse;
}

.message.received {
    align-self: flex-start;
}

.message-bubble {
    padding: 12px 18px;
    border-radius: 18px;
    font-family: 'Cairo', sans-serif;
    word-wrap: break-word;
}

.message.sent .message-bubble {
    background: linear-gradient(135deg, #667eea, #764ba2);
    color: white;
    border-bottom-left-radius: 4px;
}

.message.received .message-bubble {
    background: var(--card-bg);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-bottom-right-radius: 4px;
}

.message-time {
    font-size: 11px;
    color: var(--text-secondary);
    margin-top: 5px;
    text-align: left;
}

.message.sent .message-time {
    text-align: right;
}

.chat-input-area {
    padding: 20px;
    border-top: 2px solid var(--border-color);
    background: var(--card-bg);
    display: flex;
    gap: 10px;
    flex-shrink: 0;
}

.chat-input {
    flex: 1;
    padding: 12px 20px;
    border: 2px solid var(--border-color);
    border-radius: 25px;
    background: var(--bg-primary);
    color: var(--text-primary);
    font-family: 'Cairo', sans-serif;
    font-size: 14px;
    outline: none;
    transition: all 0.3s ease;
}

.chat-input:focus {
    border-color: #667eea;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.send-btn {
    padding: 12px 25px;
    background: linear-gradient(135deg, #667eea, #764ba2);
    color: white;
    border: none;
    border-radius: 25px;
    font-family: 'Cairo', sans-serif;
    font-weight: 700;
    cursor: pointer;
    transition: all 0.3s ease;
}

.send-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

.empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    gap: 15px;
}

.empty-state i {
    font-size: 80px;
    opacity: 0.3;
}

.upload-btn {
    padding: 12px 20px;
    background: linear-gradient(135deg, #10b981, #059669);
    color: white;
    border: none;
    border-radius: 25px;
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    align-items: center;
    gap: 8px;
    font-family: 'Cairo', sans-serif;
    font-weight: 600;
}

.upload-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 5px 15px rgba(16, 185, 129, 0.4);
}

.upload-btn i {
    font-size: 16px;
}

.image-preview-container {
    position: relative;
    display: none;
    margin: 10px 0;
    padding: 10px;
    background: var(--card-bg);
    border-radius: 15px;
    border: 2px solid var(--border-color);
}

.image-preview {
    max-width: 200px;
    max-height: 150px;
    border-radius: 10px;
    display: block;
    margin: 0 auto;
}

.remove-image-btn {
    position: absolute;
    top: 15px;
    left: 15px;
    background: #ef4444;
    color: white;
    border: none;
    border-radius: 50%;
    width: 30px;
    height: 30px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s ease;
    font-size: 14px;
}

.remove-image-btn:hover {
    background: #dc2626;
    transform: scale(1.1);
}

.message-image {
    max-width: 300px;
    max-height: 300px;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.3s ease;
    margin-top: 8px;
    display: block;
}

.message-image:hover {
    transform: scale(1.02);
    box-shadow: 0 4px 12px rgba(0,0,0,0.3);
}

/* Modal للصورة بحجم كامل */
.image-modal {
    display: none;
    position: fixed;
    z-index: 2000;
    left: 0;
    top: 0;
    width: 100%;
    height: 100%;
    background: rgba(0,0,0,0.95);
    backdrop-filter: blur(10px);
    align-items: center;
    justify-content: center;
}

.image-modal img {
    max-width: 90%;
    max-height: 90vh;
    border-radius: 10px;
    box-shadow: 0 10px 40px rgba(0,0,0,0.5);
}

.image-modal-close {
    position: absolute;
    top: 20px;
    left: 20px;
    font-size: 40px;
    color: white;
    cursor: pointer;
    background: rgba(255,255,255,0.1);
    width: 50px;
    height: 50px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s ease;
}

.image-modal-close:hover {
    background: rgba(255,255,255,0.2);
    transform: rotate(90deg);
}
</style>

<div class="messages-container">
    <div class="conversations-sidebar">
        <div class="conversations-header">
            <h2><i class="fas fa-comments"></i> المحادثات</h2>
        </div>
        <div style="padding: 15px; border-bottom: 2px solid var(--border-color);">
            <button onclick="openNewChatModal()" style="width: 100%; padding: 12px; background: linear-gradient(135deg, #10b981, #059669); color: white; border: none; border-radius: 10px; font-weight: 700; font-family: 'Cairo', sans-serif; cursor: pointer; transition: all 0.3s ease;">
                <i class="fas fa-plus-circle"></i> محادثة جديدة
            </button>
        </div>
        <div id="conversations-list">
            <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
                <i class="fas fa-spinner fa-spin"></i> جاري التحميل...
            </div>
        </div>
    </div>
    
    <div class="chat-area">
        <div id="empty-chat" class="empty-state">
            <i class="fas fa-comment-dots"></i>
            <h3>اختر محادثة لبدء الدردشة</h3>
        </div>
        
        <div id="active-chat" style="display: none; height: 100%; flex-direction: column;">
            <div class="chat-header">
                <img src="" alt="" class="chat-header-avatar" id="chat-avatar">
                <div class="chat-header-info">
                    <h3 id="chat-name"></h3>
                </div>
            </div>
            
            <div class="chat-messages" id="chat-messages">
                <!-- Messages will be loaded here -->
            </div>
            
            <div class="chat-input-area" style="flex-direction: column; align-items: stretch;">
                <div id="image-preview-container" class="image-preview-container">
                    <button class="remove-image-btn" onclick="removeImagePreview()">
                        <i class="fas fa-times"></i>
                    </button>
                    <img id="image-preview" class="image-preview" src="" alt="معاينة الصورة">
                </div>
                <div style="display: flex; gap: 10px;">
                    <input type="file" id="image-input" accept="image/*" style="display: none;" onchange="previewImage()">
                    <button class="upload-btn" onclick="document.getElementById('image-input').click()">
                        <i class="fas fa-image"></i>
                    </button>
                    <input type="text" class="chat-input" id="message-input" placeholder="اكتب رسالتك...">
                    <button class="send-btn" onclick="sendMessage()">
                        <i class="fas fa-paper-plane"></i> إرسال
                    </button>
                </div>
            </div>
        </div>
    </div>
</div>

<!-- Modal: Image Viewer -->
<div id="imageModal" class="image-modal" onclick="closeImageModal()">
    <span class="image-modal-close">&times;</span>
    <img id="modal-image" src="" alt="صورة">
</div>

<!-- Modal: New Chat -->
<div id="newChatModal" style="display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.8); backdrop-filter: blur(5px);">
    <div style="background: var(--card-bg); margin: 5% auto; padding: 30px; border-radius: 20px; width: 90%; max-width: 500px; border: 2px solid var(--border-color);">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 25px;">
            <h2 style="margin: 0; font-size: 24px; font-weight: 800; color: var(--text-primary); font-family: 'Cairo', sans-serif;">
                <i class="fas fa-user-plus"></i> محادثة جديدة
            </h2>
            <span onclick="closeNewChatModal()" style="font-size: 28px; font-weight: bold; color: var(--text-secondary); cursor: pointer; transition: all 0.3s ease;">&times;</span>
        </div>
        
        <div style="margin-bottom: 20px;">
            <input type="text" id="user-search" placeholder="ابحث عن مستخدم..." 
                   style="width: 100%; padding: 12px 20px; border: 2px solid var(--border-color); border-radius: 10px; background: var(--bg-primary); color: var(--text-primary); font-family: 'Cairo', sans-serif; font-size: 14px;" 
                   oninput="filterUsers(this.value)">
        </div>
        
        <div id="users-list" style="max-height: 400px; overflow-y: auto;">
            <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
                <i class="fas fa-spinner fa-spin"></i> جاري التحميل...
            </div>
        </div>
    </div>
</div>

<script>
let activeConversationId = null;
let activeReceiverId = null;
let allUsers = [];
let selectedImagePath = null;

// Load conversations
function loadConversations() {
    fetch('api/get_conversations.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const list = document.getElementById('conversations-list');
            
            if (data.conversations.length === 0) {
                list.innerHTML = `
                    <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
                        <i class="fas fa-inbox" style="font-size: 40px; margin-bottom: 10px;"></i><br>
                        لا توجد محادثات بعد
                    </div>
                `;
            } else {
                list.innerHTML = data.conversations.map(conv => {
                    const escapedName = conv.other_user_name.replace(/'/g, "\\'");
                    const escapedImg = conv.other_user_img.replace(/'/g, "\\'");
                    return `
                        <div class="conversation-item" onclick="openChat(${conv.conversation_id}, '${conv.other_user_id}', '${escapedName}', '${escapedImg}')">
                            <img src="${conv.other_user_img}" alt="${conv.other_user_name}" class="conversation-avatar">
                            <div class="conversation-info">
                                <div class="conversation-name">${conv.other_user_name}</div>
                                <div class="conversation-last-message">${conv.last_message || 'لا توجد رسائل'}</div>
                            </div>
                            ${conv.unread_count > 0 ? `<div class="conversation-unread">${conv.unread_count}</div>` : ''}
                        </div>
                    `;
                }).join('');
            }
        }
    });
}

function openChat(conversationId, receiverId, userName, userImg) {
    activeConversationId = conversationId;
    activeReceiverId = receiverId;
    
    // Update UI
    document.getElementById('empty-chat').style.display = 'none';
    document.getElementById('active-chat').style.display = 'flex';
    document.getElementById('chat-avatar').src = userImg;
    document.getElementById('chat-name').textContent = userName;
    
    // Mark active conversation
    document.querySelectorAll('.conversation-item').forEach(item => {
        item.classList.remove('active');
    });
    event.currentTarget.classList.add('active');
    
    // Load messages
    loadMessages(conversationId);
}

function loadMessages(conversationId) {
    fetch(`api/get_messages.php?conversation_id=${conversationId}`)
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const messagesContainer = document.getElementById('chat-messages');
            const currentUserId = '<?php echo $user_id; ?>';
            
            messagesContainer.innerHTML = data.messages.map(msg => {
                const isSent = msg.sender_id == currentUserId;
                let content = '';
                
                if (msg.image_path) {
                    content += `<img src="${msg.image_path}" class="message-image" onclick="openImageModal('${msg.image_path}')" alt="صورة">`;
                }
                
                if (msg.message) {
                    content += `<div class="message-bubble">${msg.message}</div>`;
                }
                
                return `
                    <div class="message ${isSent ? 'sent' : 'received'}">
                        <div>
                            ${content}
                            <div class="message-time">${formatTime(msg.created_at)}</div>
                        </div>
                    </div>
                `;
            }).join('');
            
            // Scroll to bottom
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
            
            // Reload conversations to update unread count
            loadConversations();
        }
    });
}

async function sendMessage() {
    const input = document.getElementById('message-input');
    const message = input.value.trim();
    const imageInput = document.getElementById('image-input');
    
    if ((!message && !imageInput.files[0]) || !activeReceiverId) return;
    
    try {
        let imagePath = null;
        
        // رفع الصورة أولاً إن وجدت
        if (imageInput.files[0]) {
            const formData = new FormData();
            formData.append('image', imageInput.files[0]);
            
            const uploadRes = await fetch('api/upload_message_image.php', {
                method: 'POST',
                body: formData
            });
            
            const uploadData = await uploadRes.json();
            
            if (!uploadData.success) {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: uploadData.message,
                    background: 'var(--card-bg)',
                    color: 'var(--text-primary)',
                    confirmButtonColor: '#667eea'
                });
                return;
            }
            
            imagePath = uploadData.image_path;
        }
        
        // إرسال الرسالة
        const res = await fetch('api/send_message.php', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                receiver_id: activeReceiverId,
                message: message,
                image_path: imagePath
            })
        });
        
        const data = await res.json();
        
        if (data.success) {
            input.value = '';
            removeImagePreview();
            
            if (!activeConversationId) {
                activeConversationId = data.conversation_id;
            }
            
            loadMessages(activeConversationId);
            loadConversations();
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message,
                background: 'var(--card-bg)',
                color: 'var(--text-primary)',
                confirmButtonColor: '#667eea'
            });
        }
    } catch (error) {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ أثناء إرسال الرسالة',
            background: 'var(--card-bg)',
            color: 'var(--text-primary)',
            confirmButtonColor: '#667eea'
        });
    }
}

// New Chat Modal Functions
function openNewChatModal() {
    document.getElementById('newChatModal').style.display = 'block';
    loadUsers();
}

function closeNewChatModal() {
    document.getElementById('newChatModal').style.display = 'none';
    document.getElementById('user-search').value = '';
}

function loadUsers() {
    fetch('api/get_users.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            allUsers = data.users;
            renderUsers(allUsers);
        }
    });
}

function renderUsers(users) {
    const list = document.getElementById('users-list');
    
    if (users.length === 0) {
        list.innerHTML = `
            <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
                <i class="fas fa-users-slash" style="font-size: 40px; margin-bottom: 10px;"></i><br>
                لا يوجد مستخدمين
            </div>
        `;
    } else {
        list.innerHTML = users.map(user => {
            const escapedName = user.name.replace(/'/g, "\\'");
            const escapedImg = user.img.replace(/'/g, "\\'");
            const adminBadge = user.is_admin == 1 ? '<span style="background: linear-gradient(135deg, #f59e0b, #d97706); color: white; padding: 2px 8px; border-radius: 10px; font-size: 11px; margin-right: 5px; font-weight: 600;"><i class="fas fa-crown"></i> أدمن</span>' : '';
            return `
                <div onclick="startNewChat('${user.id}', '${escapedName}', '${escapedImg}')" 
                     style="padding: 15px; border-bottom: 1px solid var(--border-color); cursor: pointer; transition: all 0.3s ease; display: flex; gap: 15px; align-items: center;"
                     onmouseover="this.style.background='rgba(102, 126, 234, 0.1)'" 
                     onmouseout="this.style.background='transparent'">
                    <img src="${user.img}" alt="${user.name}" style="width: 45px; height: 45px; border-radius: 50%; object-fit: cover;">
                    <div style="flex: 1;">
                        <div style="font-weight: 700; color: var(--text-primary); font-family: 'Cairo', sans-serif; display: flex; align-items: center; gap: 8px;">
                            ${user.name}
                            ${adminBadge}
                        </div>
                    </div>
                </div>
            `;
        }).join('');
    }
}

function filterUsers(searchTerm) {
    const filtered = allUsers.filter(user => 
        user.name.toLowerCase().includes(searchTerm.toLowerCase())
    );
    renderUsers(filtered);
}

function startNewChat(userId, userName, userImg) {
    closeNewChatModal();
    activeConversationId = null; // New conversation
    activeReceiverId = userId;
    
    // Update UI
    document.getElementById('empty-chat').style.display = 'none';
    document.getElementById('active-chat').style.display = 'flex';
    document.getElementById('chat-avatar').src = userImg;
    document.getElementById('chat-name').textContent = userName;
    document.getElementById('chat-messages').innerHTML = `
        <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
            <i class="fas fa-comment-medical" style="font-size: 50px; margin-bottom: 15px; opacity: 0.3;"></i><br>
            ابدأ محادثة جديدة مع ${userName}
        </div>
    `;
    
    // Focus on input
    document.getElementById('message-input').focus();
}

// Send on Enter
document.addEventListener('DOMContentLoaded', function() {
    loadConversations();
    
    document.getElementById('message-input').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendMessage();
        }
    });
    
    // Auto refresh every 10 seconds
    setInterval(() => {
        if (activeConversationId) {
            loadMessages(activeConversationId);
        }
        loadConversations();
    }, 10000);
});

function formatTime(timestamp) {
    const now = new Date();
    const time = new Date(timestamp);
    const diff = Math.floor((now - time) / 1000);
    
    if (diff < 60) return 'الآن';
    if (diff < 3600) return `منذ ${Math.floor(diff / 60)} دقيقة`;
    if (diff < 86400) return `منذ ${Math.floor(diff / 3600)} ساعة`;
    
    return time.toLocaleDateString('ar-EG', { 
        day: 'numeric', 
        month: 'short', 
        hour: '2-digit', 
        minute: '2-digit' 
    });
}

// دالة معاينة الصورة قبل الرفع
function previewImage() {
    const input = document.getElementById('image-input');
    const preview = document.getElementById('image-preview');
    const previewContainer = document.getElementById('image-preview-container');
    
    if (input.files && input.files[0]) {
        const file = input.files[0];
        
        // التحقق من نوع الملف
        if (!file.type.startsWith('image/')) {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'يرجى اختيار ملف صورة فقط',
                background: 'var(--card-bg)',
                color: 'var(--text-primary)',
                confirmButtonColor: '#667eea'
            });
            input.value = '';
            return;
        }
        
        // التحقق من حجم الملف (5MB كحد أقصى)
        if (file.size > 5 * 1024 * 1024) {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'حجم الصورة كبير جداً. الحد الأقصى 5 ميجابايت',
                background: 'var(--card-bg)',
                color: 'var(--text-primary)',
                confirmButtonColor: '#667eea'
            });
            input.value = '';
            return;
        }
        
        const reader = new FileReader();
        reader.onload = function(e) {
            preview.src = e.target.result;
            previewContainer.style.display = 'block';
        };
        reader.readAsDataURL(file);
    }
}

// دالة إزالة الصورة من المعاينة
function removeImagePreview() {
    document.getElementById('image-input').value = '';
    document.getElementById('image-preview').src = '';
    document.getElementById('image-preview-container').style.display = 'none';
    selectedImagePath = null;
}

// دالة فتح modal الصورة
function openImageModal(imageSrc) {
    document.getElementById('modal-image').src = imageSrc;
    document.getElementById('imageModal').style.display = 'flex';
}

// دالة إغلاق modal الصورة
function closeImageModal() {
    document.getElementById('imageModal').style.display = 'none';
}
</script>

<?php include 'includes/footer.php'; ?>
