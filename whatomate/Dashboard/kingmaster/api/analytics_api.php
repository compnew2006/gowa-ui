<?php
// api/analytics_api.php
// Provides analytics and exports for users, subscriptions, and packages
// Note: Adjust DB connection include/path as needed.

header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(200); exit; }

// Include PDO database helper
@include_once dirname(__DIR__).'/config/database.php';
if (!function_exists('getDB')) {
  http_response_code(500);
  echo json_encode(['success'=>false,'message'=>'Database config not found (config/database.php)']);
  exit;
}

function pdo() {
  return getDB(); // from config/database.php
}

function json_response($ok, $data = null, $msg = null) {
  header('Content-Type: application/json; charset=utf-8');
  echo json_encode(['success'=>$ok,'data'=>$data,'message'=>$msg]);
  exit;
}

function get_param($key, $default=null) {
  return $_GET[$key] ?? $_POST[$key] ?? $default;
}

function read_json_body() {
  $raw = file_get_contents('php://input');
  if (!$raw) return [];
  $data = json_decode($raw, true);
  return is_array($data) ? $data : [];
}

$action = strtolower(get_param('action','overview'));
$method = $_SERVER['REQUEST_METHOD'];
$DEBUG = (bool) get_param('debug', 0);

try {
  switch ($action) {
    case 'overview':
      overview();
      break;
    case 'geo':
      geo();
      break;
    case 'packages':
      packages_chart();
      break;
    case 'export_all_users':
      export_all_users_csv();
      break;
    case 'filter_subs':
      if ($method === 'GET' && isset($_GET['export'])) export_filtered_csv(); else filter_subs();
      break;
    case 'avg_age':
      avg_age();
      break;
    case 'birthdays':
      birthdays();
      break;
    case 'birthday_alerts':
      birthday_alerts();
      break;
    case 'export_country_users':
      export_country_users_csv();
      break;
    case 'search_users':
      search_users();
      break;
    case 'export_search_users':
      export_search_users_csv();
      break;
    case 'birthdays_stats':
      birthdays_stats();
      break;
    default:
      json_response(false,null,'Unknown action');
  }
} catch (Throwable $e) {
  http_response_code(500);
  $msg = $DEBUG ? ($e->getMessage() . (method_exists($e,'getCode')? ' [code '.$e->getCode().']':'')) : 'Server error';
  json_response(false,null,$msg);
}

function overview() {
  $db = pdo();
  $tot = $db->query("SELECT COUNT(*) c FROM users")->fetch();
  $total_users = $tot ? (int)($tot['c'] ?? 0) : 0;

  $row = $db->query("SELECT SUM(CASE WHEN status='active' THEN 1 ELSE 0 END) a, SUM(CASE WHEN status='inactive' THEN 1 ELSE 0 END) i FROM users")->fetch();
  $active = (int)($row['a'] ?? 0); $inactive = (int)($row['i'] ?? 0);

  $last_date = null;
  foreach (["SELECT MAX(created_at) m FROM users","SELECT MAX(created) m FROM users","SELECT MAX(registration_date) m FROM users"] as $sql) {
    $res = $db->query($sql); $v = $res ? ($res->fetch()['m'] ?? null) : null; if ($v) { $last_date = $v; break; }
  }

  $days = (int)(get_param('days', 10));
  $end = (new DateTime('today'))->modify("+{$days} days")->format('Y-m-d');
  $stmt = $db->prepare("SELECT COUNT(DISTINCT user_id) c FROM user_subscriptions WHERE DATE(end_date) BETWEEN CURDATE() AND :end");
  $stmt->execute([':end'=>$end]);
  $expiring = (int)($stmt->fetch()['c'] ?? 0);

  // Average age (years) from users.brithday
  $avg_age = null;
  $r = $db->query("SELECT ROUND(AVG(TIMESTAMPDIFF(YEAR, brithday, CURDATE())),1) a FROM users WHERE brithday IS NOT NULL AND brithday <> '0000-00-00'");
  if ($r) { $rowA = $r->fetch(PDO::FETCH_ASSOC); $avg_age = $rowA['a'] ?? null; }

  // Total points if column exists
  $total_points = null;
  try {
    $dbNameRow = $db->query('SELECT DATABASE() AS db')->fetch(PDO::FETCH_ASSOC);
    $dbName = $dbNameRow['db'] ?? null;
    if ($dbName) {
      $stmt = $db->prepare("SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME='users' AND COLUMN_NAME='points'");
      $stmt->execute([$dbName]);
      $hasPoints = (bool)$stmt->fetchColumn();
      if ($hasPoints) {
        $r2 = $db->query("SELECT COALESCE(SUM(points),0) s FROM users");
        if ($r2) { $rowP = $r2->fetch(PDO::FETCH_ASSOC); $total_points = $rowP['s'] ?? 0; }
      }
    }
  } catch (Throwable $e) {}

  json_response(true,[
    'total_users'=>$total_users,
    'active'=>$active,
    'inactive'=>$inactive,
    'last_user_date'=>$last_date,
    'expiring_in_days'=>$expiring,
    'avg_age'=>$avg_age,
    'total_points'=>$total_points
  ]);
}
function geo() {
  $db = pdo();
  $data = [];
  foreach ($db->query("SELECT timezone, COUNT(*) c FROM users WHERE timezone IS NOT NULL AND timezone <> '' GROUP BY timezone") as $row) {
    $tz = $row['timezone']; $count = (int)$row['c'];
    $cc = 'ZZ';
    try {
      $tzObj = @new DateTimeZone($tz);
      if ($tzObj && method_exists($tzObj, 'getLocation')) {
        $loc = @$tzObj->getLocation();
        if ($loc && !empty($loc['country_code'])) { $cc = strtoupper($loc['country_code']); }
      }
    } catch (Throwable $e) {}
    if ($cc === 'ZZ') {
      // Fallback mapping for common timezones -> country codes
      $map = [
        'Asia/Riyadh'=>'SA','Asia/Cairo'=>'EG','Asia/Dubai'=>'AE','Europe/London'=>'GB','America/New_York'=>'US',
        'Asia/Amman'=>'JO','Asia/Beirut'=>'LB','Asia/Gaza'=>'PS','Asia/Hebron'=>'PS','Africa/Casablanca'=>'MA',
        'Africa/Tunis'=>'TN','Africa/Algiers'=>'DZ','Africa/Tripoli'=>'LY','Africa/Khartoum'=>'SD','Asia/Kuwait'=>'KW',
        'Asia/Bahrain'=>'BH','Asia/Qatar'=>'QA','Asia/Muscat'=>'OM'
      ];
      if (isset($map[$tz])) $cc = $map[$tz];
    }
    if (!isset($data[$cc])) $data[$cc] = 0;
    $data[$cc] += $count;
  }
  $out = [];
  foreach ($data as $cc=>$cnt) {
    $out[] = ['code'=>$cc,'name'=>code_to_name($cc), 'count'=>$cnt];
  }
  json_response(true, $out);
}

