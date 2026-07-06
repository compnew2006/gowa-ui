<?php

require_once __DIR__ . '/../config/database.php';


/**
 * 🔹 دالة لجلب بيانات مستخدم باستخدام user_id
 */
function getUserByUserId($user_id) {
    $db = getDB();
    $query = "SELECT 
                first_name, 
                last_name, 
                email, 
                password, 
                phone, 
                img, 
                timezone, 
                job, 
                birth_date, 
                otp, 
                otp_created_at, 
                is_verified, 
                is_admin, 
                package, 
                points, 
                expiry_date, 
                created_at, 
                updated_at
              FROM users 
              WHERE user_id = :user_id
              LIMIT 1";
    
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    return $stmt->fetch(PDO::FETCH_ASSOC);
}

function getWeeklyStats($user_id) {
    $db = getDB();

    // نجيب آخر 7 أيام فقط
    $query = "
        SELECT 
            DAYOFWEEK(created_at) AS day_of_week,
            SUM(true_count) AS total_true
        FROM campaigns
        WHERE user_id = :user_id AND type_tools = 'Send'
          AND created_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
        GROUP BY DAYOFWEEK(created_at)
        ORDER BY day_of_week
    ";

    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);

    // نحول الأيام من أرقام إلى أسماء عربية
    $daysMap = [
        7 => 'السبت',
        1 => 'الأحد',
        2 => 'الأثنين',
        3 => 'الثلاثاء',
        4 => 'الأربعاء',
        5 => 'الخميس',
        6 => 'الجمعة'
    ];

    $data = array_fill_keys($daysMap, 0);

    foreach ($rows as $row) {
        $dayName = $daysMap[$row['day_of_week']];
        $data[$dayName] = (int)$row['total_true'];
    }

    return $data;
}

function getPlatformStats($user_id) {
    $db = getDB();

    $query = "
        SELECT 
            paltform, 
            COUNT(*) AS total
        FROM campaigns
        WHERE user_id = :user_id
        GROUP BY paltform
    ";

    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    return $stmt->fetchAll(PDO::FETCH_ASSOC);
}


function getLastPosts($limit = 4) {
    $query = "SELECT 
                `title`, 
                `content`, 
                `typs`, 
                `created_at`
              FROM 
                `posts`
              ORDER BY 
                `created_at` DESC
              LIMIT :limit";

    try {
        $db = getDB();
        $stmt = $db->prepare($query);
        $stmt->bindValue(':limit', (int)$limit, PDO::PARAM_INT);
        $stmt->execute();
        return $stmt->fetchAll(PDO::FETCH_ASSOC);
    } catch (PDOException $e) {
        error_log("Database Error: " . $e->getMessage());
        return [];
    }
}

function getCampaignCount($user_id) {
    $db = getDB();

    $query = "SELECT COUNT(*) AS total FROM campaigns WHERE user_id = :user_id";
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);

    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    return $result['total'] ?? 0;
}

function getMonthlyPoints($user_id) {
    $db = getDB();

    $query = "
        SELECT 
            MONTH(created_at) AS month_number,
            SUM(`count`) AS total_points
        FROM point_use
        WHERE user_id = :user_id
        GROUP BY MONTH(created_at)
        ORDER BY MONTH(created_at)
    ";

    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    $results = $stmt->fetchAll(PDO::FETCH_ASSOC);

    // تجهيز مصفوفة تحتوي على 12 شهر
    $months = array_fill(1, 12, 0);
    foreach ($results as $row) {
        $months[(int)$row['month_number']] = (int)$row['total_points'];
    }

    return array_values($months); // علشان تكون من 0 إلى 11 للـ Chart.js
}


function getExtractTrueCount($user_id) {
    $db = getDB();

    $query = "
        SELECT SUM(true_count) AS total_true_count
        FROM campaigns
        WHERE user_id = :user_id
        AND type_tools = 'Extract'
    ";

    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    return $stmt->fetch(PDO::FETCH_ASSOC)['total_true_count'] ?? 0;
}

function getPackageName($id) {
    $db = getDB();

    $query = "SELECT name FROM packages WHERE id = :id LIMIT 1";
    $stmt = $db->prepare($query);
    $stmt->execute([':id' => $id]);

    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    return $result['name'] ?? null;
}

 
function getUserIsAdmin($user_id) {
    $db = getDB();

    $query = "SELECT is_admin FROM users WHERE user_id = :user_id LIMIT 1";
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);

    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    return $result['is_admin'] ?? null;
}

