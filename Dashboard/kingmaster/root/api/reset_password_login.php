<?php
session_start();
header('Content-Type: application/json; charset=utf-8');
require_once __DIR__ . '/../config/database.php';

function respond($data, $code = 200) {
    http_response_code($code);
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

// Accept JSON or form-encoded
$inputRaw = file_get_contents('php://input');
$input = json_decode($inputRaw, true);
if (!is_array($input)) { $input = $_POST; }

$email = isset($input['email']) ? trim((string)$input['email']) : '';
if ($email === '' || !isValidEmail($email)) {
    respond([ 'success' => false, 'message' => 'يرجى إدخال بريد إلكتروني صحيح' ], 400);
}

try {
    $db = getDB();

    // Lookup user by email
    $stmt = $db->prepare('SELECT user_id, first_name, phone FROM users WHERE email = :email LIMIT 1');
    $stmt->execute([':email' => $email]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$user) {
        // To prevent user enumeration, return generic message
        respond([ 'success' => true, 'message' => 'إذا كان البريد الإلكتروني مسجلاً، سيتم إرسال كلمة مرور جديدة عبر واتساب' ]);
    }

    // Generate random password (10 chars: letters + digits)
    $alphabet = '23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';
    $len = strlen($alphabet);
    $newPlain = '';
    for ($i = 0; $i < 10; $i++) { $newPlain .= $alphabet[random_int(0, $len - 1)]; }

    // Hash password using helper (argon2id/bcrypt)
    $hashed = hashPassword($newPlain);

    // Update DB
    $upd = $db->prepare('UPDATE users SET password = :p WHERE user_id = :uid');
    $upd->execute([':p' => $hashed, ':uid' => $user['user_id']]);

    // Prepare WhatsApp message
    $phone = preg_replace('/[^0-9]/', '', (string)$user['phone']);
    if ($phone === '') {
        respond([ 'success' => false, 'message' => 'تم تحديث كلمة المرور، لكن رقم الهاتف غير متوفر لإرسال الرسالة' ], 200);
    }

    $msg  = 'مرحباً ' . ($user['first_name'] ?? '') . "\n\n";
    $msg .= "تم إعادة تعيين كلمة المرور الخاصة بك بنجاح.\n\n";
    $msg .= "🔐 كلمة المرور الجديدة: " . $newPlain . "\n\n";
    $msg .= "يرجى تسجيل الدخول وتغيير كلمة المرور فوراً لأسباب أمنية.\n\n";
    $msg .= "شكراً لك - فريق Kingmaster";
/*
    // WhatsApp API (same as used elsewhere in project)
  $payload = json_encode([
    "number" => $phone,
    "type" => "text",
    "message" => $msg,
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
    "text"   => $msg
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
 
$api = json_decode($response, true);
if (isset($api['status']) && $api['status'] == 'PENDING') {
    respond([ 'success' => true, 'message' => 'تم إرسال كلمة المرور الجديدة إلى رقم الواتساب المسجل', 'phone_last_digits' => substr($phone, -4) ]);
        }
    respond([ 'success' => true, 'message' => 'تم تحديث كلمة المرور. قد يتأخر إرسال رسالة الواتساب قليلاً.' ]);
    

    respond([ 'success' => true, 'message' => 'تم تحديث كلمة المرور. تعذر إرسال رسالة الواتساب حالياً.' ]);
} catch (Throwable $e) {
    respond([ 'success' => false, 'message' => 'حدث خطأ غير متوقع' ], 500);
}
