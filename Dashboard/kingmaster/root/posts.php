<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة الكوبونات | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';
?>


<style>
    .posts-container {
        padding: 30px;
        max-width: 1600px;
        margin: 120px auto 0 auto;
    }

    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
        flex-wrap: wrap;
        gap: 20px;
    }

    .page-title {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .page-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: bounce 2s ease-in-out infinite;
    }

    @keyframes bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    .add-post-btn {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 12px 30px;
        border-radius: 12px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
    }

    .add-post-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    /* Posts Grid */
    .posts-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 25px;
        margin-bottom: 30px;
    }

    .loading {
        grid-column: 1 / -1;
        text-align: center;
        padding: 50px;
        font-size: 18px;
        font-weight: 700;
        color: var(--text-secondary);
    }

    .loading i {
        font-size: 32px;
        color: #667eea;
        margin-bottom: 15px;
        display: block;
    }

    /* Post Card */
    .post-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        border: 2px solid var(--border-color);
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .post-card::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 4px;
        background: linear-gradient(90deg, #667eea, #764ba2, #f093fb);
    }

    .post-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 15px 35px rgba(102, 126, 234, 0.3);
        border-color: #667eea;
    }

    .post-card h3 {
        font-size: 20px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 12px;
        font-family: 'Cairo', sans-serif;
    }

    .post-type {
        display: inline-block;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 13px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        margin-bottom: 12px;
    }

    .type-new-feature {
        background: rgba(59, 130, 246, 0.1);
        color: #3b82f6;
    }

    .type-system-update {
        background: rgba(16, 185, 129, 0.1);
        color: #10b981;
    }

    .type-maintenance {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
    }

    .post-card p {
        color: var(--text-secondary);
        line-height: 1.8;
        margin-bottom: 20px;
        flex-grow: 1;
        font-family: 'Cairo', sans-serif;
    }

    .post-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-top: 15px;
        border-top: 2px solid var(--border-color);
    }

    .post-date {
        color: var(--text-secondary);
        font-size: 14px;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 6px;
        font-family: 'Cairo', sans-serif;
    }

    .post-actions {
        display: flex;
        gap: 10px;
    }

    .btn-edit,
    .btn-delete {
        padding: 8px 12px;
        border: none;
        border-radius: 8px;
        cursor: pointer;
        font-weight: 600;
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 6px;
        transition: all 0.3s ease;
    }

    .btn-edit {
        background: rgba(34, 197, 94, 0.1);
        color: #16a34a;
    }

    .btn-edit:hover {
        background: #16a34a;
        color: white;
        transform: translateY(-2px);
    }

    .btn-delete {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
    }

    .btn-delete:hover {
        background: #dc2626;
        color: white;
        transform: translateY(-2px);
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
        z-index: 9999;
        justify-content: center;
        align-items: center;
        backdrop-filter: blur(5px);
    }

    .modal.show {
        display: flex;
    }

    .modal-content {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        max-width: 600px;
        width: 90%;
        max-height: 90vh;
        overflow-y: auto;
        border: 2px solid var(--border-color);
        animation: modalSlideIn 0.3s ease;
    }

    @keyframes modalSlideIn {
        from {
            opacity: 0;
            transform: translateY(-50px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 25px;
        padding-bottom: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .modal-header h2 {
        font-size: 24px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .btn-close {
        background: none;
        border: none;
        font-size: 28px;
        color: var(--text-secondary);
        cursor: pointer;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 50%;
        transition: all 0.3s ease;
    }

    .btn-close:hover {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
        transform: rotate(90deg);
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-weight: 600;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        font-size: 15px;
    }

    .form-group input,
    .form-group textarea,
    .form-group select {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: #1e293b;
        color: #f1f5f9;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .form-group input:focus,
    .form-group textarea:focus,
    .form-group select:focus {
        outline: none;
        border-color: #667eea;
    }

    .form-group select option {
        background: #1e293b;
        color: #f1f5f9;
    }

    .form-group textarea {
        resize: vertical;
        min-height: 120px;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 25px;
        padding-top: 20px;
        border-top: 2px solid var(--border-color);
    }

    .btn-save,
    .btn-cancel {
        padding: 12px 30px;
        border: none;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .btn-save {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }

    .btn-save:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    .btn-cancel {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
    }

    .btn-cancel:hover {
        background: #dc2626;
        color: white;
    }

    .error-message,
    .success-message {
        padding: 12px 16px;
        border-radius: 12px;
        margin-bottom: 15px;
        font-weight: 700;
        font-size: 14px;
        display: none;
        font-family: 'Cairo', sans-serif;
    }

    .error-message {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
        border: 2px solid rgba(239, 68, 68, 0.2);
    }

    .success-message {
        background: rgba(16, 185, 129, 0.1);
        color: #10b981;
        border: 2px solid rgba(16, 185, 129, 0.2);
    }

    .error-message.show,
    .success-message.show {
        display: block;
    }

    /* Light Theme */
    body.light-theme .form-group input,
    body.light-theme .form-group textarea {
        background: #ffffff;
        color: #2d3436;
    }

    body.light-theme .post-card,
    body.light-theme .modal-content {
        background: rgba(255, 255, 255, 0.95);
    }

    /* Responsive */
    @media (max-width: 768px) {
        .posts-container {
            padding: 20px;
            margin-top: 100px;
        }

        .posts-grid {
            grid-template-columns: 1fr;
        }

        .modal-content {
            width: 95%;
            margin: 20px;
        }
    }
</style>

<div class="posts-container">
    <div class="page-header">
        <div class="page-title">
            <i class="fas fa-newspaper"></i>
            إدارة النشرات
        </div>
        <button class="add-post-btn" id="addPostBtn">
            <i class="fas fa-plus"></i>
            أضف نشرة
        </button>
    </div>

    <div class="posts-grid" id="postsGrid">
        <div class="loading">
            <i class="fas fa-spinner fa-spin"></i>
            جاري التحميل...
        </div>
    </div>
</div>

    <!-- Modal إضافة/تعديل نشرة -->
    <div class="modal" id="postModal">
        <div class="modal-content">
            <div class="modal-header">
                <h2 id="modalTitle">
                    <i class="fas fa-plus-circle"></i>
                    إضافة نشرة جديدة
                </h2>
                <button class="btn-close" id="closeModal">
                    <i class="fas fa-times"></i>
                </button>
            </div>

            <form id="postForm">
                <input type="hidden" id="postId" name="post_id">
                
                <div class="error-message" id="errorMessage"></div>
                <div class="success-message" id="successMessage"></div>

                <div class="form-group">
                    <label for="title">
                        <i class="fas fa-heading"></i>
                        عنوان النشرة
                    </label>
                    <input type="text" id="title" name="title" placeholder="أدخل عنوان النشرة" required>
                </div>

                <div class="form-group">
                    <label for="type">
                        <i class="fas fa-tag"></i>
                        نوع المنشور
                    </label>
                    <select id="type" name="type" class="form-select" required>
                        <option value="">اختر النوع</option>
                        <option value="New Feature">ميزة جديدة</option>
                        <option value="System Update">تحديث نظام</option>
                        <option value="Maintenance">صيانة</option>
                    </select>
                </div>

                <div class="form-group">
                    <label for="description">
                        <i class="fas fa-align-left"></i>
                        الوصف
                    </label>
                    <textarea id="description" name="description" rows="6" placeholder="أدخل وصف النشرة" required></textarea>
                </div>

                <div class="modal-footer">
                    <button type="button" class="btn-cancel" id="cancelBtn">
                        <i class="fas fa-times"></i>
                        إلغاء
                    </button>
                    <button type="submit" class="btn-save">
                        <i class="fas fa-save"></i>
                        حفظ النشرة
                    </button>
                </div>
            </form>
        </div>
    </div>

    <script src="assets/js/posts.js"></script>
    
<?php include 'includes/admin_footer.php'; ?>
