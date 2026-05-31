<?php
header('Content-Type: application/json; charset=utf-8');
require_once '../includes/functions.php';

try {
    $db = getDB();
    session_start();
    if (!isset($_SESSION['user_id'])) { http_response_code(403); echo json_encode(['success'=>false,'message'=>'unauthorized']); exit; }
    $user_id = $_SESSION['user_id'];

    // Ensure table exists (idempotent)
    $db->exec("CREATE TABLE IF NOT EXISTS wa_flows (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL,
        account_uid VARCHAR(255) NOT NULL,
        flow_name VARCHAR(255) NOT NULL,
        config LONGTEXT,
        active TINYINT(1) DEFAULT 1,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;");

    $method = $_SERVER['REQUEST_METHOD'];
    $action = $_GET['action'] ?? null;

    if ($method === 'GET') {
        if ($action === 'list') {
            $stmt = $db->prepare("SELECT id, account_uid, flow_name FROM wa_flows WHERE user_id=? ORDER BY id DESC");
            $stmt->execute([$user_id]);
            echo json_encode(['success'=>true,'flows'=>$stmt->fetchAll(PDO::FETCH_ASSOC)], JSON_UNESCAPED_UNICODE);
            exit;
        } elseif ($action === 'get') {
            $id = intval($_GET['id'] ?? 0);
            $stmt = $db->prepare("SELECT id, account_uid, flow_name, config FROM wa_flows WHERE id=? AND user_id=?");
            $stmt->execute([$id,$user_id]);
            $row = $stmt->fetch(PDO::FETCH_ASSOC);
            if (!$row) { http_response_code(404); echo json_encode(['success'=>false,'message'=>'not found']); exit; }
            echo json_encode(['success'=>true,'flow'=>$row], JSON_UNESCAPED_UNICODE); exit;
        }
        echo json_encode(['success'=>false,'message'=>'bad request']); exit;
    }

    if ($method === 'POST') {
        $raw = file_get_contents('php://input');
        $j = json_decode($raw, true);
        $a = $j['action'] ?? '';
        if ($a === 'save') {
            $id = intval($j['id'] ?? 0);
            $account_uid = trim($j['account_uid'] ?? '');
            $flow_name = trim($j['flow_name'] ?? '');
            $config = $j['config'] ?? null;
            if ($account_uid === '' || $flow_name === '') { http_response_code(400); echo json_encode(['success'=>false,'message'=>'missing fields']); exit; }
            $config_json = is_string($config) ? $config : json_encode($config, JSON_UNESCAPED_UNICODE);
            if ($id > 0) {
                $stmt = $db->prepare("UPDATE wa_flows SET account_uid=?, flow_name=?, config=? WHERE id=? AND user_id=?");
                $stmt->execute([$account_uid, $flow_name, $config_json, $id, $user_id]);
                echo json_encode(['success'=>true,'id'=>$id]); exit;
            } else {
                $stmt = $db->prepare("INSERT INTO wa_flows (user_id, account_uid, flow_name, config) VALUES (?,?,?,?)");
                $stmt->execute([$user_id, $account_uid, $flow_name, $config_json]);
                echo json_encode(['success'=>true,'id'=>$db->lastInsertId()]); exit;
            }
        } elseif ($a === 'delete') {
            $id = intval($j['id'] ?? 0);
            if ($id<=0) { http_response_code(400); echo json_encode(['success'=>false,'message'=>'bad id']); exit; }
            $stmt = $db->prepare("DELETE FROM wa_flows WHERE id=? AND user_id=?");
            $stmt->execute([$id,$user_id]);
            echo json_encode(['success'=>true]); exit;
        }
        echo json_encode(['success'=>false,'message'=>'bad action']); exit;
    }

    echo json_encode(['success'=>false,'message'=>'bad method']);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['success'=>false,'message'=>'server error','error'=>$e->getMessage()]);
}
