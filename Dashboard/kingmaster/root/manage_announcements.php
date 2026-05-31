<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}

require_once 'includes/functions.php';

// التحقق من صلاحيات الأدمن
$is_admin = getUserIsAdmin($_SESSION['user_id']);
if (!$is_admin) {
    header('Location: index.php');
    exit;
}

// معالجة الإجراءات باستخدام POST/REDIRECT/GET - قبل أي output
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (isset($_POST['action'])) {
        $action = $_POST['action'];
        
        if ($action === 'create') {
            $title = $_POST['title'] ?? '';
            $announcement_message = $_POST['message'] ?? '';
            $is_active = isset($_POST['is_active']) ? 1 : 0;
            
            if (!empty($title) && !empty($announcement_message)) {
                createAnnouncement($title, $announcement_message, $is_active);
                $_SESSION['alert'] = ['type' => 'success', 'message' => 'تم إضافة الإعلان بنجاح!'];
            } else {
                $_SESSION['alert'] = ['type' => 'error', 'message' => 'يرجى ملء جميع الحقول!'];
            }
        } elseif ($action === 'update') {
            $id = $_POST['id'] ?? 0;
            $title = $_POST['title'] ?? '';
            $announcement_message = $_POST['message'] ?? '';
            $is_active = isset($_POST['is_active']) ? 1 : 0;
            
            if ($id && !empty($title) && !empty($announcement_message)) {
                updateAnnouncement($id, $title, $announcement_message, $is_active);
                $_SESSION['alert'] = ['type' => 'success', 'message' => 'تم تحديث الإعلان بنجاح!'];
            } else {
                $_SESSION['alert'] = ['type' => 'error', 'message' => 'خطأ في تحديث الإعلان!'];
            }
        } elseif ($action === 'delete') {
            $id = $_POST['id'] ?? 0;
            if ($id) {
                deleteAnnouncement($id);
                $_SESSION['alert'] = ['type' => 'success', 'message' => 'تم حذف الإعلان بنجاح!'];
            }
        }
    }
    
    // Redirect to prevent form resubmission
    header('Location: manage_announcements.php');
    exit;
}

$page_title = "إدارة الإعلانات | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';

// Get alert from session
$alert = null;
if (isset($_SESSION['alert'])) {
    $alert = $_SESSION['alert'];
    unset($_SESSION['alert']);
}

// جلب جميع الإعلانات
$announcements = getAllAnnouncements(false);
?>

