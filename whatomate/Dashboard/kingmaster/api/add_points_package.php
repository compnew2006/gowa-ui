<?php
header('Content-Type: application/json');
require_once '../config/database.php';

try {
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (empty($input['points_count']) || empty($input['price'])) {
        echo json_encode([
            'success' => false,
            'message' => 'جميع الحقول مطلوبة'
        ]);
        exit;
    }
    
    $conn = getDB();
    
    $sql = "INSERT INTO points_packages (points_count, price, is_active) VALUES (:points_count, :price, 1)";
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':points_count', $input['points_count'], PDO::PARAM_INT);
    $stmt->bindParam(':price', $input['price'], PDO::PARAM_STR);
    
    if ($stmt->execute()) {
        echo json_encode([
            'success' => true,
            'message' => 'تم إضافة الباقة بنجاح',
            'id' => $conn->lastInsertId()
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل في إضافة الباقة'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ: ' . $e->getMessage()
    ]);
}
?>
