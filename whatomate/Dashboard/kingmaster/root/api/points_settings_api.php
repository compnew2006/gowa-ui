<?php
header('Content-Type: application/json');
require_once '../config/database.php';

$conn = getDB();
$data = json_decode(file_get_contents('php://input'), true);

if ($_SERVER['REQUEST_METHOD'] === 'GET' && isset($_GET['action']) && $_GET['action'] === 'get_all') {
    // جلب جميع الإعدادات
    $stmt = $conn->prepare("SELECT * FROM pvsettigs ORDER BY id DESC");
    $stmt->execute();
    $settings = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'settings' => $settings
    ]);
    exit;
}

if (!$data || !isset($data['action'])) {
    echo json_encode(['success' => false, 'message' => 'لم يتم تحديد الإجراء']);
    exit;
}

$action = $data['action'];

switch ($action) {
    case 'add':
        // إضافة إعداد جديد
        if (!isset($data['name']) || !isset($data['from']) || !isset($data['to']) || !isset($data['count'])) {
            echo json_encode(['success' => false, 'message' => 'جميع الحقول مطلوبة']);
            exit;
        }
        
        $name = $data['name'];
        $from = intval($data['from']);
        $to = intval($data['to']);
        $count = intval($data['count']);
        
        if ($from >= $to) {
            echo json_encode(['success' => false, 'message' => 'قيمة "من" يجب أن تكون أقل من قيمة "إلى"']);
            exit;
        }
        
        $stmt = $conn->prepare("INSERT INTO pvsettigs (name, `from`, `to`, count) VALUES (?, ?, ?, ?)");
        if ($stmt->execute([$name, $from, $to, $count])) {
            echo json_encode(['success' => true, 'message' => 'تم إضافة الإعداد بنجاح']);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في إضافة الإعداد']);
        }
        break;
        
    case 'update':
        // تعديل إعداد
        if (!isset($data['id']) || !isset($data['name']) || !isset($data['from']) || !isset($data['to']) || !isset($data['count'])) {
            echo json_encode(['success' => false, 'message' => 'جميع الحقول مطلوبة']);
            exit;
        }
        
        $id = intval($data['id']);
        $name = $data['name'];
        $from = intval($data['from']);
        $to = intval($data['to']);
        $count = intval($data['count']);
        
        if ($from >= $to) {
            echo json_encode(['success' => false, 'message' => 'قيمة "من" يجب أن تكون أقل من قيمة "إلى"']);
            exit;
        }
        
        $stmt = $conn->prepare("UPDATE pvsettigs SET name = ?, `from` = ?, `to` = ?, count = ? WHERE id = ?");
        if ($stmt->execute([$name, $from, $to, $count, $id])) {
            echo json_encode(['success' => true, 'message' => 'تم تحديث الإعداد بنجاح']);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في تحديث الإعداد']);
        }
        break;
        
    case 'delete':
        // حذف إعداد
        if (!isset($data['id'])) {
            echo json_encode(['success' => false, 'message' => 'معرف الإعداد مطلوب']);
            exit;
        }
        
        $id = intval($data['id']);
        
        $stmt = $conn->prepare("DELETE FROM pvsettigs WHERE id = ?");
        if ($stmt->execute([$id])) {
            echo json_encode(['success' => true, 'message' => 'تم حذف الإعداد بنجاح']);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في حذف الإعداد']);
        }
        break;
        
    default:
        echo json_encode(['success' => false, 'message' => 'إجراء غير معروف']);
}

$conn = null;
?>
