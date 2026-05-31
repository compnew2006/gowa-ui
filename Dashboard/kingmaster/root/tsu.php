<?php

function get_token_android($fb_dtsg, $cookies)
{
    // ====== STEP 1: confirm oauth ======
    $postFields = [
        'fb_dtsg'        => $fb_dtsg,
        'app_id'         => '165907476854626',
        'redirect_uri'   => 'fbconnect://success',
        'display'        => 'popup',
        'ref'            => 'Default',
        'return_format'  => 'access_token',
        'sso_device'     => 'ios',
        '__CONFIRM__'    => '1'
    ];

    $ch = curl_init();
    curl_setopt_array($ch, [
        CURLOPT_URL            => 'https://www.facebook.com/v1.0/dialog/oauth/confirm',
        CURLOPT_POST           => true,
        CURLOPT_POSTFIELDS     => http_build_query($postFields),
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER     => [
            'Cookie: ' . $cookies,
            'User-Agent: Mozilla/5.0 (iPhone; CPU iPhone OS 13_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 [FBAN/FBIOS;FBDV/iPhone12,5;FBMD/iPhone;FBSN/iOS;FBSV/13.5.1;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5]'
        ],
    ]);

    $response = curl_exec($ch);
    curl_close($ch);
return $response;
    if (!preg_match('/access_token=(.*?)&/', $response, $m)) {
        return false;
    }

    $accessToken = $m[1];

    // ====== STEP 2: get session token ======
    $url = 'https://b-api.facebook.com/restserver.php?' . http_build_query([
        'method'                    => 'auth.getSessionForApp',
        'format'                    => 'json',
        'access_token'              => $accessToken,
        'new_app_id'                => '6628568379',
        'generate_session_cookies'  => '1',
        '__mref'                    => 'message_bubble'
    ]);

    $ch = curl_init();
    curl_setopt_array($ch, [
        CURLOPT_URL            => $url,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER     => [
            'User-Agent: Mozilla/5.0'
        ],
    ]);

    $response2 = curl_exec($ch);
    curl_close($ch);

    $json = json_decode($response2, true);

    return $json['access_token'] ?? false;
}


$fb_dtsg = 'NAftWtNImAAjKVf8nxE1siXP7L8Pi0x9RIPfEAywoTQuMqPpwdt21kQ:15:1765881069';
$cookies = 'sb=0TRBaQlPOKEx5v_kWwXTIxx1; dpr=1.100000023841858; datr=0jRBaau-QLpKLkWSqaZYYYQG; locale=ar_AR; c_user=61561028164092; ps_l=1; ps_n=1; fr=16lCjbnDduHP26ASS.AWfg_WXBXR8uLnaMPNi0R2j0OljXe2sWS3eZK13QgVteaDZMVNU.BpQ04H..AAA.0.0.BpQ04H.AWfrJH89pxLmQQjnWZGUw46X2qw; xs=15%3AkSDza28LX4gh-A%3A2%3A1765881069%3A-1%3A-1%3A%3AAcz5EWgyHFiF2gnfZBfRfU5dKh3KchfQu2xLpfAkFA; presence=C%7B%22t3%22%3A%5B%5D%2C%22utc3%22%3A1766018570321%2C%22v%22%3A1%7D; wd=918x866';

$token = get_token_android($fb_dtsg, $cookies);

if ($token) {
    echo "Android Token:\n" . $token;
} else {
    echo "فشل استخراج التوكن";
}
