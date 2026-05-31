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

$input = json_decode(file_get_contents("php://input"), true);
$tool = isset($input['tool']) ? trim($input['tool']) : '';

try {
    $conn = getDB();
    $user_id = $_SESSION['user_id'];
    
    $stmt = $conn->prepare("
        SELECT * FROM campaigns 
        WHERE user_id = :user_id
        AND tool = :tool
        ORDER BY id DESC
    ");
    $stmt->execute([
        ':user_id' => $user_id,
        ':tool' => $tool
    ]);

    $campaigns = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'campaigns' => $campaigns
    ]);
    
} catch(PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ: ' . $e->getMessage()
    ]);
}
?>
