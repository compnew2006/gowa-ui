-- إنشاء جداول نظام MLM

-- جدول المستخدمين
CREATE TABLE IF NOT EXISTS `mlm_users` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `username` VARCHAR(50) NOT NULL UNIQUE,
  `email` VARCHAR(100) NOT NULL,
  `full_name` VARCHAR(100) NOT NULL,
  `phone` VARCHAR(20) DEFAULT NULL,
  `parent_id` INT DEFAULT NULL,
  `referral_code` VARCHAR(20) NOT NULL UNIQUE,
  `join_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `total_referrals` INT DEFAULT 0,
  `direct_referrals` INT DEFAULT 0,
  `total_earnings` DECIMAL(10,2) DEFAULT 0.00,
  `status` ENUM('active', 'inactive', 'suspended') DEFAULT 'active',
  INDEX `idx_parent_id` (`parent_id`),
  INDEX `idx_referral_code` (`referral_code`),
  FOREIGN KEY (`parent_id`) REFERENCES `mlm_users`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- جدول إعدادات MLM
CREATE TABLE IF NOT EXISTS `mlm_settings` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `level_number` INT NOT NULL,
  `level_name` VARCHAR(50) NOT NULL,
  `direct_commission_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `indirect_commission_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `is_active` TINYINT(1) DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `unique_level` (`level_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- جدول العمولات
CREATE TABLE IF NOT EXISTS `mlm_commissions` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `user_id` INT NOT NULL,
  `from_user_id` INT NOT NULL,
  `commission_type` ENUM('direct', 'indirect') NOT NULL,
  `level_number` INT NOT NULL,
  `amount` DECIMAL(10,2) NOT NULL,
  `percentage` DECIMAL(5,2) NOT NULL,
  `sale_amount` DECIMAL(10,2) NOT NULL,
  `description` VARCHAR(255) DEFAULT NULL,
  `status` ENUM('pending', 'approved', 'paid', 'cancelled') DEFAULT 'approved',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_from_user_id` (`from_user_id`),
  INDEX `idx_created_at` (`created_at`),
  FOREIGN KEY (`user_id`) REFERENCES `mlm_users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`from_user_id`) REFERENCES `mlm_users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- إدخال بيانات تجريبية للإعدادات (5 مستويات)
INSERT INTO `mlm_settings` (`level_number`, `level_name`, `direct_commission_percentage`, `indirect_commission_percentage`, `is_active`)
VALUES
  (1, 'المستوى الأول', 10.00, 10.00, 1),
  (2, 'المستوى الثاني', 5.00, 5.00, 1),
  (3, 'المستوى الثالث', 3.00, 3.00, 1),
  (4, 'المستوى الرابع', 2.00, 2.00, 1),
  (5, 'المستوى الخامس', 1.00, 1.00, 1)
ON DUPLICATE KEY UPDATE 
  `level_name` = VALUES(`level_name`),
  `direct_commission_percentage` = VALUES(`direct_commission_percentage`),
  `indirect_commission_percentage` = VALUES(`indirect_commission_percentage`);

-- إدخال بيانات تجريبية للمستخدمين
INSERT INTO `mlm_users` (`username`, `email`, `full_name`, `phone`, `parent_id`, `referral_code`, `total_referrals`, `direct_referrals`)
VALUES
  ('admin', 'admin@example.com', 'المدير العام', '0500000000', NULL, 'ADMIN001', 2, 2),
  ('user1', 'user1@example.com', 'محمد أحمد', '0501111111', 1, 'USER0001', 3, 3),
  ('user2', 'user2@example.com', 'علي حسن', '0502222222', 1, 'USER0002', 2, 2),
  ('user3', 'user3@example.com', 'فاطمة خالد', '0503333333', 2, 'USER0003', 2, 2),
  ('user4', 'user4@example.com', 'سارة عبدالله', '0504444444', 2, 'USER0004', 1, 1),
  ('user5', 'user5@example.com', 'عمر يوسف', '0505555555', 2, 'USER0005', 0, 0),
  ('user6', 'user6@example.com', 'نورا محمود', '0506666666', 3, 'USER0006', 1, 1),
  ('user7', 'user7@example.com', 'خالد سعيد', '0507777777', 3, 'USER0007', 0, 0),
  ('user8', 'user8@example.com', 'ليلى إبراهيم', '0508888888', 4, 'USER0008', 0, 0),
  ('user9', 'user9@example.com', 'أحمد عمر', '0509999999', 6, 'USER0009', 0, 0)
ON DUPLICATE KEY UPDATE 
  `email` = VALUES(`email`),
  `full_name` = VALUES(`full_name`);
