<?php
session_start();
header('Content-Type: application/json');
require_once '../config/database.php';

if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول أولاً'
    ]);
    exit;
}

$user_id = $_SESSION['user_id'];
$conn = getDB();

$data = json_decode(file_get_contents('php://input'), true);
$action = $data['action'] ?? '';

if ($action === 'send_otp') {
    // توليد OTP عشوائي من 6 أرقام
    $otp = rand(100000, 999999);
    
    // حذف أي OTP قديم للمستخدم
    $stmt = $conn->prepare("DELETE FROM otp_wall WHERE user_id = :user_id");
    $stmt->execute([':user_id' => $user_id]);
    
    // حفظ OTP الجديد
    $stmt = $conn->prepare("INSERT INTO otp_wall (user_id, otp) VALUES (:user_id, :otp)");
    $stmt->execute([
        ':user_id' => $user_id,
        ':otp' => $otp
    ]);
    
    // جلب رقم هاتف المستخدم
    $stmt = $conn->prepare("SELECT phone, first_name FROM users WHERE user_id = :user_id");
    $stmt->execute([':user_id' => $user_id]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$user) {
        echo json_encode([
            'success' => false,
            'message' => 'المستخدم غير موجود'
        ]);
        exit;
    }
    
    if (empty($user['phone'])) {
        echo json_encode([
            'success' => false,
            'message' => 'لا يوجد رقم هاتف مسجل لهذا الحساب'
        ]);
        exit;
    }
    
    // تنسيق رقم الهاتف
    $phone = preg_replace('/[^0-9]/', '', $user['phone']);
    
    // بناء رسالة الواتساب
    $message = "مرحباً " . $user['first_name'] . ",\n\n";
    $message .= "🔐 رمز التحقق من المحفظة: " . $otp . "\n\n";
    $message .= "⏰ صالح لمدة 5 دقائق فقط\n\n";
    $message .= "⚠️ لا تشارك هذا الرمز مع أي شخص\n\n";
    $message .= "شكراً لك - فريق Kingmaster";
    
    // معلومات API الواتساب
$payload = json_encode([
    "number" => $phone,
    "type" => "text",
    "message" => $message,
    "instance_id" => "6967AAB9ADA6E",
        "access_token" => "6604ac2316788"
]);


$ch = curl_init();
curl_setopt($ch, CURLOPT_URL, "https://king-master.pro/api/send");
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);

curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Content-Type: application/json',
    'Content-Length: ' . strlen($payload)
]);

curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);

$response = curl_exec($ch);
curl_close($ch);


    echo json_encode([
        'success' => true,
        'message' => 'تم إرسال رمز التحقق إلى رقم هاتفك',
        'phone_last_digits' => substr($phone, -4)
    ]);
    exit;
}

if ($action === 'verify_otp') {
    $entered_otp = $data['otp'] ?? '';
    
    if (empty($entered_otp)) {
        echo json_encode([
            'success' => false,
            'message' => 'يرجى إدخال رمز التحقق'
        ]);
        exit;
    }
    
    // جلب OTP من قاعدة البيانات
    $stmt = $conn->prepare("
        SELECT otp, created_at 
        FROM otp_wall 
        WHERE user_id = :user_id 
        ORDER BY created_at DESC 
        LIMIT 1
    ");
    $stmt->execute([':user_id' => $user_id]);
    $otp_record = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$otp_record) {
        echo json_encode([
            'success' => false,
            'message' => 'لم يتم العثور على رمز تحقق. يرجى طلب رمز جديد'
        ]);
        exit;
    }
    
    // التحقق من صلاحية الوقت (5 دقائق)
    $created_time = strtotime($otp_record['created_at']);
    $current_time = time();
    $time_diff = ($current_time - $created_time) / 60; // بالدقائق
    
 
    // التحقق من تطابق OTP
    if ($entered_otp != $otp_record['otp']) {
        echo json_encode([
            'success' => false,
            'message' => 'رمز التحقق غير صحيح'
        ]);
        exit;
    }
    
    // حذف OTP بعد التحقق الناجح
    $stmt = $conn->prepare("DELETE FROM otp_wall WHERE user_id = :user_id");
    $stmt->execute([':user_id' => $user_id]);
    
    echo json_encode([
        'success' => true,
        'message' => 'تم التحقق بنجاح'
    ]);
    exit;
}

echo json_encode([
    'success' => false,
    'message' => 'إجراء غير معروف'
]);
?>
