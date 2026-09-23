-- IMGO 新服务器数据清理（MySQL 8）。执行前停止 Go 服务并备份数据库。
-- 只保留一个有效超级管理员账号、基础配置、表结构和索引。
-- 清空其他用户、消息/公告、好友、群聊、上传记录、登录会话、财务和层级数据。
-- 不删除 public/storage 或对象存储中的实体文件，也不重置表的自增序号。
-- 仅处理已知 IMGO 表；旧 PHP 或其他应用的表不在清理范围内。

-- 必须与 .env 中 TABLE_PREFIX 一致。
SET @imgo_clear_prefix = 'yu_';
-- 要保留的超级管理员登录账号（精确匹配，且数据库中的 role 必须为 1）。
-- 此值与 .env 的 ADMIN_ACCOUNT 保持一致。
SET @imgo_keep_admin_account = 'administrator';

DROP PROCEDURE IF EXISTS imgo_clear_business_data;
DELIMITER $$
CREATE PROCEDURE imgo_clear_business_data()
BEGIN
  DECLARE done INT DEFAULT 0;
  DECLARE target_table VARCHAR(128);
  DECLARE tables_to_clear CURSOR FOR
    SELECT table_name
      FROM information_schema.tables
     WHERE table_schema = DATABASE()
       AND table_type = 'BASE TABLE'
       AND table_name IN (
         CONCAT(@imgo_clear_prefix, 'file'),
         CONCAT(@imgo_clear_prefix, 'friend'),
         CONCAT(@imgo_clear_prefix, 'group'),
         CONCAT(@imgo_clear_prefix, 'group_user'),
         CONCAT(@imgo_clear_prefix, 'message'),
         CONCAT(@imgo_clear_prefix, 'emoji'),
         CONCAT(@imgo_clear_prefix, 'imgo_online_sample'),
         CONCAT(@imgo_clear_prefix, 'imgo_object'),
         CONCAT(@imgo_clear_prefix, 'imgo_session'),
         CONCAT(@imgo_clear_prefix, 'imgo_chat_lock'),
         CONCAT(@imgo_clear_prefix, 'imgo_bank_card'),
         CONCAT(@imgo_clear_prefix, 'imgo_check_in'),
         CONCAT(@imgo_clear_prefix, 'imgo_wallet'),
         CONCAT(@imgo_clear_prefix, 'imgo_withdrawal'),
         CONCAT(@imgo_clear_prefix, 'imgo_wallet_entry'),
         CONCAT(@imgo_clear_prefix, 'imgo_recharge_order'),
         CONCAT(@imgo_clear_prefix, 'imgo_referral_path')
       )
     ORDER BY table_name;
  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;
  DECLARE EXIT HANDLER FOR SQLEXCEPTION
  BEGIN
    ROLLBACK;
    RESIGNAL;
  END;

  IF DATABASE() IS NULL
     OR @imgo_clear_prefix IS NULL
     OR @imgo_clear_prefix NOT REGEXP '^[a-zA-Z0-9_]+$' THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = 'Select database and set a valid TABLE_PREFIX first';
  END IF;

  IF @imgo_keep_admin_account IS NULL OR @imgo_keep_admin_account = '' THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = 'Set the administrator account before cleanup';
  END IF;

  IF (SELECT COUNT(*)
        FROM information_schema.tables
       WHERE table_schema = DATABASE()
         AND table_type = 'BASE TABLE'
         AND table_name IN (
           CONCAT(@imgo_clear_prefix, 'config'),
           CONCAT(@imgo_clear_prefix, 'user')
         )) <> 2 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = 'Missing config/user table; check database and prefix';
  END IF;

  SET @imgo_admin_user_id = NULL;
  SET @imgo_clear_sql = CONCAT(
    'SELECT user_id INTO @imgo_admin_user_id FROM `',
    @imgo_clear_prefix,
    'user` WHERE account=? AND role=1 AND status=1 AND COALESCE(delete_time,0)=0 LIMIT 1'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_keep_admin_account;
  DEALLOCATE PREPARE clear_statement;

  IF @imgo_admin_user_id IS NULL THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = 'The requested active role=1 administrator was not found; nothing was deleted';
  END IF;

  START TRANSACTION;

  OPEN tables_to_clear;
  clear_loop: LOOP
    FETCH tables_to_clear INTO target_table;
    IF done = 1 THEN
      LEAVE clear_loop;
    END IF;
    SET @imgo_clear_sql = CONCAT('DELETE FROM `', target_table, '`');
    PREPARE clear_statement FROM @imgo_clear_sql;
    EXECUTE clear_statement;
    DEALLOCATE PREPARE clear_statement;
  END LOOP;
  CLOSE tables_to_clear;

  -- 删除其他用户，保留超级管理员的账号、密码、昵称和权限。
  SET @imgo_clear_sql = CONCAT(
    'DELETE FROM `', @imgo_clear_prefix, 'user` WHERE user_id<>?'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  -- 清除管理员的历史头像、IP、登录计数及客服关联，避免引用已删除数据。
  SET @imgo_clear_sql = CONCAT(
    'UPDATE `', @imgo_clear_prefix,
    'user` SET avatar=NULL,cs_uid=0,login_count=0,last_login_time=0,',
    'last_login_ip=NULL,register_ip=NULL,update_time=NULL WHERE user_id=?'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  -- 保留管理员已有的邀请码，删除其他账号的邀请码和旧层级关系。
  SET @imgo_clear_sql = CONCAT(
    'DELETE FROM `', @imgo_clear_prefix, 'imgo_referral` WHERE user_id<>?'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  SET @imgo_clear_sql = CONCAT(
    'UPDATE `', @imgo_clear_prefix,
    'imgo_referral` SET parent_user_id=0 WHERE user_id=?'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  SET @imgo_clear_sql = CONCAT(
    'DELETE FROM `', @imgo_clear_prefix,
    'imgo_referral_legacy_code` WHERE user_id<>?'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  -- 只保留后台支持的基础配置，清掉自动任务进度等运行数据。
  SET @imgo_clear_sql = CONCAT(
    'DELETE FROM `', @imgo_clear_prefix,
    'config` WHERE name IS NULL OR name NOT IN (',
    '''sysInfo'',''chatInfo'',''fileUpload'',''compass'',''email'',',
    '''smtp'',''appVersion'',''sms'',''maintenance'')'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement;
  DEALLOCATE PREPARE clear_statement;

  -- 禁用并清空自动加好友/群聊配置，避免指向已删除的账号和群。
  SET @imgo_clear_sql = CONCAT(
    'UPDATE `', @imgo_clear_prefix, 'config` SET value=JSON_SET(',
    'COALESCE(value,JSON_OBJECT()),',
    '''$.autoAddUser'',JSON_OBJECT(',
    '''status'',''0'',''welcome'',''欢迎使用 Imgo'',',
    '''user_ids'',JSON_ARRAY(),''user_items'',JSON_ARRAY()),',
    '''$.autoAddGroup'',JSON_OBJECT(',
    '''status'',''0'',''name'',''交流群'',''userMax'',''100'',',
    '''owner_uid'',?,''owner_info'',JSON_ARRAY())) ',
    'WHERE name=''chatInfo'''
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  -- 保留清理任务开关和周期，仅重置历史执行状态。
  SET @imgo_clear_sql = CONCAT(
    'UPDATE `', @imgo_clear_prefix, 'config` SET value=JSON_SET(',
    'COALESCE(value,JSON_OBJECT()),',
    '''$.logs'',JSON_ARRAY(),''$.last_run_at'',0,',
    '''$.next_run_at'',0,''$.last_error'','''') ',
    'WHERE name=''maintenance'''
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement;
  DEALLOCATE PREPARE clear_statement;

  SET @imgo_clear_sql = CONCAT(
    'UPDATE `', @imgo_clear_prefix, 'config` SET create_user=?'
  );
  PREPARE clear_statement FROM @imgo_clear_sql;
  EXECUTE clear_statement USING @imgo_admin_user_id;
  DEALLOCATE PREPARE clear_statement;

  COMMIT;
  SELECT @imgo_admin_user_id AS retained_admin_user_id,
         @imgo_keep_admin_account AS retained_admin_account,
         'Cleanup completed' AS result;
END$$
DELIMITER ;

CALL imgo_clear_business_data();
DROP PROCEDURE imgo_clear_business_data;
