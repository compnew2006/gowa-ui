<?php
 
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "الإعدادات | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
 
$user = getUserData($user_id);




// بيانات وهمية - يتم استبدالها بقاعدة البيانات
$user = [
    'avatar' => $user['avatar'],
    'timezone' => $user['timezone']
];

$timezones = generateTimezoneList();

?>

<style>
    .settings-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .settings-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 25px;
    }

    .settings-grid .settings-card:first-child {
        grid-column: 1 / -1;
    }

    .settings-header {
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
        margin-bottom: 8px;
    }

    .settings-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: iconRotate 3s ease-in-out infinite;
    }

    @keyframes iconRotate {
        0%, 100% { transform: rotate(0deg); }
        50% { transform: rotate(180deg); }
    }

    .settings-subtitle {
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }

    .settings-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
        margin-bottom: 25px;
        transition: all 0.3s ease;
    }

    .settings-card:hover {
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.2);
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 25px;
        padding-bottom: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .card-title {
        font-size: 20px;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .card-icon {
        font-size: 28px;
        animation: iconFloat 3s ease-in-out infinite;
    }

    .card-icon.icon-image {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: iconRotate 4s linear infinite;
    }

    .card-icon.icon-lock {
        color: #ef4444;
        animation: lockShake 2s ease-in-out infinite;
    }

    .card-icon.icon-globe {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: globeSpin 4s linear infinite;
    }

    @keyframes iconFloat {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-5px); }
    }

    @keyframes iconRotate {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }

    @keyframes lockShake {
        0%, 100% { transform: rotate(0deg); }
        10%, 30%, 50%, 70%, 90% { transform: rotate(-5deg); }
        20%, 40%, 60%, 80% { transform: rotate(5deg); }
    }

    @keyframes globeSpin {
        0% { transform: rotateY(0deg); }
        100% { transform: rotateY(360deg); }
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-label {
        display: block;
        margin-bottom: 8px;
        font-weight: 600;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }

    .form-input, .form-select {
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

    .form-input:focus, .form-select:focus {
        outline: none;
        border-color: #667eea;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.15);
    }

    .form-select option {
        background: var(--card-bg) !important;
        background-color: var(--card-bg) !important;
        color: var(--text-primary) !important;
        padding: 10px;
    }

    /* Dark theme select */
    body:not(.light-theme) .form-select option {
        background: #1a1d29 !important;
        background-color: #1a1d29 !important;
        color: #e4e6eb !important;
    }

    .forgot-password-link {
        display: inline-block;
        margin-top: 8px;
        color: #667eea;
        font-size: 13px;
        font-weight: 600;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .forgot-password-link:hover {
        color: #764ba2;
        text-decoration: underline;
    }

    .save-btn {
        width: 100%;
        padding: 14px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
    }

    .save-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
    }

    .save-btn i {
        animation: saveIconPulse 2s ease-in-out infinite;
    }

    @keyframes saveIconPulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.2); }
    }

    /* Profile Picture Upload */
    .avatar-section {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        gap: 25px;
        margin-bottom: 30px;
    }

    .avatar-wrapper {
        position: relative;
        display: inline-block;
    }

    .current-avatar {
        width: 140px;
        height: 140px;
        border-radius: 50%;
        border: 5px solid transparent;
        background: linear-gradient(var(--card-bg), var(--card-bg)) padding-box,
                    linear-gradient(135deg, #667eea, #764ba2, #f093fb, #667eea) border-box;
        box-shadow: 0 15px 40px rgba(102, 126, 234, 0.4);
        transition: all 0.4s ease;
        object-fit: cover;
        animation: avatarGlow 3s ease-in-out infinite;
    }

    @keyframes avatarGlow {
        0%, 100% { 
            box-shadow: 0 15px 40px rgba(102, 126, 234, 0.4);
        }
        50% { 
            box-shadow: 0 20px 50px rgba(118, 75, 162, 0.6);
        }
    }

    .current-avatar:hover {
        transform: scale(1.08) rotate(5deg);
        box-shadow: 0 20px 50px rgba(102, 126, 234, 0.6);
    }

    .avatar-badge {
        position: absolute;
        bottom: 5px;
        right: 5px;
        width: 40px;
        height: 40px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.3s ease;
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.5);
        border: 3px solid var(--card-bg);
    }

    .avatar-badge:hover {
        transform: scale(1.15) rotate(15deg);
        box-shadow: 0 8px 20px rgba(102, 126, 234, 0.7);
    }

    .avatar-badge i {
        color: white;
        font-size: 18px;
        animation: cameraFloat 2s ease-in-out infinite;
    }

    @keyframes cameraFloat {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-3px); }
    }

    .avatar-info {
        width: 100%;
    }

    .avatar-info h3 {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 8px;
        font-family: 'Cairo', sans-serif;
    }

    .avatar-info p {
        color: var(--text-secondary);
        font-size: 13px;
        margin-bottom: 20px;
        font-family: 'Cairo', sans-serif;
        line-height: 1.6;
    }

    .avatar-buttons {
        display: flex;
        gap: 12px;
        justify-content: center;
        flex-wrap: wrap;
    }

    .upload-btn, .remove-btn {
        padding: 12px 24px;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
        display: flex;
        align-items: center;
        gap: 8px;
        position: relative;
        overflow: hidden;
    }

    .upload-btn::before,
    .remove-btn::before {
        content: '';
        position: absolute;
        top: 50%;
        left: 50%;
        width: 0;
        height: 0;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.3);
        transform: translate(-50%, -50%);
        transition: width 0.6s ease, height 0.6s ease;
    }

    .upload-btn:hover::before,
    .remove-btn:hover::before {
        width: 300px;
        height: 300px;
    }

    .upload-btn {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        color: white;
        box-shadow: 0 5px 15px rgba(16, 185, 129, 0.3);
    }

    .upload-btn:hover {
        transform: translateY(-3px);
        box-shadow: 0 8px 20px rgba(16, 185, 129, 0.5);
    }

    .upload-btn i {
        position: relative;
        z-index: 1;
        animation: uploadPulse 2s ease-in-out infinite;
    }

    @keyframes uploadPulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.2); }
    }

    .remove-btn {
        background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
        color: white;
        box-shadow: 0 5px 15px rgba(239, 68, 68, 0.3);
    }

    .remove-btn:hover {
        transform: translateY(-3px);
        box-shadow: 0 8px 20px rgba(239, 68, 68, 0.5);
    }

    .remove-btn i {
        position: relative;
        z-index: 1;
        animation: trashShake 2s ease-in-out infinite;
    }

    @keyframes trashShake {
        0%, 100% { transform: rotate(0deg); }
        25% { transform: rotate(-5deg); }
        75% { transform: rotate(5deg); }
    }

    .upload-btn span,
    .remove-btn span {
        position: relative;
        z-index: 1;
    }

    #avatarInput {
        display: none;
    }

    .file-requirements {
        display: flex;
        gap: 15px;
        justify-content: center;
        margin-top: 15px;
        flex-wrap: wrap;
    }

    .requirement-item {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
        padding: 6px 12px;
        background: var(--bg-primary);
        border-radius: 20px;
        border: 1px solid var(--border-color);
    }

    .requirement-item i {
        color: #667eea;
        font-size: 14px;
    }

    /* Light Theme */
    body.light-theme .settings-card {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .form-input,
    body.light-theme .form-select {
        background: #ffffff;
        color: #2d3436;
        border-color: rgba(0, 0, 0, 0.15);
    }

    body.light-theme .form-select option {
        background: #ffffff !important;
        background-color: #ffffff !important;
        color: #2d3436 !important;
    }

    body.light-theme .settings-title,
    body.light-theme .card-title,
    body.light-theme .form-label {
        color: #2d3436;
    }

    body.light-theme .settings-subtitle,
    body.light-theme .avatar-info p,
    body.light-theme .requirement-item {
        color: #636e72;
    }

    body.light-theme .avatar-info h3 {
        color: #2d3436;
    }

    body.light-theme .requirement-item {
        background: #f8f9fa;
        border-color: rgba(0, 0, 0, 0.1);
    }

    /* Responsive */
    @media (max-width: 1024px) {
        .settings-grid {
            grid-template-columns: 1fr;
        }

        .settings-grid .settings-card:first-child {
            grid-column: 1;
        }
    }

    @media (max-width: 768px) {
        .settings-container {
            padding: 20px;
            margin-top: 100px;
        }

        .avatar-section {
            flex-direction: column;
            text-align: center;
        }

        .avatar-buttons {
            justify-content: center;
        }
    }
</style>

<div class="settings-container">
    <div class="settings-header">
        <div class="settings-title">
            <i class="fas fa-cog"></i>
            الإعدادات
        </div>
        <div class="settings-subtitle">إدارة إعدادات حسابك وتفضيلاتك</div>
    </div>

    <div class="settings-grid">
        <!-- تغيير الصورة الشخصية -->
        <div class="settings-card">
            <div class="card-header">
                <i class="fas fa-image card-icon icon-image"></i>
                <span class="card-title">الصورة الشخصية</span>
            </div>

        <div class="avatar-section">
            <div class="avatar-wrapper">
                <img src="<?php echo $user['avatar']; ?>" alt="Avatar" class="current-avatar" id="previewAvatar">
                <div class="avatar-badge" onclick="document.getElementById('avatarInput').click()">
                    <i class="fas fa-camera"></i>
                </div>
            </div>
            
            <div class="avatar-info">
                <h3><i class="fas fa-image" style="color: #667eea;"></i> الصورة الشخصية</h3>
                <p>اختر صورة لملفك الشخصي تظهر في جميع أنحاء النظام</p>
                
              
                
                <div class="file-requirements">
                    <div class="requirement-item">
                        <i class="fas fa-file-image"></i>
                        <span>JPG, PNG</span>
                    </div>
                    <div class="requirement-item">
                        <i class="fas fa-weight"></i>
                        <span>حد أقصى 2MB</span>
                    </div>
                    <div class="requirement-item">
                        <i class="fas fa-crop"></i>
                        <span>مربعة مفضلة</span>
                    </div>
                </div>
                
                <input type="file" id="avatarInput" accept="image/*" onchange="previewImage(event)">
            </div>
        </div>

        <button class="save-btn" onclick="saveAvatar()">
            <i class="fas fa-save"></i>
            حفظ الصورة
        </button>
    </div>

        <!-- تغيير كلمة المرور -->
        <div class="settings-card">
            <div class="card-header">
                <i class="fas fa-lock card-icon icon-lock"></i>
                <span class="card-title">تغيير كلمة المرور</span>
            </div>

        <form onsubmit="event.preventDefault(); changePassword();">
            <div class="form-group">
                <label class="form-label">كلمة المرور الحالية</label>
                <input type="password" class="form-input" id="currentPassword" placeholder="أدخل كلمة المرور الحالية" required>
                <span class="forgot-password-link" onclick="forgotPassword()">
                    <i class="fas fa-question-circle"></i>
                    نسيت كلمة المرور؟
                </span>
            </div>

            <div class="form-group">
                <label class="form-label">كلمة المرور الجديدة</label>
                <input type="password" class="form-input" id="newPassword" placeholder="أدخل كلمة المرور الجديدة" required>
            </div>

            <div class="form-group">
                <label class="form-label">تأكيد كلمة المرور الجديدة</label>
                <input type="password" class="form-input" id="confirmPassword" placeholder="أعد إدخال كلمة المرور الجديدة" required>
            </div>

            <button type="submit" class="save-btn">
                <i class="fas fa-key"></i>
                تحديث كلمة المرور
            </button>
        </form>
    </div>

        <!-- المنطقة الزمنية -->
        <div class="settings-card">
            <div class="card-header">
                <i class="fas fa-globe card-icon icon-globe"></i>
                <span class="card-title">المنطقة الزمنية</span>
            </div>

        <div class="form-group">
            <label class="form-label">اختر المنطقة الزمنية</label>
            <select class="form-select" id="timezone">
                <?php foreach($timezones as $value => $label): ?>
                    <option value="<?php echo $value; ?>" <?php echo $value === $user['timezone'] ? 'selected' : ''; ?>>
                        <?php echo $label; ?>
                    </option>
                <?php endforeach; ?>
            </select>
        </div>

        <button class="save-btn" onclick="saveTimezone()">
            <i class="fas fa-clock"></i>
            حفظ المنطقة الزمنية
        </button>
        </div>
    </div>
</div>

<script>
// إصلاح ألوان السيلكت
function fixSelectColors() {
    const select = document.getElementById('timezone');
    const isLight = document.body.classList.contains('light-theme');
    
    if (isLight) {
        select.style.backgroundColor = '#ffffff';
        select.style.color = '#2d3436';
        // تطبيق الألوان على الoptions
        Array.from(select.options).forEach(option => {
            option.style.backgroundColor = '#ffffff';
            option.style.color = '#2d3436';
        });
    } else {
        select.style.backgroundColor = '#1a1d29';
        select.style.color = '#e4e6eb';
        Array.from(select.options).forEach(option => {
            option.style.backgroundColor = '#1a1d29';
            option.style.color = '#e4e6eb';
        });
    }
}

// تطبيق عند تحميل الصفحة
window.addEventListener('DOMContentLoaded', fixSelectColors);

// تطبيق عند تغيير الثيم
const themeObserver = new MutationObserver(function(mutations) {
    mutations.forEach(function(mutation) {
        if (mutation.attributeName === 'class') {
            fixSelectColors();
        }
    });
});
themeObserver.observe(document.body, { attributes: true });

// تغيير كلمة المرور
function changePassword() {
    const currentPassword = document.getElementById('currentPassword').value;
    const newPassword = document.getElementById('newPassword').value;
    const confirmPassword = document.getElementById('confirmPassword').value;

    if (!currentPassword || !newPassword || !confirmPassword) {
        Swal.fire({
            icon: 'warning',
            title: 'تحذير',
            text: 'يرجى ملء جميع الحقول',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        return;
    }

    if (newPassword !== confirmPassword) {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'كلمة المرور الجديدة غير متطابقة',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        return;
    }

    if (newPassword.length < 6) {
        Swal.fire({
            icon: 'warning',
            title: 'تحذير',
            text: 'كلمة المرور يجب أن تكون 6 أحرف على الأقل',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        return;
    }

    // عرض رسالة تحميل
    Swal.fire({
        title: 'جاري التحديث...',
        html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #667eea;"></i>',
        showConfirmButton: false,
        allowOutsideClick: false,
        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
    });

    // إرسال البيانات للسيرفر
    fetch('api/change_password.php', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            current_password: currentPassword,
            new_password: newPassword,
            confirm_password: confirmPassword
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم التحديث!',
                text: 'تم تغيير كلمة المرور بنجاح',
                timer: 2000,
                showConfirmButton: false,
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            }).then(() => {
                document.getElementById('currentPassword').value = '';
                document.getElementById('newPassword').value = '';
                document.getElementById('confirmPassword').value = '';
            });



        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message || 'حدث خطأ أثناء تغيير كلمة المرور',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال بالخادم',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        console.error('Error:', error);
    });
}

// نسيت كلمة المرور
function forgotPassword() {
    Swal.fire({
        title: 'استعادة كلمة المرور',
        html: `
            <div style="text-align: center; padding: 20px; font-family: 'Cairo', sans-serif;">
                <i class="fab fa-whatsapp" style="font-size: 60px; color: #25D366; margin-bottom: 15px;"></i>
                <p style="font-size: 16px; color: var(--text-primary); margin-bottom: 10px;">سيتم تغيير كلمة المرور وإرسالها عبر الواتساب</p>
                <p style="font-size: 14px; color: var(--text-secondary); margin-bottom: 15px;">إلى رقم الهاتف المسجل</p>
                <div style="background: linear-gradient(135deg, rgba(37, 211, 102, 0.1), rgba(37, 211, 102, 0.05)); padding: 15px; border-radius: 10px; border: 1px solid rgba(37, 211, 102, 0.2);">
                    <i class="fas fa-info-circle" style="color: #25D366; margin-bottom: 8px;"></i>
                    <p style="font-size: 13px; color: var(--text-secondary); margin: 0;">ستحصل على كلمة مرور جديدة عشوائية</p>
                </div>
            </div>
        `,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'إرسال كلمة المرور',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#25D366',
        cancelButtonColor: '#95a5a6',
        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
    }).then((result) => {
        if (result.isConfirmed) {
            // عرض رسالة تحميل
            Swal.fire({
                title: 'جاري إعادة تعيين كلمة المرور...',
                html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #25D366;"></i>',
                showConfirmButton: false,
                allowOutsideClick: false,
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
            
            // إرسال طلب استعادة كلمة المرور
            fetch('api/forgot_password.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
            .then(response =>response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم الإرسال!',
                        html: `
                            <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                                <i class="fab fa-whatsapp" style="font-size: 50px; color: #25D366; margin-bottom: 15px;"></i>
                                <p style="font-size: 16px; margin-bottom: 10px;">تم إرسال كلمة المرور الجديدة</p>
                                <p style="font-size: 14px; color: var(--text-secondary);">...**${data.phone_last_digits}</p>
                                <div style="background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(16, 185, 129, 0.05)); padding: 10px; border-radius: 8px; margin-top: 10px;">
                                    <p style="font-size: 12px; color: var(--text-secondary); margin: 0;">يرجى تسجيل الدخول وتغيير كلمة المرور فوراً</p>
                                </div>
                            </div>
                        `,
                        timer: 5000,
                        showConfirmButton: false,
                        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                    });
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message || 'حدث خطأ أثناء إعادة تعيين كلمة المرور',
                        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                    });
                }
            })
            .catch(error => {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: 'حدث خطأ في الاتصال بالخادم',
                    background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                    color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                });
                console.error('Error:', error);
            });
        }
    });
}

