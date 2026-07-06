<?php
header('Content-Type: application/json');
 
// الاتصال بقاعدة البيانات
require_once '../config/database.php';
$pdo = getDB();

try {
    $stats = [];
    
    // 1. إجمالي الرسائل (من جدول contacts حسب sent_count)
    try {
        $stmt = $pdo->query("SELECT SUM(sent_count) as total FROM contacts");
        $result = $stmt->fetch(PDO::FETCH_ASSOC);
        $stats['totalMessages'] = (int)($result['total'] ?? 0);
    } catch (PDOException $e) {
        $stats['totalMessages'] = 0;
    }
    
    // 2. إجمالي المستخدمين (من جدول mlm_users)
    try {
        $stmt = $pdo->query("SELECT COUNT(*) as count FROM mlm_users");
        $stats['totalUsers'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['totalUsers'] = 0;
    }
    
    // 3. إجمالي الحسابات (من جدول contacts)
    try {
        $stmt = $pdo->query("SELECT COUNT(*) as count FROM contacts");
        $stats['totalAccounts'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['totalAccounts'] = 0;
    }
    
    // 4. إجمالي الاستخراجات (مجموع count في جدول contacts)
    try {
        $stmt = $pdo->query("SELECT SUM(count) as total FROM contacts");
        $result = $stmt->fetch(PDO::FETCH_ASSOC);
        $stats['totalExtractions'] = (int)($result['total'] ?? 0);
    } catch (PDOException $e) {
        $stats['totalExtractions'] = 0;
    }
    
    // 5. المستخدمون النشطون الآن (المستخدمون النشطون في mlm_users)
    try {
        $stmt = $pdo->query("
            SELECT COUNT(*) as count 
            FROM mlm_users 
            WHERE status = 'active'
        ");
        $stats['activeUsers'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['activeUsers'] = 0;
    }
    
    // 6. المستخدمون غير النشطين
    $stats['inactiveUsers'] = $stats['totalUsers'] - $stats['activeUsers'];
    
    // 7. إجمالي الحملات (من جدول contacts)
    try {
        $stmt = $pdo->query("
            SELECT COUNT(*) as count 
            FROM contacts 
            WHERE sending_status != 'idle'
        ");
        $stats['totalCampaigns'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['totalCampaigns'] = 0;
    }
    
    // 8. الحملات الجارية (status = 'sending')
    try {
        $stmt = $pdo->query("
            SELECT COUNT(*) as count 
            FROM contacts 
            WHERE sending_status = 'sending'
        ");
        $stats['runningCampaigns'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['runningCampaigns'] = 0;
    }
    
    // 9. الحملات المتوقفة (status = 'paused')
    try {
        $stmt = $pdo->query("
            SELECT COUNT(*) as count 
            FROM contacts 
            WHERE sending_status = 'paused'
        ");
        $stats['stoppedCampaigns'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['stoppedCampaigns'] = 0;
    }
    
    // 10. الحملات المنتهية (status = 'completed')
    try {
        $stmt = $pdo->query("
            SELECT COUNT(*) as count 
            FROM contacts 
            WHERE sending_status = 'completed'
        ");
        $stats['finishedCampaigns'] = (int)$stmt->fetch(PDO::FETCH_ASSOC)['count'];
    } catch (PDOException $e) {
        $stats['finishedCampaigns'] = 0;
    }
    
    // 11. أفضل الأدوات استخداماً (من جدول contacts حسب النوع)
    try {
        $stmt = $pdo->query("
            SELECT type as name, COUNT(*) as count 
            FROM contacts 
            GROUP BY type 
            ORDER BY count DESC 
            LIMIT 5
        ");
        $topTools = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        // تحويل الأسماء للعربية
        foreach ($topTools as &$tool) {
            if ($tool['name'] == 'csv') {
                $tool['name'] = 'استيراد CSV';
            } elseif ($tool['name'] == 'text') {
                $tool['name'] = 'إدخال نصي';
            }
        }
        
        $stats['topTools'] = $topTools;
    } catch (PDOException $e) {
        $stats['topTools'] = [];
    }
    
    // 12. أفضل المنصات استخداماً (من جدول contacts)
    $platforms = [
        'facebook' => 0,
        'whatsapp' => 0,
        'instagram' => 0,
        'telegram' => 0,
        'email' => 0,
        'google_map' => 0,
        'business' => 0
    ];
    
    // حساب عدد الاستخدامات لكل منصة من جدول contacts
    try {
        $stmt = $pdo->query("
            SELECT platform, COUNT(*) as count 
            FROM contacts 
            GROUP BY platform
        ");
        $platformData = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        foreach ($platformData as $data) {
            $platform = strtolower(trim($data['platform']));
            if (isset($platforms[$platform])) {
                $platforms[$platform] = (int)$data['count'];
            }
        }
    } catch (PDOException $e) {
        // إذا حدث خطأ، اترك القيم كما هي (0)
    }
    
    $stats['topPlatforms'] = $platforms;
    
    echo json_encode([
        'success' => true,
        'stats' => $stats
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
