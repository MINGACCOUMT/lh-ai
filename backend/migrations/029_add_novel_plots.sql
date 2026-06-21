-- 情节层 + 章节分析字段（v2）
ALTER TABLE `novel_chapters`
  ADD COLUMN `characters` text NULL COMMENT '人物画像',
  ADD COLUMN `scenes` text NULL COMMENT '场景描述',
  ADD COLUMN `analysis_status` varchar(20) NOT NULL DEFAULT 'none' COMMENT '解析状态';

CREATE TABLE IF NOT EXISTS `novel_plots` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `chapter_id` bigint unsigned NOT NULL,
  `plot_index` int NOT NULL,
  `title` varchar(200) DEFAULT NULL,
  `summary` varchar(1000) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_chapter_plot` (`chapter_id`,`plot_index`),
  KEY `idx_novel_plots_chapter_id` (`chapter_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说情节';

-- 分镜挂到 plot（保留 chapter_id 列以免破坏旧迁移，但模型改用 plot_id）
ALTER TABLE `novel_shots` ADD COLUMN `plot_id` bigint unsigned NULL;
