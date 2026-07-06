<?php
 
require_once __DIR__ . '/config/database.php';

header('Content-Type: application/json; charset=utf-8');

try {
    // التحقق من fb_id
    if (empty($_GET['fb_id'])) {
        throw new Exception('fb_id is required');
    }

    if (empty($_GET['id'])) {
        throw new Exception('User not logged in');
    }

    $fb_id   = $_GET['fb_id'];
    $user_id = $_GET['id'];

    // 1️⃣ جلب نتيجة واحدة فقط
    $sql = "
        SELECT 
            fb_id,
            name,
            mobile_phone,
            gender,
            birthday,
            location,
            relationship,
            email,
            work,
            education
        FROM data_fb
        WHERE fb_id = :fb_id
        LIMIT 1
    ";

    $rows = fetchAll($sql, [
        ':fb_id' => $fb_id
    ]);

    // لو فيه نتيجة
    if (count($rows) === 1) {

        // 2️⃣ خصم 1 point من المستخدم
        $updateSql = "
            UPDATE users 
            SET points = points - 1
            WHERE user_id = :user_id
              AND points > 0
        ";

        executeQuery($updateSql, [
            ':user_id' => $user_id
        ]);
    }

    // 3️⃣ الإخراج JSON
    echo json_encode([
        'success' => true,
        'found'   => count($rows) === 1,
        'data'    => $rows[0] ?? null
    ], JSON_UNESCAPED_UNICODE);

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
