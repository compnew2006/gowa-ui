<?php
function detect_country_with_networks(string $rawNumber): array {
    $orig = $rawNumber;
    $num = preg_replace('/[^\d]/', '', $rawNumber);

    // قائمة الدول + بادئات الشبكات الرسمية لكل دولة
    $countries = [
        'مصر' => [
            'code' => '+20',
            'prefixes' => ['10','11','12','15'],
            'length' => 10
        ],
        'السعودية' => [
            'code' => '+966',
            'prefixes' => [
                '50','51','52','53','54','55','56','57','58','59'
            ],
            'length' => 9
        ],
        'الأردن' => [
            'code' => '+962',
            'prefixes' => ['77','78','79','70'],
            'length' => 9
        ],
        'سوريا' => [
            'code' => '+963',
            'prefixes' => ['93','94','95','96','97','98','99'],
            'length' => 9
        ],
        'العراق' => [
            'code' => '+964',
            'prefixes' => ['70','71','72','73','74','75','76','77','78','79'],
            'length' => 10
        ],
        'المغرب' => [
            'code' => '+212',
            'prefixes' => ['6','7'],
            'length' => 9
        ],
        'فلسطين' => [
            'code' => '+970',
            'prefixes' => ['52','56','57','58','59'],
            'length' => 9
        ],
        'قطر' => [
            'code' => '+974',
            'prefixes' => ['3','5','7','8','9'],
            'length' => 8
        ],
        'الامارات' => [
            'code' => '+971',
            'prefixes' => ['50','52','54','55','56'],
            'length' => 9
        ],
        'ليبيا' => [
            'code' => '+218',
            'prefixes' => ['91','92','93','94','95','96','97','98','99'],
            'length' => 9
        ]
    ];

    $matches = [];
    foreach ($countries as $country => $info) {
        $length_ok = (strlen($num) == $info['length']) || (strlen($num) == $info['length']+1 && $num[0]=='0');
        if (!$length_ok) continue;

        $local = $num;
        if ($num[0] == '0') $local = substr($num,1);

        foreach ($info['prefixes'] as $pre) {
            if (strpos($local, $pre) === 0) {
                $matches[] = [
                    'country' => $country,
                    'dial_code' => $info['code'],
                    'prefix' => $pre,
                    'formatted' => $info['code'] . $local
                ];
                break;
            }
        }
    }

    $result = [
        'raw'=>$orig,
        'normalized'=>$num,
        'possible'=>$matches,
        'country'=>null,
        'dial_code'=>null,
        'formatted'=>null
    ];

    if(count($matches) === 1){
        $result['country'] = $matches[0]['country'];
        $result['dial_code'] = $matches[0]['dial_code'];
        $result['formatted'] = $matches[0]['formatted'];
    } elseif(count($matches) > 1){
        // اختيار أول احتمال كافتراضي مع وجود قائمة بالاحتمالات
        $result['country'] = $matches[0]['country'];
        $result['dial_code'] = $matches[0]['dial_code'];
        $result['formatted'] = $matches[0]['formatted'];
    }

    return $result;
}



$num = "541793243";
print_r(detect_country_with_networks($num));