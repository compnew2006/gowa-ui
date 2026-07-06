<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: POST');
header('Access-Control-Allow-Headers: Content-Type');

require_once '../config/database.php';

// إنشاء اتصال mysqli
$conn = new mysqli(DB_HOST, DB_USER, DB_PASS, DB_NAME);

// التحقق من الاتصال
if ($conn->connect_error) {
    echo json_encode([
        'success' => false,
        'message' => 'فشل الاتصال بقاعدة البيانات'
    ]);
    exit();
}

$conn->set_charset("utf8mb4");

// استقبال البيانات
$data = json_decode(file_get_contents('php://input'), true);

// دعم كلا الصيغتين: input أو referral_code
$input = isset($data['referral_code']) ? trim($data['referral_code']) : 
         (isset($data['input']) ? trim($data['input']) : '');

if (empty($input)) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب إدخال رمز الإحالة'
    ]);
    exit();
}

// البحث في قاعدة البيانات عن طريق رمز الإحالة
$stmt = $conn->prepare("
    SELECT 
        user_id,
        first_name,
        last_name,
        email
    FROM users 
    WHERE user_id = ?
    LIMIT 1
");

$stmt->bind_param("s", $input);
$stmt->execute();
$result = $stmt->get_result();

if ($result->num_rows > 0) {
    $user = $result->fetch_assoc();
    
    echo json_encode([
        'success' => true,
        'message' => 'تم العثور على المستخدم',
        'first_name' => $user['first_name'],
        'last_name' => $user['last_name'],
        'email' => $user['email'],
        'referral_code' => $data['referral_code']
    ]);
} else {
    echo json_encode([
        'success' => false,
        'message' => 'رمز الإحالة غير موجود'
    ]);
}

$stmt->close();
$conn->close();
?>
