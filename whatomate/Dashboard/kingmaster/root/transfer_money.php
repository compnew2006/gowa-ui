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
    $toUserId = isset($_POST['user_id']) ? (int)$_POST['user_id'] : 0;
    $amount = isset($_POST['amount']) ? (float)$_POST['amount'] : 0;
    $fromUserId = $_SESSION['user_id'];

    if ($toUserId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    if ($amount <= 0) {
        echo json_encode(['success' => false, 'message' => 'المبلغ يجب أن يكون أكبر من صفر']);
        exit;
    }

    // جلب بيانات المرسل
    $fromUserQuery = "SELECT email FROM users WHERE user_id = ?";
    $fromUserStmt = executeQuery($fromUserQuery, [$fromUserId]);
    $fromUser = $fromUserStmt->fetch();

    if (!$fromUser) {
        echo json_encode(['success' => false, 'message' => 'بيانات المرسل غير موجودة']);
        exit;
    }

    // جلب بيانات المستقبل
    $toUserQuery = "SELECT user_id, email, first_name, last_name FROM users WHERE id = ?";
    $toUserStmt = executeQuery($toUserQuery, [$toUserId]);
    $toUser = $toUserStmt->fetch();
     $uuids = $toUser['user_id'];
    if (!$toUser) {
        echo json_encode(['success' => false, 'message' => 'المستخدم المستهدف غير موجود']);
        exit;
    }

    // التحقق من وجود محفظة للمستخدم المستهدف
    $walletCheckQuery = "SELECT balance FROM users_wallet WHERE user_id = ?";
    $walletCheckStmt = executeQuery($walletCheckQuery, [$uuids]);
    $wallet = $walletCheckStmt->fetch();

    if ($wallet) {
        // تحديث الرصيد (إضافة المبلغ)
        $newBalance = $wallet['balance'] + $amount;
        $updateWalletQuery = "UPDATE users_wallet SET balance = ? WHERE user_id = ?";
        executeQuery($updateWalletQuery, [$newBalance, $uuids]);
    } else {
        // إنشاء محفظة جديدة
        $insertWalletQuery = "INSERT INTO users_wallet (user_id, balance, points) VALUES (?, ?, 0)";
        executeQuery($insertWalletQuery, [$uuids, $amount]);
    }

    // تسجيل المعاملة في جدول transactions
    $insertTransactionQuery = "INSERT INTO transactions 
        (user_id, from_user_id, to_user_id, from_email, to_email, transaction_type, amount_type, amount) 
        VALUES (?, ?, ?, ?, ?, 'receive', 'money', ?)";
    
    executeQuery($insertTransactionQuery, [
        $uuids,           // user_id
        $fromUserId,         // from_user_id
        $toUserId,           // to_user_id
        $fromUser['email'],  // from_email
        $toUser['email'],    // to_email
        $amount              // amount
    ]);

    echo json_encode([
        'success' => true,
        'message' => "تم تحويل {$amount} جنيه إلى حساب {$toUser['first_name']} {$toUser['last_name']}"
    ]);

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء التحويل',
        'error' => $e->getMessage()
    ]);
}
?>
