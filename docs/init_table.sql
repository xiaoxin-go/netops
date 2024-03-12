create database netops charset = 'utf8';
use netops;

create table t_log
(
    id         int auto_increment primary key,
    created_at datetime default CURRENT_TIMESTAMP null,
    updated_at datetime default CURRENT_TIMESTAMP null,
    operator   varchar(50)                        null comment '操作人',
    content    varchar(5000)                      null comment '操作内容',
    created_by varchar(50)                        null,
    updated_by varchar(50)                        null
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '日志表';

CREATE TABLE `t_user`
(
    `id`         int(11)     NOT NULL AUTO_INCREMENT,
    `created_at` datetime    DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime    DEFAULT CURRENT_TIMESTAMP,
    `username`   varchar(50) NOT NULL COMMENT '用户名',
    `name_cn`    varchar(50) DEFAULT NULL,
    `enabled`    tinyint(1)  DEFAULT '1' COMMENT '用户是否启用',
    `created_by` varchar(50) DEFAULT NULL,
    `updated_by` varchar(50) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `username` (`username`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '用户表';

CREATE TABLE `t_role`
(
    `id`          int(11)     NOT NULL AUTO_INCREMENT,
    `created_at`  datetime     DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  datetime     DEFAULT CURRENT_TIMESTAMP,
    `name`        varchar(50) NOT NULL COMMENT '角色名',
    `description` varchar(255) DEFAULT NULL,
    `created_by`  varchar(50)  DEFAULT NULL,
    `updated_by`  varchar(50)  DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '角色表';

CREATE TABLE `t_user_role`
(
    `id`      int(11) NOT NULL AUTO_INCREMENT,
    `user_id` int(11) NOT NULL,
    `role_id` int(11) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `user_id` (`user_id`, `role_id`),
    KEY `user_id_2` (`user_id`, `role_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '用户角色关联表';

CREATE TABLE `t_menu`
(
    `id`         int(11)     NOT NULL AUTO_INCREMENT,
    `created_at` datetime    DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime    DEFAULT CURRENT_TIMESTAMP,
    `name`       varchar(50) NOT NULL COMMENT '菜单名称',
    `name_en`       varchar(50) NOT NULL COMMENT '菜单英文名',
    `path`        varchar(50) DEFAULT NULL COMMENT '菜单路由',
    `icon`       varchar(50) DEFAULT NULL COMMENT '图标',
    `sort`       int(11)     DEFAULT NULL COMMENT '菜单排序',
    `parent_id`  int(11)     DEFAULT NULL COMMENT '父菜单ID',
    `enabled`    tinyint(4)  DEFAULT '1' COMMENT '菜单是否启用',
    `created_by` varchar(50) DEFAULT NULL,
    `updated_by` varchar(50) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`, `path`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '菜单表';

CREATE TABLE `t_menu_api`
(
    `id`      int(11) NOT NULL AUTO_INCREMENT,
    `menu_id` int(11) NOT NULL,
    `api_id`  int(11) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `menu_id` (`menu_id`, `api_id`),
    KEY `menu_id_2` (`menu_id`, `api_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '菜单接口表';

CREATE TABLE `t_role_menu`
(
    `id`      int(11) NOT NULL AUTO_INCREMENT,
    `role_id` int(11) NOT NULL,
    `menu_id` int(11) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `role_id` (`role_id`, `menu_id`),
    KEY `t_role_menu___menu` (`menu_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '角色菜单表';

CREATE TABLE `t_role_api`
(
    `id`      int(11) NOT NULL AUTO_INCREMENT,
    `role_id` int(11) NOT NULL,
    `api_id`  int(11) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `role_id` (`role_id`, `api_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='角色权限关联';

CREATE TABLE `t_task`
(
    `id`               int(11)     NOT NULL AUTO_INCREMENT,
    `created_at`       datetime                DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`       datetime                DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
    `jira_key`         varchar(20) NOT NULL COMMENT 'JIRA工单号',
    `jira_region`      varchar(50) NOT NULL COMMENT '变更属地',
    `jira_environment` varchar(50) NOT NULL COMMENT '变更环境',
    `summary`          varchar(2048)           DEFAULT '' COMMENT '工单概要',
    `creator`          varchar(50) NOT NULL COMMENT '创建人',
    `department`       varchar(255)            DEFAULT NULL COMMENT '所属部门',
    `jira_status`      varchar(100)            DEFAULT NULL COMMENT '工单状态',
    `description`      varchar(4096)           DEFAULT NULL COMMENT '描述',
    `assignee`         varchar(50)             DEFAULT NULL COMMENT '经办人',
    `status`           varchar(50)             DEFAULT 'init' COMMENT '任务状态',
    `error_info`       varchar(2048)           DEFAULT '' COMMENT '错误信息',
    `region_id`        int(11)                 DEFAULT NULL COMMENT '网络区域',
    `implement_type`   varchar(50) NOT NULL COMMENT '实施内容',
    `execute_time`     datetime                DEFAULT NULL COMMENT '执行时间',
    `execute_end_time` datetime                DEFAULT NULL COMMENT '执行结束时间',
    `execute_use_time` int(11)                 DEFAULT NULL,
    `type`             enum ('firewall','nlb') DEFAULT 'firewall',
    `is_deleted`       int(11)                 DEFAULT '0',
    `updated_by`       varchar(50)             DEFAULT NULL,
    `created_by`       varchar(50)             DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `jira_key` (`jira_key`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '工单任务表';

CREATE TABLE `t_task_info`
(
    `id`                    int(11)                      NOT NULL AUTO_INCREMENT,
    `task_id`               int(11)                      NOT NULL COMMENT '工单任务ID',
    `src`                   varchar(2048)                NOT NULL COMMENT '源IP地址',
    `dst`                   varchar(2048)                NOT NULL COMMENT '目标IP地址',
    `dport`                 varchar(255)                 NOT NULL COMMENT '目标端口',
    `direction`             enum ('inside','outside','') NOT NULL DEFAULT '' COMMENT '策略类型，出向或者入向',
    `outbound_network_type` varchar(50)                           DEFAULT NULL COMMENT '出向网络类型',
    `created_at`            datetime                              DEFAULT CURRENT_TIMESTAMP,
    `updated_at`            datetime                              DEFAULT CURRENT_TIMESTAMP,
    `protocol`              enum ('ip','tcp','udp')               DEFAULT 'tcp' COMMENT '协议',
    `pool_name`             varchar(50)                           DEFAULT '',
    `static_ip`             varchar(50)                           DEFAULT NULL COMMENT '入向内部IP地址',
    `static_port`           varchar(20)                           DEFAULT NULL COMMENT '入向内部端口',
    `device_id`             int(11)                               DEFAULT NULL,
    `action`                varchar(50)                           DEFAULT 'deny' COMMENT '策略状态',
    `command`               text,
    `result`                text,
    `status`                varchar(50)                           DEFAULT 'init',
    `node`                  varchar(255)                          DEFAULT NULL,
    `node_port`             varchar(50)                           DEFAULT NULL,
    `s_nat`                 varchar(50)                           DEFAULT NULL,
    `vs_command`            varchar(2000)                         DEFAULT NULL,
    `pool_command`          varchar(2000)                         DEFAULT NULL,
    `exists_config`         varchar(2000)                         DEFAULT NULL,
    `nat_name`              varchar(50)                           DEFAULT NULL,
    `created_by`            varchar(50)                           DEFAULT NULL,
    `updated_by`            varchar(50)                           DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `t_task_info___line_type` (`outbound_network_type`),
    KEY `t_task_info___task` (`task_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '工单详情表';

CREATE TABLE `t_task_operate_log`
(
    `id`         int(11) NOT NULL AUTO_INCREMENT,
    `task_id`    int(11) NOT NULL COMMENT '工单任务ID',
    `operator`   varchar(50) DEFAULT NULL COMMENT '操作人',
    `content`    text COMMENT '操作内容',
    `created_at` datetime    DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime    DEFAULT CURRENT_TIMESTAMP,
    `created_by` varchar(50) DEFAULT NULL,
    `updated_by` varchar(50) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 comment '任务操作日志';

CREATE TABLE `t_device`
(
    `id`                     int(11)      NOT NULL AUTO_INCREMENT,
    `created_at`             datetime     DEFAULT CURRENT_TIMESTAMP,
    `updated_at`             datetime     DEFAULT CURRENT_TIMESTAMP,
    `region_id`              int(11)      DEFAULT NULL COMMENT '网络区域ID',
    `name`                   varchar(100) NOT NULL COMMENT '设备名',
    `host`                   varchar(100) NOT NULL COMMENT '设备IP',
    `port`                   int(11)      DEFAULT '22' COMMENT '设备端口',
    `username`               varchar(255) NOT NULL COMMENT '用户名',
    `password`               varchar(255) NOT NULL COMMENT '设备密码',
    `device_type_id`         int(11)      NOT NULL COMMENT '设备类型ID',
    `in_policy`              varchar(50)  DEFAULT NULL COMMENT '入向策略名',
    `out_policy`             varchar(50)  DEFAULT NULL COMMENT '出向策略名',
    `enabled`                tinyint(1)   DEFAULT '1' COMMENT '设备状态',
    `enable_password`        varchar(255) DEFAULT NULL COMMENT 'enable密码',
    `in_deny_policy_name`    varchar(64)  DEFAULT NULL,
    `out_deny_policy_name`   varchar(64)  DEFAULT NULL,
    `parse_status`           varchar(20)  DEFAULT 'init',
    `created_by`             varchar(50)  DEFAULT NULL,
    `updated_by`             varchar(50)  DEFAULT NULL,
    `blacklist_policy_name`  varchar(255) DEFAULT NULL,
    `in_permit_policy_name`  varchar(255) DEFAULT NULL,
    `out_permit_policy_name` varchar(255) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`),
    UNIQUE KEY `host` (`host`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='设备表';

CREATE TABLE `t_nlb_device`
(
    `id`             int(11)      NOT NULL AUTO_INCREMENT,
    `created_at`     datetime                         DEFAULT CURRENT_TIMESTAMP,
    `updated_at`     datetime                         DEFAULT CURRENT_TIMESTAMP,
    `region_id`      int(11)                          DEFAULT NULL COMMENT '网络区域ID',
    `name`           varchar(100) NOT NULL COMMENT '设备名',
    `host`           varchar(100) NOT NULL COMMENT '设备IP',
    `port`           int(11)                          DEFAULT '22' COMMENT '设备端口',
    `username`       varchar(255) NOT NULL COMMENT '用户名',
    `password`       varchar(255) NOT NULL COMMENT '设备密码',
    `device_type_id` int(11)      NOT NULL COMMENT '设备类型ID',
    `enabled`        tinyint(1)                       DEFAULT '1' COMMENT '设备状态',
    `parse_status`   enum ('init','failed','success') DEFAULT 'init' COMMENT '解析状态',
    `created_by`     varchar(50)                      DEFAULT NULL,
    `updated_by`     varchar(50)                      DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`),
    UNIQUE KEY `host` (`host`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='设备表';

CREATE TABLE `t_device_backup`
(
    `id`         int(11) NOT NULL AUTO_INCREMENT,
    `device_id`  int(11) NOT NULL COMMENT '防火墙设备',
    `created_at` datetime     DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime     DEFAULT CURRENT_TIMESTAMP,
    `filename`   varchar(255) DEFAULT NULL COMMENT '文件名',
    `md5`        varchar(64)  DEFAULT NULL COMMENT 'MD5值',
    `size`       int(11)      DEFAULT NULL COMMENT '文件大小',
    `created_by` varchar(50)  DEFAULT NULL,
    `updated_by` varchar(50)  DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT = '设备备份表';

CREATE TABLE `t_device_policy`
(
    `id`         int(11)       NOT NULL AUTO_INCREMENT,
    `device_id`  int(11)       NOT NULL COMMENT '设备ID',
    `name`       varchar(128)  NOT NULL COMMENT '策略名称',
    `direction`  varchar(50)   NOT NULL COMMENT '策略方向',
    `src`        longtext      NOT NULL COMMENT '源地址',
    `src_group`  varchar(4096) NOT NULL COMMENT '源地址组',
    `dst`        longtext      NOT NULL COMMENT '目标地址',
    `dst_group`  varchar(4096) NOT NULL COMMENT '目标地址组',
    `port`       varchar(4096) NOT NULL COMMENT '端口',
    `protocol`   varchar(20)   NOT NULL COMMENT '协议',
    `action`     varchar(20)   NOT NULL COMMENT '是否开通, deny permit',
    `command`    text,
    `port_group` varchar(255) DEFAULT NULL,
    `line`       int(11)      DEFAULT NULL,
    `valid`      tinyint(1)   DEFAULT NULL COMMENT '是否有效',
    PRIMARY KEY (`id`),
    KEY `t_device_policy_device_id_index` (`device_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT = '设备策略表';

CREATE TABLE `t_device_port`
(
    `id`        int(11)      NOT NULL AUTO_INCREMENT,
    `device_id` int(11)      NOT NULL COMMENT '设备ID',
    `name`      varchar(128) NOT NULL COMMENT 'port名称',
    `protocol`  varchar(20)  NOT NULL COMMENT '端口协议',
    `start`     int(11)      NOT NULL COMMENT '端口起始',
    `end`       int(11)      NOT NULL COMMENT '端口结束',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT = '设备端口表';

CREATE TABLE `t_device_address_group`
(
    `id`           int(11)      NOT NULL AUTO_INCREMENT,
    `device_id`    int(11)      NOT NULL COMMENT '设备ID',
    `name`         varchar(255) NOT NULL COMMENT '组名',
    `address`      varchar(128) NOT NULL COMMENT 'IP地址',
    `zone`         varchar(50) DEFAULT NULL,
    `address_type` varchar(20) DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    KEY `t_device_address_group_device_id_index` (`device_id`),
    KEY `t_device_address_group_device_id_index_2` (`device_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='设备地址组';

CREATE TABLE `t_device_nat_pool`
(
    `id`        int(11)      NOT NULL AUTO_INCREMENT,
    `device_id` int(11)      NOT NULL COMMENT '设备ID',
    `name`      varchar(128) NOT NULL COMMENT 'pool名称',
    `address`   text COMMENT '地址',
    `direction` enum ('inside','outside') DEFAULT NULL,
    `port`      varchar(255)              DEFAULT NULL,
    `command`   text,
    `nat_type`  varchar(20)               DEFAULT NULL COMMENT 'nat类型, (source, destination)',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='SRX设备出向线路类型映射地址';

CREATE TABLE `t_device_nat` (
                                `id` int(11) NOT NULL AUTO_INCREMENT,
                                `device_id` int(11) NOT NULL COMMENT '设备ID',
                                `network` varchar(2048) DEFAULT NULL,
                                `static` varchar(2048) DEFAULT NULL,
                                `protocol` varchar(50) DEFAULT NULL COMMENT '协议',
                                `network_port` varchar(50) DEFAULT NULL COMMENT '映射源端口',
                                `static_port` varchar(50) DEFAULT NULL COMMENT '映射目标端口',
                                `command` varchar(2000) DEFAULT NULL,
                                `direction` varchar(64) DEFAULT NULL,
                                `destination` varchar(2048) DEFAULT NULL,
                                `destination_group` varchar(255) DEFAULT NULL,
                                `network_group` varchar(255) DEFAULT NULL,
                                `static_group` varchar(255) DEFAULT NULL,
                                PRIMARY KEY (`id`),
                                KEY `t_device_nat_device_id_index` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 comment '设备NAT表';

CREATE TABLE `t_device_srx_nat` (
                                    `id` int(11) NOT NULL AUTO_INCREMENT,
                                    `device_id` int(11) NOT NULL COMMENT '防火墙设备',
                                    `direction` varchar(20) DEFAULT NULL COMMENT '方向',
                                    `nat_type` varchar(20) DEFAULT NULL COMMENT 'nat类型',
                                    `src` text COMMENT '源地址',
                                    `dst` text COMMENT '目标地址',
                                    `dst_port` varchar(255) DEFAULT NULL COMMENT '目标端口',
                                    `protocol` varchar(20) DEFAULT NULL COMMENT '协议',
                                    `rule` varchar(50) DEFAULT NULL COMMENT 'rule名',
                                    `pool` varchar(50) DEFAULT NULL COMMENT 'pool',
                                    `command` text COMMENT 'nat命令',
                                    PRIMARY KEY (`id`),
                                    KEY `device_id` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 comment 'SRXNat表';

CREATE TABLE `t_f5_vs` (
                           `id` int(11) NOT NULL AUTO_INCREMENT,
                           `device_id` int(11) NOT NULL COMMENT '设备',
                           `name` varchar(255) DEFAULT NULL,
                           `partition` varchar(50) DEFAULT NULL,
                           `source` varchar(50) DEFAULT NULL,
                           `destination` varchar(50) DEFAULT NULL,
                           `source_address_translation` varchar(255) DEFAULT NULL,
                           `enabled` tinyint(1) DEFAULT NULL,
                           `profiles_reference` varchar(255) DEFAULT NULL,
                           `pool` varchar(100) DEFAULT NULL,
                           `protocol` varchar(50) DEFAULT NULL,
                           `rules` varchar(100) DEFAULT NULL,
                           `persist` varchar(100) DEFAULT NULL,
                           `traffic_group` varchar(100) DEFAULT NULL,
                           PRIMARY KEY (`id`),
                           KEY `t_f5_vs_device_id_index` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT 'F5Vs';

CREATE TABLE `t_f5_pool` (
                             `id` int(11) NOT NULL AUTO_INCREMENT,
                             `device_id` int(11) NOT NULL COMMENT '设备',
                             `name` varchar(255) DEFAULT NULL,
                             `partition` varchar(50) DEFAULT NULL COMMENT 'partition',
                             `monitor` varchar(50) DEFAULT NULL,
                             PRIMARY KEY (`id`),
                             KEY `device_id` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 comment 'F5_POOL';

CREATE TABLE `t_policy_log` (
                                `device_id` int(11) NOT NULL,
                                `content` text,
                                `operator` varchar(50) DEFAULT NULL,
                                `id` int(11) NOT NULL AUTO_INCREMENT,
                                `status` varchar(50) DEFAULT NULL,
                                `device_type` varchar(20) DEFAULT NULL,
                                `created_by` varchar(50) DEFAULT NULL,
                                `updated_by` varchar(50) DEFAULT NULL,
                                PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 comment '策略解析日志';

CREATE TABLE `t_public_whitelist` (
                                      `id` int(11) NOT NULL AUTO_INCREMENT,
                                      `region_id` int(11) NOT NULL COMMENT '网络区域',
                                      `type` enum('f5-policy','nat-policy') DEFAULT NULL COMMENT '解析类型',
                                      `device_id` int(11) NOT NULL COMMENT '防火墙设备',
                                      `nlb_device_id` int(11) NOT NULL COMMENT '负载均衡设备',
                                      `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                      `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                      `description` varchar(255) DEFAULT NULL COMMENT '描述',
                                      `created_by` varchar(50) DEFAULT NULL,
                                      `updated_by` varchar(50) DEFAULT NULL,
                                      PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '公网白名单表';

CREATE TABLE `t_device_policy_hit_count` (
                                             `id` int(11) NOT NULL AUTO_INCREMENT,
                                             `device_id` int(11) NOT NULL COMMENT '防火墙设备',
                                             `name` varchar(255) DEFAULT NULL COMMENT '源目端协议唯一',
                                             `source` varchar(64) DEFAULT NULL COMMENT '源地址',
                                             `destination` varchar(64) DEFAULT NULL COMMENT '目标地址',
                                             `protocol` varchar(20) DEFAULT NULL COMMENT '协议',
                                             `port` varchar(64) DEFAULT NULL COMMENT '端口',
                                             `before_hit_count` bigint(50) DEFAULT NULL,
                                             `hit_count` bigint(50) DEFAULT NULL,
                                             `command` varchar(2000) DEFAULT NULL COMMENT '命令',
                                             `state` int(11) DEFAULT '0' COMMENT '是否有效',
                                             `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                             `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                             `created_by` varchar(50) DEFAULT NULL,
                                             `updated_by` varchar(50) DEFAULT NULL,
                                             PRIMARY KEY (`id`),
                                             UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '策略命中数';

CREATE TABLE `t_invalid_policy_task` (
                                         `id` int(11) NOT NULL AUTO_INCREMENT,
                                         `region_id` int(11) NOT NULL COMMENT '网络区域',
                                         `device_id` int(11) NOT NULL COMMENT '防火墙设备',
                                         `description` varchar(255) DEFAULT NULL COMMENT '描述',
                                         `status` enum('ready','running','failed','success') DEFAULT 'ready' COMMENT '状态',
                                         `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                         `created_by` varchar(64) DEFAULT NULL COMMENT '创建人',
                                         `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                         `updated_by` varchar(64) DEFAULT NULL COMMENT '更新人',
                                         PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '无效策略任务';

CREATE TABLE `t_device_type` (
                                 `id` int(11) NOT NULL AUTO_INCREMENT,
                                 `name` varchar(50) NOT NULL COMMENT '设备类型名称',
                                 `type` enum('firewall', 'nlb') NOT NULL COMMENT '防火墙或者负载均衡',
                                 `description` varchar(255) DEFAULT NULL,
                                 `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                 `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                 PRIMARY KEY (`id`),
                                 UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='设备类型表';

CREATE TABLE `t_implement_type` (
                                    `id` int(11) NOT NULL AUTO_INCREMENT,
                                    `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                    `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                    `name` varchar(255) NOT NULL COMMENT '实施类型名称',
                                    `description` varchar(255) DEFAULT NULL COMMENT '描述信息',
                                    PRIMARY KEY (`id`),
                                    UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='对应JIRA工单实施内容';

CREATE TABLE `t_issue_type` (
                                `id` int(11) NOT NULL AUTO_INCREMENT,
                                `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                `region_id` int(11) DEFAULT NULL COMMENT '网络区域ID',
                                `jira_region` varchar(100) NOT NULL COMMENT '工单属地',
                                `jira_environment` varchar(100) NOT NULL COMMENT '工单环境',
                                `description` varchar(255) DEFAULT NULL,
                                `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
                                `created_by` varchar(50) DEFAULT NULL,
                                `updated_by` varchar(50) DEFAULT NULL,
                                PRIMARY KEY (`id`),
                                UNIQUE KEY `region` (`jira_region`,`jira_environment`),
                                KEY `t_issue_type___region` (`region_id`),
                                CONSTRAINT `t_issue_type___region` FOREIGN KEY (`region_id`) REFERENCES `t_region` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '工单类型';

CREATE TABLE `t_outbound_network_type` (
                                           `id` int(11) NOT NULL AUTO_INCREMENT,
                                           `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                           `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                           `name` varchar(50) NOT NULL COMMENT '出向网络类型名称',
                                           `implement_type_id` int(11) NOT NULL,
                                           `description` varchar(255) DEFAULT NULL COMMENT '描述信息',
                                           `created_by` varchar(50) DEFAULT NULL,
                                           `updated_by` varchar(50) DEFAULT NULL,
                                           PRIMARY KEY (`id`),
                                           UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='出向网络类型';

CREATE TABLE `t_region` (
                            `id` int(11) NOT NULL AUTO_INCREMENT,
                            `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                            `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                            `name` varchar(100) NOT NULL COMMENT '网络区域名称',
                            `enabled` tinyint(1) DEFAULT '1',
                            `description` varchar(255) DEFAULT NULL,
                            `task_template_id` int(11) DEFAULT NULL,
                            `api_server` varchar(100) DEFAULT NULL,
                            `created_by` varchar(50) DEFAULT NULL,
                            `updated_by` varchar(50) DEFAULT NULL,
                            PRIMARY KEY (`id`),
                            UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='网络区域表';

CREATE TABLE `t_task_template` (
                                   `id` int(11) NOT NULL AUTO_INCREMENT,
                                   `name` varchar(50) NOT NULL,
                                   `content` varchar(4096) NOT NULL,
                                   `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                   `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                   `created_by` varchar(50) DEFAULT NULL,
                                   `updated_by` varchar(50) DEFAULT NULL,
                                   PRIMARY KEY (`id`),
                                   UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '任务模板表';

CREATE TABLE `t_device_subnet` (
                                   `id` int(11) NOT NULL AUTO_INCREMENT,
                                   `inner_subnet` varchar(50) NOT NULL COMMENT '内部网段，用于判断出向内网地址，源IP, 可多个地址',
                                   `outer_subnet` varchar(50) NOT NULL COMMENT '外部网段，用于判断入向公网地址，目标IP, 可多个地址',
                                   `region_id` int(11) NOT NULL COMMENT '网络区域ID',
                                   `implement_type_id` int(11) NOT NULL COMMENT '实施类型ID',
                                   `device_id` int(11) NOT NULL COMMENT '设备ID',
                                   `description` varchar(255) DEFAULT NULL COMMENT '描述信息',
                                   `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                   `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                   `created_by` varchar(50) DEFAULT NULL,
                                   `updated_by` varchar(50) DEFAULT NULL,
                                   PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '设备网段';

CREATE TABLE `t_nlb_subnet` (
                                `id` int(11) NOT NULL AUTO_INCREMENT,
                                `subnet` varchar(50) NOT NULL COMMENT '对外网段',
                                `region_id` int(11) NOT NULL COMMENT '网络区域ID',
                                `device_id` int(11) NOT NULL COMMENT '设备ID',
                                `description` varchar(255) DEFAULT NULL COMMENT '描述信息',
                                `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                `created_by` varchar(50) DEFAULT NULL,
                                `updated_by` varchar(50) DEFAULT NULL,
                                PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT '负载均衡设备网段表';

CREATE TABLE `t_device_nat_address` (
                                        `id` int(11) NOT NULL AUTO_INCREMENT,
                                        `device_id` int(11) NOT NULL COMMENT '设备ID',
                                        `static_name` varchar(255) DEFAULT NULL,
                                        `subnet` varchar(255) NOT NULL COMMENT '对应出向源IP网段, 可为多个以逗号分割',
                                        `region_id` int(11) NOT NULL COMMENT '网络区域ID',
                                        `outbound_network_type` varchar(50) DEFAULT NULL,
                                        `created_at` datetime DEFAULT NULL,
                                        `updated_at` datetime DEFAULT NULL,
                                        `nat_name` varchar(50) DEFAULT NULL COMMENT 'nat映射名称',
                                        `created_by` varchar(50) DEFAULT NULL,
                                        `updated_by` varchar(50) DEFAULT NULL,
                                        PRIMARY KEY (`id`),
                                        UNIQUE KEY `device_id` (`device_id`,`static_name`,`subnet`),
                                        KEY `t_device_nat_address_device_id_index` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='SRX出向访问源地址映射对应网段';

CREATE TABLE `t_subnet` (
                            `id` int(11) NOT NULL AUTO_INCREMENT,
                            `subnet` varchar(50) NOT NULL COMMENT 'IP地址',
                            `region` varchar(50) NOT NULL COMMENT '区域',
                            `net_type` varchar(50) NOT NULL COMMENT '网络类型',
                            `ip_type` enum('ipv4','ipv6') NOT NULL COMMENT 'IP地址类型',
                            `description` varchar(255) DEFAULT NULL COMMENT '描述信息',
                            `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                            `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                            `created_by` varchar(50) DEFAULT NULL,
                            `updated_by` varchar(50) DEFAULT NULL,
                            PRIMARY KEY (`id`),
                            UNIQUE KEY `ip` (`subnet`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='网段信息表';

CREATE TABLE `t_task_status` (
                                 `id` int(11) NOT NULL AUTO_INCREMENT,
                                 `jira_status` varchar(255) NOT NULL COMMENT '对应jira流程',
                                 `jira_next_status` varchar(255) DEFAULT NULL,
                                 `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                 `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                 `operate` varchar(50) DEFAULT NULL,
                                 `task_status` varchar(255) DEFAULT NULL,
                                 `task_next_status` varchar(50) DEFAULT NULL,
                                 `assignee` varchar(64) DEFAULT NULL,
                                 `created_by` varchar(50) DEFAULT NULL,
                                 `updated_by` varchar(50) DEFAULT NULL,
                                 PRIMARY KEY (`id`),
                                 UNIQUE KEY `operate` (`operate`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8 COMMENT='工单状态';

CREATE TABLE `t_f5_snat_pool` (
                                  `id` int(11) NOT NULL AUTO_INCREMENT,
                                  `region_id` int(11) NOT NULL COMMENT '网络区域',
                                  `device_id` int(11) NOT NULL COMMENT '设备',
                                  `name` varchar(50) DEFAULT NULL COMMENT 'pool名称',
                                  `subnet` varchar(1024) DEFAULT NULL COMMENT '网段信息',
                                  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP,
                                  PRIMARY KEY (`id`),
                                  UNIQUE KEY `device_id` (`device_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT 'F5设备SnatPool表';