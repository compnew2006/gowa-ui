<?php
/**
 * King Master - Packages API
 * API إدارة الباقات
 */

// إعداد الرؤوس
header('Content-Type: application/json; charset=UTF-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization');

// معالجة طلبات OPTIONS (للـ CORS)
if ($_SERVER['REQUEST_METHOD'] == 'OPTIONS') {
    exit(0);
}

// تضمين ملف الاتصال بقاعدة البيانات
require_once '../config/database.php';

/**
 * كلاس إدارة الباقات
 */
class PackagesAPI {
    
    /**
     * جلب جميع الباقات
     */
    public static function getAllPackages() {
        try {
            $query = "SELECT * FROM packages WHERE is_active = 1 ORDER BY display_order ASC";
            $packages = fetchAll($query);
            
            if ($packages) {
                // تحويل المميزات والمنصات من JSON إلى مصفوفة
                foreach ($packages as &$package) {
                    $package['features'] = json_decode($package['features'], true);
                    $package['supported_platforms'] = json_decode($package['supported_platforms'], true) ?? [];
                    // تحويل القيم الرقمية
                    $package['points'] = (int)$package['points'];
                    $package['messages'] = (int)$package['messages'];
                    $package['accounts'] = (int)$package['accounts'];
                }
                
                return self::sendResponse(true, 'تم جلب الباقات بنجاح', $packages);
            } else {
                return self::sendResponse(false, 'لا توجد باقات متاحة', []);
            }
        } catch (Exception $e) {
            return self::sendResponse(false, 'خطأ في جلب الباقات: ' . $e->getMessage(), null);
        }
    }

    /**
     * جلب باقة واحدة بالـ ID
     */
    public static function getPackageById($id) {
        try {
            $query = "SELECT * FROM packages WHERE id = ? AND is_active = 1";
            $package = fetchRow($query, [$id]);
            
            if ($package) {
                $package['features'] = json_decode($package['features'], true);
                $package['supported_platforms'] = json_decode($package['supported_platforms'], true) ?? [];
                // تحويل القيم الرقمية
                $package['points'] = (int)$package['points'];
                $package['messages'] = (int)$package['messages'];
                $package['accounts'] = (int)$package['accounts'];
                return self::sendResponse(true, 'تم جلب الباقة بنجاح', $package);
            } else {
                return self::sendResponse(false, 'الباقة غير موجودة', null);
            }
        } catch (Exception $e) {
            return self::sendResponse(false, 'خطأ في جلب الباقة: ' . $e->getMessage(), null);
        }
    }