function code_to_name($cc){
  $map = [
    'SA'=>'Saudi Arabia','EG'=>'Egypt','AE'=>'United Arab Emirates','QA'=>'Qatar','BH'=>'Bahrain','KW'=>'Kuwait','OM'=>'Oman','JO'=>'Jordan','LB'=>'Lebanon','IQ'=>'Iraq','SY'=>'Syria','YE'=>'Yemen','PS'=>'Palestine','MA'=>'Morocco','DZ'=>'Algeria','TN'=>'Tunisia','LY'=>'Libya','SD'=>'Sudan',
    'US'=>'United States','GB'=>'United Kingdom','DE'=>'Germany','FR'=>'France','IT'=>'Italy','ES'=>'Spain','NL'=>'Netherlands','BE'=>'Belgium','CH'=>'Switzerland','AT'=>'Austria','GR'=>'Greece','PT'=>'Portugal','PL'=>'Poland','SE'=>'Sweden','NO'=>'Norway','FI'=>'Finland','UA'=>'Ukraine',
    'CA'=>'Canada','AU'=>'Australia','RU'=>'Russia','IN'=>'India','PK'=>'Pakistan','ID'=>'Indonesia','TR'=>'Turkey','CN'=>'China','JP'=>'Japan','KR'=>'South Korea','VN'=>'Vietnam','PH'=>'Philippines','MY'=>'Malaysia','TH'=>'Thailand','SG'=>'Singapore','BR'=>'Brazil','AR'=>'Argentina','MX'=>'Mexico','ZA'=>'South Africa','NG'=>'Nigeria','KE'=>'Kenya','ET'=>'Ethiopia'
  ];
  return $map[$cc] ?? $cc;
}

function tz_to_cc($tz){
  $cc = 'ZZ';
  try {
    $tzObj = @new DateTimeZone($tz);
    if ($tzObj && method_exists($tzObj,'getLocation')){
      $loc = @$tzObj->getLocation(); if ($loc && !empty($loc['country_code'])) { $cc = strtoupper($loc['country_code']); }
    }
  } catch (Throwable $e) {}
  if ($cc==='ZZ'){
    $map = [
      'Asia/Riyadh'=>'SA','Asia/Cairo'=>'EG','Asia/Dubai'=>'AE','Europe/London'=>'GB','America/New_York'=>'US',
      'Asia/Amman'=>'JO','Asia/Beirut'=>'LB','Asia/Gaza'=>'PS','Asia/Hebron'=>'PS','Africa/Casablanca'=>'MA',
      'Africa/Tunis'=>'TN','Africa/Algiers'=>'DZ','Africa/Tripoli'=>'LY','Africa/Khartoum'=>'SD','Asia/Kuwait'=>'KW',
      'Asia/Bahrain'=>'BH','Asia/Qatar'=>'QA','Asia/Muscat'=>'OM'
    ];
    if (isset($map[$tz])) $cc = $map[$tz];
  }
  return $cc;
}

function export_country_users_csv(){
  $db = pdo();
  $country = get_param('country');
  if (!$country) { http_response_code(400); echo 'country required'; exit; }
  $needle = trim($country);
  $isCode = strlen($needle)===2;
  $needleCode = strtoupper($needle);
  $needleName = $isCode ? code_to_name($needleCode) : $needle;

  // Detect optional columns
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasCol = function($table,$col) use ($db,$dbName){
    $stmt = $db->prepare('SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME=?');
    $stmt->execute([$dbName,$table,$col]);
    return (bool)$stmt->fetchColumn();
  };
  $usersCols = ['phone'=>$hasCol('users','phone'), 'points'=>$hasCol('users','points'), 'work'=>$hasCol('users','work')];
  $subsExists = (function() use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,'user_subscriptions']); return (bool)$s->fetchColumn(); })();
  $subsHasEnd = $subsExists && $hasCol('user_subscriptions','end_date') && $hasCol('user_subscriptions','user_id');

  // Build SQL dynamically
  $select = 'u.id, u.first_name, u.last_name, u.email, u.status, u.timezone, u.brithday, u.created_at';
  $select .= $usersCols['phone'] ? ', u.phone' : ", NULL AS phone";
  $select .= $usersCols['points'] ? ', u.points' : ", NULL AS points";
  $select .= $usersCols['work'] ? ', u.work' : ", NULL AS work";
  $join = '';
  if ($subsHasEnd) {
    $select .= ', s.end_date';
    $join = ' LEFT JOIN (SELECT user_id, MAX(end_date) AS end_date FROM user_subscriptions GROUP BY user_id) s ON s.user_id = u.id';
  } else {
    $select .= ', NULL AS end_date';
  }
  $sql = "SELECT $select FROM users u$join WHERE u.timezone IS NOT NULL AND u.timezone<>''";

  $stmt = $db->query($sql);
  $rows = [];
  while ($u = $stmt->fetch(PDO::FETCH_ASSOC)){
    $cc = tz_to_cc($u['timezone']);
    $name = code_to_name($cc);
    if ($isCode) {
      if ($cc === $needleCode) { $rows[] = $u; }
    } else {
      if (strcasecmp($name, $needleName)===0) { $rows[] = $u; }
    }
  }

  header('Content-Type: text/csv; charset=utf-8');
  $fname = 'users_'.($isCode?$needleCode:preg_replace('/[^A-Za-z0-9_-]+/','_', $needleName)).'_'.date('Ymd_His').'.csv';
  header('Content-Disposition: attachment; filename='.$fname);
  $out = fopen('php://output', 'w'); fwrite($out, "\xEF\xBB\xBF");
  fputcsv($out, ['ID','First Name','Last Name','Email','Status','Timezone','Birthday','Created At','Phone','Points','Work','End Date']);
  foreach ($rows as $r){ fputcsv($out, [
    $r['id'],$r['first_name'],$r['last_name'],$r['email'],$r['status'],$r['timezone'],$r['brithday'],$r['created_at'],
    $r['phone'],$r['points'],$r['work'],$r['end_date']
  ]); }
  fclose($out); exit;
}

