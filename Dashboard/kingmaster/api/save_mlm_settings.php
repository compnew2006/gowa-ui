<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: POST');
header('Access-Control-Allow-Headers: Content-Type');

require_once '../config/database.php';

$pdo = getDB();

$input = json_decode(file_get_contents('php://input'), true);

if (!$input || !isset($input['settings'])) {
    echo json_encode(['success' => false, 'message' => 'بيانات غير صالحة']);
    exit;
}

try {
    $pdo->beginTransaction();
    
    foreach ($input['settings'] as $setting) {
        $stmt = $pdo->prepare("
            UPDATE mlm_settings 
            SET 
                direct_commission_percentage = ?,
                indirect_commission_percentage = ?,
                updated_at = CURRENT_TIMESTAMP
            WHERE level_number = ?
        ");
        
        $stmt->execute([
            $setting['direct_commission_percentage'],
            $setting['indirect_commission_percentage'],
            $setting['level_number']
        ]);
    }
    
    $pdo->commit();
    
    echo json_encode([
        'success' => true,
        'message' => 'تم حفظ الإعدادات بنجاح'
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    $pdo->rollBack();
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
