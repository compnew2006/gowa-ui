 

<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "المحفظة | Kingmaster";
$page_css = ['https://kingmaster.info/css/new.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
   
$commission = getcommission_walletsById($user_id);


?>

<!-- Main Content -->
<main class="main-content">
  <!-- Header -->
  <div class="content-card" style="margin-bottom: 2rem;">
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
      <div>
        <h2 style="margin: 0 0 0.5rem 0;">
          <i class="fab fa-whatsapp fa-beat" style="color: #25d366; --fa-animation-duration: 1.5s;"></i> محادثات WhatsApp
        </h2>
        <p style="margin: 0; color: var(--text-gray); font-size: 0.9rem;">عرض جميع جلسات ومحادثات الواتساب الخاصة بك</p>
      </div>
    </div>
  </div>

  <!-- WhatsApp Container -->
  <div class="content-card" style="padding: 0; overflow: hidden;">
    <div class="whatsapp-container">
      <!-- 1. قائمة الجلسات -->
      <div class="sessions-sidebar">
        <div class="sessions-header">
          <h2>
            <i class="fab fa-whatsapp"></i>
            الجلسات
          </h2>
        </div>
        <div class="sessions-list" id="sessionsList">
          <div class="loading">
            <i class="fas fa-spinner fa-spin"></i>
          </div>
        </div>
      </div>
      
      <!-- 2. قائمة المحادثات (الأشخاص) -->
      <div class="chats-sidebar">
        <div class="chats-header">
          <h2>
            <i class="fas fa-comments"></i>
            المحادثات
          </h2>
        </div>
        <div class="chats-list" id="chatsList">
          <div class="empty-state-wa">
            <i class="fas fa-comments"></i>
            <p>اختر جلسة لعرض المحادثات</p>
          </div>
        </div>
      </div>
      
      <!-- 3. منطقة الرسائل -->
      <div class="chat-area">
        <div id="emptyChat" class="empty-state-wa">
          <i class="fab fa-whatsapp"></i>
          <h3>WhatsApp Web</h3>
          <p>اختر محادثة لعرض الرسائل</p>
        </div>
        
        <div id="activeChat" style="display: none; height: 100%; display: flex; flex-direction: column;">
          <div class="chat-header">
            <div class="chat-avatar">
              <i class="fas fa-user"></i>
            </div>
            <div class="chat-info">
              <h3 id="chatTitle">محادثة</h3>
              <p id="chatSubtitle">جاري التحميل...</p>
            </div>
          </div>
          
          <div class="chat-messages" id="chatMessages">
            <!-- الرسائل ستظهر هنا -->
          </div>
        </div>
      </div>
    </div>
  </div>
</main>



<script>
let currentSession = null;
let currentContact = null;

// 1. تحميل الجلسات
async function loadSessions() {
    try {
        const response = await fetch('api/get_wa_sessions.php');
        const data = await response.json();
        
        const sessionsList = document.getElementById('sessionsList');
        
        if (data.success && data.sessions.length > 0) {
            sessionsList.innerHTML = data.sessions.map(session => `
                <div class="session-item" onclick="selectSession('${session.account_uid}', '${session.name}')">
                    <div class="session-info">
                        <div class="session-name">${session.name || 'جلسة واتساب'}</div>
                        <div class="session-status">
                            <span class="status-badge status-${session.status}"></span>
                            ${session.status === 'active' ? 'نشط' : 'غير نشط'}
                        </div>
                    </div>
                </div>
            `).join('');
        } else {
            sessionsList.innerHTML = `
                <div class="empty-state-wa">
                    <i class="fas fa-inbox"></i>
                    <p>لا توجد جلسات</p>
                </div>
            `;
        }
    } catch (error) {
        console.error('خطأ في تحميل الجلسات:', error);
    }
}

// 2. اختيار جلسة وتحميل المحادثات
async function selectSession(sessionId, sessionName) {
    currentSession = sessionId;
    currentContact = null;
    
    // تحديث الجلسة النشطة
    document.querySelectorAll('.session-item').forEach(item => {
        item.classList.remove('active');
    });
    event.currentTarget.classList.add('active');
    
    // إخفاء منطقة الرسائل
    document.getElementById('emptyChat').style.display = 'flex';
    document.getElementById('activeChat').style.display = 'none';
    
    // تحميل قائمة المحادثات
    await loadChats(sessionId);
}

// 3. تحميل قائمة المحادثات (الأشخاص)
async function loadChats(sessionId) {
    try {
        const chatsList = document.getElementById('chatsList');
        chatsList.innerHTML = '<div class="loading"><i class="fas fa-spinner fa-spin"></i></div>';
        
        const response = await fetch(`api/get_wa_chats.php?session_id=${sessionId}`);
        const data = await response.json();
        
        if (data.success && data.chats.length > 0) {
            chatsList.innerHTML = data.chats.map(chat => {
                const contact = chat.contact;
                const contactName = contact.replace('@c.us', '');
                const initial = contactName.charAt(0).toUpperCase();
                
                return `
                    <div class="chat-item" onclick="selectChat('${contact}', '${contactName}')">
                        <div class="chat-avatar-small">${initial}</div>
                        <div class="chat-item-info">
                            <div class="chat-item-name">${contactName}</div>
                            <div class="chat-item-last">${chat.last_message || 'لا توجد رسائل'}</div>
                        </div>
                        <div class="chat-item-count">${chat.message_count}</div>
                    </div>
                `;
            }).join('');
        } else {
            chatsList.innerHTML = `
                <div class="empty-state-wa">
                    <i class="fas fa-comments"></i>
                    <p>لا توجد محادثات</p>
                </div>
            `;
        }
    } catch (error) {
        console.error('خطأ في تحميل المحادثات:', error);
    }
}

// 4. اختيار محادثة وعرض الرسائل
async function selectChat(contact, contactName) {
    currentContact = contact;
    
    // تحديث المحادثة النشطة
    document.querySelectorAll('.chat-item').forEach(item => {
        item.classList.remove('active');
    });
    event.currentTarget.classList.add('active');
    
    // إظهار منطقة الرسائل
    document.getElementById('emptyChat').style.display = 'none';
    const activeChat = document.getElementById('activeChat');
    activeChat.style.display = 'flex';
    
    // تحديث معلومات الرأس
    const initial = contactName.charAt(0).toUpperCase();
    document.querySelector('.chat-avatar').textContent = initial;
    document.getElementById('chatTitle').textContent = contactName;
    document.getElementById('chatSubtitle').textContent = 'جاري تحميل الرسائل...';
    
    // تحميل الرسائل
    await loadMessages(currentSession, contact, contactName);
}

// 5. تحميل الرسائل
async function loadMessages(sessionId, contact, contactName) {
    try {
        const response = await fetch(`api/get_wa_conversations.php?session_id=${sessionId}&contact=${contact}`);
        const data = await response.json();
        
        const chatMessages = document.getElementById('chatMessages');
        
        if (data.success && data.conversations.length > 0) {
            document.getElementById('chatSubtitle').textContent = `${data.count} رسالة`;
            
            chatMessages.innerHTML = data.conversations.map(conv => {
                const isSent = conv.from_me === '1' || conv.from_me === 1 || conv.from_me === true;
                const time = formatTime(conv.created_at);
                
                return `
                    <div class="message ${isSent ? 'sent' : 'received'}">
                        <div class="message-content">
                            ${conv.txt || ''}
                            <div class="message-time">
                                ${time}
                                ${isSent ? '<i class="fas fa-check-double"></i>' : ''}
                            </div>
                        </div>
                    </div>
                `;
            }).join('');
            
            // التمرير إلى آخر رسالة
            chatMessages.scrollTop = chatMessages.scrollHeight;
        } else {
            chatMessages.innerHTML = `
                <div class="empty-state-wa">
                    <i class="fas fa-comments"></i>
                    <p>لا توجد رسائل</p>
                </div>
            `;
            document.getElementById('chatSubtitle').textContent = 'لا توجد رسائل';
        }
    } catch (error) {
        console.error('خطأ في تحميل الرسائل:', error);
    }
}

// تنسيق الوقت
function formatTime(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    if (diff < 86400000 && date.getDate() === now.getDate()) {
        return date.toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' });
    }
    
    if (diff < 172800000) {
        return 'أمس ' + date.toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' });
    }
    
    return date.toLocaleDateString('ar-EG', { 
        day: 'numeric', 
        month: 'short',
        hour: '2-digit', 
        minute: '2-digit' 
    });
}

// تحميل الجلسات عند فتح الصفحة
document.addEventListener('DOMContentLoaded', loadSessions);

// تحديث تلقائي كل 30 ثانية
setInterval(() => {
    if (currentSession && currentContact) {
        loadMessages(currentSession, currentContact, '');
    }
}, 30000);
</script>

<?php include 'includes/footer.php'; ?>
