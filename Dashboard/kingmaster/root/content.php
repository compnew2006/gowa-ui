 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة المحتوى | Kingmaster";
$page_css = ['/css/rightnavbar.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<!-- Main Content -->
<main class="main-content">
  <!-- Header -->
  <div class="content-card" style="margin-bottom: 2rem;">
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
      <div>
        <h2 style="margin: 0 0 0.5rem 0;">
          <i class="fas fa-file-alt" style="color: #667eea;"></i> إدارة المحتوى
        </h2>
        <p style="margin: 0; color: var(--text-gray); font-size: 0.9rem;">أنشئ وعدل محتوى الرسائل الخاصة بك</p>
      </div>
      <button onclick="openCreateModal()" class="create-btn">
        <i class="fas fa-plus-circle" style=""></i>
        إنشاء محتوى جديد
      </button>
    </div>
  </div>

  <!-- Content Grid -->
  <div class="content-card">
    <h3 style="margin: 0 0 1.5rem 0;">
      <i class="fas fa-list" style="color: #f59e0b;"></i> قائمة المحتويات
    </h3>
    <div class="content-grid" id="content-grid">
      <!-- Content cards will be loaded here -->
      <div class="empty-state">
        <i class="fas fa-file-alt" style="font-size: 4rem; color: #667eea; opacity: 0.6;"></i>
        <p style="color: var(--text-gray); margin-top: 1rem;">لا يوجد محتوى حتى الآن</p>
        <button onclick="openCreateModal()" class="create-btn" style="margin-top: 1rem;">
          <i class="fas fa-plus-circle" style=""></i>
          إنشاء أول محتوى
        </button>
      </div>
    </div>
  </div>
</main>

<!-- Create/Edit Modal -->
<div class="modal" id="content-modal">
  <div class="modal-content">
    <div class="modal-header">
      <h3>
        <i class="fas fa-plus-circle" style="color: #667eea;"></i> 
        <span id="modal-title">إنشاء محتوى جديد</span>
      </h3>
      <button class="close-btn" onclick="closeContentModal()">
        <i class="fas fa-times" style="color: #ef4444;"></i>
      </button>
    </div>
    <div class="modal-body">
      <!-- Content Name -->
      <div class="form-group">
        <label>
          <i class="fas fa-signature" style="color: #8b5cf6;"></i> اسم المحتوى
        </label>
        <input type="text" id="content-name" class="form-input" placeholder="أدخل اسم المحتوى..." />
      </div>

      <!-- Content Text -->
      <div class="form-group">
        <label>
          <i class="fas fa-align-right" style="color: #10b981;"></i> المحتوى
        </label>
        <div class="textarea-wrapper">
          <button type="button" class="emoji-btn-inside" onclick="toggleEmojiPicker()" title="إضافة إيموجي">
            <i class="fas fa-smile" style="color: #f59e0b;"></i>
          </button>
          <div class="emoji-picker" id="emoji-picker" style="display: none;">
          <div class="emoji-grid">
            <span class="emoji-item" onclick="insertEmoji('😀')">😀</span>
            <span class="emoji-item" onclick="insertEmoji('😃')">😃</span>
            <span class="emoji-item" onclick="insertEmoji('😄')">😄</span>
            <span class="emoji-item" onclick="insertEmoji('😁')">😁</span>
            <span class="emoji-item" onclick="insertEmoji('😅')">😅</span>
            <span class="emoji-item" onclick="insertEmoji('😂')">😂</span>
            <span class="emoji-item" onclick="insertEmoji('🤣')">🤣</span>
            <span class="emoji-item" onclick="insertEmoji('😊')">😊</span>
            <span class="emoji-item" onclick="insertEmoji('😇')">😇</span>
            <span class="emoji-item" onclick="insertEmoji('🙂')">🙂</span>
            <span class="emoji-item" onclick="insertEmoji('🙃')">🙃</span>
            <span class="emoji-item" onclick="insertEmoji('😉')">😉</span>
            <span class="emoji-item" onclick="insertEmoji('😌')">😌</span>
            <span class="emoji-item" onclick="insertEmoji('😍')">😍</span>
            <span class="emoji-item" onclick="insertEmoji('🥰')">🥰</span>
            <span class="emoji-item" onclick="insertEmoji('😘')">😘</span>
            <span class="emoji-item" onclick="insertEmoji('😗')">😗</span>
            <span class="emoji-item" onclick="insertEmoji('😙')">😙</span>
            <span class="emoji-item" onclick="insertEmoji('😚')">😚</span>
            <span class="emoji-item" onclick="insertEmoji('😋')">😋</span>
            <span class="emoji-item" onclick="insertEmoji('😛')">😛</span>
            <span class="emoji-item" onclick="insertEmoji('😝')">😝</span>
            <span class="emoji-item" onclick="insertEmoji('😜')">😜</span>
            <span class="emoji-item" onclick="insertEmoji('🤪')">🤪</span>
            <span class="emoji-item" onclick="insertEmoji('🤨')">🤨</span>
            <span class="emoji-item" onclick="insertEmoji('🧐')">🧐</span>
            <span class="emoji-item" onclick="insertEmoji('🤓')">🤓</span>
            <span class="emoji-item" onclick="insertEmoji('😎')">😎</span>
            <span class="emoji-item" onclick="insertEmoji('🤩')">🤩</span>
            <span class="emoji-item" onclick="insertEmoji('🥳')">🥳</span>
            <span class="emoji-item" onclick="insertEmoji('😏')">😏</span>
            <span class="emoji-item" onclick="insertEmoji('😒')">😒</span>
            <span class="emoji-item" onclick="insertEmoji('😞')">😞</span>
            <span class="emoji-item" onclick="insertEmoji('😔')">😔</span>
            <span class="emoji-item" onclick="insertEmoji('😟')">😟</span>
            <span class="emoji-item" onclick="insertEmoji('😕')">😕</span>
            <span class="emoji-item" onclick="insertEmoji('🙁')">🙁</span>
            <span class="emoji-item" onclick="insertEmoji('☹️')">☹️</span>
            <span class="emoji-item" onclick="insertEmoji('😣')">😣</span>
            <span class="emoji-item" onclick="insertEmoji('😖')">😖</span>
            <span class="emoji-item" onclick="insertEmoji('😫')">😫</span>
            <span class="emoji-item" onclick="insertEmoji('😩')">😩</span>
            <span class="emoji-item" onclick="insertEmoji('🥺')">🥺</span>
            <span class="emoji-item" onclick="insertEmoji('😢')">😢</span>
            <span class="emoji-item" onclick="insertEmoji('😭')">😭</span>
            <span class="emoji-item" onclick="insertEmoji('😤')">😤</span>
            <span class="emoji-item" onclick="insertEmoji('😠')">😠</span>
            <span class="emoji-item" onclick="insertEmoji('😡')">😡</span>
            <span class="emoji-item" onclick="insertEmoji('🤬')">🤬</span>
            <span class="emoji-item" onclick="insertEmoji('🤯')">🤯</span>
            <span class="emoji-item" onclick="insertEmoji('😳')">😳</span>
            <span class="emoji-item" onclick="insertEmoji('🥵')">🥵</span>
            <span class="emoji-item" onclick="insertEmoji('🥶')">🥶</span>
            <span class="emoji-item" onclick="insertEmoji('😱')">😱</span>
            <span class="emoji-item" onclick="insertEmoji('😨')">😨</span>
            <span class="emoji-item" onclick="insertEmoji('😰')">😰</span>
            <span class="emoji-item" onclick="insertEmoji('😥')">😥</span>
            <span class="emoji-item" onclick="insertEmoji('😓')">😓</span>
            <span class="emoji-item" onclick="insertEmoji('🤗')">🤗</span>
            <span class="emoji-item" onclick="insertEmoji('🤔')">🤔</span>
            <span class="emoji-item" onclick="insertEmoji('🤭')">🤭</span>
            <span class="emoji-item" onclick="insertEmoji('🤫')">🤫</span>
            <span class="emoji-item" onclick="insertEmoji('🤥')">🤥</span>
            <span class="emoji-item" onclick="insertEmoji('😶')">😶</span>
            <span class="emoji-item" onclick="insertEmoji('😐')">😐</span>
            <span class="emoji-item" onclick="insertEmoji('😑')">😑</span>
            <span class="emoji-item" onclick="insertEmoji('😬')">😬</span>
            <span class="emoji-item" onclick="insertEmoji('🙄')">🙄</span>
            <span class="emoji-item" onclick="insertEmoji('😯')">😯</span>
            <span class="emoji-item" onclick="insertEmoji('😦')">😦</span>
            <span class="emoji-item" onclick="insertEmoji('😧')">😧</span>
            <span class="emoji-item" onclick="insertEmoji('😮')">😮</span>
            <span class="emoji-item" onclick="insertEmoji('😲')">😲</span>
            <span class="emoji-item" onclick="insertEmoji('🥱')">🥱</span>
            <span class="emoji-item" onclick="insertEmoji('😴')">😴</span>
            <span class="emoji-item" onclick="insertEmoji('🤤')">🤤</span>
            <span class="emoji-item" onclick="insertEmoji('😪')">😪</span>
            <span class="emoji-item" onclick="insertEmoji('😵')">😵</span>
            <span class="emoji-item" onclick="insertEmoji('🤐')">🤐</span>
            <span class="emoji-item" onclick="insertEmoji('🥴')">🥴</span>
            <span class="emoji-item" onclick="insertEmoji('🤢')">🤢</span>
            <span class="emoji-item" onclick="insertEmoji('🤮')">🤮</span>
            <span class="emoji-item" onclick="insertEmoji('🤧')">🤧</span>
            <span class="emoji-item" onclick="insertEmoji('😷')">😷</span>
            <span class="emoji-item" onclick="insertEmoji('🤒')">🤒</span>
            <span class="emoji-item" onclick="insertEmoji('🤕')">🤕</span>
            <span class="emoji-item" onclick="insertEmoji('🤑')">🤑</span>
            <span class="emoji-item" onclick="insertEmoji('🤠')">🤠</span>
            <span class="emoji-item" onclick="insertEmoji('👍')">👍</span>
            <span class="emoji-item" onclick="insertEmoji('👎')">👎</span>
            <span class="emoji-item" onclick="insertEmoji('👌')">👌</span>
            <span class="emoji-item" onclick="insertEmoji('✌️')">✌️</span>
            <span class="emoji-item" onclick="insertEmoji('🤞')">🤞</span>
            <span class="emoji-item" onclick="insertEmoji('🤟')">🤟</span>
            <span class="emoji-item" onclick="insertEmoji('🤘')">🤘</span>
            <span class="emoji-item" onclick="insertEmoji('🤙')">🤙</span>
            <span class="emoji-item" onclick="insertEmoji('👏')">👏</span>
            <span class="emoji-item" onclick="insertEmoji('🙌')">🙌</span>
            <span class="emoji-item" onclick="insertEmoji('👐')">👐</span>
            <span class="emoji-item" onclick="insertEmoji('🤲')">🤲</span>
            <span class="emoji-item" onclick="insertEmoji('🤝')">🤝</span>
            <span class="emoji-item" onclick="insertEmoji('🙏')">🙏</span>
            <span class="emoji-item" onclick="insertEmoji('💪')">💪</span>
            <span class="emoji-item" onclick="insertEmoji('❤️')">❤️</span>
            <span class="emoji-item" onclick="insertEmoji('🧡')">🧡</span>
            <span class="emoji-item" onclick="insertEmoji('💛')">💛</span>
            <span class="emoji-item" onclick="insertEmoji('💚')">💚</span>
            <span class="emoji-item" onclick="insertEmoji('💙')">💙</span>
            <span class="emoji-item" onclick="insertEmoji('💜')">💜</span>
            <span class="emoji-item" onclick="insertEmoji('🖤')">🖤</span>
            <span class="emoji-item" onclick="insertEmoji('🤍')">🤍</span>
            <span class="emoji-item" onclick="insertEmoji('🤎')">🤎</span>
            <span class="emoji-item" onclick="insertEmoji('💔')">💔</span>
            <span class="emoji-item" onclick="insertEmoji('❣️')">❣️</span>
            <span class="emoji-item" onclick="insertEmoji('💕')">💕</span>
            <span class="emoji-item" onclick="insertEmoji('💞')">💞</span>
            <span class="emoji-item" onclick="insertEmoji('💓')">💓</span>
            <span class="emoji-item" onclick="insertEmoji('💗')">💗</span>
            <span class="emoji-item" onclick="insertEmoji('💖')">💖</span>
            <span class="emoji-item" onclick="insertEmoji('💘')">💘</span>
            <span class="emoji-item" onclick="insertEmoji('💝')">💝</span>
            <span class="emoji-item" onclick="insertEmoji('🔥')">🔥</span>
            <span class="emoji-item" onclick="insertEmoji('✨')">✨</span>
            <span class="emoji-item" onclick="insertEmoji('⭐')">⭐</span>
            <span class="emoji-item" onclick="insertEmoji('🌟')">🌟</span>
            <span class="emoji-item" onclick="insertEmoji('💫')">💫</span>
            <span class="emoji-item" onclick="insertEmoji('🎉')">🎉</span>
            <span class="emoji-item" onclick="insertEmoji('🎊')">🎊</span>
            <span class="emoji-item" onclick="insertEmoji('🎈')">🎈</span>
            <span class="emoji-item" onclick="insertEmoji('🎁')">🎁</span>
            <span class="emoji-item" onclick="insertEmoji('🏆')">🏆</span>
            <span class="emoji-item" onclick="insertEmoji('🥇')">🥇</span>
            <span class="emoji-item" onclick="insertEmoji('🥈')">🥈</span>
            <span class="emoji-item" onclick="insertEmoji('🥉')">🥉</span>
          </div>
          </div>
          <textarea id="content-text" class="form-textarea" rows="8" placeholder="اكتب المحتوى هنا..."></textarea>
        </div>
        <div class="char-count">
          <span><i class="fas fa-font" style="color: #10b981;"></i> الأحرف: <strong id="char-count">0</strong></span>
          <span><i class="fas fa-text-width" style="color: #3b82f6;"></i> الكلمات: <strong id="word-count">0</strong></span>
        </div>
      </div>

      <!-- Variables Info -->
      <div class="info-box">
        <div style="margin-bottom: 0.5rem;">
          <i class="fas fa-lightbulb" style="color: #f59e0b;"></i>
          <strong>كلمات متغيرة:</strong>
        </div>
        <div class="variables-info">
          <p><code>{i|مرحبا|اهلا|هلا}</code> - اختيار عشوائي من الكلمات</p>
          <p><code>{%name%}</code> - اسم جهة الاتصال</p>
          <p><code>{%date%}</code> - التاريخ الحالي</p>
          <p><code>{%time%}</code> - الوقت الحالي</p>
          <p><code>{%random%}</code> - رقم عشوائي</p>
        </div>
      </div>

      <input type="hidden" id="content-id" value="">
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeContentModal()">إلغاء</button>
      <button class="btn-save" onclick="saveContent()">
        <i class="fas fa-save" style=""></i>
        حفظ المحتوى
      </button>
    </div>
  </div>
</div>

<script src="js/content.js"></script>

<?php include 'includes/footer.php'; ?>
