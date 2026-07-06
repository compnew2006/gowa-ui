<?php
session_start();
require_once 'config/database.php';
require_once 'includes/send_otp.php';

header('Content-Type: application/json; charset=UTF-8');

// التحقق من وجود معلومات المستخدم المؤقتة
if (!isset($_SESSION['user_id']) || !isset($_SESSION['temp_phone'])) {
    echo json_encode([
        'success' => false,
        'message' => 'جلسة غير صالحة'
    ]);
    exit;
}

$user_id = $_SESSION['user_id'];
$phone = $_SESSION['temp_phone'];

try {
    // توليد OTP جديد
    $otp = str_pad(rand(0, 999999), 6, '0', STR_PAD_LEFT);
    
    // تحديث OTP في قاعدة البيانات
    $query = "UPDATE users SET otp = ?, otp_created_at = NOW() WHERE user_id = ?";
    $result = executeQuery($query, [$otp, $user_id]);
    
    if (!$result) {
        echo json_encode([
            'success' => false,
            'message' => 'فشل تحديث رمز التحقق'
        ]);
        exit;
    }
    
    // حفظ OTP الجديد في الجلسة
    $_SESSION['otp'] = $otp;
    
    // إرسال OTP عبر WhatsApp
    $sendResult = sendOTP($phone, $otp);
    
    if ($sendResult['success']) {
        echo json_encode([
            'success' => true,
            'message' => 'تم إرسال رمز جديد بنجاح',
            'otp' => $phone // للاختبار فقط - احذف في الإنتاج
        ]);
    } else {
        // حتى لو فشل الإرسال، نعتبر العملية ناجحة لأن OTP تم تحديثه
        echo json_encode([
            'success' => true,
            'message' => 'تم إنشاء رمز جديد',
            'otp' => $phone,
            'send_error' => $sendResult['response']
        ]);
    }
    
} catch (Exception $e) {
    error_log("Resend OTP Error: " . $e->getMessage());
    echo json_encode([
        'success' => false,
        'message' => $phone
    ]);
}
?>
