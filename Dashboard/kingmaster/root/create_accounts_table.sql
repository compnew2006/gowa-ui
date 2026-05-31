-- جدول الحسابات على المنصات المختلفة
-- Accounts table for different platforms

USE kingmaster_packages;

CREATE TABLE IF NOT EXISTS `accounts` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `user_id` INT(11) NOT NULL,
  `platform` ENUM('facebook', 'whatsapp', 'instagram', 'telegram', 'email') NOT NULL,
  `login_method` VARCHAR(50) DEFAULT NULL COMMENT 'credentials, cookies, qr',
  `username` VARCHAR(255) DEFAULT NULL,
  `password` VARCHAR(255) DEFAULT NULL,
  `email` VARCHAR(255) DEFAULT NULL,
  `phone` VARCHAR(50) DEFAULT NULL,
  `cookies` TEXT DEFAULT NULL,
  `two_fa_code` VARCHAR(50) DEFAULT NULL,
  `telegram_code` VARCHAR(50) DEFAULT NULL,
  `status` ENUM('active', 'inactive', 'pending', 'error') DEFAULT 'active',
  `last_login` TIMESTAMP NULL DEFAULT NULL,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  KEY `platform` (`platform`),
  KEY `status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- عرض الجدول للتأكد
DESCRIBE accounts;
