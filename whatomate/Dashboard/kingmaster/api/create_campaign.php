<?php
require_once __DIR__ . '/../config/database.php';
require_once __DIR__ . '/../includes/notification_helper.php';

$user_id = requireAuthenticatedUser();
enforceRateLimit('create_campaign:' . $user_id, 20, 60);

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('Method not allowed', 405);
}

$data = readJsonBody();
verifyCsrfToken($data['csrf_token'] ?? null);

$name = cleanText($data['name'] ?? '', 120);
$accounts = $data['accounts'] ?? [];
$page_url = filter_var(trim((string)($data['page_url'] ?? '')), FILTER_SANITIZE_URL);
$range = max(1, min((int)($data['range'] ?? 10), 100000));
$interval_id = isset($data['interval_id']) ? (int)$data['interval_id'] : null;
$tools = cleanText($data['tools'] ?? '', 80);
$paltform = cleanText($data['paltform'] ?? '', 40);
$id_action = cleanText($data['id_action'] ?? '', 255);
$contact = cleanText($data['contact'] ?? '', 255);
$speed = cleanText($data['speed'] ?? 'slow', 20);

$allowedSpeeds = ['slow', 'medium', 'fast'];
if (!in_array($speed, $allowedSpeeds, true)) {
    $speed = 'slow';
}

$toolTypes = [
    'Extract Contacts WA' => 'Extract',
    'Extract Messages WA' => 'Extract',
    'Extract Members WA' => 'Extract',
    'Wa Filter' => 'Extract',
    'Wa Sender' => 'Send',
    'sech_pg' => 'Extract',
    'sech_gb' => 'Extract',
    'sech_pp' => 'Extract',
    'like_post' => 'Send',
    'comments_post' => 'Reply',
    'mmbers_gb' => 'Extract',
];

if ($name === '' || !isset($toolTypes[$tools])) {
    respondError('يرجى ملء جميع الحقول المطلوبة', 400);
}

if (!is_array($accounts) || count($accounts) === 0 || count($accounts) > 50) {
    respondError('يجب اختيار حساب واحد على الأقل', 400);
}

if (in_array($tools, ['Extract Members WA', 'sech_pg', 'sech_gb', 'sech_pp', 'like_post', 'comments_post', 'mmbers_gb'], true)) {
    $id_action = cleanText($data['keyword'] ?? $id_action, 255);
    if ($id_action === '') {
        respondError('معرّف أو كلمة البحث مطلوبة', 400);
    }
}

if (in_array($tools, ['Wa Filter', 'Wa Sender'], true) && $contact === '') {
    respondError('قائمة جهات الاتصال مطلوبة', 400);
}

try {
    $conn = getDB();
    $campaign_id = (string)random_int(100000000, 999999999) . time();
    $token_json = json_encode($accounts, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

    $stmt = $conn->prepare("
        INSERT INTO campaigns
        (user_id, campaign_id, name, status, paltform, tool, type_tools, count, true_count, false_count, token, `range`, `interval`, page_url, pram1, contact, speed)
        VALUES
        (:user_id, :campaign_id, :name, 'pending', :paltform, :tools, :type_tools, 0, 0, 0, :token, :range, :interval, :page_url, :pram1, :contact, :speed)
    ");
    $stmt->execute([
        ':user_id' => $user_id,
        ':campaign_id' => $campaign_id,
        ':name' => $name,
        ':token' => $token_json,
        ':range' => $range,
        ':interval' => $interval_id,
        ':page_url' => $page_url,
        ':tools' => $tools,
        ':paltform' => $paltform,
        ':pram1' => $id_action,
        ':contact' => $contact,
        ':speed' => $speed,
        ':type_tools' => $toolTypes[$tools],
    ]);

    addNotification($user_id, 'حملة جديدة', "تم إنشاء الحملة '{$name}' بنجاح", 'success');
    respondJson(['success' => true, 'message' => 'تم إنشاء الحملة بنجاح', 'campaign_id' => $campaign_id]);
} catch (Throwable $e) {
    error_log('create_campaign error: ' . $e->getMessage());
    respondError('حدث خطأ أثناء إنشاء الحملة', 500);
}
