CREATE TABLE `user`
(
    `id`          bigint             NOT NULL AUTO_INCREMENT,
    `merchant_id` bigint             NOT NULL DEFAULT '1' COMMENT '所属商户(merchant.id)',
    `name`        varchar(255)        NOT NULL DEFAULT '' COMMENT '用户姓名',
    `gender`      tinyint NOT NULL DEFAULT '0' COMMENT '用户性别',
    `mobile`      varchar(255)        NOT NULL DEFAULT '' COMMENT '用户电话',
    `password`    varchar(255)        NOT NULL DEFAULT '' COMMENT '用户密码',
    `casdoor_id`  varchar(255)        NOT NULL DEFAULT '' COMMENT 'Casdoor 用户 Id（第三方登录关联键）',
    `create_time` timestamp           NULL     DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp           NULL     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_merchant_mobile` (`merchant_id`, `mobile`),
    KEY `idx_merchant_casdoor` (`merchant_id`, `casdoor_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;