function getReferralCountByReferrerId($referrer_id) {
    $db = getDB();

    $query = "SELECT COUNT(*) AS total FROM mlm_referrals WHERE referrer_id = :referrer_id";
    $stmt = $db->prepare($query);
    $stmt->execute([':referrer_id' => $referrer_id]);

    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    return $result['total'] ?? 0; // لو مفيش إحالات، ترجع صفر
}



function getReferralsByReferrerId($referrer_id) {
    $db = getDB();

    $query = "
        SELECT 
            u.user_id,
            CONCAT(u.first_name, ' ', u.last_name) AS full_name,
            u.email,
            u.created_at,
            u.img
        FROM mlm_referrals r
        JOIN users u ON r.user_id = u.user_id
        WHERE r.referrer_id = :referrer_id
        ORDER BY r.id DESC
        LIMIT 4
    ";

    $stmt = $db->prepare($query);
    $stmt->execute([':referrer_id' => $referrer_id]);

    return $stmt->fetchAll(PDO::FETCH_ASSOC);
}



function getUserData($user_id) {
    $db = getDB();

    // 1️⃣ جلب بيانات المستخدم الأساسية
    $query = "SELECT * 
              FROM users WHERE user_id = :user_id";
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$user) return null;

    // 2️⃣ عدد الإحالات
    $query = "SELECT COUNT(*) AS total_referrals FROM mlm_referrals WHERE referrer_id = :user_id";
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    $referrals = $stmt->fetch(PDO::FETCH_ASSOC)['total_referrals'] ?? 0;

    // 3️⃣ بيانات الباقة
    $query = "SELECT name, features FROM packages WHERE id = :package_id";
    $stmt = $db->prepare($query);
    $stmt->execute([':package_id' => $user['package']]);
    $package = $stmt->fetch(PDO::FETCH_ASSOC);

    // 4️⃣ النقاط من المحفظة
    $query = "SELECT points FROM users_wallet WHERE user_id = :user_id";
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    $wallet = $stmt->fetch(PDO::FETCH_ASSOC);
    $points = $wallet['points'] ?? 0;

    // 5️⃣ الصورة (افتراضية لو فاضية أو 0)
    $avatar = (!empty($user['img']) && $user['img'] != '0')
        ? $user['img']
        : 'https://ui-avatars.com/api/?name=' . urlencode($user['first_name'] . ' ' . $user['last_name']) . '&background=667eea&color=fff&size=200';

    // 6️⃣ تجهيز مميزات الباقة
    $features = [];
    if (!empty($package['features'])) {
        $decoded = json_decode($package['features'], true);
        $features = $decoded ?: explode(',', $package['features']);
    }

    // 7️⃣ حساب الأيام المتبقية
    $expiry = new DateTime($user['expiry_date']);
    $now = new DateTime();
    $days_remaining = $now->diff($expiry)->days;

    // 8️⃣ بناء النتيجة النهائية
    $userData = [
        'name' => $user['first_name'] . ' ' . $user['last_name'],
        'email' => $user['email'],
        'avatar' => $avatar,
        'plan' => $package['name'] ?? 'غير محدد',
        'plan_icon' => 'fas fa-crown',
        'plan_color' => '#f59e0b',
        'features' => $features,
        'subscription_date' => $user['created_at'],
        'expiry_date' => $user['expiry_date'],
        'days_remaining' => $days_remaining,
        'points' => $points,
        'referrals' => $referrals,
        'timezone' => $user['timezone']
    ];

    return $userData;
}


