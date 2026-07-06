<?php
$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$next = $_GET['next'] ?? null;

function fetchFacebookGroupMembers($accessToken, $id_post, $end_cursor = null)
{


    $url = "https://graph.facebook.com/graphql";


    $variables = [
    "inviteLinkKey" => null,
    "paginationPK" => $id_post,
    "group_other_member_profiles_paginating_node_after_cursor" => $end_cursor,
    "group_id" => $id_post,
    "group_other_member_profiles_paginating_node_first" => 200,
    "group_other_member_profiles_paginating_node_at_stream_use_customized_batch" => false,
    "include_workplace_fields" => false,
    "should_show_contribution_score" => false,
    "scale" => 3
    ];
    
 
    // نفس بيانات POST
    $postData = http_build_query([
        "method" => "post",
        "pretty" => "false",
        "format" => "json",
        "server_timestamps" => "true",
        "locale" => "user",
        "purpose" => "fetch",
        "fb_api_req_friendly_name" => "FBGroupAdminMembersListSeeAllViewControllerFactoryOthersQuery_At_Connection_Pagination_Group_group_other_member_profiles_paginating_node",
        "fb_api_caller_class" => "AtConnection",
"client_doc_id" => "168740769410657821597851880828",
        "fb_api_client_context" => json_encode([
            "client_connection_size" => 20
        ]),
        "variables" => json_encode($variables),
        "fb_api_analytics_tags" => json_encode(["At_Connection","pagination_framework:@connection", "GraphServices"]),
        "client_trace_id" => "1340748499"
    ]);

    // نفس الهيدرات بالضبط
    $headers = [
        "Host: graph.facebook.com",
        'X-Fb-Request-Analytics-Tags: {"network_tags":{"product":"6628568379","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"AtConnection"}',
        "X-Fb-Ta-Logging-Ids: graphql:147e67a3-6e01-4616-b159-f17068993ba7",
        "X-Fb-Rmd: state=URL_ELIGIBLE",
        "X-Fb-Sim-Hni: 6553565535",
        "X-Fb-Net-Hni: 60201",
        "Authorization: OAuth $accessToken",
        "X-Graphql-Request-Purpose: fetch",
        "User-Agent: Mozilla/5.0 (iPhone; CPU iPhone OS 18_6_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/22G100 [FBAN/FBIOS;FBAV/501.0.0.49.107;FBBV/699723644;FBDV/iPhone13,4;FBMD/iPhone;FBSN/iOS;FBSV/18.6.2;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]",
        "Content-Type: application/x-www-form-urlencoded",
        "X-Fb-Connection-Type: wifi.CTRadioAccessTechnologyLTE",
        "X-Fb-Background-State: 1",
        "X-Fb-Friendly-Name: FBGroupAdminMembersListSeeAllViewControllerFactoryOthersQuery_At_Connection_Pagination_Group_group_other_member_profiles_paginating_node",
        "X-Graphql-Client-Library: graphservice",
        "X-Fb-Privacy-Context: 1392647684458756",
        "X-Fb-Product-Log: graphql:1340748499",
        "X-Fb-Device-Group: 5449",
        "X-Tigon-Is-Retry: False",
        "Priority: u=3,i",
        "X-Fb-Http-Engine: Tigon/Liger",
        "X-Fb-Client-Ip: True",
        "X-Fb-Server-Cluster: True"
    ];

       
   
 


    // تنفيذ الطلب
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, $postData);
    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

    $response = curl_exec($ch);
    if (curl_errno($ch)) {
        throw new Exception("cURL Error: " . curl_error($ch));
    }

    curl_close($ch);

    return json_decode($response, true);
}



 

function extractGroupMembersAndCursor($jsonResponse, $token, $id)
{
    // edges
    $edges = $jsonResponse['data']['fetch__Group']['group_member_profiles']['edges'] ?? [];

    // end_cursor
    $endCursor = $jsonResponse['data']['fetch__Group']['group_member_profiles']['page_info']['end_cursor'] ?? null;


    // members array
    $members = [];

    foreach ($edges as $edge) {
        $node = $edge['node'] ?? [];

        $members[] = [
            "id" => $node['id'] ?? null,
            "name" => $node['name'] ?? '',
            "profilePicture" => $node['profile_picture']['uri'] ?? '',
            "bio" => $node['bio_text']['text'] ?? ''  // استخراج bio
        ];
    }


 return [
        "members" => $members,
        "paging" => [
            "next" => $endCursor
                ? "https://rbmcloud.com/api_get_mmbers.php?token={$token}&id={$id}&next={$endCursor}"
                : null,
            "next5" => $endCursor

        ]
    ];
    

 
}

 
// مثال تشغيل
  $result = fetchFacebookGroupMembers($token, $id, $next);
 $result= extractGroupMembersAndCursor($result, $token, $id);
 
header('Content-Type: application/json; charset=utf-8');
echo json_encode($result, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