<style>
    .announcements-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .page-header {
        margin-bottom: 30px;
    }

    .page-title {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
        margin-bottom: 10px;
    }

    .page-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: pulse 2s ease-in-out infinite;
    }

    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    /* Animated Icons */
    .bounce-icon {
        display: inline-block;
        animation: bounceAnim 2s ease-in-out infinite;
    }

    @keyframes bounceAnim {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-8px); }
    }

    .rotate-icon {
        display: inline-block;
        animation: rotateAnim 3s linear infinite;
    }

    @keyframes rotateAnim {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    .shake-icon {
        display: inline-block;
        animation: shakeAnim 2.5s ease-in-out infinite;
    }

    @keyframes shakeAnim {
        0%, 100% { transform: rotate(0deg); }
        25% { transform: rotate(-10deg); }
        75% { transform: rotate(10deg); }
    }

    .alert {
        padding: 15px 20px;
        border-radius: 12px;
        margin-bottom: 20px;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .alert.success {
        background: rgba(16, 185, 129, 0.1);
        color: #10b981;
        border: 1px solid rgba(16, 185, 129, 0.3);
    }

    .alert.error {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
        border: 1px solid rgba(239, 68, 68, 0.3);
    }

    .form-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
        margin-bottom: 30px;
    }

    .form-title {
        font-size: 20px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 20px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .form-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-label {
        display: block;
        font-size: 14px;
        font-weight: 600;
        color: var(--text-secondary);
        margin-bottom: 8px;
        font-family: 'Cairo', sans-serif;
    }

    .form-input,
    .form-textarea {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--bg-primary);
        color: var(--text-primary);
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .form-textarea {
        min-height: 120px;
        resize: vertical;
    }

    .form-input:focus,
    .form-textarea:focus {
        outline: none;
        border-color: #f093fb;
        box-shadow: 0 0 0 3px rgba(240, 147, 251, 0.15);
    }

    .form-checkbox {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 20px;
    }

    .form-checkbox input {
        width: 20px;
        height: 20px;
        cursor: pointer;
    }

    .form-checkbox label {
        font-size: 14px;
        font-weight: 600;
        color: var(--text-primary);
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
    }

    .btn {
        padding: 12px 24px;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
        transition: all 0.3s ease;
        display: inline-flex;
        align-items: center;
        gap: 8px;
    }

    .btn-primary {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }

    .btn-primary:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
    }

    .btn-edit {
        background: #3b82f6;
        color: white;
        padding: 8px 16px;
        font-size: 12px;
    }

    .btn-edit:hover {
        background: #2563eb;
    }

    .btn-delete {
        background: #ef4444;
        color: white;
        padding: 8px 16px;
        font-size: 12px;
    }

    .btn-delete:hover {
        background: #dc2626;
    }

    .announcements-list {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
    }

    .announcement-item {
        background: var(--bg-primary);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 15px;
        border: 1px solid var(--border-color);
        transition: all 0.3s ease;
    }

    .announcement-item:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.2);
    }

    .announcement-header {
        display: flex;
        justify-content: space-between;
        align-items: start;
        margin-bottom: 10px;
    }

    .announcement-title {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        margin-bottom: 5px;
    }

    .announcement-date {
        font-size: 12px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }

    .announcement-message {
        color: var(--text-secondary);
        font-size: 14px;
        line-height: 1.6;
        margin-bottom: 15px;
        font-family: 'Cairo', sans-serif;
    }

    .announcement-actions {
        display: flex;
        gap: 10px;
    }

    .status-badge {
        display: inline-block;
        padding: 4px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
    }

    .status-badge.active {
        background: rgba(16, 185, 129, 0.1);
        color: #10b981;
    }

    .status-badge.inactive {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
    }

    /* Modal Styles */
    .modal {
        display: none;
        position: fixed;
        z-index: 9999;
        left: 0;
        top: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        backdrop-filter: blur(5px);
    }

    .modal-content {
        background: var(--card-bg);
        margin: 5% auto;
        padding: 30px;
        border-radius: 20px;
        width: 90%;
        max-width: 600px;
        position: relative;
        animation: slideDown 0.3s ease;
    }

    @keyframes slideDown {
        from {
            transform: translateY(-50px);
            opacity: 0;
        }
        to {
            transform: translateY(0);
            opacity: 1;
        }
    }

    .close-modal {
        position: absolute;
        top: 15px;
        left: 15px;
        font-size: 28px;
        font-weight: bold;
        color: var(--text-secondary);
        cursor: pointer;
        transition: color 0.3s;
    }

    .close-modal:hover {
        color: #ef4444;
    }

    /* Light Theme */
    body.light-theme .form-card,
    body.light-theme .announcements-list,
    body.light-theme .modal-content {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .announcement-item {
        background: #f9fafb;
    }

    body.light-theme .form-input,
    body.light-theme .form-textarea {
        background: #ffffff;
        color: #2d3436;
        border-color: rgba(0, 0, 0, 0.15);
    }

    /* Responsive */
    @media (max-width: 768px) {
        .announcements-container {
            padding: 20px;
            margin-top: 100px;
        }

        .page-title {
            font-size: 24px;
        }

        .announcement-header {
            flex-direction: column;
        }

        .announcement-actions {
            margin-top: 10px;
        }
    }
</style>

<div class="announcements-container">
    <div class="page-header">
        <div class="page-title">
            <i class="fas fa-bullhorn"></i>
            إدارة الإعلانات
        </div>
    </div>


    <!-- Form to Add New Announcement -->
    <div class="form-card">
        <div class="form-title">
            <i class="fas fa-plus-circle bounce-icon" style="color: #10b981;"></i>
            إضافة إعلان جديد
        </div>
        <form method="POST" action="">
            <input type="hidden" name="action" value="create">
            
            <div class="form-group">
                <label class="form-label"><i class="fas fa-heading" style="color: #3b82f6;"></i> عنوان الإعلان</label>
                <input type="text" name="title" class="form-input" placeholder="مثال: عرض جديد - خصم 50%" required>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-align-right" style="color: #8b5cf6;"></i> محتوى الرسالة</label>
                <textarea name="message" class="form-textarea" placeholder="اكتب محتوى الإعلان هنا..." required></textarea>
            </div>

            <div class="form-checkbox">
                <input type="checkbox" name="is_active" id="is_active" checked>
                <label for="is_active"><i class="fas fa-toggle-on" style="color: #10b981;"></i> تفعيل الإعلان (ظهور للمستخدمين)</label>
            </div>

            <button type="submit" class="btn btn-primary">
                <i class="fas fa-save rotate-icon"></i>
                حفظ الإعلان
            </button>
        </form>
    </div>

    <!-- List of Announcements -->
    <div class="announcements-list">
        <div class="form-title">
            <i class="fas fa-list shake-icon" style="color: #f59e0b;"></i>
            جميع الإعلانات (<?php echo count($announcements); ?>)
        </div>

        <?php if (count($announcements) > 0): ?>
            <?php foreach ($announcements as $announcement): ?>
                <div class="announcement-item">
                    <div class="announcement-header">
                        <div>
                            <div class="announcement-title">
                                <?php echo htmlspecialchars($announcement['title']); ?>
                                <span class="status-badge <?php echo $announcement['is_active'] ? 'active' : 'inactive'; ?>">
                                    <?php echo $announcement['is_active'] ? 'نشط' : 'غير نشط'; ?>
                                </span>
                            </div>
                            <div class="announcement-date">
                                <i class="fas fa-calendar" style="color: #667eea;"></i>
                                <?php echo date('Y-m-d H:i', strtotime($announcement['created_at'])); ?>
                            </div>
                        </div>
                    </div>
                    <div class="announcement-message">
                        <?php echo nl2br(htmlspecialchars($announcement['message'])); ?>
                    </div>
                    <div class="announcement-actions">
                        <button class="btn btn-edit" onclick="editAnnouncement(<?php echo $announcement['id']; ?>, `<?php echo htmlspecialchars($announcement['title']); ?>`, `<?php echo htmlspecialchars($announcement['message']); ?>`, <?php echo $announcement['is_active']; ?>)">
                            <i class="fas fa-edit"></i>
                            تعديل
                        </button>
                        <button class="btn btn-delete" onclick="confirmDelete(<?php echo $announcement['id']; ?>)">
                            <i class="fas fa-trash"></i>
                            حذف
                        </button>
                    </div>
                </div>
            <?php endforeach; ?>
        <?php else: ?>
            <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
                <i class="fas fa-inbox bounce-icon" style="font-size: 48px; opacity: 0.3; margin-bottom: 15px; color: #667eea;"></i>
                <p>لا توجد إعلانات حتى الآن</p>
            </div>
        <?php endif; ?>
    </div>
</div>

<!-- Edit Modal -->
<div id="editModal" class="modal">
    <div class="modal-content">
        <span class="close-modal" onclick="closeEditModal()">&times;</span>
        <div class="form-title">
            <i class="fas fa-edit" style="color: #3b82f6;"></i>
            تعديل الإعلان
        </div>
        <form method="POST" action="">
            <input type="hidden" name="action" value="update">
            <input type="hidden" name="id" id="edit_id">
            
            <div class="form-group">
                <label class="form-label"><i class="fas fa-heading" style="color: #3b82f6;"></i> عنوان الإعلان</label>
                <input type="text" name="title" id="edit_title" class="form-input" required>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-align-right" style="color: #8b5cf6;"></i> محتوى الرسالة</label>
                <textarea name="message" id="edit_message" class="form-textarea" required></textarea>
            </div>

            <div class="form-checkbox">
                <input type="checkbox" name="is_active" id="edit_is_active">
                <label for="edit_is_active"><i class="fas fa-toggle-on" style="color: #10b981;"></i> تفعيل الإعلان</label>
            </div>

            <button type="submit" class="btn btn-primary">
                <i class="fas fa-save"></i>
                حفظ التعديلات
            </button>
        </form>
    </div>
</div>

<!-- SweetAlert2 JS -->
<script src="https://cdn.jsdelivr.net/npm/sweetalert2@11"></script>

<script>
// Show alert if exists
<?php if ($alert): ?>
Swal.fire({
    icon: '<?php echo $alert['type']; ?>',
    title: '<?php echo $alert['type'] === 'success' ? 'نجح!' : 'خطأ!'; ?>',
    text: '<?php echo addslashes($alert['message']); ?>',
    confirmButtonText: 'حسناً',
    confirmButtonColor: '#667eea',
    background: 'var(--card-bg)',
    color: 'var(--text-primary)'
});
<?php endif; ?>

function editAnnouncement(id, title, message, isActive) {
    document.getElementById('edit_id').value = id;
    document.getElementById('edit_title').value = title;
    document.getElementById('edit_message').value = message;
    document.getElementById('edit_is_active').checked = isActive == 1;
    document.getElementById('editModal').style.display = 'block';
}

function closeEditModal() {
    document.getElementById('editModal').style.display = 'none';
}

function confirmDelete(id) {
    Swal.fire({
        title: 'هل أنت متأكد؟',
        text: 'لن تتمكن من استرجاع هذا الإعلان!',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#ef4444',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، احذفه!',
        cancelButtonText: 'إلغاء',
        background: 'var(--card-bg)',
        color: 'var(--text-primary)'
    }).then((result) => {
        if (result.isConfirmed) {
            const form = document.createElement('form');
            form.method = 'POST';
            form.innerHTML = `
                <input type="hidden" name="action" value="delete">
                <input type="hidden" name="id" value="${id}">
            `;
            document.body.appendChild(form);
            form.submit();
        }
    });
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('editModal');
    if (event.target == modal) {
        closeEditModal();
    }
}
</script>

<?php include 'includes/admin_footer.php'; ?>
