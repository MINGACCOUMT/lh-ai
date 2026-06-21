-- 分镜唯一键从 (chapter_id, shot_index) 改为 (plot_id, shot_index)
-- 原因：分镜改挂 plot 后 chapter_id=0，老键导致跨情节 (0, shot_index) 冲突 (Error 1062)
ALTER TABLE `novel_shots` DROP INDEX `uniq_chapter_shot`;
ALTER TABLE `novel_shots` ADD UNIQUE KEY `uniq_plot_shot` (`plot_id`, `shot_index`);
