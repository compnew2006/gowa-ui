 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة الملفات | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<!-- Main Content -->
<main class="main-content">
  <!-- Storage Info Card -->
  <div class="content-card" style="margin-bottom: 2rem;">
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
      <div>
        <h2 style="margin: 0 0 0.5rem 0;"><i class="fas fa-hdd" style="color: #667eea;"></i> مساحة التخزين</h2>
        <p style="margin: 0; color: var(--text-gray); font-size: 0.9rem;">إدارة ملفاتك المرفوعة</p>
      </div>
      <button onclick="openUploadModal()" class="upload-btn">
        <i class="fas fa-cloud-upload-alt"></i>
        رفع ملف جديد
      </button>
    </div>
    
    <!-- Storage Progress Bar -->
    <div style="margin-top: 1.5rem;">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
        <span style="color: var(--text-gray); font-size: 0.9rem;">المستخدم</span>
        <span style="color: var(--primary-color); font-weight: 600;" id="storage-text">0 MB / 100 MB</span>
      </div>
      <div class="storage-bar">
        <div class="storage-fill" id="storage-fill" style="width: 0%"></div>
      </div>
      <p style="margin: 0.5rem 0 0 0; color: var(--text-gray); font-size: 0.85rem;">
        <i class="fas fa-info-circle" style="color: #3b82f6;"></i> متبقي <span id="remaining-storage">100</span> ميجابايت
      </p>
    </div>
  </div>

  <!-- Files Grid -->
  <div class="content-card">
    <h3 style="margin: 0 0 1.5rem 0;"><i class="fas fa-folder-open" style="color: #f59e0b;"></i> الملفات المرفوعة</h3>
    <div class="files-grid" id="files-grid">
      <!-- Files will be loaded here dynamically -->
      <div class="empty-state">
        <i class="fas fa-folder-open" style="font-size: 4rem; color: #f59e0b; opacity: 0.5;"></i>
        <p style="color: var(--text-gray); margin-top: 1rem;">لا توجد ملفات حتى الآن</p>
        <button onclick="openUploadModal()" class="upload-btn" style="margin-top: 1rem;">
          <i class="fas fa-plus"></i>
          رفع أول ملف
        </button>
      </div>
    </div>
  </div>
</main>

<!-- Upload Modal -->
<div class="modal" id="upload-modal">
  <div class="modal-content">
    <div class="modal-header">
      <h3><i class="fas fa-cloud-upload-alt" style="color: #667eea;"></i> رفع ملف جديد</h3>
      <button class="close-btn" onclick="closeUploadModal()">
        <i class="fas fa-times"></i>
      </button>
    </div>
    <div class="modal-body">
      <div class="upload-area" id="upload-area" onclick="document.getElementById('file-input').click()">
        <i class="fas fa-cloud-upload-alt" style="font-size: 3rem; color: #667eea; margin-bottom: 1rem;"></i>
        <p style="color: var(--text-light); font-weight: 600; margin-bottom: 0.5rem;">اضغط لاختيار ملف</p>
        <p style="color: var(--text-gray); font-size: 0.85rem;">PDF, صور (JPG, PNG), فيديو (MP4 ,MP3, AVI)</p>
        <input type="file" id="file-input" accept=".pdf,.jpg,.jpeg,.png,.mp4,.mp3,.avi" style="display: none;" onchange="handleFileSelect(event)">
      </div>

      <!-- Preview Area -->
      <div class="preview-area" id="preview-area" style="display: none;">
        <div id="preview-content"></div>
      </div>

      <!-- File Name Input -->
      <div class="form-group" style="margin-top: 1.5rem;">
        <label style="display: block; margin-bottom: 0.5rem; color: var(--text-light); font-weight: 600;">
          <i class="fas fa-signature" style="color: #8b5cf6;"></i> اسم الملف
        </label>
        <input type="text" id="file-name-input" class="form-input" placeholder="أدخل اسم الملف..." />
      </div>

      <!-- File Info -->
      <div class="file-info" id="file-info" style="display: none; margin-top: 1rem;">
        <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
          <span style="color: var(--text-gray); font-size: 0.85rem;">
            <i class="fas fa-file" style="color: #10b981;"></i> الحجم: <strong id="file-size">0 KB</strong>
          </span>
          <span style="color: var(--text-gray); font-size: 0.85rem;">
            <i class="fas fa-file-alt" style="color: #3b82f6;"></i> النوع: <strong id="file-type">-</strong>
          </span>
        </div>
      </div>
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeUploadModal()">إلغاء</button>
      <button class="btn-upload" id="upload-btn" onclick="uploadFile()" disabled>
        <i class="fas fa-upload"></i>
        رفع الملف
      </button>
    </div>
  </div>