// حفظ المنطقة الزمنية
function saveTimezone() {
    const timezone = document.getElementById('timezone').value;
    
    if (!timezone) {
        Swal.fire({
            icon: 'warning',
            title: 'تحذير',
            text: 'يرجى اختيار منطقة زمنية',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        return;
    }
    
    // عرض رسالة تحميل
    Swal.fire({
        title: 'جاري الحفظ...',
        html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #667eea;"></i>',
        showConfirmButton: false,
        allowOutsideClick: false,
        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
    });
    
    // إرسال البيانات للسيرفر
    fetch('api/update_timezone.php', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ timezone: timezone })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم الحفظ!',
                text: 'تم تحديث المنطقة الزمنية بنجاح',
                timer: 2000,
                showConfirmButton: false,
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message || 'حدث خطأ أثناء تحديث المنطقة الزمنية',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال بالخادم',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        console.error('Error:', error);
    });
}

// معاينة الصورة
function previewImage(event) {
    const file = event.target.files[0];
    
    if (file) {
        if (file.size > 2 * 1024 * 1024) {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'حجم الصورة يجب أن يكون أقل من 2 ميجابايت'
            });
            return;
        }

        const reader = new FileReader();
        reader.onload = function(e) {
            document.getElementById('previewAvatar').src = e.target.result;
        };
        reader.readAsDataURL(file);
    }
}

