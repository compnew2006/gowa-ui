<?php
session_start();
header('Content-Type: application/json');

require_once '../config/database.php';

/**
 * التحقق من وجود رقم في القائمة السوداء
 * يدعم GET و POST
 * 
 * GET: ?phone=01234567890&settings_id=1
 * POST: {"phone": "01234567890", "settings_id": 1}
 */

$user_id = $_SESSION['user_id'] ?? 1;

try {
    // دعم GET و POST
    if ($_SERVER['REQUEST_METHOD'] === 'GET') {
        $phone = $_GET['phone'] ?? null;
        $settings_id = $_GET['settings_id'] ?? null;
    } else {
        $input = json_decode(file_get_contents('php://input'), true);
        $phone = $input['phone'] ?? null;
        $settings_id = $input['settings_id'] ?? null;
    }
    
    if (empty($phone) || empty($settings_id)) {
        echo json_encode([
            'success' => false,
            'message' => 'رقم الهاتف ومعرف الإعدادات مطلوبان'
        ]);
        exit;
    }
    
    $conn = getDB();
    
    // البحث في القائمة السوداء باستخدام JSON_CONTAINS (سريع جداً حتى مع 10,000 رقم)
    $sql = "SELECT blacklist_enabled, 
            JSON_CONTAINS(blacklist, JSON_QUOTE(:phone)) as is_blacklisted
            FROM sending_settings 
            WHERE id = :settings_id AND user_id = :user_id";
    
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':phone', $phone, PDO::PARAM_STR);
    $stmt->bindParam(':settings_id', $settings_id, PDO::PARAM_INT);
    $stmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
    $stmt->execute();
    
    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($result && $result['blacklist_enabled']) {
        echo json_encode([
            'success' => true,
            'is_blacklisted' => (bool)$result['is_blacklisted'],
            'message' => $result['is_blacklisted'] ? 'الرقم موجود في القائمة السوداء' : 'الرقم غير موجود في القائمة السوداء'
        ]);
    } else {
        echo json_encode([
            'success' => true,
            'is_blacklisted' => false,
            'message' => 'القائمة السوداء غير مفعلة'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ: ' . $e->getMessage()
    ]);
}
?>
