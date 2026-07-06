<?php
$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$next = $_GET['next'] ?? null;
 
$one2 = searchFacebookGroups_last($next, $id, $token);
$afterCursor = getEndCursorFromResponses2($one2);

$one = extractGroupData2($one2);
header('Content-Type: application/json; charset=utf-8');
echo json_encode([
    'data' => $one,
      "paging" => [
            "next" => $afterCursor
        ]
], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
 
 
function searchFacebookGroups_last($cursor, $text, $access_token)
{
    // variables زي ما هي بس PHP Array
    $variables = [
        "allow_streaming" => false,
        "args" => [
            "callsite" => "COMET_GLOBAL_SEARCH",
            "config" => [
                "exact_match" => false,
                "high_confidence_config" => null,
                "intercept_config" => null,
                "sts_disambiguation" => null,
                "watch_config" => null
            ],
            "context" => [
                "bsid" => "f8e193ac-ece1-4802-bd30-a7bf047b91c0",
                "tsid" => "0.9537790025885376"
            ],
            "experience" => [
                "client_defined_experiences" => ["ADS_PARALLEL_FETCH"],
                "encoded_server_defined_params" => null,
                "fbid" => null,
                "type" => "GROUPS_TAB"
            ],
            "filters" => ['{"name":"public_groups","args":""}'],
            "text" => $text
        ],
        "count" => 5,
        "cursor" => $cursor,
        "feedLocation" => "SEARCH",
        "feedbackSource" => 23,
        "fetch_filters" => true,
        "focusCommentID" => null,
        "locale" => null,
        "privacySelectorRenderLocation" => "COMET_STREAM",
        "renderLocation" => "search_results_page",
        "scale" => 1,
        "stream_initial_count" => 0,
        "useDefaultActor" => false
    ];

    $url = "https://graph.facebook.com/graphql?doc_id=24114354174911239&server_timestamps=true";

    // POST Body
    $postData = http_build_query([
        "variables" => json_encode($variables),
        "fb_api_req_friendly_name" => "SearchCometResultsPaginatedResultsQuery",
        "method" => "post",
        "fb_api_caller_class" => "RelayModern",
        "access_token" => $access_token
    ]);

    // Init CURL
    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $postData,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => [
            "Content-Type: application/x-www-form-urlencoded"
        ]
    ]);

    $response = curl_exec($ch);
    $err = curl_error($ch);
    curl_close($ch);

    if ($err) {
        echo "❌ CURL Error: $err";
        return null;
    }

    $json = json_decode($response, true);

 
    return $json;
}



function extractGroupData2($response)
{
    $groups = [];

    $edges = $response['data']['serpResponse']['results']['edges'] ?? [];

    if (empty($edges)) {
        
  
        return $groups;
    }

    foreach ($edges as $edge) {
        $node = $edge['rendering_strategy'] ?? null;

        if (!$node) continue;

        $profile = $node['view_model']['profile'] ?? null;
        $profilex = $node['view_model']['primary_snippet_text_with_entities'] ?? null;

        if ($profile) {
            $groups[] = [
                "id"  => $profile['id'] ?? null,
                "name" => $profile['name'] ?? null,
                "txt" => $profilex['text'] ?? null
            ];
        }
    }

 
    return $groups;
}


function getEndCursorFromResponses2($result)
{
    try {
        return $result['data']['serpResponse']['results']['page_info']['end_cursor'] ?? null;
    } catch (Exception $e) {
        echo "❌ Error in getEndCursorFromResponses: " . $e->getMessage();
        return null;
    }
}
