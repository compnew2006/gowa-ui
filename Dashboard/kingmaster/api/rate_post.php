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

$post_id = isset($input['post_id']) ? (int)$input['post_id'] : 0;
$user_id = isset($input['user_id']) ? (int)$input['user_id'] : 0;
$rating = isset($input['rating']) ? (int)$input['rating'] : 0;

// التحقق من البيانات
if (empty($post_id) || empty($user_id)) {
    echo json_encode(['success' => false, 'message' => 'بيانات غير كاملة']);
    exit;
}

if ($rating < 1 || $rating > 5) {
    echo json_encode(['success' => false, 'message' => 'التقييم يجب أن يكون بين 1 و 5']);
    exit;
}

try {
    // التحقق من وجود البوست
    $stmt = $pdo->prepare("SELECT id FROM posts WHERE id = :post_id");
    $stmt->execute([':post_id' => $post_id]);
    
    if (!$stmt->fetch()) {
        echo json_encode(['success' => false, 'message' => 'البوست غير موجود']);
        exit;
    }
    
    // إضافة أو تحديث التقييم
    $stmt = $pdo->prepare("
        INSERT INTO post_ratings (post_id, user_id, rating)
        VALUES (:post_id, :user_id, :rating)
        ON DUPLICATE KEY UPDATE rating = VALUES(rating), updated_at = CURRENT_TIMESTAMP
    ");
    
    $stmt->execute([
        ':post_id' => $post_id,
        ':user_id' => $user_id,
        ':rating' => $rating
    ]);
    
    // جلب متوسط التقييم الجديد
    $stmt = $pdo->prepare("
        SELECT 
            COALESCE(AVG(rating), 0) as avg_rating,
            COUNT(id) as total_ratings
        FROM post_ratings 
        WHERE post_id = :post_id
    ");
    $stmt->execute([':post_id' => $post_id]);
    $stats = $stmt->fetch(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'message' => 'تم حفظ التقييم بنجاح',
        'avg_rating' => round($stats['avg_rating'], 1),
        'total_ratings' => $stats['total_ratings']
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
