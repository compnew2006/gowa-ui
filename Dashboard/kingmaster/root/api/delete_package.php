<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

$data = json_decode(file_get_contents('php://input'), true);
$id = intval($data['id'] ?? 0);

if ($id <= 0) {
    echo json_encode(['success' => false, 'message' => 'معرّف الباقة غير صالح']);
    exit;
}

try {
    $pdo = getDB();

    // حذف الباقة
    $stmt = $pdo->prepare("DELETE FROM packages WHERE id = ?");
    $result = $stmt->execute([$id]);

    if ($result && $stmt->rowCount() > 0) {
        echo json_encode([
            'success' => true,
            'message' => 'تم حذف الباقة بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'الباقة غير موجودة أو تم حذفها مسبقاً'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
