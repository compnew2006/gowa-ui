<?php
header('Content-Type: application/json');

require_once '../config/database.php';

try {
    $conn = getDB();
    
    // جلب الباقات النشطة فقط
    $sql = "SELECT * FROM points_packages WHERE is_active = 1 ORDER BY points_count ASC";
    $stmt = $conn->prepare($sql);
    $stmt->execute();
    
    $packages = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'packages' => $packages
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
