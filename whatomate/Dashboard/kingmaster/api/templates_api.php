<?php
// templates_api.php - CRUD for Messenger templates stored in DB
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

function createTemplatesTableIfNotExists() {
    $sql = "
        CREATE TABLE IF NOT EXISTS messenger_templates (
            id INT AUTO_INCREMENT PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            type VARCHAR(50) DEFAULT 'generic',
            channel VARCHAR(20) DEFAULT 'facebook',
            payload JSON NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            INDEX (type), INDEX (channel), INDEX (created_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    ";
    executeQuery($sql);
    // Ensure channel column exists for legacy tables
    @executeQuery("ALTER TABLE messenger_templates ADD COLUMN channel VARCHAR(20) DEFAULT 'facebook' AFTER type");
}

createTemplatesTableIfNotExists();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
// remove 'test/api/templates_api.php'
$segments = array_slice($segments, 3);
$idFromPath = isset($segments[0]) && ctype_digit($segments[0]) ? (int)$segments[0] : 0;

switch ($method) {
    case 'GET': {
        if ($idFromPath > 0) {
            $row = fetchRow('SELECT id, name, type, channel, payload, created_at, updated_at FROM messenger_templates WHERE id = ?', [ $idFromPath ]);
            if (!$row) respond([ 'success' => false, 'message' => 'القالب غير موجود' ], 404);
            respond([ 'success' => true, 'data' => $row ]);
        }
        $q = isset($_GET['q']) ? trim((string)$_GET['q']) : '';
        $type = isset($_GET['type']) ? trim((string)$_GET['type']) : '';
        $channel = isset($_GET['channel']) ? trim((string)$_GET['channel']) : '';
        $where = [];
        $params = [];
        if ($q !== '') { $where[] = 'LOWER(name) LIKE ?'; $params[] = '%' . strtolower($q) . '%'; }
        if ($type !== '') { $where[] = 'type = ?'; $params[] = $type; }
        if ($channel !== '') { $where[] = 'channel = ?'; $params[] = $channel; }
        $sql = 'SELECT id, name, type, channel, payload, created_at, updated_at FROM messenger_templates';
        if ($where) $sql .= ' WHERE ' . implode(' AND ', $where);
        $sql .= ' ORDER BY updated_at DESC';
        $rows = fetchAll($sql, $params);
        if ($rows === false) $rows = [];
        respond([ 'success' => true, 'data' => $rows ]);
    }
    case 'POST': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $name = trim((string)($input['name'] ?? ''));
        $type = trim((string)($input['type'] ?? 'generic'));
        $channel = trim((string)($input['channel'] ?? 'facebook'));
        $payload = $input['payload'] ?? null;
        if ($name === '' || !$payload) respond([ 'success' => false, 'message' => 'الاسم والبيانات مطلوبة' ], 400);
        // ensure payload is valid JSON object
        if (!is_array($payload)) respond([ 'success' => false, 'message' => 'صيغة JSON غير صحيحة' ], 400);
        // Workaround to use proper binding for JSON
        $sql = 'INSERT INTO messenger_templates (name, type, channel, payload) VALUES (?,?,?,?)';
        $ok = executeQuery($sql, [ $name, $type, $channel, json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ]);
        if ($ok) {
            $id = (int)getLastInsertId();
            respond([ 'success' => true, 'message' => 'تم إنشاء القالب', 'id' => $id ]);
        }
        respond([ 'success' => false, 'message' => 'فشل إنشاء القالب' ], 500);
    }
    case 'PUT': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $id = isset($input['id']) ? (int)$input['id'] : $idFromPath;
        if ($id <= 0) respond([ 'success' => false, 'message' => 'معرف غير صالح' ], 400);
        $fields = [];
        $params = [];
        if (isset($input['name'])) { $fields[] = 'name = ?'; $params[] = trim((string)$input['name']); }
        if (isset($input['type'])) { $fields[] = 'type = ?'; $params[] = trim((string)$input['type']); }
        if (isset($input['channel'])) { $fields[] = 'channel = ?'; $params[] = trim((string)$input['channel']); }
        if (isset($input['payload'])) {
            if (!is_array($input['payload'])) respond([ 'success' => false, 'message' => 'صيغة JSON غير صحيحة' ], 400);
            $fields[] = 'payload = ?';
            $params[] = json_encode($input['payload'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        }
        if (!$fields) respond([ 'success' => false, 'message' => 'لا يوجد بيانات للتعديل' ], 400);
        $params[] = $id;
        $ok = executeQuery('UPDATE messenger_templates SET ' . implode(', ', $fields) . ', updated_at = NOW() WHERE id = ?', $params);
        if ($ok) respond([ 'success' => true, 'message' => 'تم التعديل' ]);
        respond([ 'success' => false, 'message' => 'فشل التعديل' ], 500);
    }
    case 'DELETE': {
        $raw = file_get_contents('php://input');
        $payload = json_decode($raw, true) ?: [];
        if ($idFromPath > 0) {
            $ok = executeQuery('DELETE FROM messenger_templates WHERE id = ?', [ $idFromPath ]);
            if ($ok) respond([ 'success' => true, 'message' => 'تم الحذف' ]);
            respond([ 'success' => false, 'message' => 'فشل الحذف' ], 500);
        }
        $ids = isset($payload['ids']) && is_array($payload['ids']) ? array_map('intval', $payload['ids']) : [];
        if (!$ids) respond([ 'success' => false, 'message' => 'لا توجد معرفات للحذف' ], 400);
        $in = implode(',', array_fill(0, count($ids), '?'));
        $ok = executeQuery('DELETE FROM messenger_templates WHERE id IN (' . $in . ')', $ids);
        if ($ok) respond([ 'success' => true, 'message' => 'تم حذف القوالب المحددة' ]);
        respond([ 'success' => false, 'message' => 'فشل الحذف' ], 500);
    }
}

respond([ 'success' => false, 'message' => 'طريقة غير مدعومة' ], 405);
