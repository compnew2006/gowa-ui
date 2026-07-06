<?php 
 $user_id = $_SESSION['user_id'] ; // مثال

  
$user = getUserByUserId($user_id);
 
$img = $user['img'];
if ($img == "0"){
  $img = "https://i.pravatar.cc/150?img=33";
 
}
?>

    
<!-- Actions Outside Navbar -->
<div class="nav-actions-container">
  <!-- Profile Icon -->
  <div class="action-icon" onclick="toggleDropdown('profile')">
    <img src="<?php echo $img; ?>" alt="Profile">
    <div class="dropdown" id="profile-dropdown">
      <div class="dropdown-header">الملف الشخصي</div>
      <div class="dropdown-content">
        <div class="dropdown-item" onclick="window.location.href='profile.php'" style="cursor: pointer;">
          <div class="dropdown-item-title"><i class="fas fa-user"></i> عرض الملف الشخصي</div>
        </div>
        <div class="dropdown-item" onclick="window.location.href='settings.php'" style="cursor: pointer;">
          <div class="dropdown-item-title"><i class="fas fa-cog" style=""></i> الإعدادات</div>
        </div>
        <div class="dropdown-item" onclick="window.location.href='logout.php'" style="cursor: pointer;">
          <div class="dropdown-item-title" style="color: #ef4444;"><i class="fas fa-sign-out-alt"></i> تسجيل الخروج</div>
        </div>
      </div>
    </div>
  </div>
  
  
  <!-- Messages Icon -->
  <div class="action-icon" onclick="toggleDropdown('messages')">
    <i class="fas fa-envelope"></i>
    <span class="badge" id="messages-badge">0</span>
    <div class="dropdown" id="messages-dropdown">
      <div class="dropdown-header">الرسائل</div>
      <div class="dropdown-content" id="messages-content">
        <div style="text-align: center; padding: 20px; color: var(--text-secondary);">
          <i class="fas fa-spinner"></i> جاري التحميل...
        </div>
      </div>
      <div class="dropdown-footer">
        <a href="messages.php">قراءة المزيد <i class="fas fa-arrow-left"></i></a>
      </div>
    </div>
  </div>
  
  <!-- Notifications Icon -->
  <div class="action-icon" onclick="toggleDropdown('notifications')">
    <i class="fas fa-bell"></i>
    <span class="badge" id="notifications-badge">0</span>
    <div class="dropdown" id="notifications-dropdown">
      <div class="dropdown-header">الإشعارات</div>
      <div class="dropdown-content" id="notifications-content">
        <div style="text-align: center; padding: 20px; color: var(--text-secondary);">
          <i class="fas fa-spinner"></i> جاري التحميل...
        </div>
      </div>
    </div>
  </div>

<style>
.dropdown-item.unread {
    background: rgba(102, 126, 234, 0.1);
    border-left: 3px solid #667eea;
}
.dropdown-item.read {
    opacity: 0.7;
}
</style>

<script>
let notificationsLoaded = false;

function loadNotifications() {
    if (notificationsLoaded) return;
    
    fetch('api/get_notifications.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            notificationsLoaded = true;
            
            // Update badge
            const badge = document.getElementById('notifications-badge');
            badge.textContent = data.unread_count;
            if (data.unread_count > 0) {
                badge.style.display = 'flex';
            } else {
                badge.style.display = 'none';
            }
            
            // Update notifications content
            const content = document.getElementById('notifications-content');
            
            if (data.notifications.length === 0) {
                content.innerHTML = `
                    <div style="text-align: center; padding: 20px; color: var(--text-secondary);">
                        <i class="fas fa-bell-slash" style="font-size: 30px; margin-bottom: 10px;"></i><br>
                        لا توجد إشعارات
                    </div>
                `;
            } else {
                content.innerHTML = data.notifications.map(notif => `
                    <div class="dropdown-item ${notif.is_read == 0 ? 'unread' : 'read'}" 
                         onclick="markAsRead(${notif.id})" 
                         style="cursor: pointer;">
                        <div class="dropdown-item-title">
                            ${notif.is_read == 0 ? '<i class="fas fa-circle" style="font-size: 8px; color: #667eea; margin-left: 5px;"></i>' : ''}
                            ${notif.title}
                        </div>
                        <div class="dropdown-item-text">${notif.message}</div>
                        <div class="dropdown-item-time">${formatTime(notif.created_at)}</div>
                    </div>
                `).join('');
            }
        }
    })
    .catch(err => {
        console.error('Error loading notifications:', err);
    });
}

function markAsRead(notificationId) {
    fetch('api/mark_notification_read.php', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ notification_id: notificationId })
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            // Reload notifications to update UI
            notificationsLoaded = false;
            loadNotifications();
        }
    });
}

function formatTime(timestamp) {
    const now = new Date();
    const time = new Date(timestamp);
    const diff = Math.floor((now - time) / 1000); // seconds
    
    if (diff < 60) return 'منذ لحظات';
    if (diff < 3600) return `منذ ${Math.floor(diff / 60)} دقيقة`;
    if (diff < 86400) return `منذ ${Math.floor(diff / 3600)} ساعة`;
    return `منذ ${Math.floor(diff / 86400)} يوم`;
}

// Load notifications on page load
document.addEventListener('DOMContentLoaded', loadNotifications);

// Reload every 30 seconds
setInterval(() => {
    notificationsLoaded = false;
    loadNotifications();
}, 30000);

// Messages functionality
let messagesLoaded = false;

function loadMessages() {
    if (messagesLoaded) return;
    
    fetch('api/get_conversations.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            messagesLoaded = true;
            
            // Update badge
            const badge = document.getElementById('messages-badge');
            badge.textContent = data.unread_count;
            if (data.unread_count > 0) {
                badge.style.display = 'flex';
            } else {
                badge.style.display = 'none';
            }
            
            // Update messages content
            const content = document.getElementById('messages-content');
            
            if (data.conversations.length === 0) {
                content.innerHTML = `
                    <div style="text-align: center; padding: 20px; color: var(--text-secondary);">
                        <i class="fas fa-inbox" style="font-size: 30px; margin-bottom: 10px;"></i><br>
                        لا توجد رسائل
                    </div>
                `;
            } else {
                content.innerHTML = data.conversations.map(conv => `
                    <div class="dropdown-item" onclick="window.location.href='messages.php'" style="cursor: pointer;">
                        <div class="dropdown-item-title">
                            ${conv.unread_count > 0 ? '<i class="fas fa-circle" style="font-size: 8px; color: #667eea; margin-left: 5px;"></i>' : ''}
                            ${conv.other_user_name}
                        </div>
                        <div class="dropdown-item-text">${conv.last_message || 'لا توجد رسائل'}</div>
                        <div class="dropdown-item-time">${formatTime(conv.last_message_time)}</div>
                    </div>
                `).join('');
            }
        }
    })
    .catch(err => {
        console.error('Error loading messages:', err);
    });
}

// Load messages on page load
document.addEventListener('DOMContentLoaded', loadMessages);

// Reload every 30 seconds
setInterval(() => {
    messagesLoaded = false;
    loadMessages();
}, 30000);
</script>


  
</div>
