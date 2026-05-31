<?php
require_once __DIR__ . '/../config/database.php';

$userId = requireAuthenticatedUser();
verifyCsrfToken();
enforceRateLimit('contacts_add:' . $userId, 30, 60);

try {
    $input = readJsonBody(1024 * 1024);
    $name = cleanText($input['name'] ?? '', 120);
    $dataArr = $input['data'] ?? [];

    if ($name === '') {
        respondError('اسم القائمة مطلوب', 422);
    }
    if (!is_array($dataArr) || count($dataArr) === 0) {
        respondError('قائمة جهات الاتصال فارغة', 422);
    }
    if (count($dataArr) > 5000) {
        respondError('الحد الأقصى 5000 جهة اتصال في الطلب الواحد', 413);
    }

    $normalized = [];
    foreach ($dataArr as $row) {
        if (!is_array($row)) {
            continue;
        }

        $identifier = preg_replace('/[^\d+]/', '', (string)($row['identifier'] ?? $row['phone'] ?? ''));
        $displayName = cleanText($row['name'] ?? '', 120);
        if ($identifier === '' || strlen($identifier) < 5 || strlen($identifier) > 20) {
            continue;
        }

        $normalized[] = ['identifier' => $identifier, 'name' => $displayName];
    }

    if (!$normalized) {
        respondError('لا توجد جهات اتصال صالحة', 422);
    }

    $json = json_encode($normalized, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    if ($json === false) {
        respondError('تعذر تجهيز البيانات', 400);
    }

    $db = getDB();
    $stmt = $db->prepare(
        'INSERT INTO contacts (`user_id`, `name`, `platform`, `type`, `count`, `data`, `created_at`, `updated_at`)
         VALUES (:uid, :name, :platform, :type, :count, :data, NOW(), NOW())'
    );
    $stmt->execute([
        ':uid' => $userId,
        ':name' => $name,
        ':platform' => 'whatsapp',
        ':type' => 'csv',
        ':count' => count($normalized),
        ':data' => $json,
    ]);

    respondJson(['success' => true, 'id' => $db->lastInsertId(), 'count' => count($normalized)]);
} catch (Throwable $e) {
    error_log('contacts_add error: ' . $e->getMessage());
    respondError('تعذر حفظ جهات الاتصال', 500);
}
