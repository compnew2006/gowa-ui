 



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



$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إعدادات النقاط | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';
?>



<style>
    .settings-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }
    .settings-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
    }
    .settings-title {
        font-size: 28px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
    }
    .add-setting-btn {
        padding: 15px 30px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 12px;
        font-size: 16px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }
    .add-setting-btn:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 25px rgba(102, 126, 234, 0.4);
    }
    .settings-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 25px;
        margin-top: 30px;
    }
    .setting-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        box-shadow: 0 5px 20px rgba(0,0,0,0.1);
        transition: all 0.3s ease;
        border: 2px solid transparent;
    }
    .setting-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
        border-color: #667eea;
    }
    .setting-name {
        font-size: 20px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 15px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }
    .setting-info {
        display: flex;
        flex-direction: column;
        gap: 10px;
        margin-bottom: 20px;
    }
    .info-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px;
        background: var(--bg-primary);
        border-radius: 8px;
    }
    .info-label {
        font-weight: 600;
        color: var(--text-secondary);
        font-size: 14px;
    }
    .info-value {
        font-weight: 700;
        color: var(--text-primary);
        font-size: 16px;
    }
    .setting-actions {
        display: flex;
        gap: 10px;
    }
    .btn-edit, .btn-delete {
        flex: 1;
        padding: 12px;
        border: none;
        border-radius: 10px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
    }
    .btn-edit {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }
    .btn-delete {
        background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
        color: white;
    }
    .btn-edit:hover, .btn-delete:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(0,0,0,0.3);
    }
    /* Modal */
    .modal {
        display: none;
        position: fixed;
        z-index: 1000;
        left: 0;
        top: 0;
        width: 100%;
        height: 100%;
        background: rgba(0,0,0,0.7);
        backdrop-filter: blur(5px);
    }
    .modal-content {
        background: var(--card-bg);
        margin: 10% auto;
        padding: 30px;
        border-radius: 20px;
        width: 90%;
        max-width: 500px;
        animation: modalSlideIn 0.3s ease;
        font-family: 'Cairo', sans-serif;
        color: var(--text-primary);
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
    .modal-title {
        font-size: 24px;
        font-weight: 800;
        display: flex;
        align-items: center;
        gap: 12px;
    }
    .close-modal {
        font-size: 28px;
        font-weight: bold;
        color: var(--text-secondary);
        cursor: pointer;
        transition: all 0.3s ease;
    }
    .close-modal:hover {
        color: var(--primary-color);
        transform: rotate(90deg);
    }
    .form-group {
        margin-bottom: 20px;
    }
    .form-label {
        margin-bottom: 8px;
        display: block;
        font-weight: 600;
        font-size: 14px;
    }
    .form-input {
        width: 100%;
        padding: 12px 15px;
        font-size: 15px;
        border-radius: 10px;
        border: 2px solid var(--border-color);
        background: var(--bg-primary);
        color: var(--text-primary);
        transition: border-color 0.3s ease;
        box-sizing: border-box;
        font-family: 'Cairo', sans-serif;
    }
    .form-input:focus {
        border-color: #667eea;
        outline: none;
    }
    .submit-btn {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        padding: 15px;
        width: 100%;
        border: none;
        font-weight: 700;
        font-size: 16px;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
    }
    .submit-btn:hover {
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        transform: translateY(-2px);
    }
    .empty-state {
        text-align: center;
        padding: 60px 20px;
        color: var(--text-secondary);
    }
    .empty-state i {
        font-size: 64px;
        margin-bottom: 20px;
        opacity: 0.5;
    }
</style>

<div class="settings-container">
    <div class="settings-header">
        <div class="settings-title">
            <i class="fas fa-cog fa-spin" style="color: #667eea;"></i>
            إعدادات النقاط
        </div>
        <button class="add-setting-btn" onclick="openAddModal()">
            <i class="fas fa-plus-circle fa-bounce"></i>
            إضافة إعداد جديد
        </button>
    </div>

    <div class="settings-grid" id="settingsGrid">
        <!-- سيتم تحميل الإعدادات هنا -->
    </div>
</div>

<!-- Add/Edit Modal -->
<div id="settingModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title" id="modalTitle">
                <i class="fas fa-plus-circle" style="color: #667eea;"></i>
                <span id="modalTitleText">إضافة إعداد جديد</span>
            </div>
            <span class="close-modal" onclick="closeModal()">&times;</span>
        </div>
        <form id="settingForm" onsubmit="event.preventDefault(); saveSetting();">
            <input type="hidden" id="settingId">

            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-tag" style="color: #667eea;"></i>
                    اسم الإعداد
                </label>
                <input type="text" class="form-input" id="settingName" required placeholder="مثال: إعدادات النقاط الأساسية">
            </div>

            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-arrow-left" style="color: #10b981;"></i>
                    من (From)
                </label>
                <input type="number" class="form-input" id="settingFrom" required placeholder="مثال: 0" min="0">
            </div>

            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-arrow-right" style="color: #f59e0b;"></i>
                    إلى (To)
                </label>
                <input type="number" class="form-input" id="settingTo" required placeholder="مثال: 100" min="0">
            </div>

            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-hashtag" style="color: #8b5cf6;"></i>
                    العدد (Count)
                </label>
                <input type="number" class="form-input" id="settingCount" required placeholder="مثال: 10" min="0">
            </div>

            <button type="submit" class="submit-btn" id="submitBtn">
                <i class="fas fa-save"></i>
                <span id="submitBtnText">حفظ الإعداد</span>
            </button>
        </form>
    </div>
</div>

<script>
let editMode = false;

function openAddModal() {
    editMode = false;
    document.getElementById('settingId').value = '';
    document.getElementById('settingName').value = '';
    document.getElementById('settingFrom').value = '';
    document.getElementById('settingTo').value = '';
    document.getElementById('settingCount').value = '';

    document.getElementById('modalTitleText').textContent = 'إضافة إعداد جديد';
    document.getElementById('submitBtnText').textContent = 'حفظ الإعداد';
    document.getElementById('settingModal').style.display = 'block';
}

function openEditModal(setting) {
    editMode = true;
    document.getElementById('settingId').value = setting.id;
    document.getElementById('settingName').value = setting.name;
    document.getElementById('settingFrom').value = setting.from;
    document.getElementById('settingTo').value = setting.to;
    document.getElementById('settingCount').value = setting.count;

    document.getElementById('modalTitleText').textContent = 'تعديل الإعداد';
    document.getElementById('submitBtnText').textContent = 'حفظ التعديلات';
    document.getElementById('settingModal').style.display = 'block';
}

function closeModal() {
    document.getElementById('settingModal').style.display = 'none';
}

function loadSettings() {
    fetch('api/points_settings_api.php?action=get_all')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            renderSettings(data.settings);
        } else {
            Swal.fire('خطأ', data.message, 'error');
        }
    })
    .catch(() => Swal.fire('خطأ', 'حدث خطأ في تحميل الإعدادات', 'error'));
}

