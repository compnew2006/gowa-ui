<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

$pdo = getDB();

try {
    $stmt = $pdo->prepare("
        SELECT 
            id, package_id, max_levels, direct_commission_amount,
            level_1_percentage, level_2_percentage, level_3_percentage,
            level_4_percentage, level_5_percentage, level_6_percentage,
            level_7_percentage, level_8_percentage, level_9_percentage,
            level_10_percentage, is_active
        FROM package_mlm_settings
        WHERE is_active = 1
        ORDER BY package_id ASC
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
