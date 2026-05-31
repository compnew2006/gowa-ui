<?php
$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$next = $_GET['next'] ?? null;
 
 
 
 
 

// 1️⃣ دالة لجلب IDs من البحث
function getPageIds($accessToken, $query, $after) {
    
    $url = "https://graph.facebook.com/v15.0/pages/search?q=" . urlencode($query) . "&fields=id&limit=40&access_token={$accessToken}";
    if ($after) {
        $url .= "&after={$after}";
    }

    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    $response = curl_exec($ch);
    curl_close($ch);

    $data = json_decode($response, true);
    $ids = [];
    if (isset($data['data'])) {
        foreach ($data['data'] as $item) {
            $ids[] = $item['id'];
        }
    }

    $paging = $data['paging']['cursors']['after'] ?? null;

    return ['ids' => $ids, 'after' => $paging];
}

// 2️⃣ دالة لإرسال Batch Request على جميع IDs
function getBatchPageDetails($ids, $accessToken) {
    
    $fields = "id,name,category,followers_count,location,phone,website,whatsapp_number";

    $batch = [];
    foreach ($ids as $id) {
        $batch[] = [
            "method" => "GET",
            "relative_url" => "{$id}?fields={$fields}"
        ];
    }

    $batchJson = json_encode($batch);
    $postData = http_build_query([
        "access_token" => $accessToken,
        "batch" => $batchJson
    ]);

    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, "https://graph.facebook.com/v2.9");
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, $postData);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_HTTPHEADER, ["Content-Type: application/x-www-form-urlencoded"]);

    $responseText = curl_exec($ch);
    if(curl_errno($ch)) {
        $responseText = "Error: " . curl_error($ch);
    }
    curl_close($ch);

    return $responseText;
}

// 3️⃣ مثال على الاستخدام
$searchResult = getPageIds($token,$id, $next);
$pageIds = $searchResult['ids'];
$afterCursor = $searchResult['after'];
 
// جلب تفاصيل الصفحات بالباتش
$batchResponse = getBatchPageDetails($pageIds, $token);
$pagesData = json_decode($batchResponse, true);

// معالجة الردود لكل صفحة
$pages = [];
foreach ($pagesData as $item) {
    if (isset($item['body'])) {
        $data = json_decode($item['body'], true);
        $pages[] = $data;
    }
}
 
$result = [];

foreach ($pages as $page) {
    $item = [
        'id' => $page['id'] ?? null,
        'name' => $page['name'] ?? null,
        'category' => $page['category'] ?? null,
        'followers_count' => $page['followers_count'] ?? null,
        'website' => $page['website'] ?? null,
        'location' => [
            'city' => $page['location']['city'] ?? null,
            'country' => $page['location']['country'] ?? null,
            'state' => $page['location']['state'] ?? null,
            'street' => $page['location']['street'] ?? null,
            'zip' => $page['location']['zip'] ?? null,
            'latitude' => $page['location']['latitude'] ?? null,
            'longitude' => $page['location']['longitude'] ?? null
        ]
    ];
    $result[] = $item;
}

// لو فيه paging بعد الصفحات
 
// تحويل النتيجة لجيسون
header('Content-Type: application/json; charset=utf-8');
echo json_encode([
    'data' => $result,
      "paging" => [
            "next" => $afterCursor
        ]
], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
 

// لو عايز تكمل للصفحات التالية:
// while ($afterCursor) {
//     $searchResult = getPageIds("car", 5, $afterCursor);
//     $pageIds = $searchResult['ids'];
//     $afterCursor = $searchResult['after'];
//     $batchResponse = getBatchPageDetails($pageIds);
//     // معالجة batchResponse بنفس الطريقة
// }
?>

 
