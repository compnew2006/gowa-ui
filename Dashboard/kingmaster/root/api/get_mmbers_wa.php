<?php
header('Content-Type: application/json');
require_once __DIR__ . '/../includes/functions.php';

try {
    $db = getDB();
    $campaignId = isset($_GET['campaign_id']) ? (int)$_GET['campaign_id'] : (int)($_POST['campaign_id'] ?? 0);
    if ($campaignId <= 0) {
        echo json_encode(['success' => false, 'message' => 'campaign_id is required']);
        exit;
    }

    $page = isset($_GET['page']) ? (int)$_GET['page'] : (int)($_POST['page'] ?? 1);
    $perPage = isset($_GET['per_page']) ? (int)$_GET['per_page'] : (int)($_POST['per_page'] ?? 25);
    if ($perPage < 1) $perPage = 25;
    if ($perPage > 200) $perPage = 200;
    if ($page < 1) $page = 1;
    $offset = ($page - 1) * $perPage;

    $q = trim((string)($_GET['q'] ?? ($_POST['q'] ?? '')));
    $where = 'WHERE campaign_id = :cid';
    $params = [':cid' => $campaignId];
    if ($q !== '') {
        // NOTE: MySQL PDO does not allow reusing the same named placeholder twice.
        // Use distinct placeholders for each occurrence.
        $where .= ' AND (name LIKE :q1 OR phone LIKE :q2)';
        $params[':q1'] = '%' . $q . '%';
        $params[':q2'] = '%' . $q . '%';
    }

    // Count total
    $stmt = $db->prepare("SELECT COUNT(*) AS total FROM wa_members_gb $where");
    $stmt->execute($params);
    $total = (int)($stmt->fetch(PDO::FETCH_ASSOC)['total'] ?? 0);

    // Fetch page
    $sql = "SELECT name, phone, pushname FROM wa_members_gb $where ORDER BY id DESC LIMIT :limit OFFSET :offset";
    $stmt = $db->prepare($sql);
    foreach ($params as $k => $v) {
        $stmt->bindValue($k, $v);
    }
    $stmt->bindValue(':limit', $perPage, PDO::PARAM_INT);
    $stmt->bindValue(':offset', $offset, PDO::PARAM_INT);
    $stmt->execute();
    $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);

    echo json_encode([
        'success' => true,
        'campaign_id' => $campaignId,
        'page' => $page,
        'per_page' => $perPage,
        'total' => $total,
        'total_pages' => $perPage ? (int)ceil($total / $perPage) : 0,
        'data' => $rows,
    ], JSON_UNESCAPED_UNICODE);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => $e->getMessage()]);
}
