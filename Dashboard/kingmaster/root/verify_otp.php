<?php
session_start();

if (!isset($_SESSION['user_id'])) {
    header('Location: register.php');
    exit;
}

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
    <title>التحقق من الحساب - Kingmaster</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@300;400;600;700;800;900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="assets/css/auth.css">
</head>
<body>
    <div class="stars-bg" id="stars-bg"></div>
    <div class="auth-container">
        <div class="auth-card otp-card">
            <div class="auth-header">
                <div class="otp-icon">
                    <i class="fas fa-shield-alt"></i>
                </div>
                <h2>التحقق من الحساب</h2>
                <p>أدخل رمز التحقق المكون من 6 أرقام</p>
                <?php 
                // إخفاء بعض أرقام الهاتف للخصوصية
                $phone = $_SESSION['temp_phone'] ?? '';
                if ($phone) {
                    $masked_phone = substr($phone, 0, 3) . '****' . substr($phone, -2);
                }
                ?>
                <p class="email-sent">تم إرسال الرمز عبر SMS إلم: <strong><?php echo $masked_phone ?? ''; ?></strong></p>
            </div>

            <form id="otpForm" class="otp-form">
                <div class="otp-inputs">
                    <input type="text" maxlength="1" pattern="[0-9]" class="otp-input" id="otp1" autofocus>
                    <input type="text" maxlength="1" pattern="[0-9]" class="otp-input" id="otp2">
                    <input type="text" maxlength="1" pattern="[0-9]" class="otp-input" id="otp3">
                    <input type="text" maxlength="1" pattern="[0-9]" class="otp-input" id="otp4">
                    <input type="text" maxlength="1" pattern="[0-9]" class="otp-input" id="otp5">
                    <input type="text" maxlength="1" pattern="[0-9]" class="otp-input" id="otp6">
                </div>

                <div class="error-message" id="errorMessage"></div>
                <div class="success-message" id="successMessage"></div>

                <button type="submit" class="btn-submit">
                    <i class="fas fa-check-circle"></i>
                    تحقق من الرمز
                </button>

                <div class="resend-otp">
                    <p>لم تستلم الرمز؟</p>
                    <button type="button" class="btn-resend" id="resendBtn" disabled>
                        <i class="fas fa-redo"></i>
                        <span id="resendText">إعادة إرسال الرمز</span>
                        <span id="countdown">(60)</span>
                    </button>
                </div>
            </form>
        </div>
    </div>

    <script src="assets/js/verify_otp.js"></script>
</body>
</html>
