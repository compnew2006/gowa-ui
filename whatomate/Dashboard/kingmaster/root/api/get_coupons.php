<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

try {
    $pdo = getDB();
    
    $stmt = $pdo->query("
        SELECT * FROM coupons 
        ORDER BY created_at DESC
    ");
    
    $coupons = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'coupons' => $coupons
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
