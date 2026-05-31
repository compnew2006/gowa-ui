<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

$pdo = getDB();

try {
    $stmt = $pdo->prepare("
        SELECT 
            id, level_number, level_name,
            direct_commission_percentage,
            indirect_commission_percentage,
            description, is_active
        FROM mlm_settings
        ORDER BY level_number ASC
    ");
    
    $stmt->execute();
    $settings = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'settings' => $settings,
        'total' => count($settings)
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage(),
        'settings' => []
    ], JSON_UNESCAPED_UNICODE);
}