function getActivityLog($user_id) {
    $db = getDB();

    $query = "
        SELECT action, created_at 
        FROM logs 
        WHERE user_id = :user_id 
        ORDER BY created_at DESC 
        LIMIT 5
    ";
    $stmt = $db->prepare($query);
    $stmt->execute([':user_id' => $user_id]);
    $logs = $stmt->fetchAll(PDO::FETCH_ASSOC);

    // تعريف الأيقونات والألوان بناءً على نوع النشاط
    $icons = [
        'تسجيل دخول' => ['icon' => 'fa-sign-in-alt', 'color' => '#10b981'],
        'إضافة حساب Instagram' => ['icon' => 'fa-plus-circle', 'color' => '#667eea'],
        'نشر منشور' => ['icon' => 'fa-paper-plane', 'color' => '#3b82f6'],
        'تفعيل كوبون' => ['icon' => 'fa-ticket-alt', 'color' => '#f59e0b'],
        'تحديث الملف الشخصي' => ['icon' => 'fa-user-edit', 'color' => '#8b5cf6'],
        'حذف منشور' => ['icon' => 'fa-trash', 'color' => '#ef4444'],
    ];

    $activity_log = [];

    foreach ($logs as $log) {
        $action = $log['action'];
        $created_at = $log['created_at'];

        // لو النشاط مش موجود في القائمة، نخليه افتراضي
        $icon = $icons[$action]['icon'] ?? 'fa-info-circle';
        $color = $icons[$action]['color'] ?? '#6b7280'; // رمادي

        $activity_log[] = [
            'action' => $action,
            'time' => $created_at,
            'icon' => $icon,
            'color' => $color
        ];
    }

    return $activity_log;
}



function generateTimezoneList() {
    $zones = DateTimeZone::listIdentifiers();
    $output = [];

    foreach ($zones as $zone) {
        $dateTime = new DateTime('now', new DateTimeZone($zone));
        $offset = $dateTime->getOffset() / 3600;
        $sign = ($offset >= 0) ? '+' : '';
        $formattedOffset = 'GMT' . $sign . $offset;

        // نحاول استخراج المدينة فقط من اسم المنطقة
        $city = str_replace('_', ' ', explode('/', $zone)[1] ?? $zone);

        $output[$zone] = "{$city} ({$formattedOffset})";
    }

    ksort($output); // ترتيب أبجدي
    return $output;
}


function insertSyswalt($price, $typs, $created_at) {
    $db = getDB();

    $query = "INSERT INTO syswalt (price, typs, created_at) VALUES (:price, :typs, :created_at)";
    $stmt = $db->prepare($query);

    $stmt->execute([
        ':price' => $price,
        ':typs' => $typs,
        ':created_at' => $created_at
    ]);

    return $db->lastInsertId(); // ترجع ID الصف الجديد
}

/**
 * جلب جميع سجلات syswalt مع الفلاتر والpagination
 * 
 * @param int|null $user_id - معرف المستخدم (غير مستخدم حالياً)
 * @param string $filter_date - تاريخ محدد (YYYY-MM-DD)
 * @param string $filter_year - سنة محددة
 * @param string $filter_month - شهر محدد (1-12)
 * @param int $offset - بداية الصفحة
 * @param int $per_page - عدد السجلات في الصفحة
 * @return array - [
 *   'records' => مصفوفة السجلات,
 *   'total_records' => إجمالي عدد السجلات,
 *   'total_amount' => إجمالي المبلغ,
 *   'total_pages' => إجمالي عدد الصفحات
 * ]
 */
function getAllSyswalt($user_id = null, $filter_date = '', $filter_year = '', $filter_month = '', $offset = 0, $per_page = 50) {
    $db = getDB();
    
    // بناء شروط WHERE
    $where_clauses = [];
    $params = [];
    
    if (!empty($filter_date)) {
        $where_clauses[] = "DATE(created_at) = :date";
        $params[':date'] = $filter_date;
    } elseif (!empty($filter_month) && !empty($filter_year)) {
        $where_clauses[] = "YEAR(created_at) = :year AND MONTH(created_at) = :month";
        $params[':year'] = $filter_year;
        $params[':month'] = $filter_month;
    } elseif (!empty($filter_year)) {
        $where_clauses[] = "YEAR(created_at) = :year";
        $params[':year'] = $filter_year;
    }
    
    $where_sql = count($where_clauses) > 0 ? implode(' AND ', $where_clauses) : '1=1';
    
    // عد إجمالي السجلات وحساب المجموع
    $count_stmt = $db->prepare("SELECT COUNT(*) as total, SUM(CAST(price AS DECIMAL(10,2))) as total_amount FROM syswalt WHERE " . $where_sql);
    $count_stmt->execute($params);
    $count_result = $count_stmt->fetch(PDO::FETCH_ASSOC);
    $total_records = $count_result['total'];
    $total_amount = $count_result['total_amount'] ?? 0;
    $total_pages = ceil($total_records / $per_page);
    
    // جلب السجلات مع pagination
    $stmt = $db->prepare("SELECT id, price, typs, created_at FROM syswalt WHERE " . $where_sql . " ORDER BY created_at DESC LIMIT :limit OFFSET :offset");
    foreach ($params as $key => $value) {
        $stmt->bindValue($key, $value);
    }
    $stmt->bindValue(':limit', $per_page, PDO::PARAM_INT);
    $stmt->bindValue(':offset', $offset, PDO::PARAM_INT);
    $stmt->execute();
    $records = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    return [
        'records' => $records,
        'total_records' => $total_records,
        'total_amount' => $total_amount,
        'total_pages' => $total_pages
    ];
}

