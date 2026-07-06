<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: POST');
header('Access-Control-Allow-Headers: Content-Type');

// الاتصال بقاعدة البيانات
require_once '../config/database.php';

$pdo = getDB();

// قراءة البيانات المرسلة
$input = json_decode(file_get_contents('php://input'), true);

if (!$input) {
    echo json_encode(['success' => false, 'message' => 'بيانات غير صالحة']);
    exit;
}

$user_id = isset($input['user_id']) ? (int)$input['user_id'] : 0;
$user_name = isset($input['user_name']) ? trim($input['user_name']) : '';
$title = isset($input['title']) ? trim($input['title']) : '';
$content = isset($input['content']) ? trim($input['content']) : '';
$user_avatar = isset($input['user_avatar']) ? trim($input['user_avatar']) : 'images/avatar.jpg';

// التحقق من البيانات
if (empty($user_id) || empty($user_name) || empty($title) || empty($content)) {
    echo json_encode(['success' => false, 'message' => 'جميع الحقول مطلوبة']);
    exit;
}

if (strlen($title) > 255) {
    echo json_encode(['success' => false, 'message' => 'العنوان طويل جداً']);
    exit;
}

try {
    $stmt = $pdo->prepare("
        INSERT INTO posts (user_id, user_name, user_avatar, title, content)
        VALUES (:user_id, :user_name, :user_avatar, :title, :content)
    ");
    
    $stmt->execute([
        ':user_id' => $user_id,
        ':user_name' => $user_name,
        ':user_avatar' => $user_avatar,
        ':title' => $title,
        ':content' => $content
    ]);
    
    $post_id = $pdo->lastInsertId();
    
    echo json_encode([
        'success' => true,
        'message' => 'تم نشر البوست بنجاح',
        'post_id' => $post_id
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
