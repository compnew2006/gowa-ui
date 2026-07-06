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

if (!isset($data['current_password']) || !isset($data['new_password']) || !isset($data['confirm_password'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يرجى إدخال جميع الحقول المطلوبة'
    ]);
    exit;
}

$current_password = $data['current_password'];
$new_password = $data['new_password'];
$confirm_password = $data['confirm_password'];

// التحقق من تطابق كلمة المرور الجديدة
if ($new_password !== $confirm_password) {
    echo json_encode([
        'success' => false,
        'message' => 'كلمة المرور الجديدة غير متطابقة'
    ]);
    exit;
}

// التحقق من طول كلمة المرور
if (strlen($new_password) < 6) {
    echo json_encode([
        'success' => false,
        'message' => 'كلمة المرور يجب أن تكون 6 أحرف على الأقل'
    ]);
    exit;
}

try {
    $conn = getDB();
    
    // جلب كلمة المرور الحالية من قاعدة البيانات
    $stmt = $conn->prepare("SELECT password FROM users WHERE user_id = :user_id");
    $stmt->execute([':user_id' => $user_id]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$user) {
        echo json_encode([
            'success' => false,
            'message' => 'المستخدم غير موجود'
        ]);
        exit;
    }
    
    // التحقق من كلمة المرور الحالية
    if (!password_verify($current_password, $user['password'])) {
        echo json_encode([
            'success' => false,
            'message' => 'كلمة المرور الحالية غير صحيحة'
        ]);
        exit;
    }
    
    // تشفير كلمة المرور الجديدة
    $hashed_password = password_hash($new_password, PASSWORD_DEFAULT);
    
    // تحديث كلمة المرور
    $stmt = $conn->prepare("UPDATE users SET password = :password WHERE user_id = :user_id");
    $stmt->execute([
        ':password' => $hashed_password,
        ':user_id' => $user_id
    ]);
    
    if ($stmt->rowCount() > 0) {



          addLog($user_id, 'تغيير كلمه المرور');

        echo json_encode([
            'success' => true,
            'message' => 'تم تغيير كلمة المرور بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'لم يتم تحديث كلمة المرور'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