function packages_chart() {
  $db = pdo();

  // Helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasTable = function($t) use ($db, $dbName) {
    $stmt = $db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA = :db AND TABLE_NAME = :t');
    $stmt->execute([':db'=>$dbName, ':t'=>$t]);
    return (bool)$stmt->fetchColumn();
  };
  $getExistingCols = function($t, $cands) use ($db, $dbName) {
    if (!$t) return [];
    if (!is_array($cands) || empty($cands)) return [];
    $in = implode(',', array_fill(0, count($cands), '?'));
    $sql = "SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)";
    $stmt = $db->prepare($sql);
    $stmt->execute(array_merge([$dbName, $t], $cands));
    $cols = [];
    while ($r = $stmt->fetch(PDO::FETCH_ASSOC)) $cols[] = $r['COLUMN_NAME'];
    // preserve candidate order
    return array_values(array_intersect($cands, $cols));
  };
  $findFirstCol = function($t, $cands) use ($getExistingCols) {
    $cols = $getExistingCols($t, $cands);
    return $cols ? $cols[0] : null;
  };

  $mergeRows = function(&$acc, $rows){
    foreach ($rows as $r){
      $label = (string)($r['label'] ?? ''); if ($label==='') continue;
      if (!isset($acc[$label])) $acc[$label] = 0;
      $acc[$label] += (int)($r['cnt'] ?? 0);
    }
  };

  $result = [];

  // Prepare packages metadata
  $packagesExists = $hasTable('packages');
  $pkgIdCol = $packagesExists ? $findFirstCol('packages', ['id','pkg_id']) : null;
  $pkgTextCols = $packagesExists ? $getExistingCols('packages', ['name_ar','name','title_ar','title','package_name','label','arabic_name','english_name']) : [];

  // Query builder
  $querySrc = function($srcTable, $alias) use ($db, $packagesExists, $pkgIdCol, $pkgTextCols, $findFirstCol) {
    $srcCol = $findFirstCol($srcTable, ['package_id','package','plan_id','plan','pkg_id','pkg','package_name']);
    if (!$srcCol) return [];

    $labelExpr = "$alias.`$srcCol`";
    $joinSql = '';

    if ($packagesExists && ($pkgIdCol || !empty($pkgTextCols))) {
      $onParts = [];
      if ($pkgIdCol) $onParts[] = "p.`$pkgIdCol` = $alias.`$srcCol`";
      foreach ($pkgTextCols as $c) { $onParts[] = "p.`$c` = $alias.`$srcCol`"; }
      if (!empty($onParts)) {
        $joinSql = " LEFT JOIN packages p ON (".implode(' OR ', $onParts).")";
        $coalesce = [];
        foreach ($pkgTextCols as $c) { $coalesce[] = "p.`$c`"; }
        if (!empty($coalesce)) {
          $labelExpr = "COALESCE(".implode(',', $coalesce).", $alias.`$srcCol` )";
        }
      }
    }

    $sql = "SELECT $labelExpr AS label, COUNT(*) cnt FROM `$srcTable` $alias$joinSql
            WHERE $alias.`$srcCol` IS NOT NULL AND $alias.`$srcCol` <> ''
            GROUP BY label ORDER BY cnt DESC LIMIT 50";

    $rows = [];
    foreach ($db->query($sql) as $row) { $rows[] = $row; }
    return $rows;
  };

  // Try from subscriptions then users
  if ($hasTable('user_subscriptions')) {
    $rows = $querySrc('user_subscriptions', 's');
    $mergeRows($result, $rows);
  }
  if (empty($result) && $hasTable('users')) {
    $rows = $querySrc('users', 'u');
    $mergeRows($result, $rows);
  }

  // Normalize output
  $out = [];
  foreach ($result as $label=>$cnt) { $out[] = ['package_id'=>$label, 'label'=>$label, 'count'=>$cnt]; }
  usort($out, function($a,$b){ return $b['count'] <=> $a['count']; });
  json_response(true, $out);
}

