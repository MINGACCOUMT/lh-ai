-- 资产库接入 generations：小说层级 + 公共池
ALTER TABLE `generations`
  ADD COLUMN `novel_id` bigint unsigned NULL DEFAULT NULL COMMENT '所属小说ID(null=公共池)',
  ADD COLUMN `novel_asset_type` varchar(20) NULL DEFAULT NULL COMMENT 'character/scene',
  ADD COLUMN `novel_asset_name` varchar(100) NULL DEFAULT NULL COMMENT '角色/场景名';
ALTER TABLE `generations` ADD INDEX `idx_generations_novel_asset` (`novel_id`, `novel_asset_type`);
