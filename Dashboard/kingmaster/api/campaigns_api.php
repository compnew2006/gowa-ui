<?php
// campaigns_api.php - CRUD for campaigns (حملات)
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

function createCampaignsTableIfNotExists(){
    $sql = "
        CREATE TABLE IF NOT EXISTS campaigns (
            id INT AUTO_INCREMENT PRIMARY KEY,
            campaign_uid VARCHAR(12) NOT NULL UNIQUE,
            name VARCHAR(255) NOT NULL,
            names JSON NOT NULL,
            number BIGINT DEFAULT 0,
            range_from BIGINT DEFAULT NULL,
            range_to BIGINT DEFAULT NULL,
            status ENUM('pending','running','paused','stopped','finished') DEFAULT 'pending',
            count INT DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            INDEX (status), INDEX (created_at), INDEX (updated_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    ";
    executeQuery($sql);
}

function uidExists($uid){
    $row = fetchRow('SELECT id FROM campaigns WHERE campaign_uid = ?', [ $uid ]);
    return !!$row;
}

function generateUid12(){
    // Generate unique 12-digit numeric string
    for ($i=0; $i<10; $i++) {
        $uid = '';
        for ($j=0; $j<12; $j++) { $uid .= (string) random_int(0,9); }
        if (!uidExists($uid)) return $uid;
    }
    // Fallback using time + random
    $uid = substr((string)(time()) . (string)random_int(100000, 999999), 0, 12);
    return $uid;
}

createCampaignsTableIfNotExists();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
// remove 'test/api/campaigns_api.php'
$segments = array_slice($segments, 3);
$idFromPath = isset($segments[0]) && ctype_digit($segments[0]) ? (int)$segments[0] : 0;
$uidFromPath = isset($segments[0]) && preg_match('/^\d{12}$/', $segments[0]) ? $segments[0] : '';

switch ($method) {
    case 'GET': {
        if ($idFromPath > 0) {
            $row = fetchRow('SELECT * FROM campaigns WHERE id = ?', [ $idFromPath ]);
            if (!$row) respond([ 'success'=>false, 'message'=>'غير موجود' ], 404);
            respond([ 'success'=>true, 'data'=>$row ]);
        }
        if ($uidFromPath !== '') {
            $row = fetchRow('SELECT * FROM campaigns WHERE campaign_uid = ?', [ $uidFromPath ]);
            if (!$row) respond([ 'success'=>false, 'message'=>'غير موجود' ], 404);
            respond([ 'success'=>true, 'data'=>$row ]);
        }
        $q = isset($_GET['q']) ? trim((string)$_GET['q']) : '';
        $status = isset($_GET['status']) ? trim((string)$_GET['status']) : '';
        $where = [];$params = [];
        if ($q !== '') { $where[] = '(LOWER(name) LIKE ? OR campaign_uid LIKE ?)'; $params[] = '%' . strtolower($q) . '%'; $params[] = '%' . $q . '%'; }
        if ($status !== '' && in_array($status, ['pending','running','paused','stopped','finished'])) { $where[] = 'status = ?'; $params[] = $status; }
        $sql = 'SELECT * FROM campaigns';
        if ($where) $sql .= ' WHERE ' . implode(' AND ', $where);
        $sql .= ' ORDER BY updated_at DESC';
        $rows = fetchAll($sql, $params) ?: [];
        respond([ 'success'=>true, 'data'=>$rows ]);
    }
    case 'POST': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $name = trim((string)($input['name'] ?? ''));
        $names = $input['names'] ?? [];
        $number = isset($input['number']) ? (string)$input['number'] : '';
        $range_from = isset($input['range_from']) ? (string)$input['range_from'] : '';
        $range_to = isset($input['range_to']) ? (string)$input['range_to'] : '';
        if ($name === '') respond([ 'success'=>false, 'message'=>'اسم الحملة مطلوب' ], 400);
        if (!is_array($names)) $names = [];
        $uid = generateUid12();
        $ok = executeQuery('INSERT INTO campaigns (campaign_uid, name, names, number, range_from, range_to, status, count) VALUES (?,?,?,?,?,?,"pending",0)', [
            $uid, $name, json_encode($names, JSON_UNESCAPED_UNICODE), ($number!==''? (int)$number : 0), ($range_from!==''? (int)$range_from : null), ($range_to!==''? (int)$range_to : null)
        ]);
        if ($ok) {
            $id = (int)getLastInsertId();
            $row = fetchRow('SELECT * FROM campaigns WHERE id = ?', [ $id ]);
            respond([ 'success'=>true, 'message'=>'تم إنشاء الحملة', 'id'=>$id, 'uid'=>$uid, 'data'=>$row ]);
        }
        respond([ 'success'=>false, 'message'=>'فشل إنشاء الحملة' ], 500);
    }
    case 'PUT': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $id = isset($input['id']) ? (int)$input['id'] : ($idFromPath ?: 0);
        $uid = isset($input['uid']) ? (string)$input['uid'] : ($uidFromPath ?: '');
        if ($id <= 0 && $uid === '') respond([ 'success'=>false, 'message'=>'معرف غير صالح' ], 400);
        $fields = [];$params = [];$where='';
        if (isset($input['name'])) { $fields[] = 'name = ?'; $params[] = trim((string)$input['name']); }
        if (isset($input['names'])) { $fields[] = 'names = ?'; $params[] = json_encode(is_array($input['names'])? $input['names']: [], JSON_UNESCAPED_UNICODE); }
        if (isset($input['number'])) { $fields[] = 'number = ?'; $params[] = (int)$input['number']; }
        if (array_key_exists('range_from',$input)) { $fields[] = 'range_from = ?'; $params[] = ($input['range_from']!==null && $input['range_from']!=='')? (int)$input['range_from'] : null; }
        if (array_key_exists('range_to',$input)) { $fields[] = 'range_to = ?'; $params[] = ($input['range_to']!==null && $input['range_to']!=='')? (int)$input['range_to'] : null; }
        if (isset($input['status'])) {
            $st = (string)$input['status'];
            $allowed = ['pending','running','paused','stopped','finished'];
            if (!in_array($st, $allowed, true)) respond([ 'success'=>false, 'message'=>'حالة غير صحيحة' ], 400);
            $fields[] = 'status = ?'; $params[] = $st;
        }
        if (isset($input['count'])) { $fields[] = 'count = ?'; $params[] = (int)$input['count']; }
        if (!$fields) respond([ 'success'=>false, 'message'=>'لا يوجد بيانات للتعديل' ], 400);
        if ($id > 0) { $where = 'id = ?'; $params[] = $id; }
        else { $where = 'campaign_uid = ?'; $params[] = $uid; }
        $ok = executeQuery('UPDATE campaigns SET ' . implode(', ', $fields) . ', updated_at = NOW() WHERE ' . $where, $params);
        if ($ok) respond([ 'success'=>true, 'message'=>'تم التعديل' ]);
        respond([ 'success'=>false, 'message'=>'فشل التعديل' ], 500);
    }
    case 'DELETE': {
        // single by id or uid in path
        if ($idFromPath > 0) {
            $ok = executeQuery('DELETE FROM campaigns WHERE id = ?', [ $idFromPath ]);
            if ($ok) respond([ 'success'=>true, 'message'=>'تم الحذف' ]);
            respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
        }
        if ($uidFromPath !== '') {
            $ok = executeQuery('DELETE FROM campaigns WHERE campaign_uid = ?', [ $uidFromPath ]);
            if ($ok) respond([ 'success'=>true, 'message'=>'تم الحذف' ]);
            respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
        }
        $payload = json_decode(file_get_contents('php://input'), true) ?: [];
        $ids = isset($payload['ids']) && is_array($payload['ids']) ? array_map('intval', $payload['ids']) : [];
        //$uids = isset($payload['uids']) && is_array($payload['uids']) ? array_filter($payload['uids'], fn($u)=>preg_match('///^\d{12}$/',$u)) : [];
        $uids = (isset($payload['uids']) && is_array($payload['uids']))? array_filter($payload['uids'], function ($u) {return preg_match('/^\d{12}$/', (string)$u);}): [];
        if (!$ids && !$uids) respond([ 'success'=>false, 'message'=>'لا توجد معرفات' ], 400);
        $ok = true;
        if ($ids) {
            $in = implode(',', array_fill(0, count($ids), '?'));
            $ok = $ok && executeQuery('DELETE FROM campaigns WHERE id IN (' . $in . ')', $ids);
        }
        if ($uids) {
            $in = implode(',', array_fill(0, count($uids), '?'));
            $ok = $ok && executeQuery('DELETE FROM campaigns WHERE campaign_uid IN (' . $in . ')', $uids);
        }
        if ($ok) respond([ 'success'=>true, 'message'=>'تم حذف العناصر المحددة' ]);
        respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
    }
}

respond([ 'success'=>false, 'message'=>'طريقة غير مدعومة' ], 405);
