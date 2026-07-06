<?php
require_once __DIR__ . '/../config/database.php';

$userId = requireAuthenticatedUser();
enforceRateLimit('coupon_api:' . $userId, 20, 300);

$action = $_POST['action'] ?? '';
if ($action !== 'redeem') {
    respondError('إجراء غير صالح', 400);
}

verifyCsrfToken();

try {
    redeemCoupon(getDB(), $userId);
} catch (Throwable $e) {
    error_log('coupon_api error: ' . $e->getMessage());
    respondError('تعذر تفعيل الكوبون', 500);
}

function redeemCoupon(PDO $pdo, $userId) {
    $code = strtoupper(trim((string)($_POST['code'] ?? '')));
    if (!preg_match('/^[A-Z0-9_-]{3,40}$/', $code)) {
        respondError('رمز الكوبون غير صالح', 422);
    }

    $pdo->beginTransaction();
    try {
        $stmt = $pdo->prepare('SELECT * FROM coupons WHERE code = ? FOR UPDATE');
        $stmt->execute([$code]);
        $coupon = $stmt->fetch(PDO::FETCH_ASSOC);

        if (!$coupon) {
            $pdo->rollBack();
            respondError('رمز الكوبون غير صحيح', 404);
        }

        $stmt = $pdo->prepare('SELECT id FROM user_coupons WHERE user_id = ? AND coupon_id = ? LIMIT 1');
        $stmt->execute([$userId, $coupon['id']]);
        if ($stmt->fetch()) {
            $pdo->rollBack();
            respondError('لقد استخدمت هذا الكوبون من قبل', 409);
        }

        if (!empty($coupon['expires_at']) && new DateTime() > new DateTime($coupon['expires_at'])) {
            $pdo->rollBack();
            respondError('هذا الكوبون منتهي الصلاحية', 410);
        }

        if ((int)$coupon['used_count'] >= (int)$coupon['uses_limit']) {
            $pdo->rollBack();
            respondError('تم انتهاء هذا الكوبون', 409);
        }

        $stmt = $pdo->prepare('INSERT INTO user_coupons (user_id, coupon_id, coupon_code) VALUES (?, ?, ?)');
        $stmt->execute([$userId, $coupon['id'], $code]);

        $stmt = $pdo->prepare('UPDATE coupons SET used_count = used_count + 1 WHERE id = ?');
        $stmt->execute([$coupon['id']]);

        if (($coupon['discount_type'] ?? '') === 'points') {
            $pointsToAdd = max(0, (int)$coupon['discount_value']);
            if ($pointsToAdd > 0) {
                applyCouponPoints($pdo, $userId, $pointsToAdd);
            }
        }

        $pdo->commit();
        respondJson([
            'success' => true,
            'message' => 'تم تفعيل الكوبون بنجاح',
            'coupon' => [
                'code' => $coupon['code'],
                'discount_type' => $coupon['discount_type'],
                'discount_value' => $coupon['discount_value'],
                'remaining_uses' => max(0, (int)$coupon['uses_limit'] - (int)$coupon['used_count'] - 1),
            ],
        ]);
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        throw $e;
    }
}

function applyCouponPoints(PDO $pdo, $userId, $pointsToAdd) {
    $stmt = $pdo->prepare('SELECT email FROM users WHERE user_id = ? OR id = ? LIMIT 1');
    $stmt->execute([$userId, $userId]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$user) {
        respondError('المستخدم غير موجود', 404);
    }

    $stmt = $pdo->prepare('SELECT id FROM users_wallet WHERE user_id = ? LIMIT 1');
    $stmt->execute([$userId]);
    if ($stmt->fetch()) {
        $stmt = $pdo->prepare('UPDATE users_wallet SET points = points + ? WHERE user_id = ?');
        $stmt->execute([$pointsToAdd, $userId]);
    } else {
        $stmt = $pdo->prepare('INSERT INTO users_wallet (user_id, points, balance) VALUES (?, ?, 0)');
        $stmt->execute([$userId, $pointsToAdd]);
    }

    $stmt = $pdo->prepare(
        "INSERT INTO transactions
         (user_id, from_user_id, to_user_id, from_email, to_email, transaction_type, amount_type, amount)
         VALUES (?, 0, ?, ?, ?, 'receive', 'points', ?)"
    );
    $stmt->execute([$userId, $userId, 'النظام', $user['email'], $pointsToAdd]);
}
