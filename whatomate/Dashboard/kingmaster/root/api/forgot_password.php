<?php
session_start();
header('Content-Type: application/json');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول'
    ]);
    exit;
}

require_once '../config/database.php';

$user_id = $_SESSION['user_id'];

try {
    $conn = getDB();
    
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
    
    // توليد كلمة مرور عشوائية (8 أحرف: أرقام وحروف)
    $random_password = substr(str_shuffle('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'), 0, 8);
    
    // تشفير كلمة المرور
    $hashed_password = password_hash($random_password, PASSWORD_DEFAULT);
    
    // تحديث كلمة المرور في قاعدة البيانات
    $stmt = $conn->prepare("UPDATE users SET password = :password WHERE user_id = :user_id");
    $stmt->execute([
        ':password' => $hashed_password,
        ':user_id' => $user_id
    ]);
    
    // تنسيق رقم الهاتف (إزالة أي رموز غير رقمية)
    $phone = preg_replace('/[^0-9]/', '', $user['phone']);
    
    // إعداد رسالة الواتساب
    $message = "مرحباً " . $user['first_name'] . ",\n\n";
    $message .= "تم إعادة تعيين كلمة المرور الخاصة بك بنجاح.\n\n";
    $message .= "🔐 كلمة المرور الجديدة: " . $random_password . "\n\n";
    $message .= "يرجى تسجيل الدخول وتغيير كلمة المرور فوراً لأسباب أمنية.\n\n";
    $message .= "شكراً لك - فريق Kingmaster";
    
    // معلومات API الواتساب
        // إرسال الطلب
/*        
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
*/

// WhatsApp API (Evolution)
$payload = json_encode([
    "number" => $phone,
    "text"   => $message
], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

$ch = curl_init();
curl_setopt($ch, CURLOPT_URL, "http://localhost:8080/message/sendText/wa_1772808304856");
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);

curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Content-Type: application/json',
    'apikey: 429683C4C977415CAAFCCE10F7D57E11',
    'Content-Length: ' . strlen($payload)
]);

curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
// curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false); // uncomment only if your SSL is broken

$response = curl_exec($ch);
curl_close($ch);

$api_response = json_decode($response, true);
        
if (isset($api_response['status']) && $api_response['status'] == 'PENDING') {
    echo json_encode([
        'success' => true,
        'message' => 'تم إرسال كلمة المرور الجديدة إلى رقم الواتساب المسجل',
        'phone_last_digits' => substr($phone, -4)
    ]);
    exit;
}

echo json_encode([
    'success' => false,
    'message' => 'تم تغيير كلمة المرور ولكن فشل إرسال الرسالة',
    'error' => $api_response['message'] ?? 'خطأ غير معروف'
]);
exit;

} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
    exit;
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ: ' . $e->getMessage()
    ]);
    exit;
}
?>
