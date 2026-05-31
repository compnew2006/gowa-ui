<?php

$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$next = $_GET['next'] ?? null;
function getFacebookComments($postId, $accessToken, $limit = 5, $after = '')
{
    $url = "https://graph.facebook.com/v2.2/{$postId}/comments";

    $params = [
        'access_token' => $accessToken,
        'fields'       => 'from,message',
        'limit'        => $limit
    ];

    // pagination
    if (!empty($after)) {
        $params['after'] = $after;
    }

    $url .= '?' . http_build_query($params);

    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_USERAGENT      => 'Mozilla/5.0'
    ]);

    $response = curl_exec($ch);

    if (curl_errno($ch)) {
        throw new Exception(curl_error($ch));
    }

    curl_close($ch);

    return json_decode($response, true);
}



function extractCommentsAndCursor($jsonResponse)
{
    $comments = [];

    // الكومنتات
    $data = $jsonResponse['data'] ?? [];

    // after cursor
    $afterCursor = $jsonResponse['paging']['cursors']['after'] ?? null;

    foreach ($data as $item) {
        $from = $item['from'] ?? [];

        $comments[] = [
            "name"        => $from['name'] ?? null,
            "id"          => $from['id'] ?? null,
            "message"     => $item['message'] ?? null,
            "id_comments" => $item['id'] ?? null
        ];
    }

    return [
        "data" => $comments,
        "paging" => [
            "next" => $afterCursor
        ]
    ];
}

try {
 

$response = getFacebookComments($id, $token, 50, $next);
// $response = json_decode($json, true); لو جاية JSON

$result = extractCommentsAndCursor($response);

header('Content-Type: application/json; charset=utf-8');
echo json_encode($result, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);

} catch (Exception $e) {
    echo '❌ Error: ' . $e->getMessage();
}
