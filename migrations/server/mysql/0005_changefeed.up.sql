CREATE TABLE `changefeed` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
    `gun` varchar(255) NOT NULL,
    `version` int(11) NOT NULL,
    `sha256` CHAR(64) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;