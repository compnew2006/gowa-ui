<?php
require_once 'config/database.php';

try {
    $conn = getDB();
    
    // إنشاء جدول السحوبات
    $sql = "CREATE TABLE IF NOT EXISTS withdrawals (
        id INT(11) AUTO_INCREMENT PRIMARY KEY,
        user_id VARCHAR(255) NOT NULL,
        amount DECIMAL(10, 2) NOT NULL,
        withdrawal_type VARCHAR(50) NOT NULL,
        withdrawal_details TEXT,
        status ENUM('pending', 'approved', 'rejected') DEFAULT 'pending',
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX(user_id),
        INDEX(status),
        INDEX(created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    $conn->exec($sql);
    
    echo "✅ تم إنشاء جدول withdrawals بنجاح!";
    
} catch(PDOException $e) {
    echo "❌ خطأ: " . $e->getMessage();
}
?>
