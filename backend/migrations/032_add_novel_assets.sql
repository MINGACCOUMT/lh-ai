-- 小说资产库：角色 / 场景图（novel 级，跨章节复用）
CREATE TABLE IF NOT EXISTS `novel_characters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `novel_id` bigint unsigned NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` varchar(500) DEFAULT NULL,
  `image_url` varchar(500) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_novel_char` (`novel_id`,`name`),
  KEY `idx_novel_chars_novel_id` (`novel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说角色资产';

CREATE TABLE IF NOT EXISTS `novel_scenes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `novel_id` bigint unsigned NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` varchar(500) DEFAULT NULL,
  `image_url` varchar(500) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_novel_scene` (`novel_id`,`name`),
  KEY `idx_novel_scenes_novel_id` (`novel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说场景资产';

-- 章节级资产生成状态（none/generating/ready/failed）
ALTER TABLE `novel_chapters` ADD COLUMN `assets_status` varchar(20) NOT NULL DEFAULT 'none';
