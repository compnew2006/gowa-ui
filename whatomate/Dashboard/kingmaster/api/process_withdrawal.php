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

$amount = floatval($data['amount'] ?? 0);
$type = $data['type'] ?? '';
$details = $data['details'] ?? '';

// التحقق من البيانات
if ($amount <= 0) {
    echo json_encode([
        'success' => false,
        'message' => 'يرجى إدخال مبلغ صحيح'
    ]);
    exit;
}

// جلب رصيد العمولات
require_once '../includes/functions.php';
$commission = getcommission_walletsById($user_id);
$available_amount = floatval($commission['commission']);

if ($amount > $available_amount) {
    echo json_encode([
        'success' => false,
        'message' => 'المبلغ المطلوب أكبر من رصيدك المتاح'
    ]);
    exit;
}

// جلب بيانات المستخدم
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

$phone5 = preg_replace('/[^0-9]/', '', $user['phone']);

// حفظ طلب السحب في قاعدة البيانات
$stmt = $conn->prepare("
    INSERT INTO withdrawals (user_id, amount, withdrawal_type, withdrawal_details, status) 
    VALUES (:user_id, :amount, :type, :details, 'pending')
");
$stmt->execute([
    ':user_id' => $user_id,
    ':amount' => $amount,
    ':type' => $type,
    ':details' => $details
]);

// بناء رسالة الواتساب
$message = "🔔 طلب سحب جديد\n\n";
$message .= "━━━━━━━━━━━━━━━━━━━\n\n";
$message .= "👤 رقم الحساب: " . $user_id . "\n";
$message .= "📝 اسم العميل: " . $user['first_name'] . "\n";
$message .= "📞 رقم العميل: " . $phone5 . "\n\n";
$message .= "━━━━━━━━━━━━━━━━━━━\n\n";
$message .= "💰 المبلغ المطلوب: " . number_format($amount, 2) . " جنيه\n\n";
$message .= "━━━━━━━━━━━━━━━━━━━\n\n";
$message .= "📌 طريقة التحويل: " . $type . "\n\n";
$message .= "ℹ️ بيانات التحويل:\n" . $details . "\n\n";
$message .= "━━━━━━━━━━━━━━━━━━━\n\n";
$message .= "⏰ التاريخ: " . date('Y-m-d H:i:s') . "\n\n";
$message .= "شكراً - فريق Kingmaster";

// إرسال الواتساب
$phone = "201025385693";
$api_url = 'https://king-master.pro/api/send';
$instance_id = '6926C9A5115D9';
$access_token = '6604ac2316788';

$url = $api_url . '?' . http_build_query([
    'number' => $phone,
    'type' => 'text',
    'message' => $message,
    'instance_id' => $instance_id,
    'access_token' => $access_token
]);

$ch = curl_init();
curl_setopt($ch, CURLOPT_URL, $url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
$response = curl_exec($ch);
curl_close($ch);

echo json_encode([
    'success' => true,
    'message' => 'تم إرسال طلب السحب بنجاح'
]);
?>
