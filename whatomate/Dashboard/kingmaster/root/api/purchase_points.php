<?php
session_start();
header('Content-Type: application/json');

 
require_once '../includes/functions.php';

$user_id = $_SESSION['user_id'] ?? 1;

try {
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (empty($input['package_id']) || empty($input['payment_method'])) {
        echo json_encode([
            'success' => false,
            'message' => 'بيانات غير مكتملة'
        ]);
        exit;
    }
    
    $conn = getDB();
    $conn->beginTransaction();
    
    // 1. جلب معلومات الباقة
    $packageStmt = $conn->prepare("SELECT * FROM points_packages WHERE id = :id AND is_active = 1");
    $packageStmt->bindParam(':id', $input['package_id'], PDO::PARAM_INT);
    $packageStmt->execute();
    $package = $packageStmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$package) {
        $conn->rollBack();
        echo json_encode([
            'success' => false,
            'message' => 'الباقة غير موجودة'
        ]);
        exit;
    }
    
    // 2. إذا كان الدفع من الرصيد
    if ($input['payment_method'] === 'balance') {
        // جلب رصيد المستخدم من users_wallet
        $walletStmt = $conn->prepare("SELECT * FROM users_wallet WHERE user_id = :user_id");
        $walletStmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
        $walletStmt->execute();
        $wallet = $walletStmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$wallet) {
            $conn->rollBack();
            echo json_encode([
                'success' => false,
                'message' => 'المحفظة غير موجودة'
            ]);
            exit;
        }
        
        // التحقق من كفاية الرصيد
        if ($wallet['balance'] < $package['price']) {
            $conn->rollBack();
            echo json_encode([
                'success' => false,
                'message' => 'رصيدك غير كافٍ',
                'current_balance' => floatval($wallet['balance']),
                'required' => floatval($package['price']),
                'shortage' => floatval($package['price'] - $wallet['balance'])
            ]);
            exit;
        }
        
        // حفظ القيم القديمة لتسجيل المعاملة
        $balanceBefore = $wallet['balance'];
        $pointsBefore = $wallet['points'];
        
        // 3. خصم المبلغ من balance وإضافة النقاط إلى points
        $newBalance = $wallet['balance'] - $package['price'];
        $newPoints = $wallet['points'] + $package['points_count'];
        
        $updateWalletStmt = $conn->prepare(
            "UPDATE users_wallet 
             SET balance = :balance, 
                 points = :points,
                 updated_at = CURRENT_TIMESTAMP
             WHERE user_id = :user_id"
        );
        $updateWalletStmt->bindParam(':balance', $newBalance, PDO::PARAM_STR);
        $updateWalletStmt->bindParam(':points', $newPoints, PDO::PARAM_STR);
        $updateWalletStmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
        $updateWalletStmt->execute();
        
        // 4. تسجيل عملية خصم الرصيد في transactions
        $systemEmail = 'النظام';
        $transactionStmt = $conn->prepare(
            "INSERT INTO transactions 
             (user_id, from_user_id, to_user_id, from_email, to_email, transaction_type, amount_type, amount)
             VALUES (:user_id, :from_user_id, 0, :from_email, :to_email, 'send', 'money', :amount)"
        );
        $transactionStmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
        $transactionStmt->bindParam(':from_user_id', $user_id, PDO::PARAM_INT);
        $transactionStmt->bindParam(':from_email', $systemEmail, PDO::PARAM_STR);
        $transactionStmt->bindParam(':to_email', $systemEmail, PDO::PARAM_STR);
        $transactionStmt->bindParam(':amount', $package['price'], PDO::PARAM_STR);
        $transactionStmt->execute();
        
        // 5. تسجيل عملية إضافة النقاط في transactions
        $pointsTransactionStmt = $conn->prepare(
            "INSERT INTO transactions 
             (user_id, from_user_id, to_user_id, from_email, to_email, transaction_type, amount_type, amount)
             VALUES (:user_id, 0, :to_user_id, :from_email, :to_email, 'receive', 'points', :points)"
        );
        $pointsTransactionStmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
        $pointsTransactionStmt->bindParam(':to_user_id', $user_id, PDO::PARAM_INT);
        $pointsTransactionStmt->bindParam(':from_email', $systemEmail, PDO::PARAM_STR);
        $pointsTransactionStmt->bindParam(':to_email', $systemEmail, PDO::PARAM_STR);
        $pointsTransactionStmt->bindParam(':points', $package['points_count'], PDO::PARAM_STR);
        $pointsTransactionStmt->execute();
        
        $conn->commit();
        

insertSyswalt($package['price'], 'شحن نقاط', date('Y-m-d H:i:s'));


        echo json_encode([
            'success' => true,
            'message' => 'تمت عملية الشراء بنجاح',
            'data' => [
                'points_added' => intval($package['points_count']),
                'amount_paid' => floatval($package['price']),
                'new_balance' => floatval($newBalance),
                'new_points' => floatval($newPoints)
            ]
        ]);
        
    } else {
        $conn->rollBack();
        echo json_encode([
            'success' => false,
            'message' => 'طريقة دفع غير صحيحة'
        ]);
    }
    
} catch (PDOException $e) {
    if (isset($conn)) {
        $conn->rollBack();
    }
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
