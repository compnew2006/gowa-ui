<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

try {
    $pdo = getDB();
    
    // استقبال البيانات
    $data = json_decode(file_get_contents('php://input'), true);
    $id = $data['id'] ?? 0;
    
    // التحقق من المعرف
    if (empty($id)) {
        echo json_encode([
            'success' => false,
            'message' => 'معرف المنتج مطلوب'
        ]);
        exit;
    }
    
    // حذف المنتج (سيتم حذف الألوان والمقاسات تلقائياً بسبب ON DELETE CASCADE)
    $stmt = $pdo->prepare("DELETE FROM products WHERE id = ?");
    $result = $stmt->execute([$id]);
    
    if ($result) {
        echo json_encode([
            'success' => true,
            'message' => 'تم حذف المنتج بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل حذف المنتج'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
