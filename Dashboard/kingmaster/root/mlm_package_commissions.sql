-- جدول إعدادات عمولات الباقات
CREATE TABLE IF NOT EXISTS `package_mlm_settings` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `package_id` INT NOT NULL,
  `max_levels` INT NOT NULL DEFAULT 5,
  `direct_commission_amount` DECIMAL(10,2) NOT NULL DEFAULT 0.00,
  `level_1_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_2_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_3_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_4_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_5_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_6_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_7_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_8_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_9_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `level_10_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
  `is_active` TINYINT(1) DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `unique_package` (`package_id`),
  FOREIGN KEY (`package_id`) REFERENCES `packages`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- إضافة إعدادات افتراضية لكل باقة موجودة
INSERT INTO `package_mlm_settings` 
  (`package_id`, `max_levels`, `direct_commission_amount`, `level_1_percentage`, `level_2_percentage`, `level_3_percentage`, `level_4_percentage`, `level_5_percentage`)
SELECT 
  id as package_id,
  5 as max_levels,
  50.00 as direct_commission_amount,
  5.00 as level_1_percentage,
  3.00 as level_2_percentage,
  2.00 as level_3_percentage,
  1.00 as level_4_percentage,
  1.00 as level_5_percentage
FROM packages
WHERE NOT EXISTS (
  SELECT 1 FROM package_mlm_settings WHERE package_mlm_settings.package_id = packages.id
);
