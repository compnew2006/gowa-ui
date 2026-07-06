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

    $stmt = $conn->prepare("
        SELECT 
            fb_page.id_page,
            fb_page.name AS page_name,
            fb_page.facebook_id,
            MAX(accounts.name) AS account_name
        FROM fb_page
        JOIN accounts 
            ON accounts.account_uid = fb_page.facebook_id
        WHERE fb_page.user_id = :user_id
        GROUP BY 
            fb_page.id_page,
            fb_page.name,
            fb_page.facebook_id
    ");

    $stmt->execute([
        ':user_id' => $user_id
    ]);

    $accounts = $stmt->fetchAll(PDO::FETCH_ASSOC);

    foreach ($accounts as &$acc) {
        $acc['name'] = $acc['page_name'] . ' - ' . ($acc['account_name'] ?? 'بدون حساب');
    }

    echo json_encode([
        'success' => true,
        'accounts' => $accounts
    ]);

} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ: ' . $e->getMessage()
    ]);
}
?>