/**
 * ==============================================================
 * Announcements System Functions
 * ==============================================================
 */

/**
 * Get all announcements (active only for users, all for admin)
 */
function getAllAnnouncements($active_only = true) {
    $db = getDB();
    $sql = "SELECT * FROM announcements";
    if ($active_only) {
        $sql .= " WHERE is_active = 1";
    }
    $sql .= " ORDER BY created_at DESC";
    $stmt = $db->query($sql);
    return $stmt->fetchAll(PDO::FETCH_ASSOC);
}

/**
 * Get single announcement by ID
 */
function getAnnouncementById($id) {
    $db = getDB();
    $stmt = $db->prepare("SELECT * FROM announcements WHERE id = :id");
    $stmt->execute([':id' => $id]);
    return $stmt->fetch(PDO::FETCH_ASSOC);
}

/**
 * Create new announcement
 */
function createAnnouncement($title, $message, $is_active = 1) {
    $db = getDB();
    $stmt = $db->prepare("INSERT INTO announcements (title, message, is_active) VALUES (:title, :message, :is_active)");
    $stmt->execute([
        ':title' => $title,
        ':message' => $message,
        ':is_active' => $is_active
    ]);
    return $db->lastInsertId();
}

/**
 * Update announcement
 */
function updateAnnouncement($id, $title, $message, $is_active) {
    $db = getDB();
    $stmt = $db->prepare("UPDATE announcements SET title = :title, message = :message, is_active = :is_active WHERE id = :id");
    return $stmt->execute([
        ':id' => $id,
        ':title' => $title,
        ':message' => $message,
        ':is_active' => $is_active
    ]);
}

/**
 * Delete announcement
 */
function deleteAnnouncement($id) {
    $db = getDB();
    $stmt = $db->prepare("DELETE FROM announcements WHERE id = :id");
    return $stmt->execute([':id' => $id]);
}

/**
 * Get unread announcements for a user
 */
function getUnreadAnnouncements($user_id) {
    $db = getDB();
    $sql = "SELECT a.* FROM announcements a 
            WHERE a.is_active = 1 
            AND a.id NOT IN (
                SELECT announcement_id FROM user_announcement_views WHERE user_id = :user_id
            )
            ORDER BY a.created_at DESC";
    $stmt = $db->prepare($sql);
    $stmt->execute([':user_id' => $user_id]);
    return $stmt->fetchAll(PDO::FETCH_ASSOC);
}

/**
 * Mark announcement as viewed by user
 */
function markAnnouncementAsViewed($user_id, $announcement_id) {
    $db = getDB();
    try {
        $stmt = $db->prepare("INSERT IGNORE INTO user_announcement_views (user_id, announcement_id) VALUES (:user_id, :announcement_id)");
        return $stmt->execute([
            ':user_id' => $user_id,
            ':announcement_id' => $announcement_id
        ]);
    } catch (PDOException $e) {
        // If already exists, ignore
        return true;
    }
}




function getcommission_walletsById($id) {
    $db = getDB();
    $stmt = $db->prepare("SELECT * FROM commission_wallets WHERE user_id = :id");
    $stmt->execute([':id' => $id]);
    return $stmt->fetch(PDO::FETCH_ASSOC);
}


function get_Exp($id) {
    $db = getDB();
    $stmt = $db->prepare("SELECT * FROM users WHERE user_id = :id");
    $stmt->execute([':id' => $id]);
    return $stmt->fetch(PDO::FETCH_ASSOC);
}


function getContactCount($id) {
    $db = getDB();

    $query = "SELECT `count` FROM `contacts` WHERE `id` = :id LIMIT 1";
    $stmt = $db->prepare($query);
    $stmt->execute([':id' => $id]);

    $result = $stmt->fetch(PDO::FETCH_ASSOC);

    return $result['count'] ?? 0; // لو مفيش نتيجة يرجّع 0
}
