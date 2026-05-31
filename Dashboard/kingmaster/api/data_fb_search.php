<?php
require_once __DIR__ . '/db_local.php';

$user_id = requireAuthenticatedUser();
enforceRateLimit('data_fb_search:' . $user_id, 30, 60);

$payload = readJsonBody(262144);
$action = $payload['action'] ?? 'count';
if (!in_array($action, ['count', 'extract'], true)) {
    respondError('إجراء غير صالح', 400);
}

$maxExtractLimit = (int)configValue('DATA_FB_MAX_EXTRACT_LIMIT', '1000');
$requestedLimit = isset($payload['limit']) ? (int)$payload['limit'] : 100;
$requestedLimit = max(1, min($requestedLimit, $maxExtractLimit));

$type = $payload['type'] ?? 'keywords';
$country = $payload['country'] ?? 'all';
$relationship = cleanText($payload['maritalStatus'] ?? '', 80);
$where = [];
$params = [];

function addLikeGroup($field, $arr, &$where, &$params) {
    $allowed = ['work', 'education', 'location'];
    if (!in_array($field, $allowed, true) || !is_array($arr)) {
        return;
    }
    $group = [];
    foreach (array_slice($arr, 0, 20) as $v) {
        $v = cleanText($v, 80);
        if ($v === '') {
            continue;
        }
        $group[] = "$field LIKE ?";
        $params[] = '%' . $v . '%';
    }
    if ($group) {
        $where[] = '(' . implode(' OR ', $group) . ')';
    }
}

if ($country !== 'all') {
    $countriesMap = [
        'EG' => ['Egypt', 'مصر'],
        'SA' => ['Saudi Arabia', 'السعودية', 'KSA'],
        'AE' => ['UAE', 'United Arab Emirates', 'الامارات', 'الإمارات'],
        'KW' => ['Kuwait', 'الكويت'],
        'QA' => ['Qatar', 'قطر'],
        'BH' => ['Bahrain', 'البحرين'],
        'OM' => ['Oman', 'عمان'],
        'JO' => ['Jordan', 'الأردن'],
        'LB' => ['Lebanon', 'لبنان'],
        'SY' => ['Syria', 'سوريا'],
        'IQ' => ['Iraq', 'العراق'],
        'PS' => ['Palestine', 'فلسطين'],
        'YE' => ['Yemen', 'اليمن'],
        'LY' => ['Libya', 'ليبيا'],
        'TN' => ['Tunisia', 'تونس'],
        'DZ' => ['Algeria', 'الجزائر'],
        'MA' => ['Morocco', 'المغرب'],
        'SD' => ['Sudan', 'السودان'],
        'US' => ['United States', 'USA', 'الولايات المتحدة'],
        'GB' => ['United Kingdom', 'UK', 'بريطانيا'],
        'FR' => ['France', 'فرنسا'],
        'DE' => ['Germany', 'ألمانيا'],
        'IT' => ['Italy', 'إيطاليا'],
        'ES' => ['Spain', 'إسبانيا'],
        'TR' => ['Turkey', 'تركيا'],
    ];
    if (!isset($countriesMap[$country])) {
        respondError('الدولة غير صالحة', 400);
    }
    $parts = [];
    foreach ($countriesMap[$country] as $name) {
        $parts[] = 'location LIKE ?';
        $params[] = '%' . $name . '%';
    }
    $where[] = '(' . implode(' OR ', $parts) . ')';
}

addLikeGroup('work', $payload['work'] ?? [], $where, $params);
addLikeGroup('education', $payload['education'] ?? [], $where, $params);
addLikeGroup('location', $payload['location'] ?? [], $where, $params);
addLikeGroup('location', $payload['city'] ?? [], $where, $params);

if ($relationship !== '') {
    $where[] = 'LOWER(relationship) LIKE LOWER(?)';
    $params[] = '%' . $relationship . '%';
}

if ($type === 'uids') {
    $uids = is_array($payload['uids'] ?? null) ? array_slice($payload['uids'], 0, 1000) : [];
    $uids = array_values(array_filter(array_map(function ($uid) {
        return cleanText($uid, 80);
    }, $uids)));
    if ($uids) {
        $uidColumn = ($payload['uidType'] ?? 'fb_id') === 'id' ? 'id' : 'fb_id';
        $placeholders = implode(',', array_fill(0, count($uids), '?'));
        $where[] = "$uidColumn IN ($placeholders)";
        $params = array_merge($params, $uids);
    }
}

$whereClause = $where ? 'WHERE ' . implode(' AND ', $where) : '';

try {
    $countSql = "SELECT COUNT(*) FROM (SELECT id FROM data_fb $whereClause LIMIT $requestedLimit) limited";
    $stmtCount = $pdo->prepare($countSql);
    $stmtCount->execute($params);
    $totalCount = (int)$stmtCount->fetchColumn();

    if ($action === 'count') {
        respondJson(['success' => true, 'count' => $totalCount, 'limit' => $requestedLimit]);
    }

    verifyCsrfToken($payload['csrf_token'] ?? null);

    $pdo->beginTransaction();
    $stmtUser = $pdo->prepare('SELECT points FROM users WHERE user_id = ? FOR UPDATE');
    $stmtUser->execute([$user_id]);
    $user = $stmtUser->fetch(PDO::FETCH_ASSOC);

    if (!$user || (int)$user['points'] < $totalCount) {
        $pdo->rollBack();
        respondError('رصيد النقاط غير كافٍ. المطلوب: ' . $totalCount . '، المتاح: ' . ($user['points'] ?? 0), 402);
    }

    $dataSql = "SELECT id, fb_id, name, mobile_phone, gender, birthday, location, relationship, email, work, education FROM data_fb $whereClause LIMIT $requestedLimit";
    $stmtData = $pdo->prepare($dataSql);
    $stmtData->execute($params);
    $data = $stmtData->fetchAll(PDO::FETCH_ASSOC);
    $actualCount = count($data);

    if ($actualCount > 0) {
        $stmtUpdate = $pdo->prepare('UPDATE users SET points = points - ? WHERE user_id = ?');
        $stmtUpdate->execute([$actualCount, $user_id]);
        $stmtLog = $pdo->prepare("INSERT INTO point_use (user_id, points, action) VALUES (?, ?, 'data_fb_extract')");
        $stmtLog->execute([$user_id, $actualCount]);
    }

    $pdo->commit();

    respondJson([
        'success' => true,
        'count' => $actualCount,
        'limit' => $requestedLimit,
        'points_charged' => $actualCount,
        'data' => $data,
    ]);
} catch (Throwable $e) {
    if (isset($pdo) && $pdo instanceof PDO && $pdo->inTransaction()) {
        $pdo->rollBack();
    }
    error_log('data_fb_search error: ' . $e->getMessage());
    respondError('حدث خطأ في النظام', 500);
}
