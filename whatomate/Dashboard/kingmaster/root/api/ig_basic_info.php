<?php
header('Content-Type: application/json');
require_once __DIR__ . '/../includes/functions.php';

try {
    $db = getDB();
    
    // 1. استقبال المتغيرات الأساسية
    $campaignId = isset($_GET['campaign_id']) ? (int)$_GET['campaign_id'] : (int)($_POST['campaign_id'] ?? 0);
    $tableName = isset($_GET['table']) ? $_GET['table'] : ($_POST['table'] ?? '');
    $q = trim((string)($_GET['q'] ?? ($_POST['q'] ?? '')));
    
    if ($campaignId <= 0) {
        echo json_encode(['success' => false, 'message' => 'campaign_id is required']);
        exit;
    }

    // =======================================================
    // 2. القائمة البيضاء (Whitelist) + إعدادات البحث لكل جدول
    // =======================================================
    // نحدد هنا الجداول المسموحة، والأعمدة التي يتم البحث فيها لكل جدول
    $allowedTables = [
        'ig_msg'  => ['name', 'phone', 'comment'], // يبحث في الاسم والرقم والكومنت
        'ig_post' => ['shortcode', 'content'],     // يبحث في الكابشن والروابط
        'ig_retarget' => ['ig_user_id', 'status'],
        'ig_dms' => ['ig_user_id', 'username', 'full_name'],
        'ig_follow'   => ['ig_user_id', 'username', 'full_name', 'extract_type'],
        'ig_search_users'        => ['ig_user_id', 'username', 'full_name', 'profile_url', 'is_private', 'is_verified', 'keyword'],
        'ig_search_hashtags'     => ['hashtag_id', 'hashtag_name', 'hashtag_url', 'media_count', 'keyword'],
        'ig_search_locations'    => ['location_id', 'location_name', 'address', 'lat', 'lng', 'keyword']
    ];

    // حماية ضد ثغرات SQL Injection
    if (!array_key_exists($tableName, $allowedTables)) {
        echo json_encode(['success' => false, 'message' => 'Invalid or unauthorized table name']);
        exit;
    }

    // =======================================================
    // 3. بناء الفلاتر والبحث بشكل ديناميكي
    // =======================================================
    $where = 'WHERE campaign_id = :cid';
    $params = [':cid' => $campaignId];

    if ($q !== '') {
        $searchColumns = $allowedTables[$tableName];
        $searchConditions = [];
        
        // بناء جملة LIKE ديناميكياً بناءً على أعمدة الجدول
        foreach ($searchColumns as $index => $column) {
            $paramName = ":q" . $index;
            $searchConditions[] = "$column LIKE $paramName";
            $params[$paramName] = '%' . $q . '%';
        }
        
        if (!empty($searchConditions)) {
            $where .= ' AND (' . implode(' OR ', $searchConditions) . ')';
        }
    }

    // =======================================================
    // 4. إعدادات الصفحات (Pagination)
    // =======================================================
    $page = isset($_GET['page']) ? (int)$_GET['page'] : (int)($_POST['page'] ?? 1);
    $perPage = isset($_GET['per_page']) ? (int)$_GET['per_page'] : (int)($_POST['per_page'] ?? 25);
    if ($perPage < 1) $perPage = 25;
    if ($perPage > 200) $perPage = 200;
    if ($page < 1) $page = 1;
    $offset = ($page - 1) * $perPage;

    // =======================================================
    // 5. جلب البيانات (COUNT & SELECT *)
    // =======================================================
    // جلب الإجمالي
    $stmt = $db->prepare("SELECT COUNT(*) AS total FROM $tableName $where");
    $stmt->execute($params);
    $total = (int)($stmt->fetch(PDO::FETCH_ASSOC)['total'] ?? 0);

    // جلب البيانات المطلوبة باستخدام SELECT *
    // (استخدمنا orderBy id لتكون الأحدث أولاً، ويمكنك تعديلها لاحقاً إذا لزم الأمر)
    $sql = "SELECT * FROM $tableName $where ORDER BY id DESC LIMIT :limit OFFSET :offset";
    $stmt = $db->prepare($sql);
    
    foreach ($params as $k => $v) {
        $stmt->bindValue($k, $v);
    }
    $stmt->bindValue(':limit', $perPage, PDO::PARAM_INT);
    $stmt->bindValue(':offset', $offset, PDO::PARAM_INT);
    $stmt->execute();
    
    $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);

    // =======================================================
    // 6. طباعة النتيجة
    // =======================================================
    echo json_encode([
        'success' => true,
        'table' => $tableName,
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
?>