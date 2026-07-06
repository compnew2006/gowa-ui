<?php
// WhatsApp Connect Wizard

function call_api(array $payload, array $query = []) {
    $api = 'https://apis.kingmaster.info/api.php';
    if (!empty($query)) {
        $api .= '?' . http_build_query($query);
    }
    $ch = curl_init($api);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($payload));
    $res = curl_exec($ch);
    $err = curl_error($ch);
    curl_close($ch);
    return [$res, $err];
}

$session = isset($_REQUEST['session']) ? trim((string)$_REQUEST['session']) : '';
$action = $_REQUEST['op'] ?? '';

if ($action === 'check' && $session !== '') {
    [$res, $err] = call_api([], ['action' => 'check_connection', 'session' => $session]);
    header('Content-Type: application/json');
    echo $err ? json_encode(['success'=>false,'error'=>$err]) : ($res ?: json_encode(['success'=>false]));
    exit;
}

$qr = ['qrcode'=>null,'urlcode'=>null];
$msg = null; $errMsg = null; $started = null;

if ($_SERVER['REQUEST_METHOD'] === 'POST' && $session !== '') {
    // Start session with webhook and QR
    [$res1, $err1] = call_api([
        'webhook' => 'https://apis.kingmaster.info/webhook.php',
        'waitQrCode' => true,
    ], ['action' => 'start_session', 'session' => $session]);
    if ($err1) { $errMsg = $err1; }
    else { $started = json_decode($res1, true); if (!is_array($started)) $errMsg = 'Invalid response from start_session'; }

    // Try get_qr right after
    [$res2, $err2] = call_api([], ['action' => 'get_qr', 'session' => $session]);
    if (!$err2 && $res2) {
        $qrArr = json_decode($res2, true);
        if (is_array($qrArr)) { $qr['qrcode'] = $qrArr['qrcode'] ?? null; $qr['urlcode'] = $qrArr['urlcode'] ?? null; }
    }
}
?>
<!doctype html>
<html lang="ar" dir="rtl">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ربط واتساب</title>
  <style>
    body{font-family:Cairo,system-ui,Segoe UI,Roboto,Helvetica,Arial,sans-serif;margin:24px}
    .box{max-width:760px;border:1px solid #ddd;border-radius:10px;padding:16px}
    .row{margin:10px 0}
    input[type=text]{padding:10px;border:1px solid #bbb;border-radius:8px;width:280px}
    button{padding:10px 16px;border:1px solid #999;border-radius:8px;background:#fff;cursor:pointer}
    .qr img{max-width:280px}
    .status{background:#f7f7f7;border-radius:8px;padding:10px}
    .ok{color:green;font-weight:700}
    .err{color:#b00020}
    .hint{color:#666}
  </style>
</head>
<body>
  <div class="box">
    <h2>ربط حساب واتساب</h2>
    <form method="post">
      <div class="row">
        <label>اسم الجلسة</label>
        <input type="text" name="session" value="<?= htmlspecialchars($session ?: 'session_'.substr(md5(uniqid('', true)),0,6)) ?>" required>
        <button type="submit">إنشاء الجلسة</button>
      </div>
      <div class="hint">سيتم إنشاء الجلسة واسترجاع QR للربط ثم التحقق من الاتصال تلقائيًا.</div>
    </form>

    <?php if ($errMsg): ?>
      <div class="row err">خطأ: <?= htmlspecialchars($errMsg) ?></div>
    <?php endif; ?>

    <?php if ($started && $session): ?>
      <div class="row status">
        الجلسة: <b><?= htmlspecialchars($session) ?></b>
        <div id="conn">التحقق من الاتصال...</div>
      </div>

      <?php if ($qr['qrcode']): ?>
        <div class="qr">
          <div>امسح QR من واتساب على الهاتف:</div>
          <img src="data:image/png;base64,<?= htmlspecialchars($qr['qrcode']) ?>" alt="QR" />
        </div>
      <?php elseif ($qr['urlcode']): ?>
        <div class="qr">
          <a href="<?= htmlspecialchars($qr['urlcode']) ?>" target="_blank">فتح QR</a>
        </div>
      <?php else: ?>
        <div class="hint">لم يتم إيجاد QR الآن، أعد تحميل الصفحة بعد ثوانٍ إن لم يظهر.</div>
      <?php endif; ?>

      <script>
        (function(){
          const session = <?= json_encode($session) ?>;
          const connEl = document.getElementById('conn');
          let tries = 0;
          function poll(){
            fetch('whatsapp_connect.php?op=check&session='+encodeURIComponent(session), {method:'POST'})
              .then(r=>r.json()).then(j=>{
                tries++;
                const connected = !!(j && (j.connected === true || (j.data && j.data.connected === true)));
                if (connected){
                  connEl.innerHTML = '<span class="ok">تم الربط بنجاح ✓</span>';
                } else {
                  const msg = (j && (j.error || (j.data && j.data.message))) || '';
                  connEl.textContent = 'غير متصل بعد...' + (msg?(' ('+msg+')'):'');
                  if (tries < 120) setTimeout(poll, 3000);
                }
              }).catch(()=>{ if (tries < 120) setTimeout(poll, 3000); });
          }
          poll();
        })();
      </script>
    <?php endif; ?>
  </div>
</body>
</html>