    /**
     * إنشاء باقة جديدة
     */
    public static function createPackage($data) {
        try {
            // التحقق من البيانات المطلوبة
            $requiredFields = ['name_ar', 'name_en', 'name_fr', 'badge_ar', 'badge_en', 'badge_fr', 'monthly_price', 'yearly_price', 'features'];
            
            foreach ($requiredFields as $field) {
                if (!isset($data[$field]) || empty($data[$field])) {
                    return self::sendResponse(false, "الحقل {$field} مطلوب", null);
                }
            }

            // إعداد البيانات
            $name_ar = sanitizeInput($data['name_ar']);
            $name_en = sanitizeInput($data['name_en']);
            $name_fr = sanitizeInput($data['name_fr']);
            $badge_ar = sanitizeInput($data['badge_ar']);
            $badge_en = sanitizeInput($data['badge_en']);
            $badge_fr = sanitizeInput($data['badge_fr']);
            $monthly_price = floatval($data['monthly_price']);
            $yearly_price = floatval($data['yearly_price']);
            $currency = sanitizeInput($data['currency'] ?? 'ريال');
            $is_popular = isset($data['is_popular']) ? (bool)$data['is_popular'] : false;
            $features = json_encode($data['features'], JSON_UNESCAPED_UNICODE);
            $display_order = intval($data['display_order'] ?? 0);
            
            // بيانات الخصم
            $has_discount = isset($data['has_discount']) ? (bool)$data['has_discount'] : false;
            $monthly_discount = floatval($data['monthly_discount'] ?? 0);
            $yearly_discount = floatval($data['yearly_discount'] ?? 0);
            $monthly_discount_percentage = floatval($data['monthly_discount_percentage'] ?? 0);
            $yearly_discount_percentage = floatval($data['yearly_discount_percentage'] ?? 0);
            
            // الحقول الجديدة
            $points = intval($data['points'] ?? 0);
            $messages = intval($data['messages'] ?? 0);
            $accounts = intval($data['accounts'] ?? 1);
            $supported_platforms = isset($data['supported_platforms']) ? json_encode($data['supported_platforms'], JSON_UNESCAPED_UNICODE) : json_encode(['facebook', 'whatsapp', 'instagram'], JSON_UNESCAPED_UNICODE);

            // إذا كانت هذه الباقة شعبية، إزالة الشعبية من الباقات الأخرى
            if ($is_popular) {
                executeQuery("UPDATE packages SET is_popular = 0");
            }

            // إدراج الباقة الجديدة
            $query = "INSERT INTO packages (name_ar, name_en, name_fr, badge_ar, badge_en, badge_fr, monthly_price, yearly_price, currency, is_popular, features, display_order, has_discount, monthly_discount, yearly_discount, monthly_discount_percentage, yearly_discount_percentage, points, messages, accounts, supported_platforms, is_active) 
                     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)";
            
            $params = [$name_ar, $name_en, $name_fr, $badge_ar, $badge_en, $badge_fr, $monthly_price, $yearly_price, $currency, $is_popular, $features, $display_order, $has_discount, $monthly_discount, $yearly_discount, $monthly_discount_percentage, $yearly_discount_percentage, $points, $messages, $accounts, $supported_platforms];
            
            $stmt = executeQuery($query, $params);
            
            if ($stmt) {
                $newId = getLastInsertId();
                return self::sendResponse(true, 'تم إنشاء الباقة بنجاح', ['id' => $newId]);
            } else {
                return self::sendResponse(false, 'فشل في إنشاء الباقة', null);
            }

        } catch (Exception $e) {
            return self::sendResponse(false, 'خطأ في إنشاء الباقة: ' . $e->getMessage(), null);
        }
    }

    /**
     * تحديث باقة موجودة
     */
    public static function updatePackage($id, $data) {
        try {
            // التحقق من وجود الباقة
            $existingPackage = fetchRow("SELECT id FROM packages WHERE id = ?", [$id]);
            if (!$existingPackage) {
                return self::sendResponse(false, 'الباقة غير موجودة', null);
            }

            // إعداد البيانات
            $name_ar = sanitizeInput($data['name_ar']);
            $name_en = sanitizeInput($data['name_en']);
            $name_fr = sanitizeInput($data['name_fr']);
            $badge_ar = sanitizeInput($data['badge_ar']);
            $badge_en = sanitizeInput($data['badge_en']);
            $badge_fr = sanitizeInput($data['badge_fr']);
            $monthly_price = floatval($data['monthly_price']);
            $yearly_price = floatval($data['yearly_price']);
            $currency = sanitizeInput($data['currency'] ?? 'ريال');
            $is_popular = isset($data['is_popular']) ? (bool)$data['is_popular'] : false;
            $features = json_encode($data['features'], JSON_UNESCAPED_UNICODE);
            $display_order = intval($data['display_order'] ?? 0);
            $is_active = isset($data['is_active']) ? (bool)$data['is_active'] : true;
            
            // بيانات الخصم
            $has_discount = isset($data['has_discount']) ? (bool)$data['has_discount'] : false;
            $monthly_discount = floatval($data['monthly_discount'] ?? 0);
            $yearly_discount = floatval($data['yearly_discount'] ?? 0);
            $monthly_discount_percentage = floatval($data['monthly_discount_percentage'] ?? 0);
            $yearly_discount_percentage = floatval($data['yearly_discount_percentage'] ?? 0);
            
            // الحقول الجديدة
            $points = intval($data['points'] ?? 0);
            $messages = intval($data['messages'] ?? 0);
            $accounts = intval($data['accounts'] ?? 1);
            $supported_platforms = isset($data['supported_platforms']) ? json_encode($data['supported_platforms'], JSON_UNESCAPED_UNICODE) : json_encode(['facebook', 'whatsapp', 'instagram'], JSON_UNESCAPED_UNICODE);

            // إذا كانت هذه الباقة شعبية، إزالة الشعبية من الباقات الأخرى
            if ($is_popular) {
                executeQuery("UPDATE packages SET is_popular = 0 WHERE id != ?", [$id]);
            }

            // تحديث الباقة
            $query = "UPDATE packages SET 
                     name_ar = ?, name_en = ?, name_fr = ?, 
                     badge_ar = ?, badge_en = ?, badge_fr = ?,
                     monthly_price = ?, yearly_price = ?, currency = ?,
                     is_popular = ?, features = ?, display_order = ?, is_active = ?,
                     has_discount = ?, monthly_discount = ?, yearly_discount = ?,
                     monthly_discount_percentage = ?, yearly_discount_percentage = ?,
                     points = ?, messages = ?, accounts = ?, supported_platforms = ?
                     WHERE id = ?";
            
            $params = [$name_ar, $name_en, $name_fr, $badge_ar, $badge_en, $badge_fr, $monthly_price, $yearly_price, $currency, $is_popular, $features, $display_order, $is_active, $has_discount, $monthly_discount, $yearly_discount, $monthly_discount_percentage, $yearly_discount_percentage, $points, $messages, $accounts, $supported_platforms, $id];
            
            $stmt = executeQuery($query, $params);
            
            if ($stmt && $stmt->rowCount() > 0) {
                return self::sendResponse(true, 'تم تحديث الباقة بنجاح', null);
            } else {
                return self::sendResponse(false, 'لم يتم تحديث أي بيانات', null);
            }

        } catch (Exception $e) {
            return self::sendResponse(false, 'خطأ في تحديث الباقة: ' . $e->getMessage(), null);
        }
    }

    /**
     * حذف باقة
     */
    public static function deletePackage($id) {
        try {
            // التحقق من وجود الباقة
            $existingPackage = fetchRow("SELECT id FROM packages WHERE id = ?", [$id]);
            if (!$existingPackage) {
                return self::sendResponse(false, 'الباقة غير موجودة', null);
            }

            // حذف الباقة (soft delete)
            $query = "DELETE FROM `packages` WHERE id = ?";
            $stmt = executeQuery($query, [$id]);
            
            if ($stmt && $stmt->rowCount() > 0) {
                return self::sendResponse(true, 'تم حذف الباقة بنجاح', null);
            } else {
                return self::sendResponse(false, 'فشل في حذف الباقة', null);
            }

        } catch (Exception $e) {
            return self::sendResponse(false, 'خطأ في حذف الباقة: ' . $e->getMessage(), null);
        }
    }

    /**
     * تعيين باقة كشعبية
     */
    public static function setPopularPackage($id) {
        try {
            // إزالة الشعبية من جميع الباقات
            executeQuery("UPDATE packages SET is_popular = 0");
            
            // تعيين الباقة المحددة كشعبية
            $query = "UPDATE packages SET is_popular = 1 WHERE id = ? AND is_active = 1";
            $stmt = executeQuery($query, [$id]);
            
            if ($stmt && $stmt->rowCount() > 0) {
                return self::sendResponse(true, 'تم تعيين الباقة كشعبية بنجاح', null);
            } else {
                return self::sendResponse(false, 'الباقة غير موجودة أو غير نشطة', null);
            }

        } catch (Exception $e) {
            return self::sendResponse(false, 'خطأ في تعيين الباقة الشعبية: ' . $e->getMessage(), null);
        }
    }

    /**
     * إرسال الرد بتنسيق JSON
     */
    private static function sendResponse($success, $message, $data = null) {
        $response = [
            'success' => $success,
            'message' => $message,
            'timestamp' => date('Y-m-d H:i:s')
        ];
        
        if ($data !== null) {
            $response['data'] = $data;
        }
        
        echo json_encode($response, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
        exit;
    }
}

