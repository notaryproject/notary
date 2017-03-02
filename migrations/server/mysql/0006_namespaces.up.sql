ALTER TABLE `tuf_files`
  ADD COLUMN `namespace` VARCHAR(255) DEFAULT "published",
  DROP INDEX `gun`,
  ADD UNIQUE KEY  `gun` (`gun`,`role`,`version`, `namespace`);

ALTER TABLE `changefeed`
  ADD COLUMN `namespace` VARCHAR(255) DEFAULT "published",
  DROP INDEX `idx_changefeed_gun`,
  ADD INDEX `idx_changefeed_gun` (`gun`, `namespace`);