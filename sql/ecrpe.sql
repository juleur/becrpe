-- MySQL Script generated by MySQL Workbench
-- Sat Feb 22 16:29:24 2020
-- Model: New Model    Version: 1.0
-- MySQL Workbench Forward Engineering

SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

-- -----------------------------------------------------
-- Schema ecrpe
-- -----------------------------------------------------
DROP SCHEMA IF EXISTS `ecrpe` ;

-- -----------------------------------------------------
-- Schema ecrpe
-- -----------------------------------------------------
CREATE SCHEMA IF NOT EXISTS `ecrpe` DEFAULT CHARACTER SET utf8 ;
USE `ecrpe` ;

-- -----------------------------------------------------
-- Table `ecrpe`.`users`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`users` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`users` (
  `id` SMALLINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(20) NOT NULL,
  `fullname` VARCHAR(20) NULL,
  `email` VARCHAR(50) NOT NULL,
  `encrypted_pwd` VARCHAR(100) NOT NULL,
  `is_teacher` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `users_email_unique` (`email` ASC) VISIBLE,
  INDEX `users_email_idx` (`email` ASC) VISIBLE,
  UNIQUE INDEX `users_username_unique` (`username` ASC) VISIBLE,
  UNIQUE INDEX `users_fullname_unique` (`fullname` ASC) VISIBLE,
  INDEX `users_is_teacher_idx` (`is_teacher` ASC) VISIBLE)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`refresher_courses`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`refresher_courses` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`refresher_courses` (
  `id` SMALLINT NOT NULL AUTO_INCREMENT,
  `subject` ENUM('ECONOMICS', 'FRENCH', 'MATHETIMATICS') NOT NULL,
  `year` VARCHAR(4) NOT NULL,
  `is_finished` TINYINT(1) NOT NULL DEFAULT 0,
  `price` FLOAT NOT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `rc_subject` (`subject` ASC) VISIBLE)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`sessions`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`sessions` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`sessions` (
  `id` MEDIUMINT NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(50) NOT NULL,
  `section` ENUM('DIALECTICAL', 'SCIENTIFIC') NOT NULL,
  `type` ENUM('EXERCISE', 'LESSON') NOT NULL,
  `description` VARCHAR(255) NULL DEFAULT NULL,
  `session_number` TINYINT NULL,
  `recorded_on` DATE NOT NULL,
  `is_ready` TINYINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NULL DEFAULT NULL,
  `refresher_course_id` SMALLINT NOT NULL,
  `user_id` SMALLINT NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `sessions_refresher_course_id_idx` (`refresher_course_id` ASC) VISIBLE,
  INDEX `sessions_user_id_idx` (`user_id` ASC) VISIBLE,
  INDEX `sessions_is_ready_idx` (`is_ready` ASC) VISIBLE,
  CONSTRAINT `fk_refresher_course_id_sessions`
    FOREIGN KEY (`refresher_course_id`)
    REFERENCES `ecrpe`.`refresher_courses` (`id`)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT `fk_user_id_sessions`
    FOREIGN KEY (`user_id`)
    REFERENCES `ecrpe`.`users` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`payments`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`payments` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`payments` (
  `id` MEDIUMINT NOT NULL AUTO_INCREMENT,
  `paypal_payer_id` VARCHAR(25) NOT NULL,
  `paypal_order_id` VARCHAR(25) NOT NULL,
  `created_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `payments_paypal_order_id_unique` (`paypal_order_id` ASC) VISIBLE)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`users_refresher_courses`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`users_refresher_courses` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`users_refresher_courses` (
  `payment_id` MEDIUMINT NULL,
  `user_id` SMALLINT NULL,
  `refresher_course_id` SMALLINT NULL,
  INDEX `urc_payment_id_idx` (`payment_id` ASC) VISIBLE,
  INDEX `urc_composite_idx` (`refresher_course_id` ASC, `user_id` ASC) VISIBLE,
  INDEX `urc_user_id_idx` (`user_id` ASC) VISIBLE,
  CONSTRAINT `fk_user_id_urc`
    FOREIGN KEY (`user_id`)
    REFERENCES `ecrpe`.`users` (`id`)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT `fk_refresher_course_id_urc`
    FOREIGN KEY (`refresher_course_id`)
    REFERENCES `ecrpe`.`refresher_courses` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `fk_payment_id_urc`
    FOREIGN KEY (`payment_id`)
    REFERENCES `ecrpe`.`payments` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`videos`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`videos` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`videos` (
  `id` MEDIUMINT NOT NULL AUTO_INCREMENT,
  `path` VARCHAR(100) NULL DEFAULT NULL,
  `duration` VARCHAR(7) NULL DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NULL DEFAULT NULL,
  `session_id` MEDIUMINT NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `videos_path_unique` (`path` ASC) VISIBLE,
  UNIQUE INDEX `videos_session_id_unique` (`session_id` ASC) VISIBLE,
  INDEX `videos_session_id_idx` (`session_id` ASC) VISIBLE,
  CONSTRAINT `fk_session_id_videos`
    FOREIGN KEY (`session_id`)
    REFERENCES `ecrpe`.`sessions` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`user_auths`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`user_auths` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`user_auths` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `user_agent` VARCHAR(150) NOT NULL,
  `ip_address` VARCHAR(40) NOT NULL,
  `refresh_token` VARCHAR(16) NOT NULL,
  `delivered_at` DATETIME NOT NULL,
  `is_revoked` TINYINT NOT NULL DEFAULT 0,
  `revoked_at` DATETIME NULL,
  `on_login` TINYINT NOT NULL DEFAULT 0,
  `on_refresh` TINYINT NOT NULL DEFAULT 0,
  `user_id` SMALLINT NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `ua_refresh_token_unique` (`refresh_token` ASC) VISIBLE,
  INDEX `ua_user_id_idx` (`user_id` ASC) VISIBLE,
  INDEX `ua_composite_idx` (`is_revoked` ASC, `revoked_at` ASC, `user_id` ASC) VISIBLE,
  CONSTRAINT `fk_user_id_user_auths`
    FOREIGN KEY (`user_id`)
    REFERENCES `ecrpe`.`users` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `ecrpe`.`class_papers`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `ecrpe`.`class_papers` ;

CREATE TABLE IF NOT EXISTS `ecrpe`.`class_papers` (
  `id` MEDIUMINT NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(50) NULL,
  `path` VARCHAR(100) NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NULL,
  `session_id` MEDIUMINT NULL,
  PRIMARY KEY (`id`),
  INDEX `class_papers_session_id_idx` (`session_id` ASC) VISIBLE,
  UNIQUE INDEX `class_papers_path_unique` (`path` ASC) VISIBLE,
  CONSTRAINT `fk_session_id_class_papers`
    FOREIGN KEY (`session_id`)
    REFERENCES `ecrpe`.`sessions` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;

USE `ecrpe`;

DELIMITER $$

USE `ecrpe`$$
DROP TRIGGER IF EXISTS `ecrpe`.`users_AFTER_DELETE` $$
USE `ecrpe`$$
CREATE DEFINER = CURRENT_USER TRIGGER `ecrpe`.`users_AFTER_DELETE` AFTER DELETE ON `users` FOR EACH ROW
BEGIN
	delete from user_auths AS ua where ua.user_id = OLD.id;
    delete from users_refresher_courses AS urc where urc.user_id = OLD.id;
END$$


DELIMITER ;

SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
