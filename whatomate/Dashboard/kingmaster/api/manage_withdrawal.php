<?php
session_start();
header('Content-Type: application/json');
require_once '../config/database.php';

// Check if admin (you should have your own admin check)
if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول أولاً'
    ]);
    exit;
}

$conn = getDB();
$data = json_decode(file_get_contents('php://input'), true);

$action = $data['action'] ?? '';
$withdrawal_id = $data['withdrawal_id'] ?? 0;
$new_status = $data['status'] ?? '';

if ($action !== 'update_status' || !$withdrawal_id || !$new_status) {
    echo json_encode([
        'success' => false,
        'message' => 'بيانات غير كاملة'
    ]);
    exit;
}

try {
    // Get withdrawal details
    $stmt = $conn->prepare("SELECT * FROM withdrawals WHERE id = :id");
    $stmt->execute([':id' => $withdrawal_id]);
    $withdrawal = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$withdrawal) {
        echo json_encode([
            'success' => false,
            'message' => 'الطلب غير موجود'
        ]);
        exit;
    }
    
    $old_status = $withdrawal['status'];
    
    // If approving from rejected, need to check balance again
    // If approving from pending, deduct amount and send notification
    if ($new_status === 'approved' && ($old_status === 'pending' || $old_status === 'rejected')) {
        $user_id = $withdrawal['user_id'];
        $amount = floatval($withdrawal['amount']);
        
        // Check current commission balance
        $stmt = $conn->prepare("SELECT commission FROM commission_wallets WHERE user_id = :user_id");
        $stmt->execute([':user_id' => $user_id]);
        $wallet = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$wallet) {
            echo json_encode([
                'success' => false,
                'message' => 'محفظة المستخدم غير موجودة'
            ]);
            exit;
        }
        
        $current_balance = floatval($wallet['commission']);
        
        if ($current_balance < $amount) {
            echo json_encode([
                'success' => false,
                'message' => 'رصيد المستخدم غير كافي'
            ]);
            exit;
        }
        
        // Deduct amount
        $new_balance = $current_balance - $amount;
        $stmt = $conn->prepare("UPDATE commission_wallets SET commission = :new_balance WHERE user_id = :user_id");
        $stmt->execute([
            ':new_balance' => $new_balance,
            ':user_id' => $user_id
        ]);
        
        // Get user phone
        $stmt = $conn->prepare("SELECT phone, first_name FROM users WHERE user_id = :user_id");
        $stmt->execute([':user_id' => $user_id]);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if ($user && !empty($user['phone'])) {
            $phone = preg_replace('/[^0-9]/', '', $user['phone']);
            
            // Send WhatsApp notification
            $message = "مرحباً " . $user['first_name'] . ",\n\n";
            $message .= "✅ تم قبول طلب السحب الخاص بك\n\n";
            $message .= "━━━━━━━━━━━━━━━━━━━\n\n";
            $message .= "💰 المبلغ: " . number_format($amount, 2) . " جنيه\n";
            $message .= "📌 طريقة السحب: " . $withdrawal['withdrawal_type'] . "\n\n";
            $message .= "ℹ️ بيانات التحويل:\n" . $withdrawal['withdrawal_details'] . "\n\n";
            $message .= "━━━━━━━━━━━━━━━━━━━\n\n";
            $message .= "سيتم تحويل المبلغ الان\n\n";
            $message .= "شكراً - فريق Kingmaster";
            
            $api_url = 'https://king-master.pro/api/send';
            $instance_id = '6905CDECE64DF';
            $access_token = '6604ac2316788';
            
            $url = $api_url . '?' . http_build_query([
                'number' => $phone,
                'type' => 'text',
                'message' => $message,
                'instance_id' => $instance_id,
                'access_token' => $access_token
            ]);
            
            $ch = curl_init();
            curl_setopt($ch, CURLOPT_URL, $url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
            curl_exec($ch);
            curl_close($ch);
        }
    }
    
    // If changing from approved back to rejected or pending, need to refund the amount
    if ($old_status === 'approved' && ($new_status === 'rejected' || $new_status === 'pending')) {
        $user_id = $withdrawal['user_id'];
        $amount = floatval($withdrawal['amount']);
        
        // Refund the amount back to commission wallet
        $stmt = $conn->prepare("UPDATE commission_wallets SET commission = commission + :amount WHERE user_id = :user_id");
        $stmt->execute([
            ':amount' => $amount,
            ':user_id' => $user_id
        ]);
    }
    
    // If rejecting from pending, send notification
    if ($new_status === 'rejected' && $old_status === 'pending') {
        $user_id = $withdrawal['user_id'];
        $amount = floatval($withdrawal['amount']);
        
        // Get user phone
        $stmt = $conn->prepare("SELECT phone, first_name FROM users WHERE user_id = :user_id");
        $stmt->execute([':user_id' => $user_id]);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if ($user && !empty($user['phone'])) {
            $phone = preg_replace('/[^0-9]/', '', $user['phone']);
            
            // Send WhatsApp notification
            $message = "مرحباً " . $user['first_name'] . ",\n\n";
            $message .= "❌ تم رفض طلب السحب الخاص بك\n\n";
            $message .= "━━━━━━━━━━━━━━━━━━━\n\n";
            $message .= "💰 المبلغ: " . number_format($amount, 2) . " جنيه\n";
            $message .= "📌 طريقة السحب: " . $withdrawal['withdrawal_type'] . "\n\n";
            $message .= "━━━━━━━━━━━━━━━━━━━\n\n";
            $message .= "📞 يمكنك التواصل مع الإدارة لمعرفة سبب الرفض\n\n";
            $message .= "شكراً - فريق Kingmaster";
            
            $api_url = 'https://king-master.pro/api/send';
            $instance_id = '692EFAA37496F';
            $access_token = '6604ac2316788';
            
            $url = $api_url . '?' . http_build_query([
                'number' => $phone,
                'type' => 'text',
                'message' => $message,
                'instance_id' => $instance_id,
                'access_token' => $access_token
            ]);
            
            $ch = curl_init();
            curl_setopt($ch, CURLOPT_URL, $url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
            curl_exec($ch);
            curl_close($ch);
        }
    }
    
    // Update withdrawal status
    $stmt = $conn->prepare("UPDATE withdrawals SET status = :status, updated_at = NOW() WHERE id = :id");
    $stmt->execute([
        ':status' => $new_status,
        ':id' => $withdrawal_id
    ]);
    
    $statusText = [
        'pending' => 'جاري المعالجة',
        'approved' => 'مقبول',
        'rejected' => 'مرفوض'
    ][$new_status] ?? $new_status;
    
    echo json_encode([
        'success' => true,
        'message' => 'تم تحديث الحالة إلى: ' . $statusText
    ]);
    
} catch(PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ: ' . $e->getMessage()
    ]);
}
?>
