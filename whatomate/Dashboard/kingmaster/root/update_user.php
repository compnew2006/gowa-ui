<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح بالدخول']);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    echo json_encode(['success' => false, 'message' => 'طريقة الطلب غير صحيحة']);
    exit;
}

try {
    $userId = isset($_POST['user_id']) ? (int)$_POST['user_id'] : 0;
    $isAdmin = isset($_POST['is_admin']) ? (int)$_POST['is_admin'] : 0;
    $pints = $_POST['pints'];

    $package = isset($_POST['package']) ? (int)$_POST['package'] : 0;
    $expiryDate = isset($_POST['expiry_date']) ? sanitizeInput($_POST['expiry_date']) : '';

    if ($userId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    if (empty($expiryDate)) {
        echo json_encode(['success' => false, 'message' => 'يرجى تحديد تاريخ الانتهاء']);
        exit;
    }



    // جلب user_id الأصلي من جدول users
    $getUserIdQuerys = "SELECT user_id FROM users WHERE id = ?";
    $userIdStmtx = executeQuery($getUserIdQuerys, [$userId]);
    $userRowx = $userIdStmtx->fetch(PDO::FETCH_ASSOC);

    if (!$userRowx) {
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        exit;
    }

    $user_id_tox = $userRowx['user_id'];



    // جلب الباقة الحالية للمستخدم
    $currentPackageQuery = "SELECT package FROM users WHERE id = ?";
    $currentPackageStmt = executeQuery($currentPackageQuery, [$userId]);
    $currentPackage = $currentPackageStmt->fetch(PDO::FETCH_ASSOC);
    $oldPackage = $currentPackage ? (int)$currentPackage['package'] : 0;

    // تحديث بيانات المستخدم
    $updateUserQuery = "UPDATE users SET is_admin = ?, package = ?, expiry_date = ?, points = ? WHERE id = ?";
    $updateResult = executeQuery($updateUserQuery, [$isAdmin, $package, $expiryDate, $pints, $userId]);

    if ($updateResult) {
        // إذا تغيرت الباقة وليست الباقة التجريبية (0)
             // جلب عدد النقاط من الباقة الجديدة
            $pointsQuery = "SELECT points FROM packages WHERE id = ?";
            $pointsStmt = executeQuery($pointsQuery, [$package]);
            $pointsData = $pointsStmt->fetch(PDO::FETCH_ASSOC);

            if ($pointsData) {
                $points = (int)$pointsData['points'];

                // جلب user_id الحقيقي (المستخدم في wallet)
                $getUserIdQuery = "SELECT user_id FROM users WHERE id = ?";
                $userIdStmt = executeQuery($getUserIdQuery, [$userId]);
                $userRow = $userIdStmt->fetch(PDO::FETCH_ASSOC);

                if ($userRow) {
                    $user_id_to = $userRow['user_id'];

                    // التحقق من وجود سجل في users_wallet
                    $walletCheckQuery = "SELECT points FROM users_wallet WHERE user_id = ?";
                    $walletCheckStmt = executeQuery($walletCheckQuery, [$user_id_to]);
                    $walletExists = $walletCheckStmt->fetch(PDO::FETCH_ASSOC);

                    if ($walletExists) {
                        // تحديث النقاط (إضافة إلى الرصيد الحالي)
                        $currentPoints = (int)$walletExists['points'];
                        $newPoints = $currentPoints + $points;

                        $updateWalletQuery = "UPDATE users_wallet SET points = ? WHERE user_id = ?";
                        executeQuery($updateWalletQuery, [$pints, $user_id_tox]);
                    } else {
                        // إنشاء سجل جديد
                        $insertWalletQuery = "INSERT INTO users_wallet (user_id, points) VALUES (?, ?)";
                        executeQuery($insertWalletQuery, [$user_id_to, $points]);
                    }

                    echo json_encode([
                        'success' => true,
                        'message' => 'تم تحديث بيانات المستخدم وإضافة النقاط بنجاح',
                        'user_id' => $user_id_to,
                        'points_added' => $points
                    ]);
                    exit;
                }
            }
         

        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث بيانات المستخدم بنجاح (بدون تعديل في النقاط)'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل تحديث بيانات المستخدم'
        ]);
    }

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء تنفيذ العملية',
        'error' => $e->getMessage()
    ]);
}

?>
