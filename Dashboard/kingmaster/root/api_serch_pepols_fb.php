<?php

function searchFacebookGraphQLPepelo($accessToken, $endCursor = '', $keyword = 'car') {
    $url = 'https://graph.facebook.com/graphql';

    // إعدادات الهيدر
    $headers = [
        'Host: graph.facebook.com',
        'X-Fb-Request-Analytics-Tags: ' . json_encode([
            'network_tags' => [
                'product' => '350685531728',
                'purpose' => 'none',
                'request_category' => 'graphql',
                'retry_attempt' => '0'
            ],
            'application_tags' => 'graphservice'
        ]),
        'X-Fb-Ta-Logging-Ids: graphql:d64eaa1b-463c-4be0-b965-057f21ba0c44',
        'X-Fb-Rmd: state=URL_ELIGIBLE',
        'X-Fb-Sim-Hni: 60201',
        'X-Fb-Net-Hni: 60201',
        'User-Agent: FBAN/FB4A;FBAV/417.0.0.33.65;FBBV/480086274;FBDM/{density=1.5,width=720,height=1280};FBLC/en_US;FBRV/483172840;FBCR/EMS - Mobinil;FBMF/google;FBBD/google;FBPN/com.facebook.katana;FBDV/G011A;FBSV/9;FBOP/1;FBCA/x86:armeabi-v7a;',
        'Authorization: OAuth ' . $accessToken,
        'X-Fb-Background-State: 1',
        'X-Fb-Friendly-Name: SearchResultsGraphQL-pagination_query',
        'X-Graphql-Client-Library: graphservice',
        'Content-Type: application/x-www-form-urlencoded',
        'X-Fb-Connection-Type: WIFI',
        'X-Fb-Device-Group: 5449',
        'X-Tigon-Is-Retry: False',
        'Priority: u=3,i',
        'X-Fb-Http-Engine: Liger',
        'X-Fb-Client-Ip: True',
        'X-Fb-Server-Cluster: True'
    ];

    // إعدادات الـ POST data
    $postData = http_build_query([
        'method' => 'post',
        'pretty' => 'false',
        'format' => 'json',
        'server_timestamps' => 'true',
        'locale' => 'en_US',
        'fb_api_req_friendly_name' => 'SearchResultsGraphQL-pagination_query',
        'fb_api_caller_class' => 'graphservice',
        'client_doc_id' => '395907910716207519270483411726',
        'variables' => json_encode([
            'end_cursor' => $endCursor,
            'bsid' => '2936913f-868c-40b2-89a3-dcbae69e9c87',
            'supported_experiences' => [
                'FAST_FILTERS', 'FILTERS', 'FILTERS_AS_SEE_MORE', 'INSTANT_FILTERS',
                'MARKETPLACE_ON_GLOBAL', 'MIXED_MEDIA', 'NATIVE_TEMPLATES',
                'NT_ENABLED_FOR_TAB', 'NT_SPLIT_VIEWS', 'PEOPLE_RADIUS_FILTER',
                'PHOTO_STREAM_VIEWER', 'SEARCH_INTERCEPT', 'SEARCH_SNIPPETS_ICONS_ENABLED',
                'USAGE_COLOR_SERP', 'commerce_groups_search', 'keyword_only'
            ],
            'bqf' => "keywords_users({$keyword})",
            'image_large_aspect_width' => 720,
            'ui_theme_name' => 'APOLLO_FULL_BLEED',
            'query_source' => 'graph_search_v2_results_page_see_more',
            'extra_data' => json_encode(['request_params' => []]),
            'inline_comments_location' => 'search'
        ]),
        'fb_api_analytics_tags' => json_encode(['pagination_query', 'GraphServices']),
        'client_trace_id' => 'd64eaa1b-463c-4be0-b965-057f21ba0c44'
    ]);

    // تهيئة cURL
    $ch = curl_init($url);
    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, $postData);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

    $response = curl_exec($ch);
    $error = curl_error($ch);
    curl_close($ch);

    if ($error) {
        throw new Exception("❌ GraphQL Request Failed: " . $error);
    }

    return json_decode($response, true); // يرجع array
}



function extractGroupData2($response)
{
    $groups = [];

    $edges = $response['data']['search_query']['combined_results']['edges'] ?? [];

    if (empty($edges)) {
        
  
        return $groups;
    }

    foreach ($edges as $edge) {
        $node = $edge['node'] ?? null;

        if (!$node) continue;

        $profile = $node;
       
        if ($profile) {
            $groups[] = [
                "id"  => $profile['id'] ?? null,
                "name" => $profile['name'] ?? null
            ];
        }
    }

 
    return $groups;
}


function getEndCursorFromResponses2($result)
{
    try {
        return $result['data']['search_query']['combined_results']['page_info']['end_cursor'] ?? null;
    } catch (Exception $e) {
        echo "❌ Error in getEndCursorFromResponses: " . $e->getMessage();
        return null;
    }
}



// مثال للاستخدام
try {
    
    $token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$next = $_GET['next'] ?? null;
 
 
 
    $data = searchFacebookGraphQLPepelo($token, $next, $id);
 
    $one = extractGroupData2($data);
    $afterCursor = getEndCursorFromResponses2($data);

    header('Content-Type: application/json; charset=utf-8');
    echo json_encode([
        'data' => $one,
        "paging" => [
            "next" => $afterCursor
        ]
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);

} catch (Exception $e) {
    echo $e->getMessage();
}

