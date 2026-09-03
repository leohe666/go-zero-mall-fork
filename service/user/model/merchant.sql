CREATE TABLE `merchant`
(
    `id`                BIGINT NOT NULL AUTO_INCREMENT,
    `name`              VARCHAR(128) NOT NULL DEFAULT '' COMMENT '商户名称',
    `status`            TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1启用 0停用',
    `casdoor_endpoint`  VARCHAR(255) NOT NULL DEFAULT '' COMMENT '该商户的 Casdoor 实例地址',
    `casdoor_client_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '该商户在 Casdoor 的应用 clientId',
    `casdoor_client_secret_enc` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Casdoor 应用 clientSecret（AES-GCM 加密后）',
    `casdoor_org`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'Casdoor organization（租户名）',
    `casdoor_app`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'Casdoor application 名',
    `casdoor_cert_pem`  TEXT COMMENT '该商户 Casdoor 应用证书公钥（校验 JWT）',
    `wx_app_id`         VARCHAR(64) NOT NULL DEFAULT '' COMMENT '微信小程序 AppID（公开）',
    `wx_app_secret_enc` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '微信 AppSecret（AES-GCM 加密后）',
    `create_time`       TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time`       TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_casdoor_client_id` (`casdoor_client_id`),
    UNIQUE KEY `idx_wx_app_id` (`wx_app_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;