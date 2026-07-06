<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إعدادات الإرسال | Kingmaster";
$page_css = ['/css/toppages.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<div class="settings-container">
    <div class="settings-header">
        <div class="settings-title">
            <i class="fas fa-cog"></i>
            إعدادات الإرسال
        </div>
        <button class="add-settings-btn" onclick="openAddModal()">
            <i class="fas fa-plus"></i>
            إضافة إعدادات
        </button>
    </div>

    <div id="settingsGrid" class="settings-grid">
        <!-- سيتم تحميل الإعدادات هنا -->
    </div>
</div>

<!-- Add Settings Modal -->
<div id="addSettingsModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title sendingsetting-modal">
                <i class="fas fa-cog"></i>
                إضافة إعدادات إرسال جديدة
            </div>
            <span class="close-modal" onclick="closeModal('addSettingsModal')">&times;</span>
        </div>
        <form id="settingsForm" onsubmit="event.preventDefault(); saveSettings();">
            <div class="form-group">
                <label class="form-label">المنصة</label>
                <select class="form-select" id="platform" required>
                    <option value="">اختر المنصة</option>
                    <option value="facebook">فيسبوك</option>
                    <option value="whatsapp">واتساب</option>
                    <option value="instagram">انستجرام</option>
                    <option value="telegram">تليجرام</option>
                    <option value="email">بريد إلكتروني</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label">اسم الإعدادات</label>
                <input type="text" class="form-input" id="settingsName" placeholder="أدخل اسم الإعدادات" required>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">فاصل زمني من (ثانية)</label>
                    <input type="number" class="form-input" id="intervalFrom" placeholder="مثال: 5" min="1" required>
                </div>
                <div class="form-group">
                    <label class="form-label">فاصل زمني إلى (ثانية)</label>
                    <input type="number" class="form-input" id="intervalTo" placeholder="مثال: 10" min="1" required>
                </div>
            </div>

            <!-- Protection Settings -->
            <div class="checkbox-wrapper">
                <label class="checkbox-container">
                    <input type="checkbox" class="checkbox-input" id="protectionEnabled" onchange="toggleProtection()">
                    <span class="checkbox-custom"></span>
                    <span class="checkbox-label"><i class="fas fa-shield-alt"></i> إعدادات الحماية</span>
                </label>
                
                <div id="protectionFields" class="conditional-fields">
                    <div class="form-group">
                        <label class="form-label">عدد الرسائل</label>
                        <input type="number" class="form-input" id="msgCount" placeholder="مثال: 20" min="1">
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">فاصل زمني من (دقيقة)</label>
                            <input type="number" class="form-input" id="protectionIntervalFrom" placeholder="مثال: 5" min="1">
                        </div>
                        <div class="form-group">
                            <label class="form-label">فاصل زمني إلى (دقيقة)</label>
                            <input type="number" class="form-input" id="protectionIntervalTo" placeholder="مثال: 10" min="1">
                        </div>
                    </div>
                </div>
            </div>

            <!-- Blacklist Settings -->
            <div class="checkbox-wrapper">
                <label class="checkbox-container">
                    <input type="checkbox" class="checkbox-input" id="blacklistEnabled" onchange="toggleBlacklist()">
                    <span class="checkbox-custom"></span>
                    <span class="checkbox-label"><i class="fas fa-ban"></i> القائمة السوداء</span>
                </label>
                
                <div id="blacklistFields" class="conditional-fields">
                    <div class="form-group">
                        <label class="form-label">الأرقام المحظورة</label>
                        <textarea class="form-input" id="blacklist" rows="6" placeholder="أدخل الأرقام، كل رقم في سطر منفصل&#10;مثال:&#10;01234567890&#10;01098765432&#10;01155555555"></textarea>
                        <small style="display: block; margin-top: 5px; color: var(--text-secondary); font-size: 12px; font-family: 'Cairo', sans-serif;">يمكنك إدخال عدة أرقام، واحد في كل سطر</small>
                    </div>
                </div>
            </div>

            <button type="submit" class="submit-btn">
                <i class="fas fa-save"></i>
                حفظ الإعدادات
            </button>
        </form>
    </div>
</div>

<script>
function openAddModal() {
    document.getElementById('addSettingsModal').style.display = 'block';
}

function closeModal(modalId) {
    document.getElementById(modalId).style.display = 'none';
}

function toggleProtection() {
    const checkbox = document.getElementById('protectionEnabled');
    const fields = document.getElementById('protectionFields');
    
    if (checkbox.checked) {
        fields.classList.add('show');
    } else {
        fields.classList.remove('show');
    }
}

function toggleBlacklist() {
    const checkbox = document.getElementById('blacklistEnabled');
    const fields = document.getElementById('blacklistFields');
    
    if (checkbox.checked) {
        fields.classList.add('show');
    } else {
        fields.classList.remove('show');
    }
}

function saveSettings() {
    // تحويل القائمة السوداء إلى JSON
    let blacklistArray = [];
    const blacklistText = document.getElementById('blacklist').value;
    if (blacklistText && blacklistText.trim()) {
        blacklistArray = blacklistText.trim().split('\n')
            .map(num => num.trim())
            .filter(num => num !== '');
    }
    
    // التحقق من وضع التعديل
    const editId = document.getElementById('settingsForm').getAttribute('data-edit-id');
    const isEditing = editId !== null;
    
    const formData = {
        platform: document.getElementById('platform').value,
        settingsName: document.getElementById('settingsName').value,
        intervalFrom: document.getElementById('intervalFrom').value,
        intervalTo: document.getElementById('intervalTo').value,
        protectionEnabled: document.getElementById('protectionEnabled').checked ? 1 : 0,
        msgCount: document.getElementById('msgCount').value || null,
        protectionIntervalFrom: document.getElementById('protectionIntervalFrom').value || null,
        protectionIntervalTo: document.getElementById('protectionIntervalTo').value || null,
        blacklistEnabled: document.getElementById('blacklistEnabled').checked ? 1 : 0,
        blacklist: blacklistArray.length > 0 ? JSON.stringify(blacklistArray) : null
    };
    
    // إضافة ID في حالة التعديل
    if (isEditing) {
        formData.id = editId;
    }
    
    const apiUrl = isEditing ? 'api/update_sending_settings.php' : 'api/save_sending_settings.php';
    
    fetch(apiUrl, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(formData)
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            const isEditing = document.getElementById('settingsForm').getAttribute('data-edit-id') !== null;
            Swal.fire({
                icon: 'success',
                title: 'تم!',
                text: isEditing ? 'تم تحديث الأعدادات بنجاح' : 'تم حفظ الإعدادات بنجاح',
                timer: 2000,
                showConfirmButton: false
            });
            closeModal('addSettingsModal');
            resetForm();
            loadSettings();
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message || 'حدث خطأ أثناء الحفظ'
            });
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال'
        });
    });
}

