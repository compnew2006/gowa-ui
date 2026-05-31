<?php
// accounts_api.php - CRUD for accounts management
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

function createAccountsTableIfNotExists(){
  $sql = "
    CREATE TABLE IF NOT EXISTS accounts (
      id INT AUTO_INCREMENT PRIMARY KEY,
      name VARCHAR(255) NOT NULL DEFAULT 'name account',
      account_uid VARCHAR(64) NOT NULL DEFAULT '100000033',
      channel ENUM('facebook','whatsapp','instagram','telegram','email','sms','tiktok','linkedin') NOT NULL,
      status ENUM('active','closed','inactive') NOT NULL DEFAULT 'inactive',
      method ENUM('cookies','data','') NOT NULL DEFAULT '',
      cookies_text LONGTEXT NULL,
      data JSON NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      INDEX (channel), INDEX (status), INDEX (created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
  ";
  executeQuery($sql);
}

createAccountsTableIfNotExists();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
// remove 'test/api/accounts_api.php'
$segments = array_slice($segments, 3);
$idFromPath = isset($segments[0]) && ctype_digit($segments[0]) ? (int)$segments[0] : 0;

switch ($method) {
  case 'GET': {
    if ($idFromPath > 0) {
      $row = fetchRow('SELECT * FROM accounts WHERE id = ?', [ $idFromPath ]);
      if (!$row) respond([ 'success' => false, 'message' => 'غير موجود' ], 404);
      respond([ 'success' => true, 'data' => $row ]);
    }
    $q = isset($_GET['q']) ? trim((string)$_GET['q']) : '';
    $status = isset($_GET['status']) ? trim((string)$_GET['status']) : '';
    $channel = isset($_GET['channel']) ? trim((string)$_GET['channel']) : '';
    $where = [];$params = [];
    if ($q !== '') { $where[] = '(LOWER(name) LIKE ? OR account_uid LIKE ?)'; $params[] = '%' . strtolower($q) . '%'; $params[] = '%' . $q . '%'; }
    if ($status !== '' && in_array($status, ['active','closed','inactive'], true)) { $where[] = 'status = ?'; $params[] = $status; }
    if ($channel !== '' && in_array($channel, ['facebook','whatsapp','instagram','telegram','email','sms','tiktok','linkedin'], true)) { $where[] = 'channel = ?'; $params[] = $channel; }
    $sql = 'SELECT * FROM accounts';
    if ($where) $sql .= ' WHERE ' . implode(' AND ', $where);
    $sql .= ' ORDER BY updated_at DESC';
    $rows = fetchAll($sql, $params) ?: [];
    respond([ 'success' => true, 'data' => $rows ]);
  }
  case 'POST': {
    $input = json_decode(file_get_contents('php://input'), true) ?: [];
    $name = trim((string)($input['name'] ?? 'name account'));
    $account_uid = trim((string)($input['account_uid'] ?? '100000033'));
    $channel = trim((string)($input['channel'] ?? 'facebook'));
    $status = trim((string)($input['status'] ?? 'inactive'));
    $method = trim((string)($input['method'] ?? ''));
    $cookies_text = isset($input['cookies_text']) ? (string)$input['cookies_text'] : null;
    $data = isset($input['data']) ? $input['data'] : null;
    if ($name === '') $name = 'name account';
    if ($account_uid === '') $account_uid = '100000033';
    $allowedChannels = ['facebook','whatsapp','instagram','telegram','email','sms','tiktok','linkedin'];
    if (!in_array($channel, $allowedChannels, true)) respond([ 'success'=>false, 'message'=>'قناة غير صحيحة' ], 400);
    $allowedStatus = ['active','closed','inactive'];
    if (!in_array($status, $allowedStatus, true)) $status = 'inactive';
    if ($method !== '' && !in_array($method, ['cookies','data'], true)) $method = '';
    $ok = executeQuery('INSERT INTO accounts (name, account_uid, channel, status, method, cookies_text, data) VALUES (?,?,?,?,?,?,?)', [
      $name, $account_uid, $channel, $status, $method, $cookies_text, $data ? json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) : null
    ]);
    if ($ok) { $id = (int)getLastInsertId(); $row = fetchRow('SELECT * FROM accounts WHERE id = ?', [ $id ]); respond([ 'success'=>true, 'message'=>'تم إنشاء الحساب', 'id'=>$id, 'data'=>$row ]); }
    respond([ 'success'=>false, 'message'=>'فشل الإنشاء' ], 500);
  }
  case 'PUT': {
    $input = json_decode(file_get_contents('php://input'), true) ?: [];
    $id = isset($input['id']) ? (int)$input['id'] : $idFromPath;
    if ($id <= 0) respond([ 'success'=>false, 'message'=>'معرف غير صالح' ], 400);
    $fields = [];$params = [];
    if (isset($input['name'])) { $fields[]='name=?'; $params[] = trim((string)$input['name']); }
    if (isset($input['account_uid'])) { $fields[]='account_uid=?'; $params[] = trim((string)$input['account_uid']); }
    if (isset($input['channel'])) { $fields[]='channel=?'; $params[] = trim((string)$input['channel']); }
    if (isset($input['status'])) { $fields[]='status=?'; $params[] = trim((string)$input['status']); }
    if (isset($input['method'])) { $fields[]='method=?'; $params[] = trim((string)$input['method']); }
    if (array_key_exists('cookies_text', $input)) { $fields[]='cookies_text=?'; $params[] = (string)$input['cookies_text']; }
    if (array_key_exists('data', $input)) { $fields[]='data=?'; $params[] = $input['data'] ? json_encode($input['data'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) : null; }
    if (!$fields) respond([ 'success'=>false, 'message'=>'لا يوجد بيانات للتعديل' ], 400);
    $params[] = $id;
    $ok = executeQuery('UPDATE accounts SET ' . implode(', ', $fields) . ', updated_at=NOW() WHERE id=?', $params);
    if ($ok) respond([ 'success'=>true, 'message'=>'تم التعديل' ]);
    respond([ 'success'=>false, 'message'=>'فشل التعديل' ], 500);
  }
  case 'DELETE': {
    if ($idFromPath > 0) {
      $ok = executeQuery('DELETE FROM accounts WHERE id = ?', [ $idFromPath ]);
      if ($ok) respond([ 'success'=>true, 'message'=>'تم الحذف' ]);
      respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
    }
    $payload = json_decode(file_get_contents('php://input'), true) ?: [];
    $ids = isset($payload['ids']) && is_array($payload['ids']) ? array_map('intval', $payload['ids']) : [];
    if (!$ids) respond([ 'success'=>false, 'message'=>'لا توجد معرفات' ], 400);
    $in = implode(',', array_fill(0, count($ids), '?'));
    $ok = executeQuery('DELETE FROM accounts WHERE id IN (' . $in . ')', $ids);
    if ($ok) respond([ 'success'=>true, 'message'=>'تم حذف الحسابات المحددة' ]);
    respond([ 'success'=>false, 'message'=>'فشل الحذف' ], 500);
  }
}

respond([ 'success'=>false, 'message'=>'طريقة غير مدعومة' ], 405);