function export_all_users_csv() {
  $db = pdo();
  // Schema helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $getCols = function($t,$cands) use ($db,$dbName){ if(!$t) return []; $in=implode(',',array_fill(0,count($cands),'?')); $sql="SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)"; $stmt=$db->prepare($sql); $stmt->execute(array_merge([$dbName,$t], $cands)); $cols=[]; while($r=$stmt->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME']; return array_values(array_intersect($cands,$cols)); };
  $hasTable = function($t) use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,$t]); return (bool)$s->fetchColumn(); };

  $uCandidates = ['id','first_name','last_name','email','status','user_type','timezone','package_id','brithday','created_at','phone','work','points'];
  $uCols = $getCols('users', $uCandidates);

  // Build select list preserving a known order, substituting NULL for missing optional fields
  $selects = [];
  $csvHeaders = ['ID','First Name','Last Name','Email','Status','User Type','Timezone','Package','Birthday','Created At','Phone','Work','Points','End Date'];
  $map = [
    'id'=>'u.id','first_name'=>'u.first_name','last_name'=>'u.last_name','email'=>'u.email','status'=>'u.status','user_type'=>'u.user_type','timezone'=>'u.timezone','package_id'=>'u.package_id','brithday'=>'u.brithday','created_at'=>'u.created_at','phone'=>'u.phone','work'=>'u.work','points'=>'u.points'
  ];
  $order = ['id','first_name','last_name','email','status','user_type','timezone','package_id','brithday','created_at','phone','work','points'];
  foreach ($order as $c) { $selects[] = in_array($c,$uCols,true) ? $map[$c] : ("NULL AS $c"); }

  // Latest end_date
  $join = '';
  if ($hasTable('user_subscriptions')) {
    $join = ' LEFT JOIN (SELECT user_id, MAX(end_date) AS end_date FROM user_subscriptions GROUP BY user_id) s ON s.user_id = u.id';
    $selects[] = 's.end_date';
  } else {
    $selects[] = 'NULL AS end_date';
  }

  // Package label: try packages table if available
  $pkgLabel = 'NULL AS package_label';
  if ($hasTable('packages')) {
    // Try to derive label from packages if package_id exists, else keep as NULL
    if (in_array('package_id',$uCols,true)) {
      // Try common name columns
      $nameCols = $getCols('packages', ['name_ar','name','title_ar','title','package_name']);
      if (!empty($nameCols)) {
        $co = 'COALESCE('.implode(',', array_map(function($c){ return 'p.`'.$c.'`'; }, $nameCols)).')';
        $join .= ' LEFT JOIN packages p ON p.id = u.package_id';
        $pkgLabel = $co.' AS package_label';
      }
    }
  }

  // Rebuild selects to include package label in a known place (after timezone)
  // We'll create final SELECT list explicitly
  $sql = 'SELECT '.implode(', ', [
    in_array('id',$uCols)?'u.id':'NULL AS id',
    in_array('first_name',$uCols)?'u.first_name':'NULL AS first_name',
    in_array('last_name',$uCols)?'u.last_name':'NULL AS last_name',
    in_array('email',$uCols)?'u.email':'NULL AS email',
    in_array('status',$uCols)?'u.status':'NULL AS status',
    in_array('user_type',$uCols)?'u.user_type':'NULL AS user_type',
    in_array('timezone',$uCols)?'u.timezone':'NULL AS timezone',
    $pkgLabel,
    in_array('brithday',$uCols)?'u.brithday':'NULL AS brithday',
    in_array('created_at',$uCols)?'u.created_at':'NULL AS created_at',
    in_array('phone',$uCols)?'u.phone':'NULL AS phone',
    in_array('work',$uCols)?'u.work':'NULL AS work',
    in_array('points',$uCols)?'u.points':'NULL AS points',
    (strpos(implode(',',$selects),'s.end_date')!==false)?'s.end_date':'NULL AS end_date'
  ]).' FROM users u'.$join.' ORDER BY u.id ASC';

  $stmt = $db->query($sql);

  header('Content-Type: text/csv; charset=utf-8');
  header('Content-Disposition: attachment; filename=users_export_'.date('Ymd_His').'.csv');
  $out = fopen('php://output', 'w');
  fwrite($out, "\xEF\xBB\xBF");
  fputcsv($out, $csvHeaders);
  while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
    fputcsv($out, [
      $row['id'],$row['first_name'],$row['last_name'],$row['email'],$row['status'],$row['user_type'],$row['timezone'],
      $row['package_label'],$row['brithday'],$row['created_at'],$row['phone'],$row['work'],$row['points'],$row['end_date']
    ]);
  }
  fclose($out); exit;
}

