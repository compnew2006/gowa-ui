-- Kingmaster Schema Only
-- 2026-05-26 21:25:16
-- Tables: 76

CREATE DATABASE IF NOT EXISTS `kingmaster` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `kingmaster`;

-- ------------------------------------------
-- Table: accounts (596 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `accounts`;
CREATE TABLE `accounts` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) DEFAULT '1',
  `name` varchar(255) NOT NULL DEFAULT 'name account',
  `account_uid` varchar(64) NOT NULL DEFAULT '100000033',
  `channel` enum('facebook','whatsapp','instagram','telegram','email','sms','tiktok','linkedin') NOT NULL,
  `status` enum('active','closed','inactive') NOT NULL DEFAULT 'inactive',
  `method` enum('cookies','data','qr','') NOT NULL,
  `cookies_text` longtext DEFAULT NULL,
  `data` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`data`)),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `channel` (`channel`),
  KEY `status` (`status`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4451 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: announcements (0 rows)
-- ------------------------------------------
SET FOREIGN_KEY_CHECKS = 0;
DROP TABLE IF EXISTS `announcements`;
SET FOREIGN_KEY_CHECKS = 1;
CREATE TABLE `announcements` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `message` text NOT NULL,
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: campaigns (5011 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `campaigns`;
CREATE TABLE `campaigns` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(50) NOT NULL,
  `campaign_id` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `status` enum('pending','running','paused','stopped','finished') DEFAULT 'pending',
  `paltform` varchar(255) DEFAULT NULL,
  `tool` varchar(255) DEFAULT NULL,
  `type_tools` varchar(255) NOT NULL,
  `count` int(11) DEFAULT 0,
  `true_count` varchar(50) NOT NULL DEFAULT '0',
  `false_count` varchar(50) NOT NULL DEFAULT '0',
  `token` text DEFAULT NULL,
  `page_url` varchar(255) DEFAULT NULL,
  `range` varchar(255) NOT NULL DEFAULT '0',
  `interval` int(255) DEFAULT NULL,
  `content_id` varchar(255) DEFAULT NULL,
  `can_like` varchar(50) DEFAULT NULL,
  `pram1` text DEFAULT NULL,
  `pram2` varchar(255) DEFAULT NULL,
  `pram3` varchar(255) DEFAULT NULL,
  `pram4` varchar(255) DEFAULT NULL,
  `contact` varchar(255) DEFAULT NULL,
  `speed` varchar(50) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `campaign_uid` (`campaign_id`),
  KEY `status` (`status`),
  KEY `created_at` (`created_at`),
  KEY `updated_at` (`updated_at`),
  KEY `user_id` (`user_id`),
  KEY `paltform` (`paltform`,`tool`,`type_tools`),
  KEY `paltform_2` (`paltform`,`tool`,`type_tools`),
  KEY `count` (`count`,`true_count`,`false_count`),
  KEY `name` (`name`),
  KEY `page_url` (`page_url`),
  KEY `token` (`token`(768))
) ENGINE=InnoDB AUTO_INCREMENT=12903 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: commission_wallets (475 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `commission_wallets`;
CREATE TABLE `commission_wallets` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `commission` varchar(255) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`,`commission`,`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=476 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: contacts (2720 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `contacts`;
CREATE TABLE `contacts` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `platform` varchar(50) NOT NULL,
  `type` enum('csv','text') NOT NULL,
  `count` int(11) NOT NULL DEFAULT 0,
  `data` longtext NOT NULL COMMENT 'JSON data of contacts',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=5158 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: content (843 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `content`;
CREATE TABLE `content` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL DEFAULT '1',
  `name` varchar(255) NOT NULL,
  `text` text NOT NULL,
  `char_count` int(11) NOT NULL DEFAULT 0,
  `word_count` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1215 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: content_messages (4 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `content_messages`;
CREATE TABLE `content_messages` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `category` varchar(100) DEFAULT '',
  `tags` text DEFAULT '',
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: conversations (5 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `conversations`;
CREATE TABLE `conversations` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user1_id` varchar(50) NOT NULL,
  `user2_id` varchar(50) NOT NULL,
  `last_message` text DEFAULT NULL,
  `last_message_time` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_conversation` (`user1_id`,`user2_id`),
  KEY `user1_id` (`user1_id`),
  KEY `user2_id` (`user2_id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: coupons (7 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `coupons`;
CREATE TABLE `coupons` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `code` varchar(50) NOT NULL,
  `discount_type` varchar(200) DEFAULT NULL,
  `discount_value` int(200) NOT NULL DEFAULT 0,
  `uses_limit` int(11) DEFAULT 1,
  `used_count` int(11) DEFAULT 0,
  `expires_at` datetime NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `code` (`code`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: data_fb (87542089 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `data_fb`;
CREATE TABLE `data_fb` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `fb_id` varchar(255) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `mobile_phone` varchar(255) NOT NULL,
  `gender` varchar(255) NOT NULL,
  `birthday` varchar(255) NOT NULL,
  `location` varchar(255) DEFAULT NULL,
  `relationship` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `work` varchar(255) DEFAULT NULL,
  `education` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fb_id` (`fb_id`)
) ENGINE=InnoDB AUTO_INCREMENT=87542090 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: db_camp (202874 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `db_camp`;
CREATE TABLE `db_camp` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `fb_id` varchar(255) DEFAULT NULL,
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `gender` varchar(255) DEFAULT NULL,
  `birthday` varchar(255) DEFAULT NULL,
  `location` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `relashan` longtext DEFAULT NULL,
  `email` longtext DEFAULT NULL,
  `work` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `educ` longtext DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`)
) ENGINE=InnoDB AUTO_INCREMENT=202875 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: fb_page (1967 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `fb_page`;
CREATE TABLE `fb_page` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `facebook_id` varchar(255) DEFAULT NULL,
  `user_id` varchar(255) DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `token` varchar(255) DEFAULT NULL,
  `id_page` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `facebook_id` (`facebook_id`),
  KEY `user_id` (`user_id`),
  KEY `id_page` (`id_page`)
) ENGINE=InnoDB AUTO_INCREMENT=18967 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci;

-- ------------------------------------------
-- Table: fb_serch (38394474 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `fb_serch`;
CREATE TABLE `fb_serch` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) DEFAULT NULL,
  `page_id` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `followers_count` longtext DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`)
) ENGINE=InnoDB AUTO_INCREMENT=38398226 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: files (644 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `files`;
CREATE TABLE `files` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `original_name` varchar(255) NOT NULL,
  `file_path` varchar(500) NOT NULL,
  `file_type` varchar(50) NOT NULL COMMENT 'image, video, pdf',
  `mime_type` varchar(100) NOT NULL,
  `file_size` bigint(20) NOT NULL COMMENT 'بالبايت',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1021 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: filter_wa (304815 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `filter_wa`;
CREATE TABLE `filter_wa` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ty` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`),
  KEY `phone` (`phone`),
  KEY `ty` (`ty`),
  FULLTEXT KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=319353 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: gb_wa (1473 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `gb_wa`;
CREATE TABLE `gb_wa` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `gb_id` varchar(255) DEFAULT NULL,
  `participantsCount` varchar(50) DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`),
  KEY `participantsCount` (`participantsCount`),
  FULLTEXT KEY `name` (`name`),
  FULLTEXT KEY `gb_id` (`gb_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1474 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: groups_list (5063 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `groups_list`;
CREATE TABLE `groups_list` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `groupId` int(11) NOT NULL,
  `groupName` varchar(255) NOT NULL,
  `groupLink` text DEFAULT NULL,
  `country` varchar(100) DEFAULT NULL,
  `Language` varchar(100) DEFAULT NULL,
  `groupDesc` text DEFAULT NULL,
  `GroupImage` text DEFAULT NULL,
  `categoryName` varchar(255) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  FULLTEXT KEY `groupName` (`groupName`)
) ENGINE=InnoDB AUTO_INCREMENT=5064 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: ig_dms (37988 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_dms`;
CREATE TABLE `ig_dms` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `ig_user_id` varchar(255) NOT NULL,
  `username` varchar(255) DEFAULT NULL,
  `full_name` varchar(255) DEFAULT NULL,
  `last_message_date` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_user_campaign` (`campaign_id`,`ig_user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=37992 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: ig_follow (5237 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_follow`;
CREATE TABLE `ig_follow` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `ig_user_id` varchar(255) NOT NULL,
  `username` varchar(255) DEFAULT NULL,
  `full_name` varchar(255) DEFAULT NULL,
  `extract_type` varchar(50) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_follower_camp` (`campaign_id`,`ig_user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=5882 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: ig_msg (8996 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_msg`;
CREATE TABLE `ig_msg` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `comment` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `unreadCount` varchar(255) NOT NULL DEFAULT '0',
  `isMe` varchar(50) DEFAULT NULL,
  `comment_date` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`,`phone`),
  FULLTEXT KEY `name` (`name`,`comment`)
) ENGINE=InnoDB AUTO_INCREMENT=9650 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: ig_post (3805 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_post`;
CREATE TABLE `ig_post` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(50) DEFAULT NULL,
  `shortcode` varchar(50) NOT NULL,
  `post_type` varchar(50) DEFAULT 'Image',
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `likes_count` int(11) DEFAULT 0,
  `comments_count` int(11) DEFAULT 0,
  `post_date` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3806 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: ig_retarget (3105 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_retarget`;
CREATE TABLE `ig_retarget` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) DEFAULT NULL,
  `ig_user_id` varchar(255) DEFAULT NULL,
  `status` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3106 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci;

-- ------------------------------------------
-- Table: ig_search_hashtags (55 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_search_hashtags`;
CREATE TABLE `ig_search_hashtags` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `keyword` varchar(255) DEFAULT NULL,
  `hashtag_id` varchar(255) NOT NULL,
  `hashtag_name` varchar(255) DEFAULT NULL,
  `hashtag_url` varchar(255) DEFAULT NULL,
  `media_count` varchar(50) DEFAULT '0',
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_search_tag` (`campaign_id`,`hashtag_name`)
) ENGINE=InnoDB AUTO_INCREMENT=56 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: ig_search_locations (60 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_search_locations`;
CREATE TABLE `ig_search_locations` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `keyword` varchar(255) DEFAULT NULL,
  `location_id` varchar(255) NOT NULL,
  `location_name` varchar(255) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `lat` varchar(50) DEFAULT NULL,
  `lng` varchar(50) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_search_loc` (`campaign_id`,`location_id`)
) ENGINE=InnoDB AUTO_INCREMENT=61 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: ig_search_users (253 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ig_search_users`;
CREATE TABLE `ig_search_users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `keyword` varchar(255) DEFAULT NULL,
  `ig_user_id` varchar(255) NOT NULL,
  `username` varchar(255) DEFAULT NULL,
  `full_name` varchar(255) DEFAULT NULL,
  `profile_url` varchar(255) DEFAULT NULL,
  `is_private` varchar(10) DEFAULT '0',
  `is_verified` varchar(10) DEFAULT '0',
  `search_type` varchar(50) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_search_user` (`campaign_id`,`ig_user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=254 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: logs (7069 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `logs`;
CREATE TABLE `logs` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `action` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7070 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: media_files (4 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `media_files`;
CREATE TABLE `media_files` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) DEFAULT '',
  `original_name` varchar(255) NOT NULL,
  `filename` varchar(255) NOT NULL,
  `url` varchar(255) NOT NULL,
  `mime_type` varchar(100) NOT NULL,
  `type` enum('image','video','pdf','file') DEFAULT 'file',
  `size` bigint(20) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `type` (`type`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: messages (17 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `messages`;
CREATE TABLE `messages` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `conversation_id` int(11) NOT NULL,
  `sender_id` varchar(50) NOT NULL,
  `receiver_id` varchar(50) NOT NULL,
  `message` text NOT NULL,
  `image_path` varchar(500) DEFAULT NULL,
  `is_read` tinyint(1) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `conversation_id` (`conversation_id`),
  KEY `sender_id` (`sender_id`),
  KEY `receiver_id` (`receiver_id`),
  KEY `is_read` (`is_read`),
  KEY `created_at` (`created_at`),
  KEY `idx_image_path` (`image_path`),
  CONSTRAINT `messages_ibfk_1` FOREIGN KEY (`conversation_id`) REFERENCES `conversations` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: messenger_templates (4 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `messenger_templates`;
CREATE TABLE `messenger_templates` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `type` varchar(50) DEFAULT 'generic',
  `channel` varchar(20) DEFAULT 'facebook',
  `payload` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`payload`)),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `type` (`type`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: mlm_commissions (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `mlm_commissions`;
CREATE TABLE `mlm_commissions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `from_user_id` varchar(255) NOT NULL,
  `commission_type` enum('direct','indirect') NOT NULL,
  `level_number` int(11) NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `percentage` decimal(5,2) NOT NULL,
  `sale_amount` decimal(10,2) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `status` enum('pending','approved','paid','cancelled') DEFAULT 'approved',
  `created_at` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_from_user_id` (`from_user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=20 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: mlm_referrals (33 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `mlm_referrals`;
CREATE TABLE `mlm_referrals` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL COMMENT '???????? ???????????????? ???????? ???? ??????????',
  `referrer_id` varchar(255) NOT NULL COMMENT '???????? ???????????????? ???????? ????????',
  `referral_code` varchar(50) NOT NULL COMMENT '?????? ?????????????? ????????????????',
  `level` int(11) DEFAULT 1 COMMENT '?????????????? ???? ???????????? (1-4)',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_user` (`user_id`),
  KEY `idx_referrer` (`referrer_id`),
  KEY `idx_user` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=39 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='???????? ???????????????? ?????????????? ????????????????';

-- ------------------------------------------
-- Table: mlm_settings (5 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `mlm_settings`;
CREATE TABLE `mlm_settings` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `level_number` int(11) NOT NULL,
  `level_name` varchar(50) NOT NULL,
  `direct_commission_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `indirect_commission_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_level` (`level_number`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: mlm_users (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `mlm_users`;
CREATE TABLE `mlm_users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL,
  `email` varchar(100) NOT NULL,
  `full_name` varchar(100) NOT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `parent_id` int(11) DEFAULT NULL,
  `referral_code` varchar(20) NOT NULL,
  `join_date` datetime DEFAULT current_timestamp(),
  `total_referrals` int(11) DEFAULT 0,
  `direct_referrals` int(11) DEFAULT 0,
  `total_earnings` decimal(10,2) DEFAULT 0.00,
  `status` enum('active','inactive','suspended') DEFAULT 'active',
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`),
  UNIQUE KEY `referral_code` (`referral_code`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_referral_code` (`referral_code`),
  CONSTRAINT `mlm_users_ibfk_1` FOREIGN KEY (`parent_id`) REFERENCES `mlm_users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=52 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: notifications (19863 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `notifications`;
CREATE TABLE `notifications` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(50) NOT NULL,
  `title` varchar(255) NOT NULL,
  `message` text NOT NULL,
  `type` varchar(50) DEFAULT 'info',
  `is_read` tinyint(1) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  KEY `is_read` (`is_read`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=19864 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: order_status_history (35 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `order_status_history`;
CREATE TABLE `order_status_history` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `order_id` int(11) NOT NULL,
  `status` enum('pending','verified','preparing','shipped','delivered','rejected') NOT NULL,
  `notes` text DEFAULT NULL,
  `created_by` varchar(100) DEFAULT 'النظام',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_status` (`status`),
  CONSTRAINT `order_status_history_ibfk_1` FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=36 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: orders (10 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `orders`;
CREATE TABLE `orders` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `product_id` int(11) NOT NULL,
  `product_name` varchar(255) NOT NULL,
  `quantity` int(11) NOT NULL,
  `color` varchar(50) DEFAULT NULL,
  `size` varchar(50) DEFAULT NULL,
  `price` decimal(10,2) NOT NULL,
  `total` decimal(10,2) NOT NULL,
  `commission` decimal(10,2) DEFAULT 0.00,
  `address` text DEFAULT NULL,
  `phone` varchar(50) DEFAULT NULL,
  `payment_method` varchar(50) DEFAULT 'balance',
  `status` enum('pending','approved','preparing','shipping','delivered','completed','rejected','cancelled') DEFAULT 'pending',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: otp_wall (50 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `otp_wall`;
CREATE TABLE `otp_wall` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `otp` int(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`,`otp`)
) ENGINE=InnoDB AUTO_INCREMENT=94 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: package_mlm_settings (3 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `package_mlm_settings`;
CREATE TABLE `package_mlm_settings` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `package_id` int(11) NOT NULL,
  `max_levels` int(11) NOT NULL DEFAULT 5,
  `direct_commission_amount` decimal(10,2) NOT NULL DEFAULT 0.00,
  `level_1_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_2_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_3_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_4_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_5_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_6_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_7_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_8_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_9_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `level_10_percentage` decimal(5,2) NOT NULL DEFAULT 0.00,
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_package` (`package_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: packages (5 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `packages`;
CREATE TABLE `packages` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `features` text DEFAULT NULL COMMENT 'JSON array of features',
  `price` decimal(10,2) NOT NULL,
  `original_price` decimal(10,2) DEFAULT NULL COMMENT 'السعر قبل الخصم',
  `has_discount` tinyint(1) DEFAULT 0,
  `is_popular` tinyint(1) DEFAULT 0,
  `platforms` text DEFAULT NULL COMMENT 'JSON array: facebook, whatsapp, telegram, instagram, email, business',
  `accounts_count` int(11) DEFAULT 0,
  `messages_count` int(11) DEFAULT 0,
  `points` int(11) DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `currency` varchar(200) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `points` (`points`),
  KEY `messages_count` (`messages_count`),
  KEY `platforms` (`platforms`(768)),
  KEY `price` (`price`),
  KEY `accounts_count` (`accounts_count`,`messages_count`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: point_use (2 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `point_use`;
CREATE TABLE `point_use` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) DEFAULT NULL,
  `count` varchar(255) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: points_packages (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `points_packages`;
CREATE TABLE `points_packages` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `points_count` int(11) NOT NULL,
  `price` decimal(10,2) NOT NULL,
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: points_pricing (4 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `points_pricing`;
CREATE TABLE `points_pricing` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `points` int(11) NOT NULL,
  `price` decimal(10,2) NOT NULL,
  `name` varchar(100) NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_points` (`points`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: post_ratings (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `post_ratings`;
CREATE TABLE `post_ratings` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `post_id` int(11) NOT NULL,
  `user_id` varchar(255) NOT NULL,
  `rating` tinyint(4) NOT NULL CHECK (`rating` >= 1 and `rating` <= 5),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_user_post` (`user_id`,`post_id`),
  KEY `idx_post_id` (`post_id`),
  KEY `idx_user_id` (`user_id`),
  CONSTRAINT `post_ratings_ibfk_1` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: posts (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `posts`;
CREATE TABLE `posts` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `typs` enum('New Feature','System Update','Maintenance') NOT NULL DEFAULT 'New Feature',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: product_colors (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `product_colors`;
CREATE TABLE `product_colors` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `product_id` int(11) NOT NULL,
  `color_name` varchar(50) DEFAULT NULL,
  `color_hex` varchar(7) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `product_id` (`product_id`),
  CONSTRAINT `product_colors_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: product_sizes (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `product_sizes`;
CREATE TABLE `product_sizes` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `product_id` int(11) NOT NULL,
  `size_name` varchar(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `product_id` (`product_id`),
  CONSTRAINT `product_sizes_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=23 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: products (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `products`;
CREATE TABLE `products` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `image` varchar(255) DEFAULT NULL,
  `price` decimal(10,2) NOT NULL,
  `commission` decimal(10,2) DEFAULT 0.00,
  `stock` int(11) DEFAULT 0,
  `colors` text DEFAULT NULL,
  `sizes` text DEFAULT NULL,
  `is_digital` tinyint(1) DEFAULT 0,
  `category` varchar(100) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `discount_percentage` decimal(5,2) DEFAULT 0.00,
  `stock_quantity` int(11) DEFAULT 0,
  `rating` decimal(3,2) DEFAULT 0.00,
  `reviews_count` int(11) DEFAULT 0,
  `image_url` varchar(500) DEFAULT NULL,
  `is_new` tinyint(1) DEFAULT 0,
  `is_featured` tinyint(1) DEFAULT 0,
  `status` enum('active','inactive','out_of_stock') DEFAULT 'active',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: pvsettigs (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `pvsettigs`;
CREATE TABLE `pvsettigs` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `from` int(50) NOT NULL,
  `to` int(50) NOT NULL,
  `count` int(50) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `from` (`from`,`to`,`count`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: rb_wa (752006 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `rb_wa`;
CREATE TABLE `rb_wa` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `st` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=752007 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci;

-- ------------------------------------------
-- Table: ref (4 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `ref`;
CREATE TABLE `ref` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(50) NOT NULL,
  `ref_code` varchar(50) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`,`ref_code`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: representative (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `representative`;
CREATE TABLE `representative` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: retarget_rep (52008804 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `retarget_rep`;
CREATE TABLE `retarget_rep` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `st` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`)
) ENGINE=InnoDB AUTO_INCREMENT=52008805 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci;

-- ------------------------------------------
-- Table: sales_customers (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `sales_customers`;
CREATE TABLE `sales_customers` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `phone` varchar(20) NOT NULL,
  `email` varchar(255) DEFAULT NULL,
  `points` int(11) NOT NULL DEFAULT 0,
  `purchase_amount` decimal(10,2) NOT NULL DEFAULT 0.00,
  `expiry_date` date DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_phone` (`phone`),
  KEY `idx_expiry_date` (`expiry_date`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: sales_target (3 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `sales_target`;
CREATE TABLE `sales_target` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `target_amount` decimal(10,2) NOT NULL DEFAULT 0.00,
  `current_amount` decimal(10,2) NOT NULL DEFAULT 0.00,
  `bonus_amount` decimal(10,2) NOT NULL DEFAULT 0.00 COMMENT '???????????? ?????? ?????????? ?????????????? (50% ????????)',
  `target_month` varchar(7) NOT NULL COMMENT 'YYYY-MM',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_target_month` (`target_month`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: sending_settings (351 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `sending_settings`;
CREATE TABLE `sending_settings` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(50) NOT NULL,
  `settings_name` varchar(255) NOT NULL,
  `platform` enum('facebook','whatsapp','instagram','telegram','email') NOT NULL,
  `interval_from` int(11) NOT NULL,
  `interval_to` int(11) NOT NULL,
  `protection_enabled` tinyint(1) DEFAULT 0,
  `msg_count` int(11) DEFAULT NULL,
  `protection_interval_from` int(11) DEFAULT NULL,
  `protection_interval_to` int(11) DEFAULT NULL,
  `blacklist_enabled` tinyint(1) DEFAULT 0,
  `blacklist` text DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_platform` (`platform`)
) ENGINE=InnoDB AUTO_INCREMENT=470 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: syswalt (13 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `syswalt`;
CREATE TABLE `syswalt` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `price` varchar(255) NOT NULL,
  `typs` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `price` (`price`,`typs`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: tools (2 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `tools`;
CREATE TABLE `tools` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `status` enum('working','not_working') NOT NULL DEFAULT 'working',
  `visible` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `status` (`status`),
  KEY `visible` (`visible`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: transactions (296 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `transactions`;
CREATE TABLE `transactions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `from_user_id` varchar(255) DEFAULT NULL,
  `to_user_id` varchar(255) DEFAULT NULL,
  `from_email` varchar(255) DEFAULT NULL,
  `to_email` varchar(255) DEFAULT NULL,
  `transaction_type` enum('send','receive') NOT NULL,
  `amount_type` enum('money','points') NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  KEY `from_user_id` (`from_user_id`),
  KEY `to_user_id` (`to_user_id`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=348 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: user_announcement_views (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `user_announcement_views`;
CREATE TABLE `user_announcement_views` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(55) NOT NULL,
  `announcement_id` int(11) NOT NULL,
  `viewed_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_announcement` (`user_id`,`announcement_id`),
  KEY `announcement_id` (`announcement_id`),
  CONSTRAINT `user_announcement_views_ibfk_1` FOREIGN KEY (`announcement_id`) REFERENCES `announcements` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=625 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: user_coupons (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `user_coupons`;
CREATE TABLE `user_coupons` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `coupon_id` int(11) NOT NULL,
  `coupon_code` varchar(50) NOT NULL,
  `used_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_coupon_unique` (`user_id`,`coupon_id`),
  KEY `user_id` (`user_id`),
  KEY `coupon_id` (`coupon_id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: user_subscriptions (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `user_subscriptions`;
CREATE TABLE `user_subscriptions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `package_id` int(11) NOT NULL,
  `start_date` date NOT NULL,
  `end_date` date NOT NULL,
  `status` enum('active','expired','cancelled') DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  KEY `package_id` (`package_id`),
  CONSTRAINT `user_subscriptions_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `user_subscriptions_ibfk_2` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: users (381 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(50) NOT NULL,
  `first_name` varchar(100) NOT NULL,
  `last_name` varchar(100) NOT NULL,
  `email` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `phone` varchar(20) NOT NULL,
  `timezone` varchar(100) DEFAULT 'UTC',
  `job` varchar(150) DEFAULT NULL,
  `birth_date` date DEFAULT NULL,
  `otp` varchar(6) DEFAULT NULL,
  `otp_created_at` datetime DEFAULT NULL,
  `is_verified` tinyint(1) DEFAULT 0,
  `is_admin` tinyint(1) DEFAULT 0,
  `package` int(11) DEFAULT 8,
  `points` int(11) DEFAULT 100,
  `msg_count` varchar(50) NOT NULL DEFAULT '0',
  `account_count` varchar(50) NOT NULL DEFAULT '0',
  `expiry_date` datetime DEFAULT (current_timestamp() + interval 48 hour),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `img` varchar(255) NOT NULL DEFAULT '0',
  `referrer_id` varchar(255) DEFAULT NULL COMMENT '???????? ???????????????? ???????? ?????? ????????????????',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_user_id` (`user_id`),
  UNIQUE KEY `unique_email` (`email`),
  UNIQUE KEY `unique_phone` (`phone`),
  KEY `idx_is_verified` (`is_verified`),
  KEY `idx_referrer` (`referrer_id`)
) ENGINE=InnoDB AUTO_INCREMENT=482 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: users_wallet (480 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `users_wallet`;
CREATE TABLE `users_wallet` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) DEFAULT NULL,
  `referrer_id` int(11) DEFAULT NULL,
  `balance` decimal(10,2) NOT NULL DEFAULT 0.00,
  `points` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=505 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wa_contacts (146066 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wa_contacts`;
CREATE TABLE `wa_contacts` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `pushname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`),
  KEY `phone` (`phone`,`name`,`pushname`),
  FULLTEXT KEY `name` (`name`,`pushname`)
) ENGINE=InnoDB AUTO_INCREMENT=146067 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wa_conv (2 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wa_conv`;
CREATE TABLE `wa_conv` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) DEFAULT NULL,
  `session_id` varchar(255) DEFAULT NULL,
  `txt` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `froms` varchar(255) DEFAULT NULL,
  `tos` varchar(255) DEFAULT NULL,
  `from_me` varchar(50) NOT NULL DEFAULT 'false',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`,`session_id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wa_flows (70 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wa_flows`;
CREATE TABLE `wa_flows` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `account_uid` varchar(255) NOT NULL,
  `flow_name` varchar(255) NOT NULL,
  `config` longtext DEFAULT NULL,
  `active` tinyint(1) DEFAULT 1,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=171 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: wa_members_gb (28799 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wa_members_gb`;
CREATE TABLE `wa_members_gb` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `pushname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`),
  KEY `phone` (`phone`),
  FULLTEXT KEY `name` (`name`),
  FULLTEXT KEY `pushname` (`pushname`)
) ENGINE=InnoDB AUTO_INCREMENT=28800 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wa_msg (67873 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wa_msg`;
CREATE TABLE `wa_msg` (
  `id` int(255) NOT NULL AUTO_INCREMENT,
  `campaign_id` varchar(255) DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `pushname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `unreadCount` varchar(255) NOT NULL DEFAULT '0',
  `isMe` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `campaign_id` (`campaign_id`,`phone`),
  FULLTEXT KEY `name` (`name`,`pushname`)
) ENGINE=InnoDB AUTO_INCREMENT=67874 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wallet_transactions (0 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wallet_transactions`;
CREATE TABLE `wallet_transactions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `transaction_type` enum('points','money') NOT NULL,
  `sender_id` int(11) NOT NULL,
  `sender_name` varchar(100) NOT NULL,
  `receiver_id` int(11) NOT NULL,
  `receiver_name` varchar(100) NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `note` text DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_sender` (`sender_id`),
  KEY `idx_receiver` (`receiver_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wallets (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wallets`;
CREATE TABLE `wallets` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `user_name` varchar(100) NOT NULL,
  `points_balance` decimal(10,2) DEFAULT 0.00,
  `money_balance` decimal(10,2) DEFAULT 0.00,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: whatsapp_lists (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `whatsapp_lists`;
CREATE TABLE `whatsapp_lists` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `base_url` varchar(255) NOT NULL,
  `api_key` varchar(128) NOT NULL,
  `instance_id` varchar(128) NOT NULL,
  `phone` varchar(32) NOT NULL,
  `description` text NOT NULL,
  `button_text` varchar(100) NOT NULL,
  `sections` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`sections`)),
  `url` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `created_at` (`created_at`),
  KEY `phone` (`phone`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: whatsapp_polls (1 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `whatsapp_polls`;
CREATE TABLE `whatsapp_polls` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `base_url` varchar(255) NOT NULL,
  `api_key` varchar(128) NOT NULL,
  `instance_id` varchar(128) NOT NULL,
  `phone` varchar(32) NOT NULL,
  `poll_name` varchar(255) NOT NULL,
  `choices` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`choices`)),
  `url` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `created_at` (`created_at`),
  KEY `phone` (`phone`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ------------------------------------------
-- Table: withdrawals (4 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `withdrawals`;
CREATE TABLE `withdrawals` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(255) NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `withdrawal_type` varchar(50) NOT NULL,
  `withdrawal_details` text DEFAULT NULL,
  `status` enum('pending','approved','rejected') DEFAULT 'pending',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  KEY `status` (`status`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wpp_events (7357840 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wpp_events`;
CREATE TABLE `wpp_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `session` varchar(191) DEFAULT NULL,
  `event` varchar(191) DEFAULT NULL,
  `type` varchar(191) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `payload` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`payload`)),
  PRIMARY KEY (`id`),
  KEY `idx_session` (`session`),
  KEY `idx_event` (`event`)
) ENGINE=InnoDB AUTO_INCREMENT=7357911 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wpp_messages (2246870 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wpp_messages`;
CREATE TABLE `wpp_messages` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `session` varchar(191) DEFAULT NULL,
  `direction` enum('in','out') DEFAULT NULL,
  `remote_jid` varchar(191) DEFAULT NULL,
  `participant` varchar(191) DEFAULT NULL,
  `msg_key_id` varchar(255) DEFAULT NULL,
  `timestamp` bigint(20) DEFAULT NULL,
  `body` mediumtext DEFAULT NULL,
  `msg_type` varchar(64) DEFAULT NULL,
  `ack` tinyint(4) DEFAULT NULL,
  `raw` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`raw`)),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_msg_key_id` (`msg_key_id`),
  KEY `idx_remote_jid` (`remote_jid`),
  KEY `idx_session_direction` (`session`,`direction`)
) ENGINE=InnoDB AUTO_INCREMENT=3051377 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------
-- Table: wpp_polls (1024 rows)
-- ------------------------------------------
DROP TABLE IF EXISTS `wpp_polls`;
CREATE TABLE `wpp_polls` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `session` varchar(191) DEFAULT NULL,
  `chat_jid` varchar(191) DEFAULT NULL,
  `sender_jid` varchar(191) DEFAULT NULL,
  `msg_key_id` varchar(255) DEFAULT NULL,
  `selected_options` mediumtext DEFAULT NULL,
  `timestamp` bigint(20) DEFAULT NULL,
  `raw` mediumtext DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_session_chat` (`session`,`chat_jid`),
  KEY `idx_msg_key` (`msg_key_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1025 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

