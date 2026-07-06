<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json; charset=UTF-8');

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    echo json_encode(['success' => false, 'message' => 'طريقة الطلب غير صحيحة']);
    exit;
}

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'جلسة غير صالحة']);
    exit;
}

$input_otp = sanitizeInput($_POST['otp'] ?? '');
$user_id = $_SESSION['user_id'];

if (empty($input_otp) || strlen($input_otp) != 6) {
    echo json_encode(['success' => false, 'message' => 'رمز التحقق يجب أن يكون 6 أرقام']);
    exit;
}

try {
    // جلب بيانات المستخدم و OTP من قاعدة البيانات
    $user = fetchRow("SELECT id, otp, otp_created_at, is_verified FROM users WHERE user_id = ?", [$user_id]);
    
    if (!$user) {
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        exit;
    }
    
    if ($user['is_verified'] == 1) {
        echo json_encode(['success' => false, 'message' => 'الحساب تم تفعيله بالفعل']);
        exit;
    }
    
    // التحقق من صلاحية OTP (15 دقيقة)
    $otp_time = strtotime($user['otp_created_at']);
    $current_time = time();
    $time_diff = ($current_time - $otp_time) / 60; // بالدقائق
    
   
    
    // مقارنة OTP
    if ($input_otp !== $user['otp']) {
        echo json_encode(['success' => false, 'message' => 'رمز التحقق غير صحيح']);
        exit;
    }
    
    // تحديث حالة المستخدم إلى "تم التحقق"
    $update_query = "UPDATE users SET is_verified = 1, otp = NULL, otp_created_at = NULL WHERE user_id = ?";
    $result = executeQuery($update_query, [$user_id]);
    
    if ($result) {
        // حذف البيانات المؤقتة
        unset($_SESSION['user_id']);
        unset($_SESSION['temp_email']);
        unset($_SESSION['temp_phone']);
        unset($_SESSION['otp']);
        
        echo json_encode([
            'success' => true,
            'message' => 'تم تفعيل حسابك بنجاح! يمكنك الآن تسجيل الدخول',
            'redirect' => 'login.php'
        ]);
    } else {
        echo json_encode(['success' => false, 'message' => 'حدث خطأ أثناء التحقق']);
    }
    
} catch (Exception $e) {
    echo json_encode(['success' => false, 'message' => 'خطأ: ' . $e->getMessage()]);
}
?>
