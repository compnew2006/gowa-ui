<?php
// اسم الملف اللي فيه البروكسيات
$jsonFile = 'proxies.json';

// اقرأ محتوى الملف
$data = file_get_contents($jsonFile);

// حوله لمصفوفة PHP
$proxiesData = json_decode($data, true);

// الدالة لاختبار البروكسي
function testProxy($proxyUrl) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, "https://www.facebook.com/"); // موقع لاختبار البروكسي
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_PROXY, $proxyUrl);
    curl_setopt($ch, CURLOPT_TIMEOUT, 60);
    curl_setopt($ch, CURLOPT_FAILONERROR, true);

    $res = curl_exec($ch);
    $err = curl_error($ch);
    curl_close($ch);

    if ($res) {
        return ["status" => "OK", "proxy" => $proxyUrl];
    } else {
        return ["status" => "BAD", "proxy" => $proxyUrl, "error" => $err];
    }
}

// اختبر كل البروكسيات
foreach ($proxiesData['proxies'] as $p) {
    $proxyUrl = $p['ip'] . ":" . $p['port'];
    $result = testProxy($proxyUrl);
    echo json_encode($result) . "\n";
}
?>
