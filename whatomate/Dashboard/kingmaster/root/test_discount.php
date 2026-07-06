<?php
require_once 'config/database.php';
header('Content-Type: application/json; charset=UTF-8');

try {
    $db = getDB();
    $stmt = $db->query("SELECT * FROM packages ORDER BY display_order ASC");
    $packages = $stmt->fetchAll();
    
    foreach ($packages as &$package) {
        $package['features'] = json_decode($package['features'], true);
        
        // تحويل القيم الرقمية
        $package['monthly_price'] = floatval($package['monthly_price']);
        $package['yearly_price'] = floatval($package['yearly_price']);
        $package['monthly_discount'] = floatval($package['monthly_discount']);
        $package['yearly_discount'] = floatval($package['yearly_discount']);
        $package['monthly_discount_percentage'] = floatval($package['monthly_discount_percentage']);
        $package['yearly_discount_percentage'] = floatval($package['yearly_discount_percentage']);
        $package['has_discount'] = (bool)$package['has_discount'];
    }
    
    echo json_encode([
        'success' => true,
        'data' => $packages
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'error' => $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
?>