CREATE TABLE `namespaces` (
  `name` VARCHAR(255) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `namespaces` VALUES ("default");

ALTER TABLE `tuf_files`
  ADD COLUMN `namespace` VARCHAR(255) DEFAULT "default",
  ADD FOREIGN KEY (`namespace`) REFERENCES `namespaces`(`name`),
  DROP INDEX `gun`,
  ADD UNIQUE KEY  `gun` (`gun`,`role`,`version`, `namespace`);

ALTER TABLE `changefeed`
  ADD COLUMN `namespace` VARCHAR(255) DEFAULT "default",
  ADD FOREIGN KEY (`namespace`) REFERENCES `namespaces`(`name`),
  DROP INDEX `idx_changefeed_gun`,
  ADD INDEX `idx_changefeed_gun` (`gun`, `namespace`);