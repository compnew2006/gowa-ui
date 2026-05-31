<?php
header('Content-Type: application/json');
require_once '../config/database.php';

try {
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (empty($input['id'])) {
        echo json_encode([
            'success' => false,
            'message' => 'معرف الباقة مطلوب'
        ]);
        exit;
    }
    
    $conn = getDB();
    
    $sql = "DELETE FROM points_packages WHERE id = :id";
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':id', $input['id'], PDO::PARAM_INT);
    
    if ($stmt->execute()) {
        if ($stmt->rowCount() > 0) {
            echo json_encode([
                'success' => true,
                'message' => 'تم حذف الباقة بنجاح'
            ]);
        } else {
            echo json_encode([
                'success' => false,
                'message' => 'الباقة غير موجودة'
            ]);
        }
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل في حذف الباقة'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ: ' . $e->getMessage()
    ]);
}
?>
