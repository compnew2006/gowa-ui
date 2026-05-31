<?php
// whatsapp_polls_api.php - CRUD for WhatsApp poll links
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') { http_response_code(204); exit; }

require_once '../config/database.php';

function respond($data, $code = 200) {
    http_response_code($code);
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function createTableIfNotExists() {
    $sql = "
        CREATE TABLE IF NOT EXISTS whatsapp_polls (
            id INT AUTO_INCREMENT PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            base_url VARCHAR(255) NOT NULL,
            api_key VARCHAR(128) NOT NULL,
            instance_id VARCHAR(128) NOT NULL,
            phone VARCHAR(32) NOT NULL,
            poll_name VARCHAR(255) NOT NULL,
            choices JSON NOT NULL,
            url TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            INDEX (created_at), INDEX (phone)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    ";
    executeQuery($sql);
}

function buildPollUrl($base, $key, $instance, $phone, $name, $choices) {
    $q = [];
    $q[] = 'key=' . rawurlencode($key);
    $q[] = 'instance_id=' . rawurlencode($instance);
    $q[] = 'phone=' . rawurlencode($phone);
    $q[] = 'name=' . rawurlencode($name);
    foreach ($choices as $c) {
        $q[] = 'choices[]=' . rawurlencode($c);
    }
    return rtrim($base, '?&') . '?' . implode('&', $q);
}

createTableIfNotExists();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
// remove 'test/api/whatsapp_polls_api.php'
$segments = array_slice($segments, 3);
$idFromPath = isset($segments[0]) && ctype_digit($segments[0]) ? (int)$segments[0] : 0;

switch ($method) {
    case 'GET': {
        $q = isset($_GET['q']) ? trim((string)$_GET['q']) : '';
        $where = [];
        $params = [];
        if ($idFromPath > 0) {
            $row = fetchRow('SELECT * FROM whatsapp_polls WHERE id = ?', [ $idFromPath ]);
            if (!$row) respond([ 'success' => false, 'message' => 'غير موجود' ], 404);
            respond([ 'success' => true, 'data' => $row ]);
        }
        if ($q !== '') { $where[] = '(LOWER(name) LIKE ? OR phone LIKE ?)'; $params[] = '%' . strtolower($q) . '%'; $params[] = '%' . $q . '%'; }
        $sql = 'SELECT * FROM whatsapp_polls';
        if ($where) $sql .= ' WHERE ' . implode(' AND ', $where);
        $sql .= ' ORDER BY updated_at DESC';
        $rows = fetchAll($sql, $params) ?: [];
        respond([ 'success' => true, 'data' => $rows ]);
    }
    case 'POST': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $name = trim((string)($input['name'] ?? ''));
        $base_url = trim((string)($input['base_url'] ?? ''));
        $api_key = trim((string)($input['api_key'] ?? ''));
        $instance_id = trim((string)($input['instance_id'] ?? ''));
        $phone = trim((string)($input['phone'] ?? ''));
        $poll_name = trim((string)($input['poll_name'] ?? ''));
        $choices = $input['choices'] ?? [];
        if ($name === '' || $base_url === '' || $api_key === '' || $instance_id === '' || $phone === '' || $poll_name === '' || !is_array($choices) || count($choices) === 0) {
            respond([ 'success' => false, 'message' => 'بيانات غير مكتملة' ], 400);
        }
        $url = buildPollUrl($base_url, $api_key, $instance_id, $phone, $poll_name, $choices);
        $ok = executeQuery('INSERT INTO whatsapp_polls (name, base_url, api_key, instance_id, phone, poll_name, choices, url) VALUES (?,?,?,?,?,?,?,?)', [
            $name, $base_url, $api_key, $instance_id, $phone, $poll_name, json_encode($choices, JSON_UNESCAPED_UNICODE), $url
        ]);
        if ($ok) {
            $id = (int)getLastInsertId();
            respond([ 'success' => true, 'message' => 'تم الحفظ', 'id' => $id, 'url' => $url ]);
        }
        respond([ 'success' => false, 'message' => 'فشل الحفظ' ], 500);
    }
    case 'PUT': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $id = isset($input['id']) ? (int)$input['id'] : $idFromPath;
        if ($id <= 0) respond([ 'success' => false, 'message' => 'معرف غير صالح' ], 400);
        $row = fetchRow('SELECT * FROM whatsapp_polls WHERE id = ?', [ $id ]);
        if (!$row) respond([ 'success' => false, 'message' => 'غير موجود' ], 404);
        $name = isset($input['name']) ? trim((string)$input['name']) : $row['name'];
        $base_url = isset($input['base_url']) ? trim((string)$input['base_url']) : $row['base_url'];
        $api_key = isset($input['api_key']) ? trim((string)$input['api_key']) : $row['api_key'];
        $instance_id = isset($input['instance_id']) ? trim((string)$input['instance_id']) : $row['instance_id'];
        $phone = isset($input['phone']) ? trim((string)$input['phone']) : $row['phone'];
        $poll_name = isset($input['poll_name']) ? trim((string)$input['poll_name']) : $row['poll_name'];
        $choices = isset($input['choices']) ? $input['choices'] : json_decode($row['choices'], true);
        if (!is_array($choices) || count($choices) === 0) $choices = [];
        $url = buildPollUrl($base_url, $api_key, $instance_id, $phone, $poll_name, $choices);
        $ok = executeQuery('UPDATE whatsapp_polls SET name=?, base_url=?, api_key=?, instance_id=?, phone=?, poll_name=?, choices=?, url=?, updated_at=NOW() WHERE id=?', [
            $name, $base_url, $api_key, $instance_id, $phone, $poll_name, json_encode($choices, JSON_UNESCAPED_UNICODE), $url, $id
        ]);
        if ($ok) respond([ 'success' => true, 'message' => 'تم التعديل', 'url' => $url ]);
        respond([ 'success' => false, 'message' => 'فشل التعديل' ], 500);
    }
    case 'DELETE': {
        if ($idFromPath > 0) {
            $ok = executeQuery('DELETE FROM whatsapp_polls WHERE id = ?', [ $idFromPath ]);
            if ($ok) respond([ 'success' => true, 'message' => 'تم الحذف' ]);
            respond([ 'success' => false, 'message' => 'فشل الحذف' ], 500);
        }
        $payload = json_decode(file_get_contents('php://input'), true) ?: [];
        $ids = isset($payload['ids']) && is_array($payload['ids']) ? array_map('intval', $payload['ids']) : [];
        if (!$ids) respond([ 'success' => false, 'message' => 'لا توجد معرفات' ], 400);
        $in = implode(',', array_fill(0, count($ids), '?'));
        $ok = executeQuery('DELETE FROM whatsapp_polls WHERE id IN (' . $in . ')', $ids);
        if ($ok) respond([ 'success' => true, 'message' => 'تم حذف العناصر المحددة' ]);
        respond([ 'success' => false, 'message' => 'فشل الحذف' ], 500);
    }
}

respond([ 'success' => false, 'message' => 'طريقة غير مدعومة' ], 405);
