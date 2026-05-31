<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح بالدخول']);
    exit;
}

try {
    $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
    $search = isset($_GET['search']) ? sanitizeInput($_GET['search']) : '';
    $limit = 20;
    $offset = ($page - 1) * $limit;

    // بناء استعلام البحث
    $searchCondition = '';
    $params = [];
    
    if (!empty($search)) {
        $searchCondition = " WHERE u.first_name LIKE ? OR u.last_name LIKE ? OR u.email LIKE ? OR u.phone LIKE ?";
        $searchTerm = "%{$search}%";
        $params = [$searchTerm, $searchTerm, $searchTerm, $searchTerm];
    }

    // حساب إجمالي المستخدمين
    $countQuery = "SELECT COUNT(*) as total FROM users u" . $searchCondition;
    $countStmt = empty($params) ? 
        executeQuery($countQuery) : 
        executeQuery($countQuery, $params);
    
    $countResult = $countStmt->fetch();
    $totalUsers = $countResult['total'];
    $totalPages = ceil($totalUsers / $limit);

    // جلب المستخدمين مع اسم الباقة من جدول packages
    $query = "SELECT u.id, u.first_name, u.last_name, u.email, u.phone, u.is_admin, u.is_verified, 
                     u.package, u.expiry_date, u.created_at, u.timezone, u.points,
                     CASE 
                         WHEN u.package = 0 OR u.package IS NULL THEN 'باقة تجريبية'
                         ELSE COALESCE(p.name, 'باقة تجريبية')
                     END as package_name 
              FROM users u
              LEFT JOIN packages p ON u.package = p.id" . $searchCondition . " 
              ORDER BY u.created_at DESC 
              LIMIT ? OFFSET ?";
    
    $queryParams = array_merge($params, [$limit, $offset]);
    $usersStmt = executeQuery($query, $queryParams);
    $users = $usersStmt->fetchAll();

    echo json_encode([
        'success' => true,
        'users' => $users,
        'total_pages' => $totalPages,
        'current_page' => $page,
        'total_users' => $totalUsers
    ]);

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء جلب البيانات',
        'error' => $e->getMessage()
    ]);
}
?>
