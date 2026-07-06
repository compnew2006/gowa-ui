<?php
// coupons_api.php - CRUD for discount coupons
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') { http_response_code(204); exit; }

require_once '../config/database.php';

function respond($data, $code = 200){ http_response_code($code); echo json_encode($data, JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES); exit; }

function createCouponsTableIfNotExists(){
    $sql = "
        CREATE TABLE IF NOT EXISTS coupons (
            id INT AUTO_INCREMENT PRIMARY KEY,
            code VARCHAR(64) NOT NULL UNIQUE,
            type ENUM('extra_time','points','amount','discount') NOT NULL,
            value DECIMAL(12,2) DEFAULT 0,
            duration_days INT DEFAULT 0,
            uses_limit INT DEFAULT 1,
            used_count INT DEFAULT 0,
            expires_at DATETIME NULL,
            status ENUM('active','inactive') DEFAULT 'active',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            INDEX (type), INDEX (status), INDEX (expires_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    ";
    executeQuery($sql);
}

createCouponsTableIfNotExists();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
// remove 'test/api/coupons_api.php'
$segments = array_slice($segments, 3);
$idFromPath = isset($segments[0]) && ctype_digit($segments[0]) ? (int)$segments[0] : 0;

switch ($method) {
    case 'GET': {
        if ($idFromPath > 0){
            $row = fetchRow('SELECT * FROM coupons WHERE id = ?', [ $idFromPath ]);
            if (!$row) respond([ 'success'=>false, 'message'=>'غير موجود' ], 404);
            respond([ 'success'=>true, 'data'=>$row ]);
        }
        $q = isset($_GET['q']) ? trim((string)$_GET['q']) : '';
        $status = isset($_GET['status']) ? trim((string)$_GET['status']) : '';
        $type = isset($_GET['type']) ? trim((string)$_GET['type']) : '';
        $where=[]; $params=[];
        if ($q !== '') { $where[] = 'LOWER(code) LIKE ?'; $params[] = '%' . strtolower($q) . '%'; }
        if ($status !== '' && in_array($status, ['active','inactive'], true)) { $where[] = 'status = ?'; $params[] = $status; }
        if ($type !== '' && in_array($type, ['extra_time','points','amount','discount'], true)) { $where[] = 'type = ?'; $params[] = $type; }
        $sql = 'SELECT * FROM coupons';
        if ($where) $sql .= ' WHERE ' . implode(' AND ', $where);
        $sql .= ' ORDER BY updated_at DESC';
        $rows = fetchAll($sql, $params) ?: [];
        respond([ 'success'=>true, 'data'=>$rows ]);
    }
    case 'POST': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $code = trim((string)($input['code'] ?? ''));
        $type = trim((string)($input['type'] ?? ''));
        $value = isset($input['value']) ? (float)$input['value'] : 0;
        $duration_days = isset($input['duration_days']) ? (int)$input['duration_days'] : 0;
        $uses_limit = isset($input['uses_limit']) ? (int)$input['uses_limit'] : 1;
        $expires_at = isset($input['expires_at']) ? trim((string)$input['expires_at']) : null;
        $status = isset($input['status']) ? trim((string)$input['status']) : 'active';
        if ($code === '' || !in_array($type, ['extra_time','points','amount','discount'], true)) respond([ 'success'=>false, 'message'=>'بيانات غير مكتملة' ], 400);
        $ok = executeQuery('INSERT INTO coupons (code, type, value, duration_days, uses_limit, expires_at, status) VALUES (?,?,?,?,?,?,?)', [ $code, $type, $value, $duration_days, $uses_limit, $expires_at ?: null, in_array($status,['active','inactive'],true)?$status:'active' ]);
        if ($ok) { $id=(int)getLastInsertId(); respond([ 'success'=>true, 'message'=>'تم الإنشاء', 'id'=>$id ]); }
        respond([ 'success'=>false, 'message'=>'فشل الإنشاء' ], 500);
    }
    case 'PUT': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $id = isset($input['id']) ? (int)$input['id'] : $idFromPath;
        if ($id <= 0) respond([ 'success'=>false, 'message'=>'معرف غير صالح' ], 400);
        $fields=[]; $params=[];
        if (isset($input['code'])) { $fields[]='code=?'; $params[] = trim((string)$input['code']); }
        if (isset($input['type'])) { $fields[]='type=?'; $params[] = trim((string)$input['type']); }
        if (isset($input['value'])) { $fields[]='value=?'; $params[] = (float)$input['value']; }
        if (isset($input['duration_days'])) { $fields[]='duration_days=?'; $params[] = (int)$input['duration_days']; }
        if (isset($input['uses_limit'])) { $fields[]='uses_limit=?'; $params[] = (int)$input['uses_limit']; }
        if (isset($input['used_count'])) { $fields[]='used_count=?'; $params[] = (int)$input['used_count']; }
        if (array_key_exists('expires_at', $input)) { $fields[]='expires_at=?'; $params[] = $input['expires_at'] ?: null; }
        if (isset($input['status'])) { $fields[]='status=?'; $params[] = trim((string)$input['status']); }
        if (!$fields) respond([ 'success'=>false, 'message'=>'لا يوجد بيانات للتعديل' ], 400);
        $params[] = $id;
        $ok = executeQuery('UPDATE coupons SET ' . implode(', ', $fields) . ', updated_at = NOW() WHERE id = ?', $params);
        if ($ok) respond([ 'success'=>true, 'message'=>'تم التعديل' ]);
        respond([ 'success'=>false, 'message'=>'فشل التعديل' ], 500);
    }
    case 'DELETE': {
        if ($idFromPath > 0) {
            $ok = executeQuery('DELETE FROM coupons WHERE id = ?', [ $idFromPath ]);
            if ($ok) respond([ 'success'=>true, 'message'=>'تم الحذف' ]);
            respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
        }
        $payload = json_decode(file_get_contents('php://input'), true) ?: [];
        $ids = isset($payload['ids']) && is_array($payload['ids']) ? array_map('intval', $payload['ids']) : [];
        if (!$ids) respond([ 'success'=>false, 'message'=>'لا توجد معرفات' ], 400);
        $in = implode(',', array_fill(0, count($ids), '?'));
        $ok = executeQuery('DELETE FROM coupons WHERE id IN (' . $in . ')', $ids);
        if ($ok) respond([ 'success'=>true, 'message'=>'تم حذف العناصر المحددة' ]);
        respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
    }
}

respond([ 'success'=>false, 'message'=>'طريقة غير مدعومة' ], 405);
