<?php
// api/media_api.php
// Returns media files from media_files table: id, title, url
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(200); exit; }

@include_once dirname(__DIR__).'/config/database.php';
if (!function_exists('getDB')) {
  http_response_code(500);
  echo json_encode(['success'=>false,'message'=>'Database config not found']);
  exit;
}

function json_ok($data){ header('Content-Type: application/json; charset=utf-8'); echo json_encode(['success'=>true,'data'=>$data]); exit; }
function json_err($msg){ header('Content-Type: application/json; charset=utf-8'); echo json_encode(['success'=>false,'message'=>$msg]); exit; }

try {
  $db = getDB();
  // Ensure table exists? If not, return empty
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch(PDO::FETCH_ASSOC)['db'] ?? null;
  $stmt = $db->prepare("SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME='media_files'");
  $stmt->execute([$dbName]);
  if (!$stmt->fetchColumn()) { json_ok([]); }

  $q = $db->query("SELECT id, title, url FROM media_files ORDER BY id DESC LIMIT 500");
  $rows = [];
  if ($q) while ($r = $q->fetch(PDO::FETCH_ASSOC)) { $rows[] = $r; }
  json_ok($rows);
} catch (Throwable $e) {
  json_err('Server error');
}
