<?php

function getFacebookConversations($accessToken, $after = '') {
    // رابط الـ API مع الباراميتر after لو فيه
    $url = "https://graph.facebook.com/v2.9/me/conversations?fields=can_reply,senders&limit=300&access_token={$accessToken}";
    if (!empty($after)) {
        $url .= "&after=" . urlencode($after);
    }

    // إعداد curl
    $ch = curl_init($url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_HTTPGET, true);

    // تنفيذ الطلب
    $response = curl_exec($ch);

    // التحقق من الأخطاء
    if (curl_errno($ch)) {
        echo 'Curl error: ' . curl_error($ch);
        curl_close($ch);
        return null;
    }

    curl_close($ch);

    // تحويل JSON إلى مصفوفة PHP
    $data = json_decode($response, true);

    return $response;
}

    $token = $_GET['token'] ?? '';

$next = $_GET['next'] ?? null;
 
 
 
 
// مثال على الاستخدام
$accessToken =  $token;
$afterCursor = $next; // لو عايز تبدأ من cursor معين

$conversations = getFacebookConversations($accessToken, $afterCursor);

// طباعة النتائج


$data = json_decode($conversations, true);

// مصفوفة لتخزين أول مرسل لكل محادثة
$results = [];

// استعراض الـ conversations
foreach ($data['data'] as $conversation) {
    if (!empty($conversation['senders']['data'][0])) {
        $firstSender = $conversation['senders']['data'][0];
        $results[] = [
            'name' => $firstSender['name'],
            'id' => $firstSender['id']
        ];
    }
}

// قيمة after
$afterCursor = $data['paging']['cursors']['after'] ?? null;

// عرض النتائج
$output = [
    'first_senders' => $results,
     "paging" => [
            "next" => $afterCursor
        ]
 
];

// عرض النتيجة كـ JSON
header('Content-Type: application/json');
echo json_encode($output, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);


?>
