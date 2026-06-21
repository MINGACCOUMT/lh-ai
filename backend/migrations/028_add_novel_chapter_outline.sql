-- 章节大纲（分镜抽取时由 LLM 产出的本章简短概述）
ALTER TABLE `novel_chapters` ADD COLUMN `outline` text NULL COMMENT 'chapter outline';
