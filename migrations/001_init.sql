-- BakaRay 数据库初始化脚本
-- 支持 MySQL/PostgreSQL

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `username` VARCHAR(64) NOT NULL UNIQUE,
    `password_hash` VARCHAR(128) NOT NULL,
    `balance` BIGINT NOT NULL DEFAULT 0 COMMENT '余额（分）',
    `user_group_id` BIGINT UNSIGNED DEFAULT 0,
    `role` VARCHAR(20) NOT NULL DEFAULT 'user' COMMENT 'admin/user',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_username` (`username`),
    INDEX `idx_user_group` (`user_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 用户组表
CREATE TABLE IF NOT EXISTS `user_groups` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(64) NOT NULL,
    `description` VARCHAR(255) DEFAULT '',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户组表';

-- 节点组表
CREATE TABLE IF NOT EXISTS `node_groups` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(128) NOT NULL,
    `type` VARCHAR(20) NOT NULL COMMENT 'entry/target',
    `description` VARCHAR(255) DEFAULT '',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点组表';

-- 节点表
CREATE TABLE IF NOT EXISTS `nodes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(128) NOT NULL,
    `host` VARCHAR(255) NOT NULL COMMENT '节点IP/域名',
    `port` INT NOT NULL COMMENT 'API端口',
    `secret` VARCHAR(128) NOT NULL COMMENT '通信密钥',
    `status` VARCHAR(20) NOT NULL DEFAULT 'offline' COMMENT 'online/offline',
    `node_group_id` BIGINT UNSIGNED DEFAULT 0,
    `protocols` VARCHAR(255) DEFAULT 'gost,iptables' COMMENT '支持的协议',
    `multiplier` DECIMAL(10,2) NOT NULL DEFAULT 1.00 COMMENT '倍率',
    `region` VARCHAR(64) DEFAULT '' COMMENT '节点地区',
    `last_seen` DATETIME DEFAULT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_status` (`status`),
    INDEX `idx_node_group` (`node_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点表';

-- 节点-用户组关联表
CREATE TABLE IF NOT EXISTS `node_allowed_groups` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `node_id` BIGINT UNSIGNED NOT NULL,
    `user_group_id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_node_group` (`node_id`, `user_group_id`),
    INDEX `idx_node` (`node_id`),
    INDEX `idx_group` (`user_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点-用户组关联表';

-- 转发规则表
CREATE TABLE IF NOT EXISTS `forwarding_rules` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `node_id` BIGINT UNSIGNED NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(128) NOT NULL,
    `protocol` VARCHAR(20) NOT NULL COMMENT 'gost/iptables',
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `traffic_used` BIGINT NOT NULL DEFAULT 0 COMMENT '已用流量（字节）',
    `traffic_limit` BIGINT NOT NULL DEFAULT 0 COMMENT '流量限制（字节）',
    `speed_limit` BIGINT NOT NULL DEFAULT 0 COMMENT '限速（kbps）',
    `mode` VARCHAR(20) NOT NULL DEFAULT 'direct' COMMENT 'direct/rr/lb',
    `listen_port` INT NOT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_node` (`node_id`),
    INDEX `idx_user` (`user_id`),
    INDEX `idx_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='转发规则表';

-- 转发目标表
CREATE TABLE IF NOT EXISTS `targets` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `rule_id` BIGINT UNSIGNED NOT NULL,
    `host` VARCHAR(255) NOT NULL,
    `port` INT NOT NULL,
    `weight` INT NOT NULL DEFAULT 1,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_rule` (`rule_id`),
    INDEX `idx_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='转发目标表';

-- gost 协议专用配置表
CREATE TABLE IF NOT EXISTS `gost_rules` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `rule_id` BIGINT UNSIGNED NOT NULL UNIQUE,
    `transport` VARCHAR(20) DEFAULT 'tcp' COMMENT 'tcp/udp/quic',
    `tls` TINYINT(1) DEFAULT 0,
    `chain` VARCHAR(255) DEFAULT '' COMMENT '代理链配置',
    `timeout` INT DEFAULT 0 COMMENT '超时（秒）',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_rule` (`rule_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='gost协议配置表';

-- iptables 协议专用配置表
CREATE TABLE IF NOT EXISTS `iptables_rules` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `rule_id` BIGINT UNSIGNED NOT NULL UNIQUE,
    `proto` VARCHAR(10) DEFAULT 'tcp',
    `snat` TINYINT(1) DEFAULT 0,
    `iface` VARCHAR(64) DEFAULT '' COMMENT '网络接口',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_rule` (`rule_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='iptables协议配置表';

-- 套餐表
CREATE TABLE IF NOT EXISTS `packages` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(128) NOT NULL,
    `traffic` BIGINT NOT NULL COMMENT '流量（字节）',
    `price` BIGINT NOT NULL COMMENT '价格（分）',
    `user_group_id` BIGINT UNSIGNED DEFAULT 0,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='套餐表';

-- 订单表
CREATE TABLE IF NOT EXISTS `orders` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `package_id` BIGINT UNSIGNED DEFAULT 0,
    `amount` BIGINT NOT NULL COMMENT '金额（分）',
    `status` VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT 'pending/success/failed',
    `trade_no` VARCHAR(64) NOT NULL UNIQUE,
    `pay_type` VARCHAR(32) DEFAULT '',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_user` (`user_id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_trade_no` (`trade_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';

-- 支付配置表
CREATE TABLE IF NOT EXISTS `payment_configs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(128) NOT NULL,
    `provider` VARCHAR(32) NOT NULL COMMENT 'epay/custom',
    `merchant_id` VARCHAR(64) DEFAULT '',
    `merchant_key` VARCHAR(255) DEFAULT '',
    `api_url` VARCHAR(255) DEFAULT '',
    `notify_url` VARCHAR(255) DEFAULT '',
    `extra_params` TEXT COMMENT '额外配置（JSON）',
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付配置表';

-- 支付提供商扩展表
CREATE TABLE IF NOT EXISTS `payment_providers` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(64) NOT NULL,
    `code` VARCHAR(32) NOT NULL UNIQUE,
    `description` VARCHAR(255) DEFAULT '',
    `config_schema` TEXT COMMENT '配置Schema（JSON）',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付提供商表';

-- 站点配置表
CREATE TABLE IF NOT EXISTS `site_config` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `site_name` VARCHAR(128) NOT NULL DEFAULT 'BakaRay',
    `site_domain` VARCHAR(255) DEFAULT '',
    `node_secret` VARCHAR(128) DEFAULT '',
    `node_report_interval` INT DEFAULT 30 COMMENT '上报频率（秒）',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='站点配置表';

-- 流量日志表
CREATE TABLE IF NOT EXISTS `traffic_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `rule_id` BIGINT UNSIGNED NOT NULL,
    `node_id` BIGINT UNSIGNED NOT NULL,
    `bytes_in` BIGINT NOT NULL DEFAULT 0,
    `bytes_out` BIGINT NOT NULL DEFAULT 0,
    `timestamp` DATETIME NOT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_rule` (`rule_id`),
    INDEX `idx_node` (`node_id`),
    INDEX `idx_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='流量日志表';

-- 插入默认管理员用户（密码: admin123）
INSERT INTO `users` (`username`, `password_hash`, `balance`, `role`) VALUES
('admin', 'admin123', 0, 'admin');

-- 插入默认站点配置
INSERT INTO `site_config` (`site_name`, `site_domain`, `node_secret`, `node_report_interval`) VALUES
('BakaRay', 'http://localhost:8080', 'your-node-secret-key', 30);