function filter_subs() {
  $db = pdo();
  $payload = array_merge($_GET, read_json_body());
  $start = $payload['start_date'] ?? null; $end = $payload['end_date'] ?? null;
  if (!$start || !$end) json_response(false, null, 'start_date and end_date are required');

  // Schema helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasTable = function($t) use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,$t]); return (bool)$s->fetchColumn(); };
  $getExistingCols = function($t,$cands) use ($db,$dbName){ if(!$t) return []; $in=implode(',',array_fill(0,count($cands),'?')); $sql="SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)"; $stmt=$db->prepare($sql); $stmt->execute(array_merge([$dbName,$t], $cands)); $cols=[]; while($r=$stmt->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME']; return array_values(array_intersect($cands,$cols)); };
  $findFirstCol = function($t,$cands) use ($getExistingCols){ $cols=$getExistingCols($t,$cands); return $cols? $cols[0]: null; };

  // Detect subscription package column
  $subCol = $findFirstCol('user_subscriptions', ['package_id','package','plan_id','plan','pkg_id','pkg','package_name']);
  if (!$subCol) { // no package column, still return rows w/o package info
    $sql = "SELECT u.id, u.first_name, u.last_name, u.email, s.end_date, NULL AS package_id, NULL AS package_name
            FROM user_subscriptions s JOIN users u ON u.id=s.user_id
            WHERE DATE(s.end_date) BETWEEN DATE(:s) AND DATE(:e)
            ORDER BY s.end_date ASC";
    $stmt=$db->prepare($sql); $stmt->execute([':s'=>$start, ':e'=>$end]); $rows=$stmt->fetchAll(PDO::FETCH_ASSOC); json_response(true,$rows);
  }

  // Prepare packages join/label
  $packagesExists = $hasTable('packages');
  $pkgIdCol = $packagesExists ? $findFirstCol('packages',['id','pkg_id']) : null;
  $pkgTextCols = $packagesExists ? $getExistingCols('packages',['name_ar','name','title_ar','title','package_name','label','arabic_name','english_name']) : [];
  $onParts = [];
  if ($packagesExists) {
    if ($pkgIdCol) $onParts[] = "p.`$pkgIdCol` = s.`$subCol`";
    foreach ($pkgTextCols as $c) { $onParts[] = "p.`$c` = s.`$subCol`"; }
  }
  $joinSql = !empty($onParts) ? (" LEFT JOIN packages p ON (".implode(' OR ',$onParts).")") : '';
  $coalesce = [];
  foreach ($pkgTextCols as $c) { $coalesce[] = "p.`$c`"; }
  $labelExpr = !empty($coalesce) ? ("COALESCE(".implode(',', $coalesce).", s.`$subCol`) AS package_name") : ("s.`$subCol` AS package_name");
  $pkgIdSelect = ($subCol==='package_id') ? "s.`$subCol` AS package_id" : "NULL AS package_id";

  // Optional user columns
  $uOptCols = $getExistingCols('users', ['phone','brithday','work','points']);
  $uPhoneSel = in_array('phone',$uOptCols,true) ? 'u.phone' : 'NULL AS phone';
  $uBirthSel = in_array('brithday',$uOptCols,true) ? 'u.brithday' : 'NULL AS brithday';
  $uWorkSel  = in_array('work',$uOptCols,true) ? 'u.work' : 'NULL AS work';
  $uPtsSel   = in_array('points',$uOptCols,true) ? 'u.points' : 'NULL AS points';

  $sql = "SELECT u.id, u.first_name, u.last_name, u.email, $uPhoneSel, $uBirthSel, $uWorkSel, $uPtsSel, s.end_date, $pkgIdSelect, $labelExpr
          FROM user_subscriptions s JOIN users u ON u.id = s.user_id$joinSql
          WHERE DATE(s.end_date) BETWEEN DATE(:s) AND DATE(:e)
          ORDER BY s.end_date ASC";
  $stmt = $db->prepare($sql);
  $stmt->execute([':s'=>$start, ':e'=>$end]);
  $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);
  json_response(true, $rows);
}

function export_filtered_csv() {
  $db = pdo();
  $start = $_GET['start_date'] ?? null; $end = $_GET['end_date'] ?? null;
  if (!$start || !$end) { http_response_code(400); echo 'start_date and end_date required'; exit; }

  // Schema helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasTable = function($t) use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,$t]); return (bool)$s->fetchColumn(); };
  $getExistingCols = function($t,$cands) use ($db,$dbName){ if(!$t) return []; $in=implode(',',array_fill(0,count($cands),'?')); $sql="SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)"; $stmt=$db->prepare($sql); $stmt->execute(array_merge([$dbName,$t], $cands)); $cols=[]; while($r=$stmt->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME']; return array_values(array_intersect($cands,$cols)); };
  $findFirstCol = function($t,$cands) use ($getExistingCols){ $cols=$getExistingCols($t,$cands); return $cols? $cols[0]: null; };

  $subCol = $findFirstCol('user_subscriptions', ['package_id','package','plan_id','plan','pkg_id','pkg','package_name']);
  $packagesExists = $hasTable('packages');
  $pkgIdCol = $packagesExists ? $findFirstCol('packages',['id','pkg_id']) : null;
  $pkgTextCols = $packagesExists ? $getExistingCols('packages',['name_ar','name','title_ar','title','package_name','label','arabic_name','english_name']) : [];
  $onParts = [];
  if ($packagesExists && $subCol) {
    if ($pkgIdCol) $onParts[] = "p.`$pkgIdCol` = s.`$subCol`";
    foreach ($pkgTextCols as $c) { $onParts[] = "p.`$c` = s.`$subCol`"; }
  }
  $joinSql = !empty($onParts) ? (" LEFT JOIN packages p ON (".implode(' OR ',$onParts).")") : '';
  $coalesce = [];
  foreach ($pkgTextCols as $c) { $coalesce[] = "p.`$c`"; }
  $labelExpr = !empty($coalesce) && $subCol ? ("COALESCE(".implode(',', $coalesce).", s.`$subCol`) AS package_name") : ($subCol?"s.`$subCol` AS package_name":"NULL AS package_name");
  $pkgIdSelect = ($subCol==='package_id') ? "s.`$subCol` AS package_id" : "NULL AS package_id";

  // Optional user columns
  $uOptCols = $getExistingCols('users', ['phone','brithday','work','points']);
  $uPhoneSel = in_array('phone',$uOptCols,true) ? 'u.phone' : 'NULL AS phone';
  $uBirthSel = in_array('brithday',$uOptCols,true) ? 'u.brithday' : 'NULL AS brithday';
  $uWorkSel  = in_array('work',$uOptCols,true) ? 'u.work' : 'NULL AS work';
  $uPtsSel   = in_array('points',$uOptCols,true) ? 'u.points' : 'NULL AS points';

  $sql = "SELECT u.id, u.first_name, u.last_name, u.email, $uPhoneSel, $uBirthSel, $uWorkSel, $uPtsSel, $pkgIdSelect, $labelExpr, s.end_date
          FROM user_subscriptions s JOIN users u ON u.id = s.user_id$joinSql
          WHERE DATE(s.end_date) BETWEEN DATE(:s) AND DATE(:e)
          ORDER BY s.end_date ASC";
  $stmt = $db->prepare($sql);
  $stmt->execute([':s'=>$start, ':e'=>$end]);

  header('Content-Type: text/csv; charset=utf-8');
  header('Content-Disposition: attachment; filename=subscriptions_'.date('Ymd').'_'.preg_replace('/[^0-9]/','',$start).'_to_'.preg_replace('/[^0-9]/','',$end).'.csv');
  $out = fopen('php://output', 'w'); fwrite($out, "\xEF\xBB\xBF");
  fputcsv($out, ['ID','First Name','Last Name','Email','Phone','Birthday','Work','Points','Package ID','Package Name','End Date']);
  while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
    fputcsv($out, [
      $row['id'],$row['first_name'],$row['last_name'],$row['email'],
      $row['phone'],$row['brithday'],$row['work'],$row['points'],
      $row['package_id'],$row['package_name'],$row['end_date']
    ]);
  }
  fclose($out); exit;
}

function avg_age() {
  $db = pdo();
  $r = $db->query("SELECT ROUND(AVG(TIMESTAMPDIFF(YEAR, brithday, CURDATE())),1) a FROM users WHERE brithday IS NOT NULL AND brithday <> '0000-00-00'")->fetch();
  $a = $r['a'] ?? null;
  json_response(true, ['avg_age'=>$a]);
}

function birthdays() {
  $db = pdo();
  $month = (int)(get_param('month', 0));
  $day = (int)(get_param('day', 0));
  if ($month<=0 || $month>12) json_response(false,null,'Invalid month');
  if ($day<0 || $day>31) json_response(false,null,'Invalid day');

  // Schema helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasTable = function($t) use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,$t]); return (bool)$s->fetchColumn(); };
  $getExistingCols = function($t,$cands) use ($db,$dbName){ if(!$t) return []; $in=implode(',',array_fill(0,count($cands),'?')); $sql="SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)"; $stmt=$db->prepare($sql); $stmt->execute(array_merge([$dbName,$t], $cands)); $cols=[]; while($r=$stmt->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME']; return array_values(array_intersect($cands,$cols)); };
  $findFirstCol = function($t,$cands) use ($getExistingCols){ $cols=$getExistingCols($t,$cands); return $cols? $cols[0]: null; };

  // Optional user columns
  $uOpt = $getExistingCols('users', ['phone','work','points']);
  $uPhoneSel = in_array('phone',$uOpt,true) ? 'u.phone' : 'NULL AS phone';
  $uWorkSel  = in_array('work',$uOpt,true) ? 'u.work' : 'NULL AS work';
  $uPtsSel   = in_array('points',$uOpt,true) ? 'u.points' : 'NULL AS points';

  // Latest subscription join (optional) + package label
  $subExists = $hasTable('user_subscriptions');
  $subCol = $subExists ? $findFirstCol('user_subscriptions', ['package_id','package','plan_id','plan','pkg_id','pkg','package_name']) : null;
  $joinSubs = $subExists ? ' LEFT JOIN (SELECT user_id, MAX(end_date) AS end_date FROM user_subscriptions GROUP BY user_id) s ON s.user_id = u.id' : '';

  $packagesExists = $hasTable('packages');
  $pkgIdCol = $packagesExists ? $findFirstCol('packages',['id','pkg_id']) : null;
  $pkgTextCols = $packagesExists ? $getExistingCols('packages',['name_ar','name','title_ar','title','package_name','label','arabic_name','english_name']) : [];

  $labelExpr = 'NULL AS package_name';
  $joinPkg = '';
  if ($subCol) {
    $onParts = [];
    if ($packagesExists) {
      if ($pkgIdCol) $onParts[] = "p.`$pkgIdCol` = us.`$subCol`";
      foreach ($pkgTextCols as $c) { $onParts[] = "p.`$c` = us.`$subCol`"; }
    }
    $coalesce = [];
    foreach ($pkgTextCols as $c) { $coalesce[] = "p.`$c`"; }
    $labelExpr = !empty($coalesce) ? ("COALESCE(".implode(',', $coalesce).", us.`$subCol`) AS package_name") : ("us.`$subCol` AS package_name");
    if (!empty($onParts)) {
      $joinPkg = " LEFT JOIN (SELECT user_id, $subCol FROM user_subscriptions WHERE end_date = (SELECT MAX(end_date) FROM user_subscriptions us2 WHERE us2.user_id = user_subscriptions.user_id)) us ON us.user_id = u.id LEFT JOIN packages p ON (".implode(' OR ', $onParts).")";
    } else {
      $joinPkg = " LEFT JOIN (SELECT user_id, $subCol FROM user_subscriptions WHERE end_date = (SELECT MAX(end_date) FROM user_subscriptions us2 WHERE us2.user_id = user_subscriptions.user_id)) us ON us.user_id = u.id";
    }
  }

  $where = $day>0 ? 'MONTH(u.brithday)=:m AND DAY(u.brithday)=:d' : 'MONTH(u.brithday)=:m';
  $sql = "SELECT u.id, u.first_name, u.last_name, u.email, u.brithday, $uPhoneSel, $uWorkSel, $uPtsSel, s.end_date, $labelExpr
          FROM users u$joinSubs$joinPkg
          WHERE $where
          ORDER BY DAY(u.brithday), u.last_name, u.first_name";
  $stmt = $db->prepare($sql);
  $params = [':m'=>$month]; if ($day>0) $params[':d']=$day; $stmt->execute($params);
  $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);
  json_response(true, $rows);
}

function birthday_alerts() {
  $db = pdo();
  $sql = "SELECT id, first_name, last_name, email, brithday,
                 CASE WHEN DATE_FORMAT(brithday,'%m-%d') = DATE_FORMAT(CURDATE(),'%m-%d') THEN 'today'
                      WHEN DATE_FORMAT(brithday,'%m-%d') = DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 1 DAY),'%m-%d') THEN 'tomorrow'
                 END AS when_tag
          FROM users
          WHERE DATE_FORMAT(brithday,'%m-%d') IN (
            DATE_FORMAT(CURDATE(),'%m-%d'), DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 1 DAY),'%m-%d')
          )
          ORDER BY when_tag, last_name, first_name";
  $rows = [];
  foreach ($db->query($sql) as $r) { $rows[] = $r; }
  json_response(true, $rows);
}

