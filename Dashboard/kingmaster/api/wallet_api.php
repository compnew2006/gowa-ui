<?php
require_once __DIR__ . '/../config/database.php';

$user_id = requireAuthenticatedUser();
enforceRateLimit('wallet_api:' . $user_id, 60, 60);
$pdo = getDB();
$action = $_POST['action'] ?? ($_GET['action'] ?? '');

switch ($action) {
    case 'get_wallet':
        getWallet($pdo, $user_id);
        break;
    case 'get_transactions':
        getTransactions($pdo, $user_id);
        break;
    case 'transfer':
        verifyCsrfToken();
        transfer($pdo, $user_id);
        break;
    default:
        respondError('Invalid action', 400);
}

function getWallet($pdo, $user_id) {
    try {
        $stmt = $pdo->prepare('SELECT * FROM users_wallet WHERE user_id = ?');
        $stmt->execute([$user_id]);
        $wallet = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$wallet) {
            $stmt = $pdo->prepare('INSERT INTO users_wallet (user_id, balance, points) VALUES (?, 0, 0)');
            $stmt->execute([$user_id]);
            $stmt = $pdo->prepare('SELECT * FROM users_wallet WHERE user_id = ?');
            $stmt->execute([$user_id]);
            $wallet = $stmt->fetch(PDO::FETCH_ASSOC);
        }
        $stmt = $pdo->prepare('SELECT COUNT(*) FROM transactions WHERE user_id = ?');
        $stmt->execute([$user_id]);
        respondJson(['success' => true, 'wallet' => $wallet, 'total_transactions' => (int)$stmt->fetchColumn()]);
    } catch (Throwable $e) {
        error_log('wallet get error: ' . $e->getMessage());
        respondError('خطأ في جلب البيانات', 500);
    }
}

function getTransactions($pdo, $user_id) {
    try {
        $date = $_GET['date'] ?? '';
        $month = $_GET['month'] ?? '';
        $sql = 'SELECT * FROM transactions WHERE user_id = ?';
        $params = [$user_id];
        if ($date !== '' && preg_match('/^\d{4}-\d{2}-\d{2}$/', $date)) {
            $sql .= ' AND DATE(created_at) = ?';
            $params[] = $date;
        }
        if ($month !== '' && preg_match('/^\d{4}-\d{2}$/', $month)) {
            $sql .= " AND DATE_FORMAT(created_at, '%Y-%m') = ?";
            $params[] = $month;
        }
        $sql .= ' ORDER BY created_at DESC LIMIT 200';
        $stmt = $pdo->prepare($sql);
        $stmt->execute($params);
        respondJson(['success' => true, 'transactions' => $stmt->fetchAll(PDO::FETCH_ASSOC)]);
    } catch (Throwable $e) {
        error_log('wallet transactions error: ' . $e->getMessage());
        respondError('خطأ في جلب المعاملات', 500);
    }
}

function transfer($pdo, $user_id) {
    enforceRateLimit('wallet_transfer:' . $user_id, 10, 300);

    $to_email = filter_var($_POST['to_email'] ?? '', FILTER_SANITIZE_EMAIL);
    $amount_type = $_POST['amount_type'] ?? '';
    $amount = (float)($_POST['amount'] ?? 0);
    $password = $_POST['password'] ?? '';

    if (!isValidEmail($to_email) || !in_array($amount_type, ['money', 'points'], true) || $amount <= 0 || $password === '') {
        respondError('البيانات المطلوبة مفقودة', 400);
    }

    try {
        $pdo->beginTransaction();

        $stmt = $pdo->prepare('SELECT user_id, email, password FROM users WHERE user_id = ? LIMIT 1');
        $stmt->execute([$user_id]);
        $sender = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$sender || !verifyPassword($password, $sender['password'])) {
            $pdo->rollBack();
            respondError('كلمة المرور غير صحيحة', 401);
        }

        $stmt = $pdo->prepare('SELECT user_id, email FROM users WHERE email = ? LIMIT 1');
        $stmt->execute([$to_email]);
        $receiver = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$receiver) {
            $pdo->rollBack();
            respondError('المستخدم غير موجود', 404);
        }

        $to_user_id = $receiver['user_id'];
        if ((string)$to_user_id === (string)$user_id || strcasecmp((string)$receiver['email'], (string)$sender['email']) === 0) {
            $pdo->rollBack();
            respondError('لا يمكنك التحويل لنفسك', 400);
        }

        $stmt = $pdo->prepare('SELECT * FROM users_wallet WHERE user_id = ? FOR UPDATE');
        $stmt->execute([$user_id]);
        $sender_wallet = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$sender_wallet) {
            $pdo->rollBack();
            respondError('المحفظة غير موجودة', 404);
        }

        $balanceField = $amount_type === 'money' ? 'balance' : 'points';
        if ((float)$sender_wallet[$balanceField] < $amount) {
            $pdo->rollBack();
            respondError('الرصيد غير كافٍ', 400);
        }

        $stmt = $pdo->prepare('SELECT user_id FROM users_wallet WHERE user_id = ? FOR UPDATE');
        $stmt->execute([$to_user_id]);
        if (!$stmt->fetch()) {
            $stmt = $pdo->prepare('INSERT INTO users_wallet (user_id, balance, points) VALUES (?, 0, 0)');
            $stmt->execute([$to_user_id]);
        }

        $stmt = $pdo->prepare("UPDATE users_wallet SET $balanceField = $balanceField - ? WHERE user_id = ?");
        $stmt->execute([$amount, $user_id]);
        $stmt = $pdo->prepare("UPDATE users_wallet SET $balanceField = $balanceField + ? WHERE user_id = ?");
        $stmt->execute([$amount, $to_user_id]);

        $stmt = $pdo->prepare("INSERT INTO transactions (user_id, from_user_id, to_user_id, from_email, to_email, transaction_type, amount_type, amount) VALUES (?, ?, ?, ?, ?, 'send', ?, ?)");
        $stmt->execute([$user_id, $user_id, $to_user_id, $sender['email'], $receiver['email'], $amount_type, $amount]);
        $stmt = $pdo->prepare("INSERT INTO transactions (user_id, from_user_id, to_user_id, from_email, to_email, transaction_type, amount_type, amount) VALUES (?, ?, ?, ?, ?, 'receive', ?, ?)");
        $stmt->execute([$to_user_id, $user_id, $to_user_id, $sender['email'], $receiver['email'], $amount_type, $amount]);

        $pdo->commit();
        respondJson(['success' => true, 'message' => 'تم التحويل بنجاح']);
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        error_log('wallet transfer error: ' . $e->getMessage());
        respondError('خطأ في التحويل', 500);
    }
}
