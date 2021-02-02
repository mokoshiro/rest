CREATE DATABASE sample_db;
use sample_db;

CREATE TABLE `peer` (
  `peer_id` VARCHAR(512) NOT NULL UNIQUE,
  `addr` VARCHAR(32) NOT NULL,
  `credential` VARCHAR(256) NOT NULL, 
  `location` POINT NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`peer_id`)
  -- SPATIAL KEY `location` (`location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
