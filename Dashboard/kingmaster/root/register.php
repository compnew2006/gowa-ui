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
    <title>إنشاء حساب جديد - Kingmaster</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@300;400;600;700;800;900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="assets/css/auth.css">
</head>
<body>
    <div class="stars-bg" id="stars-bg"></div>
    <div class="auth-container">
        <div class="auth-card">
            <div class="auth-header">
                <div class="logo">
                    <i class="fas fa-rocket"></i>
                    <h1>Kingmaster</h1>
                </div>
                <h2>إنشاء حساب جديد</h2>
             </div>

            <form id="registerForm" action="register_handler.php" method="POST" class="auth-form">
                <?php echo csrfInput(); ?>
                <div class="form-row">
                    <div class="form-group">
                        <label><i class="fas fa-user"></i> الاسم الأول</label>
                        <input type="text" name="first_name" id="first_name" required>
                    </div>
                    <div class="form-group">
                        <label><i class="fas fa-user"></i> الاسم الأخير</label>
                        <input type="text" name="last_name" id="last_name" required>
                    </div>
                </div>

                <div class="form-group">
                    <label><i class="fas fa-envelope"></i> البريد الإلكتروني</label>
                    <input type="email" name="email" id="email" required>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label><i class="fas fa-lock"></i> كلمة المرور</label>
                        <div class="password-input">
                            <input type="password" name="password" id="password" required>
                            <i class="fas fa-eye toggle-password" onclick="togglePassword('password')"></i>
                        </div>
                        <div class="password-strength" id="passwordStrength">
                            <div class="strength-bar">
                                <div class="strength-fill" id="strengthFill"></div>
                            </div>
                            <span class="strength-text" id="strengthText">ضعيفة</span>
                        </div>
                    </div>

                    <div class="form-group">
                        <label><i class="fas fa-lock"></i> تأكيد كلمة المرور</label>
                        <div class="password-input">
                            <input type="password" name="confirm_password" id="confirm_password" required>
                            <i class="fas fa-eye toggle-password" onclick="togglePassword('confirm_password')"></i>
                        </div>
                        <span class="password-match" id="passwordMatch"></span>
                    </div>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label><i class="fas fa-phone"></i> رقم الهاتف</label>
                        <input type="tel" name="phone" id="phone" placeholder="+966 50 123 4567" required>
                    </div>

                    <div class="form-group">
                        <label><i class="fas fa-globe"></i> المنطقة الزمنية</label>
                        <select name="timezone" id="timezone" required>
                        <option value="">اختر المنطقة الزمنية</option>
                        <?php
                        $timezones = DateTimeZone::listIdentifiers();
                        foreach ($timezones as $tz) {
                            $selected = ($tz == 'Asia/Riyadh') ? 'selected' : '';
                            echo "<option value=\"$tz\" $selected>$tz</option>";
                        }
                        ?>
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label><i class="fas fa-briefcase"></i> العمل (اختياري)</label>
                    <input type="text" name="job" id="job">
                </div>

                <div class="form-group">
                    <label><i class="fas fa-calendar"></i> تاريخ الميلاد</label>
                    <div class="date-inputs">
                        <select name="birth_day" id="birth_day" required>
                            <option value="">اليوم</option>
                            <?php for($i=1; $i<=31; $i++) echo "<option value='$i'>$i</option>"; ?>
                        </select>
                        <select name="birth_month" id="birth_month" required>
                            <option value="">الشهر</option>
                            <option value="1">يناير</option>
                            <option value="2">فبراير</option>
                            <option value="3">مارس</option>
                            <option value="4">أبريل</option>
                            <option value="5">مايو</option>
                            <option value="6">يونيو</option>
                            <option value="7">يوليو</option>
                            <option value="8">أغسطس</option>
                            <option value="9">سبتمبر</option>
                            <option value="10">أكتوبر</option>
                            <option value="11">نوفمبر</option>
                            <option value="12">ديسمبر</option>
                        </select>
                        <select name="birth_year" id="birth_year" required>
                            <option value="">السنة</option>
                            <?php for($i=date('Y'); $i>=1950; $i--) echo "<option value='$i'>$i</option>"; ?>
                        </select>
                    </div>
                </div>

                <div class="form-group checkbox-group">
                    <label class="checkbox-label">
                        <input type="checkbox" name="terms" id="terms" required>
                        <span>أوافق على <a href="#terms" target="_blank">الشروط والأحكام</a> و <a href="#privacy" target="_blank">سياسة الخصوصية</a></span>
                    </label>
                </div>

                <div class="error-message" id="errorMessage"></div>
                <div class="success-message" id="successMessage"></div>

                <button type="submit" class="btn-submit">
                    <i class="fas fa-user-plus"></i>
                    إنشاء حساب
                </button>

                <div class="auth-footer">
                    <p>لديك حساب بالفعل؟ <a href="login.php">تسجيل الدخول</a></p>
                </div>
            </form>
        </div>
    </div>

    <script src="assets/js/register.js"></script>
</body>
</html>
