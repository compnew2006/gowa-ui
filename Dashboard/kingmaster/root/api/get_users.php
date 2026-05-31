<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../config/database.php';
require_once '../includes/functions.php';

$user_id = $_SESSION['user_id'];

try {
    $conn = getDB();
    
    // Get all users except current user (admins first)
    $stmt = $conn->prepare("
        SELECT user_id as id, CONCAT(first_name, ' ', last_name) as name, img, is_admin 
        FROM users 
        WHERE user_id != :user_id
        ORDER BY is_admin DESC, first_name ASC
        LIMIT 50
    ");
    $stmt->execute([':user_id' => $user_id]);
    $users = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    // Format image URLs
    foreach ($users as &$user) {
        if ($user['img'] == '0' || empty($user['img'])) {
            $user['img'] = 'https://i.pravatar.cc/150?img=33';
        }
    }
    
    echo json_encode([
        'success' => true,
        'users' => $users
    ]);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب المستخدمين: ' . $e->getMessage()
    ]);
}
