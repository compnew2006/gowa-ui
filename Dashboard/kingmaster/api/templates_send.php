<?php
// templates_send.php - send stored template to Facebook Messenger
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') { http_response_code(204); exit; }

require_once '../config/database.php';

function respond($data, $code = 200) {
    http_response_code($code);
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

$input = json_decode(file_get_contents('php://input'), true) ?: [];
$templateId = isset($input['template_id']) ? (int)$input['template_id'] : 0;
$recipientId = trim((string)($input['recipient_id'] ?? ''));
$accessToken = trim((string)($input['page_access_token'] ?? ''));

if ($recipientId === '' || $accessToken === '') {
    respond([ 'success' => false, 'message' => 'recipient_id و page_access_token مطلوبة' ], 400);
}

$payload = null;
$type = 'generic';
if ($templateId > 0) {
    $row = fetchRow('SELECT type, payload FROM messenger_templates WHERE id = ?', [ $templateId ]);
    if (!$row) respond([ 'success' => false, 'message' => 'القالب غير موجود' ], 404);
    $type = $row['type'] ?: 'generic';
    $payload = json_decode($row['payload'], true);
} else if (isset($input['payload']) && is_array($input['payload'])) {
    $payload = $input['payload'];
    $type = isset($input['type']) ? (string)$input['type'] : 'generic';
}

if (!$payload) { respond([ 'success' => false, 'message' => 'لا يوجد payload لإرساله' ], 400); }

$body = [
    'messaging_type' => 'MESSAGE_TAG',
    'tag' => 'CUSTOMER_FEEDBACK',
    'recipient' => [ 'id' => $recipientId ],
    'message' => [
        'attachment' => [
            'type' => 'template',
            'payload' => array_merge([ 'template_type' => $type ], $payload)
        ]
    ]
];

$url = 'https://graph.facebook.com/v15.0/me/messages?access_token=' . rawurlencode($accessToken);

$ch = curl_init($url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_HTTPHEADER, [ 'Content-Type: application/json' ]);
curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
$response = curl_exec($ch);
$errno = curl_errno($ch);
$http = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

if ($errno) {
    respond([ 'success' => false, 'message' => 'خطأ في الاتصال بخدمة فيسبوك', 'code' => $errno ], 502);
}

$fb = json_decode($response, true);
if ($http >= 200 && $http < 300) {
    respond([ 'success' => true, 'message' => 'تم الإرسال', 'facebook' => $fb ]);
}
respond([ 'success' => false, 'message' => 'فشل الإرسال', 'facebook' => $fb, 'status' => $http ], 400);
