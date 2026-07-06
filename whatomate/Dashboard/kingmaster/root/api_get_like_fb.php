<?php
$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$next = $_GET['next'] ?? null;
$uuid = 'feedback:'.$id;
$base64 = base64_encode($uuid);

function fetchFeedbackReactors($OAuth, $feedbackId, $afterCursor)
{
    $url = "https://graph.facebook.com/graphql";

    $headers = [
        "Content-Type: application/x-www-form-urlencoded",
        "User-Agent: [FBAN/FB4A;FBAV/417.0.0.33.65;FBBV/480086274;FBDM/{density=1.5,width=720,height=1280};FBLC/en_US;FBRV/483172840;FBCR/EMS - Mobinil;FBMF/google;FBBD/google;FBPN/com.facebook.katana;FBDV/G011A;FBSV/9;FBOP/1;FBCA/x86:armeabi-v7a;]",
        "X-Fb-Request-Analytics-Tags: {\"network_tags\":{\"product\":\"350685531728\",\"purpose\":\"fetch\",\"request_category\":\"graphql\",\"retry_attempt\":\"0\"},\"application_tags\":\"ConnectionManager\"}",
        "X-Fb-Ta-Logging-Ids: graphql:d07725f2-e5ee-4ddd-b72a-3d6c4645a05c",
        "X-Fb-Rmd: state=URL_ELIGIBLE",
        "X-Fb-Device-Group: 5449",
        "X-Fb-Session-Id: nid=DsiCFVtIcd2f;tid=2076;nc=0;fc=1;bc=1;cid=d54a1db57c461c6fe363a92364788772",
        "X-Fb-Product-Log: graphql:d07725f2-e5ee-4ddd-b72a-3d6c4645a05c",
        "X-Fb-Privacy-Context: 902684366915547",
        "X-Graphql-Client-Library: graphservice",
        "X-Fb-Qpl-Active-Flows-Json: {\"schema_version\":\"v2\",\"inprogress_qpls\":[{\"marker_id\":25952257,\"annotations\":{\"current_endpoint\":\"PermalinkReactorsListFragment:permalink_reactors_list\"}}],\"snapshot_attributes\":{}}",
        "X-Fb-Friendly-Name: FeedbackReactorsGraphService_At_Connection_Pagination_Feedback_reactors_connection",
        "X-Fb-Background-State: 1",
        "X-Fb-Connection-Type: WIFI",
        "X-Graphql-Request-Purpose: fetch",
        "Authorization: OAuth $OAuth",
        "X-Fb-Net-Hni: 60201",
        "X-Fb-Sim-Hni: 60201",
        "X-Tigon-Is-Retry: False",
        "Accept-Encoding: gzip, deflate, br",
        "X-Fb-Http-Engine: Liger",
        "X-Fb-Client-Ip: True",
        "X-Fb-Server-Cluster: True",
        "X-Fb-Connection-Token: d54a1db57c461c6fe363a92364788772"
    ];

    $payload = [
        "method" => "post",
        "pretty" => "false",
        "format" => "json",
        "server_timestamps" => "true",
        "locale" => "user",
        "purpose" => "fetch",
        "fb_api_req_friendly_name" => "FeedbackReactorsGraphService_At_Connection_Pagination_Feedback_reactors_connection",
        "fb_api_caller_class" => "ConnectionManager",
        "client_doc_id" => "14962755647493038689879967568",
        "fb_api_client_context" => json_encode([
            "load_next_page_counter" => 3,
            "client_connection_size" => 75
        ]),
        "variables" => json_encode([
            "reactors_connection_first" => 25,
            "feedback_id" => $feedbackId,
            "paginationPK" => $feedbackId,
            "reactors_connection_after_cursor" => $afterCursor
        ]),
        "fb_api_analytics_tags" => json_encode(["At_Connection", "GraphServices"]),
        "client_trace_id" => "d07725f2-e5ee-4ddd-b72a-3d6c4645a05c"
    ];

    $postData = http_build_query($payload);

    $ch = curl_init($url);

    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $postData,
        CURLOPT_HTTPHEADER => $headers,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_ENCODING => "" // لقراءة gzip
    ]);

    $result = curl_exec($ch);

    if (curl_errno($ch)) {
        curl_close($ch);
        return null;
    }

    curl_close($ch);
    return json_decode($result, true);
}

 function extractfeedsAndCursor($jsonResponse, $token, $id)
{
    $edges = $jsonResponse['data']['node']['reactors']['edges'] ?? [];
    $endCursor = $jsonResponse['data']['node']['reactors']['page_info']['end_cursor'] ?? null;

    $feeds = [];

    foreach ($edges as $edge) {
        $node = $edge['node'] ?? [];
        $feeds[] = [
            "id" => $node['id'] ?? null,
            "name" => $node['name'] ?? ''
        ];
    }


 return [
     "test" => $jsonResponse,
        "user" => $feeds,
        "paging" => [
            "next" => $endCursor
                ? "https://kingmaster.info/api_get_like_fb.php?token={$token}&id={$id}&next={$endCursor}"
                : null,
                "next2" =>  $endCursor
        ]
    ];
    
    
    
}

// مثال تشغيل
  $result = fetchFeedbackReactors($token, $base64, $next);
 $result= extractfeedsAndCursor($result, $token, $id);
 
header('Content-Type: application/json; charset=utf-8');
echo json_encode($result, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