function loadSettings() {
    fetch('api/get_sending_settings.php')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            renderSettings(data.settings);
        }
    })
    .catch(error => {
        console.error('Error loading settings:', error);
    });
}

function renderSettings(settings) {
    const grid = document.getElementById('settingsGrid');
    
    if (settings.length === 0) {
        grid.innerHTML = `
            <div style="grid-column: 1/-1; text-align: center; padding: 60px 20px;">
                <i class="fas fa-cog" style="font-size: 80px; color: #667eea; margin-bottom: 20px; opacity: 0.5;"></i>
                <h3 style="color: var(--text-primary); font-family: 'Cairo', sans-serif;">لا توجد إعدادات</h3>
                <p style="color: var(--text-secondary); font-family: 'Cairo', sans-serif;">ابدأ بإضافة إعدادات إرسال جديدة</p>
            </div>
        `;
        return;
    }
    
    grid.innerHTML = settings.map(setting => `
        <div class="settings-card">
            <div class="card-header">
                <div class="card-name">${setting.settings_name}</div>
                <span class="platform-badge platform-${setting.platform}">
                    ${getPlatformIcon(setting.platform)} ${getPlatformName(setting.platform)}
                </span>
            </div>
            <div class="card-info">
                <div class="info-row">
                    <span class="info-label">الفاصل الزمني</span>
                    <span class="info-value">${setting.interval_from} - ${setting.interval_to} ثانية</span>
                </div>
                <div class="info-row">
                    <span class="info-label">إعدادات الحماية</span>
                    <span class="status-badge ${setting.protection_enabled ? 'status-active' : 'status-inactive'}">
                        ${setting.protection_enabled ? 'مفعّل' : 'غير مفعّل'}
                    </span>
                </div>
                ${setting.protection_enabled ? `
                <div class="info-row">
                    <span class="info-label">عدد الرسائل</span>
                    <span class="info-value">${setting.msg_count} رسالة</span>
                </div>
                <div class="info-row">
                    <span class="info-label">فاصل الحماية</span>
                    <span class="info-value">${setting.protection_interval_from} - ${setting.protection_interval_to} دقيقة</span>
                </div>
                ` : ''}
                <div class="info-row">
                    <span class="info-label">القائمة السوداء</span>
                    <span class="status-badge ${setting.blacklist_enabled ? 'status-active' : 'status-inactive'}">
                        ${setting.blacklist_enabled ? 'مفعّل' : 'غير مفعّل'}
                    </span>
                </div>
                ${setting.blacklist_enabled && setting.blacklist ? `
                <div class="info-row">
                    <span class="info-label">عدد الأرقام المحظورة</span>
                    <span class="info-value">${getBlacklistCount(setting.blacklist)} رقم</span>
                </div>
                ` : ''}
            </div>
            <div class="card-actions">
                <button class="action-btn btn-edit" onclick="editSettings(${setting.id})">
                    <i class="fas fa-edit"></i>
                    تعديل
                </button>
                <button class="action-btn btn-delete" onclick="deleteSettings(${setting.id})">
                    <i class="fas fa-trash"></i>
                    حذف
                </button>
            </div>
        </div>
    `).join('');
}