function birthdays_stats(){
  $db = pdo();
  $group = strtolower((string) get_param('group','year')); // 'year' or 'month'
  // Ensure users.brithday exists
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $stmt = $db->prepare("SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME='users' AND COLUMN_NAME='brithday'");
  $stmt->execute([$dbName]);
  if (!$stmt->fetchColumn()) {
    if ($group==='month') { json_response(true, array_map(function($i){ return ['month'=>$i,'count'=>0]; }, range(1,12))); }
    else { json_response(true, []); }
  }

  if ($group==='month') {
    $res = $db->query("SELECT MONTH(brithday) m, COUNT(*) cnt FROM users WHERE brithday IS NOT NULL AND brithday <> '0000-00-00' GROUP BY m");
    $map = array_fill(1,12,0);
    if ($res) { while ($r = $res->fetch(PDO::FETCH_ASSOC)) { $m = (int)($r['m']??0); if ($m>=1 && $m<=12) $map[$m] = (int)$r['cnt']; } }
    $out = [];
    for ($i=1;$i<=12;$i++) $out[] = ['month'=>$i,'count'=>$map[$i]];
    json_response(true, $out);
  } else {
    // Group by year
    $rows = [];
    $res = $db->query("SELECT YEAR(brithday) y, COUNT(*) cnt FROM users WHERE brithday IS NOT NULL AND brithday <> '0000-00-00' GROUP BY y ORDER BY y ASC");
    if ($res) { while ($r = $res->fetch(PDO::FETCH_ASSOC)) { if (!empty($r['y'])) $rows[] = ['year'=>(int)$r['y'], 'count'=>(int)$r['cnt']]; } }
    json_response(true, $rows);
  }
}

