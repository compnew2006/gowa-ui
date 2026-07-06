<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../config/database.php';

$user_id = $_SESSION['user_id'];

try {
    $conn = getDB();
    
    // جلب معلومات المستخدم والباقة
    $stmt = $conn->prepare("
        SELECT 
            u.account_count,
            p.accounts_count as max_accounts,
            p.name as package_name
        FROM users u
        LEFT JOIN packages p ON u.package = p.id
        WHERE u.user_id = :user_id
    ");
    
    $stmt->execute([':user_id' => $user_id]);
    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($result) {
        echo json_encode([
            'success' => true,
            'current_count' => (int)$result['account_count'],
            'max_count' => (int)$result['max_accounts'],
            'package_name' => $result['package_name'],
            'can_add' => (int)$result['account_count'] < (int)$result['max_accounts']
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'لم يتم العثور على بيانات المستخدم'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في الخادم: ' . $e->getMessage()
    ]);
}
?>
