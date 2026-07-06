<?php
header('Content-Type: application/json; charset=utf-8');

/**
 * ✅ Fix: ensure session cookie works for /api
 * Minimal change فقط
 */
ini_set('session.cookie_path', '/');
session_set_cookie_params([
    'lifetime' => 0,
    'path' => '/',
    'httponly' => true,
    'samesite' => 'Lax',
]);
session_start();

if (!isset($_SESSION['user_id'])) {
    http_response_code(401);
    echo json_encode(["success"=>false,"message"=>"غير مصرح"], JSON_UNESCAPED_UNICODE);
    exit;
}

require_once __DIR__ . '/db_local.php';

$sid = (string)$_SESSION['user_id'];
$sidTrim = trim($sid);
$sidInt = ctype_digit($sidTrim) ? (int)$sidTrim : -1;

function getUser(PDO $pdo, string $sidTrim, int $sidInt) {
    $stmt = $pdo->prepare("
        SELECT id, user_id, email, points
        FROM users
        WHERE (id = :id) OR (user_id = :sid) OR (email = :sid)
        LIMIT 1
    ");
    $stmt->execute([
        ':id'  => $sidInt,
        ':sid' => $sidTrim
    ]);
    return $stmt->fetch(PDO::FETCH_ASSOC);
}

// ✅ GET points
if ($_SERVER['REQUEST_METHOD'] === 'GET') {
    $user = getUser($pdo, $sidTrim, $sidInt);
    if (!$user) {
        echo json_encode([
            "success"=>false,
            "message"=>"المستخدم غير موجود",
            "debug_session_user_id"=>$sidTrim
        ], JSON_UNESCAPED_UNICODE);
        exit;
    }

    echo json_encode([
        "success"=>true,
        "points"=>(int)($user['points'] ?? 0)
    ], JSON_UNESCAPED_UNICODE);
    exit;
}

// ✅ POST deduct
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $payload = json_decode(file_get_contents("php://input"), true);
    $count = (int)($payload['count'] ?? 0);

    if ($count <= 0) {
        echo json_encode(["success"=>false,"message"=>"عدد غير صحيح"], JSON_UNESCAPED_UNICODE);
        exit;
    }

    try {
        $pdo->beginTransaction();

        $stmt = $pdo->prepare("
            SELECT id, points
            FROM users
            WHERE (id = :id) OR (user_id = :sid) OR (email = :sid)
            FOR UPDATE
        ");
        $stmt->execute([':id'=>$sidInt, ':sid'=>$sidTrim]);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);

        if (!$user) {
            $pdo->rollBack();
            echo json_encode(["success"=>false,"message"=>"المستخدم غير موجود"], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $have = (int)($user['points'] ?? 0);
        if ($have < $count) {
            $pdo->rollBack();
            echo json_encode([
                "success"=>false,
                "message"=>"رصيد النقاط غير كافٍ",
                "need"=>$count,
                "have"=>$have
            ], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $upd = $pdo->prepare("UPDATE users SET points = points - :c WHERE id = :uid");
        $upd->execute([':c'=>$count, ':uid'=>(int)$user['id']]);

        // log (اختياري)
        try {
            $log = $pdo->prepare("INSERT INTO point_use (user_id, points, action) VALUES (?, ?, 'download_data')");
            $log->execute([(int)$user['id'], $count]);
        } catch (Throwable $e) {}

        $pdo->commit();

        echo json_encode([
            "success"=>true,
            "points_used"=>$count,
            "remaining_points"=>$have - $count
        ], JSON_UNESCAPED_UNICODE);
        exit;

    } catch (Throwable $e) {
        if ($pdo->inTransaction()) $pdo->rollBack();
        http_response_code(500);
        echo json_encode(["success"=>false,"message"=>"Server error: ".$e->getMessage()], JSON_UNESCAPED_UNICODE);
        exit;
    }
}

http_response_code(405);
echo json_encode(["success"=>false,"message"=>"Method not allowed"], JSON_UNESCAPED_UNICODE);