function search_users(){
  $db = pdo();
  $q = trim((string) get_param('q',''));
  if ($q==='') json_response(true, []);

  // Schema helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasTable = function($t) use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,$t]); return (bool)$s->fetchColumn(); };
  $getExistingCols = function($t,$cands) use ($db,$dbName){ if(!$t) return []; $in=implode(',',array_fill(0,count($cands),'?')); $sql="SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)"; $stmt=$db->prepare($sql); $stmt->execute(array_merge([$dbName,$t], $cands)); $cols=[]; while($r=$stmt->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME']; return array_values(array_intersect($cands,$cols)); };
  $findFirstCol = function($t,$cands) use ($getExistingCols){ $cols=$getExistingCols($t,$cands); return $cols? $cols[0]: null; };

  $uCols = $getExistingCols('users', ['id','first_name','last_name','email','phone','work','points','status','timezone','package_id','brithday','created_at']);
  $conds = [];
  $params = [];
  $i = 0;
  if (in_array('phone',$uCols,true)) { $i++; $conds[] = "u.phone LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (in_array('email',$uCols,true)) { $i++; $conds[] = "u.email LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (in_array('first_name',$uCols,true)) { $i++; $conds[] = "u.first_name LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (in_array('last_name',$uCols,true)) { $i++; $conds[] = "u.last_name LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (empty($conds)) json_response(true, []);
  $where = '('.implode(' OR ', $conds).')';

  // Latest sub end_date
  $hasSubs = $hasTable('user_subscriptions');
  $join = $hasSubs ? ' LEFT JOIN (SELECT user_id, MAX(end_date) AS end_date FROM user_subscriptions GROUP BY user_id) s ON s.user_id = u.id' : '';

  // Package label: prefer latest subscription's package, else fallback to users.package_id
  $pkgLabel = 'NULL AS package_name';
  $joinPkg = '';
  $subCol = $hasSubs ? $findFirstCol('user_subscriptions', ['package_id','package','plan_id','plan','pkg_id','pkg','package_name']) : null;
  if ($subCol) {
    // Join latest subscription's package column as 'us.pkg'
    $joinPkg = " LEFT JOIN (SELECT user_id, $subCol AS pkg FROM user_subscriptions t WHERE end_date = (SELECT MAX(end_date) FROM user_subscriptions x WHERE x.user_id=t.user_id)) us ON us.user_id = u.id";
    $nameCols = $getExistingCols('packages', ['name_ar','name','title_ar','title','package_name']);
    if ($hasTable('packages') && !empty($nameCols)) {
      $co = 'COALESCE('.implode(',', array_map(function($c){ return 'p.`'.$c.'`'; }, $nameCols)).', us.pkg)';
      // map by id or text
      $onParts = ['p.id = us.pkg'];
      foreach ($nameCols as $c) { $onParts[] = 'p.`'.$c.'` = us.pkg'; }
      $joinPkg .= ' LEFT JOIN packages p ON ('.implode(' OR ', $onParts).')';
      $pkgLabel = $co.' AS package_name';
    } else {
      $pkgLabel = 'us.pkg AS package_name';
    }
  } else if ($hasTable('packages') && in_array('package_id',$uCols,true)){
    // Fallback: users.package_id -> packages
    $nameCols = $getExistingCols('packages', ['name_ar','name','title_ar','title','package_name']);
    if (!empty($nameCols)){
      $co = 'COALESCE('.implode(',', array_map(function($c){ return 'p.`'.$c.'`'; }, $nameCols)).')';
      $joinPkg = ' LEFT JOIN packages p ON p.id = u.package_id';
      $pkgLabel = $co.' AS package_name';
    }
  }
  $join .= $joinPkg;

  // Optional columns selections
  $sel = [
    'u.id','u.first_name','u.last_name','u.email',
    in_array('phone',$uCols,true)?'u.phone':'NULL AS phone',
    in_array('brithday',$uCols,true)?'u.brithday':'NULL AS brithday',
    in_array('work',$uCols,true)?'u.work':'NULL AS work',
    in_array('points',$uCols,true)?'u.points':'NULL AS points',
    $pkgLabel,
    $hasSubs ? 's.end_date' : 'NULL AS end_date'
  ];
  $sql = 'SELECT '.implode(',', $sel).' FROM users u'.$join.' WHERE '.$where.' ORDER BY u.id DESC LIMIT 200';
  $stmt = $db->prepare($sql);
  $stmt->execute($params);
  $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);
  json_response(true, $rows);
}

function export_search_users_csv(){
  $db = pdo();
  $q = trim((string) get_param('q',''));
  header('Content-Type: text/csv; charset=utf-8');
  header('Content-Disposition: attachment; filename=search_users_'.date('Ymd_His').'.csv');
  $out = fopen('php://output', 'w'); fwrite($out, "\xEF\xBB\xBF");
  fputcsv($out, ['ID','First Name','Last Name','Email','Phone','Birthday','Work','Points','Package Name','End Date']);
  if ($q==='') { fclose($out); exit; }

  // Schema helpers
  $dbName = $db->query('SELECT DATABASE() AS db')->fetch()['db'] ?? null;
  $hasTable = function($t) use ($db,$dbName){ $s=$db->prepare('SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?'); $s->execute([$dbName,$t]); return (bool)$s->fetchColumn(); };
  $getExistingCols = function($t,$cands) use ($db,$dbName){ if(!$t) return []; $in=implode(',',array_fill(0,count($cands),'?')); $sql="SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME=? AND COLUMN_NAME IN ($in)"; $stmt=$db->prepare($sql); $stmt->execute(array_merge([$dbName,$t], $cands)); $cols=[]; while($r=$stmt->fetch(PDO::FETCH_ASSOC)) $cols[]=$r['COLUMN_NAME']; return array_values(array_intersect($cands,$cols)); };
  $findFirstCol = function($t,$cands) use ($getExistingCols){ $cols=$getExistingCols($t,$cands); return $cols? $cols[0]: null; };

  $uCols = $getExistingCols('users', ['id','first_name','last_name','email','phone','work','points','status','timezone','package_id','brithday','created_at']);
  $conds = [];
  $params = [];
  $i = 0;
  if (in_array('phone',$uCols,true)) { $i++; $conds[] = "u.phone LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (in_array('email',$uCols,true)) { $i++; $conds[] = "u.email LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (in_array('first_name',$uCols,true)) { $i++; $conds[] = "u.first_name LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (in_array('last_name',$uCols,true)) { $i++; $conds[] = "u.last_name LIKE :q$i"; $params[":q$i"] = '%'.$q.'%'; }
  if (empty($conds)) { fclose($out); exit; }
  $where = '('.implode(' OR ', $conds).')';

  // Latest sub end_date
  $hasSubs = $hasTable('user_subscriptions');
  $join = $hasSubs ? ' LEFT JOIN (SELECT user_id, MAX(end_date) AS end_date FROM user_subscriptions GROUP BY user_id) s ON s.user_id = u.id' : '';

  // Package label similar to search_users
  $pkgLabel = 'NULL AS package_name';
  $joinPkg = '';
  $subCol = $hasSubs ? $findFirstCol('user_subscriptions', ['package_id','package','plan_id','plan','pkg_id','pkg','package_name']) : null;
  if ($subCol) {
    $joinPkg = " LEFT JOIN (SELECT user_id, $subCol AS pkg FROM user_subscriptions t WHERE end_date = (SELECT MAX(end_date) FROM user_subscriptions x WHERE x.user_id=t.user_id)) us ON us.user_id = u.id";
    $nameCols = $getExistingCols('packages', ['name_ar','name','title_ar','title','package_name']);
    if ($hasTable('packages') && !empty($nameCols)) {
      $co = 'COALESCE('.implode(',', array_map(function($c){ return 'p.`'.$c.'`'; }, $nameCols)).', us.pkg)';
      $onParts = ['p.id = us.pkg']; foreach ($nameCols as $c) { $onParts[] = 'p.`'.$c.'` = us.pkg'; }
      $joinPkg .= ' LEFT JOIN packages p ON ('.implode(' OR ', $onParts).')';
      $pkgLabel = $co.' AS package_name';
    } else {
      $pkgLabel = 'us.pkg AS package_name';
    }
  } else if ($hasTable('packages') && in_array('package_id',$uCols,true)){
    $nameCols = $getExistingCols('packages', ['name_ar','name','title_ar','title','package_name']);
    if (!empty($nameCols)){
      $co = 'COALESCE('.implode(',', array_map(function($c){ return 'p.`'.$c.'`'; }, $nameCols)).')';
      $joinPkg = ' LEFT JOIN packages p ON p.id = u.package_id';
      $pkgLabel = $co.' AS package_name';
    }
  }
  $join .= $joinPkg;

  $sel = [
    'u.id','u.first_name','u.last_name','u.email',
    in_array('phone',$uCols,true)?'u.phone':'NULL AS phone',
    in_array('brithday',$uCols,true)?'u.brithday':'NULL AS brithday',
    in_array('work',$uCols,true)?'u.work':'NULL AS work',
    in_array('points',$uCols,true)?'u.points':'NULL AS points',
    $pkgLabel,
    $hasSubs ? 's.end_date' : 'NULL AS end_date'
  ];
  $sql = 'SELECT '.implode(',', $sel).' FROM users u'.$join.' WHERE '.$where.' ORDER BY u.id DESC LIMIT 200';
  $stmt = $db->prepare($sql);
  $stmt->execute($params);
  while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
    fputcsv($out, [
      $row['id'],$row['first_name'],$row['last_name'],$row['email'],
      $row['phone'],$row['brithday'],$row['work'],$row['points'],
      $row['package_name'],$row['end_date']
    ]);
  }
  fclose($out); exit;
}

?>
