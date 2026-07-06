<?php
header('Content-Type: application/json');
require_once '../config/database.php';

try {
    $conn = getDB();
    
    // جلب جميع الباقات (النشطة وغير النشطة)
    $sql = "SELECT * FROM points_packages ORDER BY points_count ASC";
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
        'message' => 'خطأ: ' . $e->getMessage()
    ]);
}
?>
