<?php
require_once __DIR__ . '/../config/database.php';

$sessionUserId = requireAuthenticatedUser();
enforceRateLimit('transfer:' . $sessionUserId, 10, 300);

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('Method not allowed', 405);
}

$pdo = getDB();
$input = readJsonBody();
verifyCsrfToken($input['csrf_token'] ?? null);

$receiverId = (int)($input['receiver_id'] ?? 0);
$transferType = trim((string)($input['transfer_type'] ?? ''));
$amount = (float)($input['amount'] ?? 0);
$password = (string)($input['password'] ?? '');
$note = cleanText($input['note'] ?? '', 255);

if ($receiverId <= 0 || !in_array($transferType, ['points', 'money'], true) || $amount <= 0 || $password === '') {
    respondError('جميع الحقول مطلوبة', 400);
}

try {
    $pdo->beginTransaction();

    $stmt = $pdo->prepare('SELECT id, user_id, email, password FROM users WHERE user_id = ? OR id = ? LIMIT 1');
    $stmt->execute([$sessionUserId, $sessionUserId]);
    $senderUser = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$senderUser || !verifyPassword($password, $senderUser['password'])) {
        $pdo->rollBack();
        respondError('كلمة المرور غير صحيحة', 401);
    }

    $senderId = (int)$senderUser['id'];
    if ($senderId === $receiverId || (string)$senderUser['user_id'] === (string)$receiverId) {
        $pdo->rollBack();
        respondError('لا يمكنك التحويل لنفسك', 400);
    }

    $stmt = $pdo->prepare('SELECT id, user_id, email FROM users WHERE id = ? OR user_id = ? LIMIT 1');
    $stmt->execute([$receiverId, $receiverId]);
    $receiverUser = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$receiverUser) {
        $pdo->rollBack();
        respondError('المستخدم المستلم غير موجود', 404);
    }
    if ((string)$receiverUser['user_id'] === (string)$senderUser['user_id'] || strcasecmp($receiverUser['email'] ?? '', $senderUser['email'] ?? '') === 0) {
        $pdo->rollBack();
        respondError('لا يمكنك التحويل لنفسك', 400);
    }

    $stmt = $pdo->prepare('SELECT points_balance, money_balance, user_name FROM wallets WHERE user_id = ? FOR UPDATE');
    $stmt->execute([$senderId]);
    $senderWallet = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$senderWallet) {
        $pdo->rollBack();
        respondError('محفظة المرسل غير موجودة', 404);
    }

    $stmt = $pdo->prepare('SELECT user_name FROM wallets WHERE user_id = ? FOR UPDATE');
    $stmt->execute([(int)$receiverUser['id']]);
    $receiverWallet = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$receiverWallet) {
        $pdo->rollBack();
        respondError('محفظة المستلم غير موجودة', 404);
    }

    $balanceField = $transferType === 'points' ? 'points_balance' : 'money_balance';
    if ((float)$senderWallet[$balanceField] < $amount) {
        $pdo->rollBack();
        respondError('رصيد غير كافٍ', 400);
    }

    $stmt = $pdo->prepare("UPDATE wallets SET $balanceField = $balanceField - ? WHERE user_id = ?");
    $stmt->execute([$amount, $senderId]);
    $stmt = $pdo->prepare("UPDATE wallets SET $balanceField = $balanceField + ? WHERE user_id = ?");
    $stmt->execute([$amount, (int)$receiverUser['id']]);

    $stmt = $pdo->prepare('INSERT INTO wallet_transactions (transaction_type, sender_id, sender_name, receiver_id, receiver_name, amount, note) VALUES (?, ?, ?, ?, ?, ?, ?)');
    $stmt->execute([$transferType, $senderId, $senderWallet['user_name'], (int)$receiverUser['id'], $receiverWallet['user_name'], $amount, $note]);

    $pdo->commit();
    respondJson(['success' => true, 'message' => 'تم التحويل بنجاح']);
} catch (Throwable $e) {
    if ($pdo->inTransaction()) {
        $pdo->rollBack();
    }
    error_log('transfer error: ' . $e->getMessage());
    respondError('خطأ في التحويل', 500);
}