// معالجة الطلبات
try {
    $method = $_SERVER['REQUEST_METHOD'];
    $input = json_decode(file_get_contents('php://input'), true);
    
    // استخراج الـ ID من الـ URL إن وجد
    $pathInfo = $_SERVER['PATH_INFO'] ?? '';
    $pathParts = explode('/', trim($pathInfo, '/'));
    $packageId = isset($pathParts[0]) && is_numeric($pathParts[0]) ? intval($pathParts[0]) : null;
    
    switch ($method) {
        case 'GET':
            if ($packageId) {
                PackagesAPI::getPackageById($packageId);
            } else {
                PackagesAPI::getAllPackages();
            }
            break;
            
        case 'POST':
            if (isset($_GET['action']) && $_GET['action'] === 'set_popular' && $packageId) {
                PackagesAPI::setPopularPackage($packageId);
            } else {
                if (!$input) {
                    PackagesAPI::sendResponse(false, 'بيانات غير صحيحة', null);
                }
                PackagesAPI::createPackage($input);
            }
            break;
            
        case 'PUT':
            if (!$packageId) {
                PackagesAPI::sendResponse(false, 'معرف الباقة مطلوب', null);
            }
            if (!$input) {
                PackagesAPI::sendResponse(false, 'بيانات غير صحيحة', null);
            }
            PackagesAPI::updatePackage($packageId, $input);
            break;
            
        case 'DELETE':
            if (!$packageId) {
                PackagesAPI::sendResponse(false, 'معرف الباقة مطلوب', null);
            }
            PackagesAPI::deletePackage($packageId);
            break;
            
        default:
            PackagesAPI::sendResponse(false, 'طريقة الطلب غير مدعومة', null);
            break;
    }
    
} catch (Exception $e) {
    PackagesAPI::sendResponse(false, 'خطأ في معالجة الطلب: ' . $e->getMessage(), null);
}
?>