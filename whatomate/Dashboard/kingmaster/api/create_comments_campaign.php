<?php
require_once __DIR__ . '/../config/database.php';

$user_id = requireAuthenticatedUser();
enforceRateLimit('create_comments_campaign:' . $user_id, 20, 60);

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('Method not allowed', 405);
}

$data = readJsonBody();
verifyCsrfToken($data['csrf_token'] ?? null);

$name = cleanText($data['name'] ?? '', 120);
$accounts = $data['accounts'] ?? [];
$content_id = (int)($data['content_id'] ?? 0);
$interval_id = isset($data['interval_id']) ? (int)$data['interval_id'] : null;
$can_like = !empty($data['can_like']) ? 1 : 0;

if ($name === '' || !is_array($accounts) || count($accounts) === 0 || count($accounts) > 50 || $content_id <= 0) {
    respondError('يرجى ملء جميع الحقول المطلوبة', 400);
}

try {
    $conn = getDB();
    $campaign_id = (string)random_int(100000000, 999999999) . time();
    $token_json = json_encode($accounts, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

    $stmt = $conn->prepare("
        INSERT INTO campaigns
        (user_id, campaign_id, name, status, paltform, tool, type_tools, count, true_count, false_count, token, content_id, `interval`, can_like)
        VALUES
        (:user_id, :campaign_id, :name, 'pending', 'Facebook', 'Reply Comments FB', 'Reply', 0, 0, 0, :token, :content_id, :interval, :can_like)
    ");
    $stmt->execute([
        ':user_id' => $user_id,
        ':campaign_id' => $campaign_id,
        ':name' => $name,
        ':token' => $token_json,
        ':content_id' => $content_id,
        ':interval' => $interval_id,
        ':can_like' => $can_like,
    ]);

    respondJson(['success' => true, 'message' => 'تم إنشاء الحملة بنجاح', 'campaign_id' => $campaign_id]);
} catch (Throwable $e) {
    error_log('create_comments_campaign error: ' . $e->getMessage());
    respondError('حدث خطأ أثناء إنشاء الحملة', 500);
}