</div>

<!-- Edit Modal -->
<div class="modal" id="edit-modal">
  <div class="modal-content">
    <div class="modal-header">
      <h3><i class="fas fa-edit" style="color: #667eea;"></i> تعديل الملف</h3>
      <button class="close-btn" onclick="closeEditModal()">
        <i class="fas fa-times"></i>
      </button>
    </div>
    <div class="modal-body">
      <div class="form-group">
        <label style="display: block; margin-bottom: 0.5rem; color: var(--text-light); font-weight: 600;">
          <i class="fas fa-signature" style="color: #8b5cf6;"></i> اسم الملف
        </label>
        <input type="text" id="edit-file-name" class="form-input" />
      </div>
      <input type="hidden" id="edit-file-id">
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeEditModal()">إلغاء</button>
      <button class="btn-upload" onclick="saveEdit()">
        <i class="fas fa-save"></i>
        حفظ التغييرات
      </button>
    </div>
  </div>
</div>

<style>
  /* Upload Button */
  .upload-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.8rem 1.5rem;
    background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
    color: #fff;
    border: none;
    border-radius: 10px;
    font-weight: 600;
    font-size: 0.95rem;
    cursor: pointer;
    transition: all 0.3s ease;
    box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
  }

  .upload-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(102, 126, 234, 0.6);
  }

  /* Storage Bar */
  .storage-bar {
    width: 100%;
    height: 20px;
    background: rgba(102, 126, 234, 0.1);
    border-radius: 10px;
    overflow: hidden;
    border: 1px solid var(--border-color);
  }

  .storage-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--primary-color), var(--secondary-color));
    transition: width 0.5s ease;
    border-radius: 10px;
  }

  /* Files Grid */
  .files-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
    gap: 1.5rem;
    min-height: 200px;
  }

  .empty-state {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem;
  }

  /* File Card */
  .file-card {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 1rem;
    transition: all 0.3s ease;
    position: relative;
    overflow: hidden;
  }

  .file-card:hover {
    transform: translateY(-5px);
    box-shadow: 0 8px 20px rgba(102, 126, 234, 0.3);
  }

  .file-preview {
    width: 100%;
    height: 150px;
    border-radius: 8px;
    margin-bottom: 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(102, 126, 234, 0.1);
    overflow: hidden;
  }

  .file-preview img,
  .file-preview video {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .file-preview i {
    font-size: 3rem;
    color: var(--primary-color);
  }

  /* File type icon colors */
  .file-preview i.fa-file-pdf {
    color: #ef4444;
  }

  .file-preview i.fa-file {
    color: #6b7280;
  }

  .file-info-text {
    margin-bottom: 1rem;
  }

  .file-name {
    font-weight: 600;
    color: var(--text-light);
    margin-bottom: 0.3rem;
    font-size: 0.95rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-size {
    color: var(--text-gray);
    font-size: 0.8rem;
  }

  .file-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-edit,
  .btn-delete {
    flex: 1;
    padding: 0.6rem;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.3rem;
  }

  .btn-edit {
    background: rgba(102, 126, 234, 0.1);
    color: var(--primary-color);
    border: 1px solid rgba(102, 126, 234, 0.3);
  }

  .btn-edit:hover {
    background: rgba(102, 126, 234, 0.2);
  }

  .btn-delete {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .btn-delete:hover {
    background: rgba(239, 68, 68, 0.2);
  }

  /* Modal */
  .modal {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.7);
    backdrop-filter: blur(5px);
    z-index: 10000;
    align-items: center;
    justify-content: center;
  }

  .modal.active {
    display: flex;
  }

  .modal-content {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: 16px;
    width: 90%;
    max-width: 600px;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem;
    border-bottom: 1px solid var(--border-color);
  }

  .modal-header h3 {
    margin: 0;
    color: var(--text-light);
    font-size: 1.3rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    color: var(--text-gray);
    cursor: pointer;
    width: 35px;
    height: 35px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s ease;
  }

  .close-btn:hover {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  .modal-body {
    padding: 1.5rem;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    padding: 1.5rem;
    border-top: 1px solid var(--border-color);
  }

  /* Upload Area */
  .upload-area {
    border: 2px dashed var(--border-color);
    border-radius: 12px;
    padding: 2rem;
    text-align: center;
    cursor: pointer;
    transition: all 0.3s ease;
    background: rgba(102, 126, 234, 0.05);
  }

  .upload-area:hover {
    border-color: var(--primary-color);
    background: rgba(102, 126, 234, 0.1);
  }

  /* Preview Area */
  .preview-area {
    margin-top: 1rem;
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid var(--border-color);
  }

  .preview-area img,
  .preview-area video {
    width: 100%;
    max-height: 300px;
    object-fit: contain;
    background: rgba(0, 0, 0, 0.5);
  }

  .preview-area iframe {
    width: 100%;
    height: 300px;
    border: none;
  }

  /* Form Input */
  .form-input {
    width: 100%;
    padding: 0.8rem 1rem;
    border: 1px solid var(--border-color);
    border-radius: 8px;
    background: rgba(102, 126, 234, 0.05);
    color: var(--text-light);
    font-size: 0.95rem;
    transition: all 0.3s ease;
  }

  .form-input:focus {
    outline: none;
    border-color: var(--primary-color);
    background: rgba(102, 126, 234, 0.1);
  }

  /* Modal Buttons */
  .btn-cancel,
  .btn-upload {
    padding: 0.8rem 1.5rem;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
  }

  .btn-cancel {
    background: rgba(102, 126, 234, 0.1);
    color: var(--primary-color);
    border: 1px solid rgba(102, 126, 234, 0.3);
  }

  .btn-cancel:hover {
    background: rgba(102, 126, 234, 0.2);
  }

  .btn-upload {
    background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
    color: #fff;
    box-shadow: 0 4px 10px rgba(102, 126, 234, 0.3);
  }

  .btn-upload:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 6px 15px rgba(102, 126, 234, 0.5);
  }

  .btn-upload:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Light Theme */
  body.light-theme .file-name {
    color: #2d3436;
  }

  body.light-theme .modal-header h3 {
    color: #2d3436;
  }

  body.light-theme .file-card {
    background: #ffffff;
    border: 2px solid #e5e7eb;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }

  body.light-theme .file-card:hover {
    border-color: #667eea;
    box-shadow: 0 8px 20px rgba(102, 126, 234, 0.2);
  }

  body.light-theme .file-preview {
    background: #f3f4f6;
    border: 1px solid #e5e7eb;
  }

  body.light-theme .file-size {
    color: #6b7280;
  }

  body.light-theme .btn-edit {
    background: rgba(102, 126, 234, 0.1);
    color: #667eea;
    border: 2px solid rgba(102, 126, 234, 0.4);
  }

  body.light-theme .btn-edit:hover {
    background: rgba(102, 126, 234, 0.2);
    border-color: #667eea;
  }

  body.light-theme .btn-delete {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 2px solid rgba(239, 68, 68, 0.4);
  }

  body.light-theme .btn-delete:hover {
    background: rgba(239, 68, 68, 0.2);
    border-color: #ef4444;
  }

  /* Mobile tuning */
  @media (max-width: 768px){
    .upload-btn{
      width: 100%;
      justify-content: center;
    }
    .files-grid{
      grid-template-columns: 1fr;
      gap: 1rem;
    }
    .empty-state{
      padding: 1.5rem;
      text-align: center;
    }
    .modal-footer{
      flex-direction: column;
    }
    .btn-cancel, .btn-upload{
      width: 100%;
      justify-content: center;
    }
  }

  @media (max-width: 420px){
    .file-actions{ flex-direction: column; }
  }
</style>

<script src="js/files.js"></script>

<?php include 'includes/footer.php'; ?>
