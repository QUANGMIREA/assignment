SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

# DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` INT(4) ZEROFILL NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `is_active` BOOL DEFAULT TRUE NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

# DROP TABLE IF EXISTS `segments`;
CREATE TABLE `segments` (
    `id` INT(3)  NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `slug` VARCHAR(50) NOT NULL UNIQUE,
    `is_active` BOOL DEFAULT TRUE NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

# DROP TABLE IF EXISTS `user_segment_relation`;
CREATE TABLE `user_segment_relation` (
    `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user_id` INT(4) ZEROFILL NOT NULL,
    `segment_id` INT(3) NOT NULL,
    `is_active` BOOL DEFAULT TRUE NOT NULL,
    `date_assigned` DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    `date_unassigned` DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (segment_id) REFERENCES segments(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

# Auto users creation
DELIMITER //
CREATE PROCEDURE AutoInsertValuesToTable()
BEGIN
    DECLARE startingRange INT DEFAULT 1000;
    WHILE startingRange <= 2000 DO
            INSERT INTO `users` (`id`) VALUES (startingRange);
            SET startingRange = startingRange + 1;
        END WHILE;
END
//
DELIMITER ;
CALL AutoInsertValuesToTable();

# INSERT INTO `segments` (`id`, `slug`) VALUES
# (100, 'AVITO_VOICE_MESSAGES'),
# (120, 'AVITO_PERFORMANCE_VAS'),
# (256, 'AVITO_DISCOUNT_30'),
# (588, 'AVITO_DISCOUNT_50');
#
# INSERT INTO `user_segment_relation` (`user_id`, `segment_id`) VALUES
# (1000, 100),
# (1000, 120),
# (1000, 256),
# (1000, 588),
# (1002, 100),
# (1002, 588),
# (1003, 256),
# (1004, 588),
# (1005, 256);
