-- 小说系统：小说 / 章节 / 分镜
CREATE TABLE IF NOT EXISTS `novels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `title` varchar(200) NOT NULL,
  `source_filename` varchar(255) DEFAULT NULL,
  `raw_content` longtext,
  `chapter_count` int NOT NULL DEFAULT 0,
  `status` varchar(20) NOT NULL DEFAULT 'active',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_novels_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说';

CREATE TABLE IF NOT EXISTS `novel_chapters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `novel_id` bigint unsigned NOT NULL,
  `chapter_index` int NOT NULL,
  `title` varchar(200) DEFAULT NULL,
  `content` longtext,
  `storyboard_status` varchar(20) NOT NULL DEFAULT 'none',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_novel_chapter` (`novel_id`,`chapter_index`),
  KEY `idx_novel_chapters_novel_id` (`novel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说章节';

CREATE TABLE IF NOT EXISTS `novel_shots` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `chapter_id` bigint unsigned NOT NULL,
  `shot_index` int NOT NULL,
  `scene` varchar(1000) DEFAULT NULL,
  `characters` varchar(500) DEFAULT NULL,
  `prompt` varchar(2000) DEFAULT NULL,
  `dialogue` varchar(1000) DEFAULT NULL,
  `camera` varchar(100) DEFAULT NULL,
  `image_url` varchar(500) DEFAULT NULL,
  `video_url` varchar(500) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_chapter_shot` (`chapter_id`,`shot_index`),
  KEY `idx_novel_shots_chapter_id` (`chapter_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说分镜';