function getPlatformIcon(platform) {
    const icons = {
        facebook: '<i class="fab fa-facebook-f"></i>',
        whatsapp: '<i class="fab fa-whatsapp"></i>',
        instagram: '<i class="fab fa-instagram"></i>',
        telegram: '<i class="fab fa-telegram-plane"></i>',
        email: '<i class="fas fa-envelope"></i>'
    };
    return icons[platform] || '';
}

function getPlatformName(platform) {
    const names = {
        facebook: 'فيسبوك',
        whatsapp: 'واتساب',
        instagram: 'انستجرام',
        telegram: 'تليجرام',
        email: 'بريد إلكتروني'
    };
    return names[platform] || platform;
}

function getBlacklistCount(blacklist) {
    try {
        const parsed = JSON.parse(blacklist);
        return Array.isArray(parsed) ? parsed.length : 0;
    } catch (e) {
        return 0;
    }
}

function deleteSettings(id) {
    Swal.fire({
        title: 'تأكيد الحذف',
        text: 'هل أنت متأكد من حذف هذه الإعدادات؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#ef4444'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('api/delete_sending_settings.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ id: id })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم الحذف!',
                        timer: 1500,
                        showConfirmButton: false
                    });
                    loadSettings();
                }
            });
        }
    });
}

function editSettings(id) {
    // جلب بيانات الإعدادات
    fetch(`api/get_single_setting.php?id=${id}`)
    .then(response => response.json())
    .then(data => {
        if (data.success && data.setting) {
            const setting = data.setting;
            
            // ملء الحقول في المودال
            document.getElementById('platform').value = setting.platform;
            document.getElementById('settingsName').value = setting.settings_name;
            document.getElementById('intervalFrom').value = setting.interval_from;
            document.getElementById('intervalTo').value = setting.interval_to;
            
            // إعدادات الحماية
            document.getElementById('protectionEnabled').checked = setting.protection_enabled == 1;
            if (setting.protection_enabled == 1) {
                document.getElementById('protectionFields').classList.add('show');
                document.getElementById('msgCount').value = setting.msg_count || '';
                document.getElementById('protectionIntervalFrom').value = setting.protection_interval_from || '';
                document.getElementById('protectionIntervalTo').value = setting.protection_interval_to || '';
            }
            
            // القائمة السوداء
            document.getElementById('blacklistEnabled').checked = setting.blacklist_enabled == 1;
            if (setting.blacklist_enabled == 1) {
                document.getElementById('blacklistFields').classList.add('show');
                // تحويل JSON إلى نص (كل رقم في سطر)
                if (setting.blacklist) {
                    try {
                        const blacklistArray = JSON.parse(setting.blacklist);
                        document.getElementById('blacklist').value = blacklistArray.join('\n');
                    } catch (e) {
                        document.getElementById('blacklist').value = '';
                    }
                }
            }
            
            // تغيير نص الزر والعنوان
            document.querySelector('.modal-title').innerHTML = '<i class="fas fa-edit"></i> تعديل الإعدادات';
            document.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ التعديلات';
            
            // حفظ الـ ID للتعديل
            document.getElementById('settingsForm').setAttribute('data-edit-id', id);
            
            // فتح المودال
            document.getElementById('addSettingsModal').style.display = 'block';
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'فشل في تحميل الإعدادات'
            });
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال'
        });
    });
}

function resetForm() {
    document.getElementById('settingsForm').reset();
    document.getElementById('protectionFields').classList.remove('show');
    document.getElementById('blacklistFields').classList.remove('show');
    document.getElementById('settingsForm').removeAttribute('data-edit-id');
    
    // إعادة العنوان والزر للوضع الأفتراضي
    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-cog"></i> إضافة إعدادات إرسال جديدة';
    document.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ الإعدادات';
}

// Close modal when clicking outside
window.onclick = function(event) {
    if (event.target.classList.contains('modal')) {
        event.target.style.display = 'none';
    }
}

// Load settings on page load
loadSettings();
</script>

<?php include 'includes/footer.php'; ?>
