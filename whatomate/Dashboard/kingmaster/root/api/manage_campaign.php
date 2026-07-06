<?php
session_start();
header('Content-Type: application/json');
require_once '../config/database.php';

if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول أولاً'
    ]);
    exit;
}

$data = json_decode(file_get_contents('php://input'), true);
$action = $data['action'] ?? '';
$campaign_id = $data['campaign_id'] ?? 0;

if (!$action || !$campaign_id) {
    echo json_encode([
        'success' => false,
        'message' => 'بيانات غير كاملة'
    ]);
    exit;
}

try {
    $conn = getDB();
    $user_id = $_SESSION['user_id'];
    
    // Verify campaign belongs to user
    $stmt = $conn->prepare("SELECT * FROM campaigns WHERE id = :id AND user_id = :user_id");
    $stmt->execute([':id' => $campaign_id, ':user_id' => $user_id]);
    $campaign = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$campaign) {
        echo json_encode([
            'success' => false,
            'message' => 'الحملة غير موجودة'
        ]);
        exit;
    }
    
    switch ($action) {
        case 'change_status':
            $new_status = $data['status'] ?? '';
            if (!in_array($new_status, ['pending', 'running', 'paused', 'stopped', 'finished'])) {
                echo json_encode([
                    'success' => false,
                    'message' => 'حالة غير صالحة'
                ]);
                exit;
            }
            
            $stmt = $conn->prepare("UPDATE campaigns SET status = :status WHERE id = :id");
            $stmt->execute([':status' => $new_status, ':id' => $campaign_id]);
            
            echo json_encode([
                'success' => true,
                'message' => 'تم تغيير حالة الحملة بنجاح'
            ]);
            break;
            
        case 'update':
            $name = $data['name'] ?? $campaign['name'];
            $accounts = $data['accounts'] ?? json_decode($campaign['token'], true);
            $interval = $data['interval_id'] ?? $campaign['interval'];
            
            if (empty($accounts)) {
                echo json_encode([
                    'success' => false,
                    'message' => 'يجب اختيار حساب واحد على الأقل'
                ]);
                exit;
            }
            
            $token_json = is_array($accounts) ? json_encode($accounts) : $accounts;
            
            $stmt = $conn->prepare("
                UPDATE campaigns 
                SET name = :name, token = :token, `interval` = :interval 
                WHERE id = :id
            ");
            $stmt->execute([
                ':name' => $name,
                ':token' => $token_json,
                ':interval' => $interval,
                ':id' => $campaign_id
            ]);
            
            echo json_encode([
                'success' => true,
                'message' => 'تم تحديث الحملة بنجاح'
            ]);
            break;
            
        case 'delete':
            $stmt = $conn->prepare("DELETE FROM campaigns WHERE id = :id");
            $stmt->execute([':id' => $campaign_id]);
            
            echo json_encode([
                'success' => true,
                'message' => 'تم حذف الحملة بنجاح'
            ]);
            break;
            
        default:
            echo json_encode([
                'success' => false,
                'message' => 'إجراء غير معروف'
            ]);
    }
    
} catch(PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ: ' . $e->getMessage()
    ]);
}
?>