function renderSettings(settings) {
    const grid = document.getElementById('settingsGrid');
    
    if (settings.length === 0) {
        grid.innerHTML = `
            <div class="empty-state" style="grid-column: 1/-1;">
                <i class="fas fa-inbox"></i>
                <p style="font-size: 18px; font-weight: 600;">لا توجد إعدادات بعد</p>
                <p>ابدأ بإضافة إعداد جديد</p>
            </div>
        `;
        return;
    }

    grid.innerHTML = settings.map(setting => `
        <div class="setting-card">
            <div class="setting-name">
                <i class="fas fa-star" style="color: #f59e0b;"></i>
                ${setting.name}
            </div>
            <div class="setting-info">
                <div class="info-row">
                    <span class="info-label"><i class="fas fa-arrow-left" style="color: #10b981;"></i> من</span>
                    <span class="info-value">${setting.from}</span>
                </div>
                <div class="info-row">
                    <span class="info-label"><i class="fas fa-arrow-right" style="color: #f59e0b;"></i> إلى</span>
                    <span class="info-value">${setting.to}</span>
                </div>
                <div class="info-row">
                    <span class="info-label"><i class="fas fa-hashtag" style="color: #8b5cf6;"></i> العدد</span>
                    <span class="info-value">${setting.count}</span>
                </div>
            </div>
            <div class="setting-actions">
                <button class="btn-edit" onclick='openEditModal(${JSON.stringify(setting)})'>
                    <i class="fas fa-edit"></i>
                    تعديل
                </button>
                <button class="btn-delete" onclick="deleteSetting(${setting.id})">
                    <i class="fas fa-trash"></i>
                    حذف
                </button>
            </div>
        </div>
    `).join('');
}

function saveSetting() {
    const id = document.getElementById('settingId').value;
    const setting = {
        name: document.getElementById('settingName').value,
        from: parseInt(document.getElementById('settingFrom').value),
        to: parseInt(document.getElementById('settingTo').value),
        count: parseInt(document.getElementById('settingCount').value)
    };

    if (setting.from >= setting.to) {
        Swal.fire('خطأ', 'قيمة "من" يجب أن تكون أقل من قيمة "إلى"', 'error');
        return;
    }

    const action = id ? 'update' : 'add';
    if (id) setting.id = parseInt(id);

    fetch('api/points_settings_api.php', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({action, ...setting})
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم!',
                text: id ? 'تم تحديث الإعداد بنجاح' : 'تم إضافة الإعداد بنجاح',
                showConfirmButton: false,
                timer: 1500
            });
            closeModal();
            loadSettings();
        } else {
            Swal.fire('خطأ', data.message, 'error');
        }
    })
    .catch(() => Swal.fire('خطأ', 'فشل في حفظ الإعداد', 'error'));
}

function deleteSetting(id) {
    Swal.fire({
        title: 'تأكيد الحذف',
        text: 'هل أنت متأكد من حذف هذا الإعداد؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#ef4444',
        cancelButtonColor: '#6b7280'
    }).then(result => {
        if (result.isConfirmed) {
            fetch('api/points_settings_api.php', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({action: 'delete', id})
            })
            .then(res => res.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم الحذف!',
                        text: data.message,
                        showConfirmButton: false,
                        timer: 1500
                    });
                    loadSettings();
                } else {
                    Swal.fire('خطأ', data.message, 'error');
                }
            })
            .catch(() => Swal.fire('خطأ', 'فشل في حذف الإعداد', 'error'));
        }
    });
}

// إغلاق المودال بالنقر خارج المحتوى
window.onclick = function(event) {
    if (event.target.id === 'settingModal') {
        closeModal();
    }
}

// تحميل الإعدادات عند تحميل الصفحة
loadSettings();
</script>

<?php include 'includes/admin_footer.php'; ?>
