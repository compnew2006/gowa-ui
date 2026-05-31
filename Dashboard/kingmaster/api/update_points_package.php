<?php
header('Content-Type: application/json');
require_once '../config/database.php';

try {
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (empty($input['id']) || empty($input['points_count']) || empty($input['price'])) {
        echo json_encode([
            'success' => false,
            'message' => 'جميع الحقول مطلوبة'
        ]);
        exit;
    }
    
    $conn = getDB();
    
    $sql = "UPDATE points_packages SET points_count = :points_count, price = :price WHERE id = :id";
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':id', $input['id'], PDO::PARAM_INT);
    $stmt->bindParam(':points_count', $input['points_count'], PDO::PARAM_INT);
    $stmt->bindParam(':price', $input['price'], PDO::PARAM_STR);
    
    if ($stmt->execute()) {
        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث الباقة بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل في تحديث الباقة'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ: ' . $e->getMessage()
    ]);
}
?>
