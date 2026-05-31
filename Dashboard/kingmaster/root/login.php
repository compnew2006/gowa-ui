<?php
require_once __DIR__ . '/config/database.php';
startSecureSession();
applySecurityHeaders();

// إذا كان المستخدم مسجل دخول بالفعل، إعادة توجيهه
if (isset($_SESSION['user_id']) && isset($_SESSION['is_logged_in'])) {
    header('Location: index.php');
    exit;
}
?>
<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>تسجيل الدخول - Kingmaster</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@300;400;600;700;800;900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="assets/css/auth.css">
    <style>
        /* Modal styles aligned with auth theme */
        .modal-overlay { display:none; position:fixed; inset:0; z-index:1000; background: radial-gradient(ellipse at top, rgba(102,126,234,0.15) 0%, transparent 50%), radial-gradient(ellipse at bottom, rgba(118,75,162,0.15) 0%, transparent 50%), rgba(0,0,0,0.55); align-items:center; justify-content:center; }
        .modal { direction:rtl; background: rgba(30,41,59,0.92); backdrop-filter: blur(20px); color:#f1f5f9; width:95%; max-width:520px; border-radius:20px; box-shadow: 0 25px 70px rgba(0,0,0,0.5), 0 50px 100px rgba(102,126,234,0.2), inset 0 0 100px rgba(102,126,234,0.05); border:2px solid rgba(102,126,234,0.3); animation: fadeInUp 0.35s ease; }
        .modal-header { padding:18px 22px; border-bottom:2px solid rgba(102,126,234,0.2); display:flex; align-items:center; justify-content:space-between; }
        .modal-header h3 { margin:0; font-size:19px; font-weight:800; }
        .modal-body { padding:22px; }
        .modal-actions { padding:18px 22px; display:flex; gap:10px; justify-content:flex-start; border-top:2px solid rgba(102,126,234,0.12); }
        .btn { cursor:pointer; border:none; border-radius:12px; padding:10px 14px; font-family:inherit; display:inline-flex; align-items:center; gap:8px; }
        .btn-primary { background: linear-gradient(135deg, var(--primary), var(--secondary)); color:#fff; box-shadow: 0 8px 25px rgba(102,126,234,0.4), 0 15px 40px rgba(118,75,162,0.3); }
        .btn-secondary { background:#1f2937; color:#fff; border:2px solid rgba(102,126,234,0.25); }
        .btn i { animation: pulse 2.5s infinite ease-in-out, float3d 8s ease-in-out infinite; }
        .link { color:#60a5fa; cursor:pointer; text-decoration:underline; font-size:14px; }
        .small-note { font-size:12px; color:#9ca3af; margin-top:8px; }
        .inline-feedback { margin-top:10px; font-size:14px; }
        .inline-feedback.error { color:#ef4444; }
        .inline-feedback.success { color:#10b981; }
        /* Animate all icons inside modal */
        .modal i { will-change: transform; animation: pulse 3s ease-in-out infinite, float3d 10s ease-in-out infinite; }
        .modal-header i { animation: rotate 10s linear infinite, float3d 8s ease-in-out infinite; }
        /* Inputs inside modal to match theme */
        .modal input[type="email"], .modal input[type="text"], .modal input[type="password"] { width:100%; margin-top:8px; padding:12px 14px; border-radius:12px; border:2px solid rgba(102,126,234,0.3); background: rgba(15,23,42,0.5); color:#f1f5f9; font-family:'Cairo', sans-serif; transition: all .25s ease; }
        .modal input:focus { outline:none; border-color: var(--primary); background: rgba(15,23,42,0.7); box-shadow: 0 0 0 4px rgba(102,126,234,0.2); }
        .modal label { display:flex; align-items:center; gap:8px; font-weight:800; color:#f1f5f9; margin-bottom:8px; }
    </style>
</head>
<body>
    <div class="stars-bg" id="stars-bg"></div>
    
    <div class="auth-container">
        <div class="auth-card">
            <div class="auth-header">
                <div class="logo">
                    <i class="fas fa-crown"></i>
                    <h1>Kingmaster</h1>
                </div>
                <h2>مرحباً بعودتك</h2>
                <p>سجل دخولك للوصول إلى حسابك</p>
            </div>

            <form id="loginForm" class="auth-form">
                <?php echo csrfInput(); ?>
                <div class="error-message" id="errorMessage"></div>
                <div class="success-message" id="successMessage"></div>

                <div class="form-group">
                    <label for="email">
                        <i class="fas fa-envelope"></i>
                        البريد الإلكتروني
                    </label>
                    <input type="email" id="email" name="email" placeholder="أدخل بريدك الإلكتروني" required>
                </div>

                <div class="form-group">
                    <label for="password">
                        <i class="fas fa-lock"></i>
                        كلمة المرور
                    </label>
                    <div class="password-input">
                        <input type="password" id="password" name="password" placeholder="أدخل كلمة المرور" required>
                        <i class="fas fa-eye toggle-password" onclick="togglePassword('password')"></i>
                    </div>
                    <div style="margin-top:8px">
                        <span class="link" id="openForgotModal">هل نسيت كلمة المرور؟</span>
                    </div>
                </div>

                <div class="checkbox-group">
                    <label class="checkbox-label">
                        <input type="checkbox" name="terms" id="terms" required>
                        <span>أوافق على <a href="terms_privacy">الشروط والأحكام</a> و <a href="privacy">سياسة الخصوصية</a></span>
                    </label>
                </div>

                <button type="submit" class="btn-submit">
                    <i class="fas fa-sign-in-alt"></i>
                    تسجيل الدخول
                </button>
            </form>

            <div class="auth-footer">
                <p>ليس لديك حساب؟ <a href="register.php">سجل الآن</a></p>
            </div>
        </div>
    </div>

    <!-- Forgot Password Modal -->
    <div class="modal-overlay" id="forgotModalOverlay" aria-hidden="true">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="forgotModalTitle">
            <div class="modal-header">
                <h3 id="forgotModalTitle"><i class="fas fa-unlock-keyhole"></i> استعادة كلمة المرور</h3>
                <button class="btn btn-secondary" id="closeForgotModal" title="إغلاق"><i class="fas fa-times"></i></button>
            </div>
            <div class="modal-body">
                <label for="forgotEmail"><i class="fas fa-envelope"></i> البريد الإلكتروني المسجّل</label>
                <input type="email" id="forgotEmail" placeholder="example@email.com" style="width:100%;margin-top:8px;padding:12px;border-radius:10px;border:1px solid rgba(255,255,255,.1);background:#0b1220;color:#fff">
                <div class="small-note">سيتم إرسال كلمة مرور جديدة إلى رقم الواتساب المرتبط بحسابك.</div>
                <div id="forgotInlineMsg" class="inline-feedback" aria-live="polite"></div>
            </div>
            <div class="modal-actions">
                <button class="btn btn-primary" id="submitForgot"><i class="fas fa-paper-plane"></i> إرسال كلمة مرور جديدة</button>
                <button class="btn btn-secondary" id="cancelForgot">إلغاء</button>
            </div>
        </div>
    </div>

    <script src="assets/js/login.js"></script>
</body>
</html>
