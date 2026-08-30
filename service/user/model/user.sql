CREATE TABLE `user`
(
    `id`           bigint unsigned     NOT NULL AUTO_INCREMENT,
    `name`         varchar(255)        NOT NULL DEFAULT '' COMMENT '用户姓名',
    `gender`       tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '用户性别',
    `mobile`       varchar(255)        NOT NULL DEFAULT '' COMMENT '用户电话',
    `password`     varchar(255)        NOT NULL DEFAULT '' COMMENT '用户密码',
    `casdoor_name` varchar(255)        NOT NULL DEFAULT '' COMMENT 'Casdoor 用户名(微信小程序登录为 wechat-openid)',
    `create_time`  timestamp           NULL     DEFAULT CURRENT_TIMESTAMP,
    `update_time`  timestamp           NULL     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_mobile_unique` (`mobile`),
    KEY `idx_casdoor_name` (`casdoor_name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;