<?php
session_start();
header('Content-Type: application/json');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول'
    ]);
    exit;
}

require_once '../config/database.php';

$user_id = $_SESSION['user_id'];


function addLog($user_id, $action) {
    $db = getDB();

    $query = "INSERT INTO logs (user_id, action) VALUES (:user_id, :action)";
    $stmt = $db->prepare($query);

    $stmt->execute([
        ':user_id' => $user_id,
        ':action' => $action
    ]);

    return $db->lastInsertId(); // ترجع رقم السجل الجديد
}


// قراءة البيانات
$data = json_decode(file_get_contents('php://input'), true);

if (!isset($data['timezone']) || empty($data['timezone'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يرجى تحديد المنطقة الزمنية'
    ]);
    exit;
}

$timezone = $data['timezone'];

try {
    $conn = getDB();
    
    // تحديث المنطقة الزمنية
    $stmt = $conn->prepare("UPDATE users SET timezone = :timezone WHERE user_id = :user_id");
    $stmt->execute([
        ':timezone' => $timezone,
        ':user_id' => $user_id
    ]);
    
    if ($stmt->rowCount() > 0) {

          addLog($user_id, 'تغيير المنطقة الزمنية');

        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث المنطقة الزمنية بنجاح',
            'timezone' => $timezone
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'لم يتم العثور على المستخدم أو المنطقة الزمنية لم تتغير'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
