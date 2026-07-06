<?php
// tools_api.php - CRUD for tools entries
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') { http_response_code(204); exit; }

require_once '../config/database.php';

function respond($data, $code = 200){ http_response_code($code); echo json_encode($data, JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES); exit; }

function createToolsTableIfNotExists(){
    $sql = "
        CREATE TABLE IF NOT EXISTS tools (
            id INT AUTO_INCREMENT PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            status ENUM('working','not_working') NOT NULL DEFAULT 'working',
            visible TINYINT(1) NOT NULL DEFAULT 1,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            INDEX(status), INDEX(visible)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    ";
    executeQuery($sql);
}

createToolsTableIfNotExists();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
$segments = array_slice($segments, 3); // remove 'test/api/tools_api.php'
$idFromPath = isset($segments[0]) && ctype_digit($segments[0]) ? (int)$segments[0] : 0;

switch ($method) {
    case 'GET': {
        if ($idFromPath > 0){
            $row = fetchRow('SELECT * FROM tools WHERE id = ?', [ $idFromPath ]);
            if (!$row) respond(['success'=>false,'message'=>'غير موجود'],404);
            respond(['success'=>true,'data'=>$row]);
        }
        $q = isset($_GET['q']) ? trim((string)$_GET['q']) : '';
        $status = isset($_GET['status']) ? trim((string)$_GET['status']) : '';
        $visible = isset($_GET['visible']) ? trim((string)$_GET['visible']) : '';
        $where = [];$params=[];
        if ($q !== '') { $where[]='LOWER(name) LIKE ?'; $params[]='%'.strtolower($q).'%'; }
        if ($status !== '' && in_array($status,['working','not_working'],true)) { $where[]='status = ?'; $params[]=$status; }
        if ($visible !== '' && ($visible==='0' || $visible==='1')) { $where[]='visible = ?'; $params[]=(int)$visible; }
        $sql = 'SELECT * FROM tools';
        if ($where) $sql .= ' WHERE ' . implode(' AND ', $where);
        $sql .= ' ORDER BY updated_at DESC';
        $rows = fetchAll($sql,$params) ?: [];
        respond(['success'=>true,'data'=>$rows]);
    }
    case 'POST': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $name = trim((string)($input['name'] ?? ''));
        $status = in_array(($input['status'] ?? 'working'), ['working','not_working'], true) ? $input['status'] : 'working';
        $visible = isset($input['visible']) ? (int)!!$input['visible'] : 1;
        if ($name === '') respond(['success'=>false,'message'=>'الاسم مطلوب'],400);
        $ok = executeQuery('INSERT INTO tools (name,status,visible) VALUES (?,?,?)',[ $name, $status, $visible ]);
        if ($ok){ $id=(int)getLastInsertId(); respond(['success'=>true,'message'=>'تم الإنشاء','id'=>$id]); }
        respond(['success'=>false,'message'=>'فشل الإنشاء'],500);
    }
    case 'PUT': {
        $input = json_decode(file_get_contents('php://input'), true) ?: [];
        $id = isset($input['id']) ? (int)$input['id'] : $idFromPath;
        if ($id <= 0) respond(['success'=>false,'message'=>'معرف غير صالح'],400);
        $fields=[]; $params=[];
        if (isset($input['name'])) { $fields[]='name=?'; $params[] = trim((string)$input['name']); }
        if (isset($input['status'])) { $fields[]='status=?'; $params[] = in_array($input['status'],['working','not_working'],true)?$input['status']:'working'; }
        if (isset($input['visible'])) { $fields[]='visible=?'; $params[] = (int)!!$input['visible']; }
        if (!$fields) respond(['success'=>false,'message'=>'لا يوجد بيانات للتعديل'],400);
        $params[] = $id;
        $ok = executeQuery('UPDATE tools SET '.implode(', ',$fields).', updated_at=NOW() WHERE id=?',$params);
        if ($ok) respond(['success'=>true,'message'=>'تم التعديل']);
        respond(['success'=>false,'message'=>'فشل التعديل'],500);
    }
    case 'DELETE': {
        if ($idFromPath > 0){
            $ok = executeQuery('DELETE FROM tools WHERE id = ?', [ $idFromPath ]);
            if ($ok) respond(['success'=>true,'message'=>'تم الحذف']);
            respond(['success'=>false,'message'=>'فشل الحذف'],500);
        }
        $payload = json_decode(file_get_contents('php://input'), true) ?: [];
        $ids = isset($payload['ids']) && is_array($payload['ids']) ? array_map('intval',$payload['ids']) : [];
        if (!$ids) respond(['success'=>false,'message'=>'لا توجد معرفات'],400);
        $in = implode(',', array_fill(0,count($ids),'?'));
        $ok = executeQuery('DELETE FROM tools WHERE id IN ('.$in.')', $ids);
        if ($ok) respond(['success'=>true,'message'=>'تم حذف العناصر المحددة']);
        respond(['success'=>false,'message'=>'فشل الحذف'],500);
    }
}

respond(['success'=>false,'message'=>'طريقة غير مدعومة'],405);
