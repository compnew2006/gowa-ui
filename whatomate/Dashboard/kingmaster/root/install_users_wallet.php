<?php
/**
 * سكريبت إنشاء جدول users_wallet و transactions
 */

require_once 'config/database.php';

try {
    $conn = getDB();
    
    // إنشاء جدول users_wallet
    $walletSql = "CREATE TABLE IF NOT EXISTS users_wallet (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL UNIQUE,
        balance DECIMAL(10, 2) DEFAULT 0.00,
        points DECIMAL(10, 2) DEFAULT 0.00,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX idx_user_id (user_id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    $conn->exec($walletSql);
    echo "✅ تم إنشاء جدول users_wallet بنجاح!<br>";
    
    // إنشاء جدول transactions
    $transactionsSql = "CREATE TABLE IF NOT EXISTS transactions (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL,
        type ENUM('purchase_points', 'add_balance', 'deduct_balance', 'transfer', 'other') NOT NULL,
        amount DECIMAL(10, 2) NOT NULL,
        points DECIMAL(10, 2) DEFAULT NULL,
        description TEXT,
        balance_before DECIMAL(10, 2),
        balance_after DECIMAL(10, 2),
        points_before DECIMAL(10, 2),
        points_after DECIMAL(10, 2),
        status ENUM('pending', 'completed', 'failed', 'cancelled') DEFAULT 'completed',
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_user_id (user_id),
        INDEX idx_type (type),
        INDEX idx_created_at (created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    $conn->exec($transactionsSql);
    echo "✅ تم إنشاء جدول transactions بنجاح!<br>";
    
    // إدراج بيانات تجريبية
    $insertSql = "INSERT INTO users_wallet (user_id, balance, points) VALUES
        (1, 500.00, 100.00),
        (2, 250.00, 50.00),
        (3, 1000.00, 500.00)";
    
    $conn->exec($insertSql);
    echo "✅ تم إضافة البيانات التجريبية بنجاح!<br>";
    
    echo "🎉 الآن يمكنك استخدام نظام شحن النقاط<br>";
    echo "<a href='points.php' style='display: inline-block; margin-top: 20px; padding: 10px 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; text-decoration: none; border-radius: 8px; font-family: Arial;'>الانتقال إلى صفحة النقاط</a>";
    
} catch (PDOException $e) {
    echo "❌ خطأ: " . $e->getMessage();
}
?>
