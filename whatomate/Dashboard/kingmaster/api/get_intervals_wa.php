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

try {
    $conn = getDB();
    $user_id = $_SESSION['user_id'];

    $stmt = $conn->prepare("SELECT id, settings_name FROM sending_settings WHERE user_id = :user_id AND platform = 'whatsapp'");
    $stmt->execute([':user_id' => $user_id]);
    $intervals = $stmt->fetchAll(PDO::FETCH_ASSOC);

    echo json_encode([
        'success' => true,
        'intervals' => $intervals
    ]);

} catch(PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ: ' . $e->getMessage()
    ]);
}
?>
