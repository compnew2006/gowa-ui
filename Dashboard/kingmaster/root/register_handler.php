<?php
require_once 'config/database.php';
require_once 'includes/send_otp.php';

startSecureSession();
applySecurityHeaders();

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('طريقة الطلب غير صحيحة', 405);
}

verifyCsrfToken();
enforceRateLimit('register', 3, 3600);

// استقبال البيانات
$first_name = cleanText($_POST['first_name'] ?? '', 80);
$last_name = cleanText($_POST['last_name'] ?? '', 80);
$email = filter_var($_POST['email'] ?? '', FILTER_SANITIZE_EMAIL);
$password = $_POST['password'] ?? '';
$confirm_password = $_POST['confirm_password'] ?? '';
$phone = cleanText($_POST['phone'] ?? '', 30);
$timezone = cleanText($_POST['timezone'] ?? '', 80);
$job = cleanText($_POST['job'] ?? '', 120);
$birth_day = $_POST['birth_day'] ?? '';
$birth_month = $_POST['birth_month'] ?? '';
$birth_year = $_POST['birth_year'] ?? '';
$terms = isset($_POST['terms']) && $_POST['terms'] == 'on';

// التحقق من الحقول المطلوبة
if (empty($first_name) || empty($last_name) || empty($email) || empty($password) || 
    empty($phone) || empty($timezone) || empty($birth_day) || empty($birth_month) || empty($birth_year)) {
    respondError('جميع الحقول مطلوبة', 400);
}

// التحقق من الموافقة على الشروط
if (!$terms) {
    respondError('يجب الموافقة على الشروط والأحكام', 400);
}

// التحقق من صحة البريد الإلكتروني
if (!isValidEmail($email)) {
    respondError('البريد الإلكتروني غير صحيح', 400);
}

// التحقق من تطابق كلمتي المرور
if ($password !== $confirm_password) {
    respondError('كلمتا المرور غير متطابقتين', 400);
}

// التحقق من قوة كلمة المرور
if (strlen($password) < 12 || !preg_match('/[A-Z]/', $password) || !preg_match('/[a-z]/', $password) || !preg_match('/[0-9]/', $password)) {
    respondError('كلمة المرور يجب أن تكون 12 حرفاً على الأقل وتحتوي على حروف كبيرة وصغيرة وأرقام', 400);
}

if (!in_array($timezone, DateTimeZone::listIdentifiers(), true)) {
    respondError('المنطقة الزمنية غير صالحة', 400);
}

if (!checkdate((int)$birth_month, (int)$birth_day, (int)$birth_year)) {
    respondError('تاريخ الميلاد غير صالح', 400);
}

// تكوين تاريخ الميلاد
$birth_date = sprintf('%04d-%02d-%02d', $birth_year, $birth_month, $birth_day);

// التحقق من أن البريد الإلكتروني غير مسجل
$check_email = fetchRow("SELECT id FROM users WHERE email = ?", [$email]);
if ($check_email) {
    respondError('البريد الإلكتروني مسجل بالفعل', 409);
}

// التحقق من أن رقم الهاتف غير مسجل
$check_phone = fetchRow("SELECT id FROM users WHERE phone = ?", [$phone]);
if ($check_phone) {
    respondError('رقم الهاتف مسجل بالفعل', 409);
}

try {
    // توليد user_id فريد
    $user_id = 'USR' . strtoupper(bin2hex(random_bytes(8)));
    
    // تشفير كلمة المرور
    $hashed_password = hashPassword($password);
    
    // توليد OTP من 6 أرقام
    $otp = str_pad((string)random_int(0, 999999), 6, '0', STR_PAD_LEFT);
    
    // إدخال البيانات في قاعدة البيانات
    $query = "INSERT INTO users (user_id, first_name, last_name, email, password, phone, timezone, job, birth_date, otp, otp_created_at, is_verified, is_admin, expiry_date) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), 0, 0, NOW() + INTERVAL 48 HOUR)";
    
    $result = executeQuery($query, [
        $user_id,
        $first_name,
        $last_name,
        $email,
        $hashed_password,
        $phone,
        $timezone,
        $job,
        $birth_date,
        $otp
    ]);
    
    if ($result) {
        $startingPoints = max(0, (int)configValue('REGISTRATION_STARTING_POINTS', '0'));
        $walletQuery = "INSERT INTO users_wallet (user_id, balance, points) 
                        VALUES (?, 0.00, ?)";
        
        try {
            executeQuery($walletQuery, [$user_id, $startingPoints]);
        } catch (Exception $e) {
            error_log("Wallet creation error: " . $e->getMessage());
        }



          $commission_wallets = "INSERT INTO commission_wallets (user_id, commission) 
                        VALUES (?, 0.00)";
        
        try {
            $commission_walletsResult = executeQuery($commission_wallets, [$user_id]);
            
        } catch (Exception $e) {
            error_log("Wallet creation error: " . $e->getMessage());
        }



        
        
        // حفظ معلومات المستخدم في الجلسة
        $_SESSION['user_id'] = $user_id;
        $_SESSION['temp_email'] = $email;
        $_SESSION['temp_phone'] = $phone;
        $_SESSION['otp'] = $otp;
        $_SESSION['first_name']=$first_name;
        // تسجيل للتأكد
        $_SESSION['csrf_token'] = bin2hex(random_bytes(32));
        
        // إرسال OTP عبر WhatsApp
        //$sendResult = sendOTP($phone, $otp);
        
        //if (!$sendResult['success']) {
           // error_log("Failed to send OTP: " . $sendResult['message']);
            // لا نوقف التسجيل، فقط نسجل الخطأ
        //}
       
        respondJson([
            'success' => true, 
            'message' => 'تم التسجيل بنجاح. جاري التحويل لصفحة التحقق...',
            'redirect' => 'index.php'
        ]);
    } else {
        respondError('حدث خطأ أثناء التسجيل', 500);
    }
    
} catch (Exception $e) {
    error_log("Registration Error: " . $e->getMessage());
    error_log("Stack Trace: " . $e->getTraceAsString());
    respondError('حدث خطأ أثناء التسجيل', 500);
}
?>
