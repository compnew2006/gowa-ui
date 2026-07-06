<?php
/**
 * King Master - Get Packages for Landing Page
 * جلب الباقات لصفحة اللاندينغ
 */

// إعداد الرؤوس
header('Content-Type: application/json; charset=UTF-8');
header('Access-Control-Allow-Origin: *');

// تضمين ملف الاتصال بقاعدة البيانات
require_once 'config/database.php';

try {
    // جلب جميع الباقات
    $query = "SELECT * FROM packages ORDER BY id ASC";
    $packages = fetchAll($query);
    
    if ($packages && count($packages) > 0) {
        // تحويل البيانات للصيغة المطلوبة
        foreach ($packages as &$package) {
            // تحويل المميزات من JSON إلى مصفوفة
            $featuresJson = $package['features'] ?? '[]';
            $featuresArray = json_decode($featuresJson, true);
            if (!is_array($featuresArray)) {
                $featuresArray = [];
            }
            
            $package['features'] = [
                'ar' => $featuresArray,
                'en' => $featuresArray,
                'fr' => $featuresArray
            ];
            
            // تحويل المنصات من JSON إلى مصفوفة
            $platformsJson = $package['platforms'] ?? '[]';
            $platformsArray = json_decode($platformsJson, true);
            if (!is_array($platformsArray)) {
                $platformsArray = [];
            }
            $package['supported_platforms'] = $platformsArray;
            
            // تحويل القيم الرقمية
            $package['monthly_price'] = floatval($package['price'] ?? 0);
            $package['yearly_price'] = floatval($package['price'] ?? 0) * 10; // سعر افتراضي للسنة
            $package['is_popular'] = (bool)($package['is_popular'] ?? 0);
            $package['points'] = intval($package['points'] ?? 0);
            $package['messages'] = intval($package['messages_count'] ?? 0);
            $package['accounts'] = intval($package['accounts_count'] ?? 1);
            
            // حساب الخصم
            $package['has_discount'] = (bool)($package['has_discount'] ?? 0);
            $originalPrice = floatval($package['original_price'] ?? 0);
            $currentPrice = floatval($package['price'] ?? 0);
            
            if ($package['has_discount'] && $originalPrice > $currentPrice) {
                $package['monthly_discount'] = $originalPrice - $currentPrice;
                $package['monthly_discount_percentage'] = (($originalPrice - $currentPrice) / $originalPrice) * 100;
            } else {
                $package['monthly_discount'] = 0;
                $package['monthly_discount_percentage'] = 0;
            }
            
            // إضافة أسماء باللغات
            $package['name_ar'] = $package['name'];
            $package['name_en'] = $package['name'];
            $package['description_ar'] = $package['description'] ?? '';
            $package['description_en'] = $package['description'] ?? '';
        }
        
        // إرسال الاستجابة الناجحة
        echo json_encode([
            'success' => true,
            'message' => 'تم جلب الباقات بنجاح',
            'data' => $packages,
            'count' => count($packages),
            'timestamp' => date('Y-m-d H:i:s')
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
        
    } else {
        // لا توجد باقات
        echo json_encode([
            'success' => false,
            'message' => 'لا توجد باقات متاحة حالياً',
            'data' => [],
            'count' => 0,
            'timestamp' => date('Y-m-d H:i:s')
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    }
    
} catch (Exception $e) {
    // خطأ في قاعدة البيانات
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب البيانات: ' . $e->getMessage(),
        'data' => [],
        'count' => 0,
        'timestamp' => date('Y-m-d H:i:s')
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
}
?>