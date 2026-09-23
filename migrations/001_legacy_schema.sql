-- Legacy table structure only. No sample users or credentials.
CREATE TABLE `yu_config` (
  `id` int(11) NOT NULL,
  `name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `value` json DEFAULT NULL,
  `create_user` int(11) NOT NULL DEFAULT '0',
  `update_time` int(11) NOT NULL DEFAULT '0',
  `create_time` int(11) NOT NULL DEFAULT '0',
  `remark` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='配置表';

CREATE TABLE `yu_file` (
  `file_id` int(11) NOT NULL,
  `cate` tinyint(1) NOT NULL DEFAULT '9' COMMENT '文件分类',
  `file_type` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '文件类型',
  `parent_id` int(11) NOT NULL DEFAULT '0' COMMENT '父Id',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '名称',
  `src` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '链接',
  `size` int(11) DEFAULT '0' COMMENT '大小',
  `ext` varchar(16) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '文件后缀',
  `md5` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'md5',
  `user_id` int(11) NOT NULL DEFAULT '0' COMMENT '创建人',
  `create_time` int(11) NOT NULL DEFAULT '0' COMMENT '创建时间',
  `update_time` int(11) NOT NULL DEFAULT '0' COMMENT '更新时间',
  `delete_time` int(11) NOT NULL DEFAULT '0' COMMENT '删除时间',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态',
  `is_lock` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否锁定'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文件库';

CREATE TABLE `yu_friend` (
  `friend_id` int(11) NOT NULL,
  `friend_user_id` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '好友ID',
  `nickname` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '好友备注',
  `is_invite` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为邀请方',
  `is_top` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否置顶',
  `is_notice` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否消息提醒',
  `create_user` int(11) NOT NULL DEFAULT '0',
  `update_time` int(11) NOT NULL DEFAULT '0',
  `create_time` int(11) NOT NULL DEFAULT '0',
  `delete_time` int(11) NOT NULL DEFAULT '0',
  `remark` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '申请备注',
  `status` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='联系人置顶表';

CREATE TABLE `yu_group` (
  `group_id` int(11) NOT NULL,
  `name` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '团队名称',
  `name_py` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '团队的拼音',
  `avatar` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '群聊头像',
  `level` tinyint(1) NOT NULL DEFAULT '1' COMMENT '等级',
  `create_user` int(11) NOT NULL DEFAULT '0' COMMENT '创建人',
  `create_time` int(11) NOT NULL DEFAULT '0' COMMENT '创建时间',
  `owner_id` int(11) NOT NULL DEFAULT '0' COMMENT '拥有者',
  `is_public` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否公开',
  `notice` mediumtext COLLATE utf8mb4_unicode_ci COMMENT '公告',
  `setting` mediumtext COLLATE utf8mb4_unicode_ci COMMENT '设置',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态',
  `delete_time` int(11) NOT NULL DEFAULT '0' COMMENT '删除时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `yu_group_user` (
  `id` int(11) NOT NULL,
  `group_id` int(11) NOT NULL DEFAULT '0' COMMENT '团队Id',
  `user_id` int(11) NOT NULL DEFAULT '0' COMMENT '用户Id',
  `role` tinyint(1) NOT NULL DEFAULT '2' COMMENT '角色 1拥有者，2管理员，3成员',
  `invite_id` int(11) NOT NULL DEFAULT '0' COMMENT '邀请人',
  `create_time` int(11) NOT NULL DEFAULT '0' COMMENT '创建时间',
  `unread` int(11) NOT NULL DEFAULT '0' COMMENT '群未读消息',
  `is_notice` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1是否提醒',
  `is_top` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否置顶',
  `no_speak_time` int(11) NOT NULL DEFAULT '0' COMMENT '禁言到期时间',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态 0 ，未同意邀请，1，同意',
  `delete_time` int(11) NOT NULL DEFAULT '0' COMMENT '删除时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `yu_message` (
  `msg_id` int(11) NOT NULL,
  `id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '消息id',
  `from_user` int(11) NOT NULL DEFAULT '0' COMMENT '发送者',
  `to_user` int(11) NOT NULL DEFAULT '0' COMMENT '接受收者',
  `content` text COLLATE utf8mb4_unicode_ci COMMENT '消息内容，如果为文件或图片就是url',
  `chat_identify` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '标识 ：a与b聊天，b与a聊天。记录 a-b',
  `type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'text' COMMENT '消息类型：text、file、image...',
  `is_group` tinyint(1) NOT NULL DEFAULT '0' COMMENT '群聊消息',
  `is_read` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否阅读',
  `is_last` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否是最后一条消息',
  `create_time` int(13) NOT NULL DEFAULT '0' COMMENT '发送时间',
  `is_undo` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否撤回',
  `at` text COLLATE utf8mb4_unicode_ci COMMENT '提醒某人',
  `pid` int(11) DEFAULT '0' COMMENT '引用id',
  `file_id` int(11) NOT NULL DEFAULT '0' COMMENT '文件id',
  `file_cate` tinyint(1) NOT NULL DEFAULT '0' COMMENT '文件类型',
  `file_size` int(11) NOT NULL DEFAULT '0' COMMENT '文件大小',
  `file_name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '文件名称',
  `extends` json DEFAULT NULL COMMENT '消息扩展内容',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态',
  `del_user` text COLLATE utf8mb4_unicode_ci COMMENT '已删除成员'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `yu_emoji` (
  `id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL DEFAULT '0' COMMENT '用户id，0为系统',
  `type` tinyint(1) NOT NULL DEFAULT '1' COMMENT '类型',
  `name` varchar(255) DEFAULT NULL COMMENT '名称',
  `src` varchar(255) DEFAULT NULL COMMENT '链接',
  `file_id` INT(11) NOT NULL DEFAULT '0' COMMENT '文件id',
  `create_time` int(11) NOT NULL DEFAULT '0',
  `update_time` int(11) NOT NULL DEFAULT '0',
  `delete_time` int(11) NOT NULL DEFAULT '0',
  `status` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='表情表';

CREATE TABLE `yu_user` (
  `user_id` int(11) NOT NULL,
  `account` char(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `realname` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `password` char(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `salt` varchar(4) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '加密盐',
  `avatar` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '头像',
  `email` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '电子邮箱',
  `sex` tinyint(1) NOT NULL DEFAULT '2' COMMENT '性别，0女，1男，2未知',
  `role` tinyint(1) NOT NULL DEFAULT '0' COMMENT '角色，0无角色，1超管，2普管',
  `motto` text COLLATE utf8mb4_unicode_ci COMMENT '个性签名',
  `remark` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  `name_py` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '名字的拼音',
  `cs_uid` int(11) NOT NULL DEFAULT '0' COMMENT '客服ID',
  `setting` json DEFAULT NULL COMMENT '用户设置',
  `friend_limit` int(11) NOT NULL DEFAULT '0' COMMENT '好友上限',
  `group_limit` int(11) NOT NULL DEFAULT '0' COMMENT '群聊上限',
  `create_time` int(11) UNSIGNED NOT NULL COMMENT '创建时间',
  `update_time` int(11) UNSIGNED DEFAULT NULL,
  `login_count` mediumint(8) UNSIGNED DEFAULT '0' COMMENT '登录次数',
  `is_auth` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否认证',
  `last_login_time` int(11) UNSIGNED DEFAULT '0' COMMENT '最后登录时间',
  `last_login_ip` char(15) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '最后登录Ip\n',
  `last_chat_time` bigint UNSIGNED NOT NULL DEFAULT '0' COMMENT '最近聊天连接时间',
  `last_chat_ip` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '最近聊天IP',
  `register_ip` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '注册IP',
  `delete_time` int(11) UNSIGNED NOT NULL DEFAULT '0',
  `status` tinyint(1) UNSIGNED DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPACT;

ALTER TABLE `yu_config`
  ADD PRIMARY KEY (`id`);

ALTER TABLE `yu_file`
  ADD PRIMARY KEY (`file_id`);

ALTER TABLE `yu_friend`
  ADD PRIMARY KEY (`friend_id`);

ALTER TABLE `yu_group`
  ADD PRIMARY KEY (`group_id`);

ALTER TABLE `yu_group_user`
  ADD PRIMARY KEY (`id`);

ALTER TABLE `yu_message`
  ADD PRIMARY KEY (`msg_id`);

ALTER TABLE `yu_emoji`
  ADD PRIMARY KEY (`id`);

ALTER TABLE `yu_user`
  ADD PRIMARY KEY (`user_id`),
  ADD UNIQUE KEY `account` (`account`) USING BTREE,
  ADD KEY `accountpassword` (`account`,`password`);

ALTER TABLE `yu_config`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_file`
  MODIFY `file_id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_friend`
  MODIFY `friend_id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_group`
  MODIFY `group_id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_group_user`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_message`
  MODIFY `msg_id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_emoji`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

ALTER TABLE `yu_user`
  MODIFY `user_id` int(11) NOT NULL AUTO_INCREMENT;
