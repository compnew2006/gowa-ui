<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

$pdo = getDB();

$user_id = isset($_GET['user_id']) ? (int)$_GET['user_id'] : 1;
$max_levels = isset($_GET['max_levels']) ? (int)$_GET['max_levels'] : 5;

// دالة rekursive لبناء الشجرة
function buildTree($pdo, $parent_id, $current_level, $max_levels) {
    if ($current_level > $max_levels) {
        return [];
    }
    
    $stmt = $pdo->prepare("
        SELECT 
            id, username, full_name, email, phone, 
            referral_code, direct_referrals, total_referrals, 
            total_earnings, status, join_date
        FROM mlm_users 
        WHERE parent_id = ?
        ORDER BY join_date ASC
    ");
    $stmt->execute([$parent_id]);
    $users = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    $result = [];
    foreach ($users as $user) {
        $user['children'] = buildTree($pdo, $user['id'], $current_level + 1, $max_levels);
        $user['children_count'] = count($user['children']);
        $user['current_level'] = $current_level;
        $result[] = $user;
    }
    
    return $result;
}

try {
    // جلب بيانات المستخدم الرئيسي
    $stmt = $pdo->prepare("
        SELECT 
            id, username, full_name, email, phone, 
            parent_id, referral_code, direct_referrals, total_referrals, 
            total_earnings, status, join_date
        FROM mlm_users 
        WHERE id = ?
    ");
    $stmt->execute([$user_id]);
    $root_user = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$root_user) {
        echo json_encode([
            'success' => false,
            'message' => 'المستخدم غير موجود'
        ], JSON_UNESCAPED_UNICODE);
        exit;
    }
    
    // بناء الشجرة
    // الإحالات المباشرة تبدأ من مستوى 0 (نفس مستوى المستخدم الرئيسي)
    $root_user['children'] = buildTree($pdo, $user_id, 0, $max_levels);
    $root_user['children_count'] = count($root_user['children']);
    $root_user['current_level'] = 0;
    
    // حساب إجمالي الفرع
    function countTotalDownline($node) {
        $total = count($node['children']);
        foreach ($node['children'] as $child) {
            $total += countTotalDownline($child);
        }
        return $total;
    }
    
    $total_downline = countTotalDownline($root_user);
    
    echo json_encode([
        'success' => true,
        'user' => $root_user,
        'total_downline' => $total_downline,
        'max_levels' => $max_levels
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