// حفظ الصورة
function saveAvatar() {
    const fileInput = document.getElementById('avatarInput');
    
    if (!fileInput.files[0]) {
        Swal.fire({
            icon: 'warning',
            title: 'تحذير',
            text: 'يرجى اختيار صورة أولاً',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        return;
    }

    // عرض رسالة تحميل
    Swal.fire({
        title: 'جاري رفع الصورة...',
        html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #667eea;"></i>',
        showConfirmButton: false,
        allowOutsideClick: false,
        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
    });

    // إعداد البيانات للرفع
    const formData = new FormData();
    formData.append('avatar', fileInput.files[0]);

    // رفع الصورة للسيرفر
    fetch('api/upload_avatar.php', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم الحفظ!',
                text: 'تم تحديث الصورة الشخصية بنجاح',
                timer: 2000,
                showConfirmButton: false,
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            }).then(() => {
                // تحديث الصورة في الصفحة
                location.reload();
            });
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message || 'حدث خطأ أثناء رفع الصورة',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال بالخادم',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
        console.error('Error:', error);
    });
}

// إزالة الصورة
function removeAvatar() {
    Swal.fire({
        title: 'تأكيد الحذف',
        text: 'هل تريد حذف الصورة الشخصية؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#ef4444'
    }).then((result) => {
        if (result.isConfirmed) {
            // العودة للصورة الافتراضية
            document.getElementById('previewAvatar').src = 'https://ui-avatars.com/api/?name=User&background=667eea&color=fff&size=200';
            document.getElementById('avatarInput').value = '';
            
            Swal.fire({
                icon: 'success',
                title: 'تم الحذف!',
                text: 'تم حذف الصورة الشخصية',
                timer: 2000,
                showConfirmButton: false
            });
        }
    });
}
</script>

<?php include 'includes/footer.php'; ?>
