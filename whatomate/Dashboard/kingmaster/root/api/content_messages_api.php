<?php
// api/content_messages_api.php
// Returns saved message presets from content_messages
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(200); exit; }

@include_once dirname(__DIR__).'/config/database.php';
if (!function_exists('getDB')) { header('Content-Type: application/json'); echo json_encode(['success'=>false,'message'=>'DB config missing']); exit; }

function ok($d){ header('Content-Type: application/json'); echo json_encode(['success'=>true,'data'=>$d]); exit; }
function err($m){ header('Content-Type: application/json'); echo json_encode(['success'=>false,'message'=>$m]); exit; }

try{
  $db = getDB();
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch(PDO::FETCH_ASSOC)['db'] ?? null;
  $chk = $db->prepare("SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME='content_messages'");
  $chk->execute([$dbName]);
  if (!$chk->fetchColumn()) ok([]);

  // Detect columns
  $cand = ['id','title','name','label','kind','type','text','body','content','template_id','media_id'];
  $in = implode(',', array_fill(0, count($cand), '?'));
  $colQ = $db->prepare("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME='content_messages' AND COLUMN_NAME IN ($in)");
  $colQ->execute(array_merge([$dbName], $cand));
  $cols = [];
  while($r=$colQ->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME'];
  $has = function($c) use ($cols){ return in_array($c,$cols,true); };

  $sel = [];
  $sel[] = $has('id') ? 'id' : 'NULL AS id';
  if ($has('title')) $sel[] = 'title';
  else if ($has('name')) $sel[] = 'name AS title';
  else if ($has('label')) $sel[] = 'label AS title';
  else $sel[] = "CONCAT('Preset ', id) AS title";

  if ($has('kind')) $sel[] = 'kind'; else if ($has('type')) $sel[] = 'type AS kind'; else $sel[] = "NULL AS kind";
  if ($has('text')) $sel[] = 'text'; else if ($has('body')) $sel[] = 'body AS text'; else if ($has('content')) $sel[] = 'content AS text'; else $sel[] = 'NULL AS text';
  $sel[] = $has('template_id') ? 'template_id' : 'NULL AS template_id';
  $sel[] = $has('media_id') ? 'media_id' : 'NULL AS media_id';

  $sql = 'SELECT '.implode(',', $sel).' FROM content_messages ORDER BY id DESC LIMIT 500';
  $rows = [];
  foreach ($db->query($sql) as $row) $rows[] = $row;
  ok($rows);
}catch(Throwable $e){ err('Server error'); }
