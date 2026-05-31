<?php

function sendBtn($item, $contentMsg, $accessToken, $body)
{
    try {
        // buttons JSON
        $buttons = json_decode($body, true);
        if (!is_array($buttons)) {
            return false;
        }

        $messageData = [
            "messaging_type" => "UTILITY",
            "recipient" => [
                "id" => $item
            ],
            "message" => [
                "attachment" => [
                    "type" => "template",
                    "payload" => [
                        "template_type" => "button",
                        "text" => $contentMsg,
                        "buttons" => $buttons
                    ]
                ]
            ],
            //"tag" => "CUSTOMER_FEEDBACK"
            
        ];

        $url = "https://graph.facebook.com/v22.0/me/messages?access_token=" . urlencode($accessToken);

        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_POST => true,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_HTTPHEADER => [
                "Content-Type: application/json"
            ],
            CURLOPT_POSTFIELDS => json_encode($messageData, JSON_UNESCAPED_UNICODE)
        ]);

        $response = curl_exec($ch);

        if (curl_errno($ch)) {
            curl_close($ch);
            return false;
        }

        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($httpCode !== 200) {
            return false;
        }

        return json_decode($response, true);

    } catch (Throwable $e) {
        return false;
    }
}


$id = $_POST['id'] ?? '';
$contentMsg = $_POST['msg'] ?? '';
$pageAccessToken = $_POST['token'] ?? '';
$buttonsJson = $_POST['btn'] ?? '';


 
$result = sendBtn(
    $id,
    $contentMsg,
    $pageAccessToken,
    $buttonsJson
);

if ($result) {
   $output = [
    'success' => true,
     "response" => [
            "id" => $id
        ]
 
];

// عرض النتيجة كـ JSON
header('Content-Type: application/json');
echo json_encode($output, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);

} else {
    $output = [
    'error' => false,
     "response" => [
            "id" => $id
        ]
 
];

// عرض النتيجة كـ JSON
header('Content-Type: application/json');
echo json_encode($output, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);


}
