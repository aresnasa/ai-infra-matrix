-- 混合认证系统迁移脚本
-- 为用户表添加认证源和LDAP DN字段

-- 添加认证源字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_source VARCHAR(50) DEFAULT 'local';

-- 添加LDAP DN字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS ldap_dn VARCHAR(500);

-- 为现有用户设置默认的认证源
UPDATE users SET auth_source = 'local' WHERE auth_source IS NULL;

-- 添加注释
COMMENT ON COLUMN users.auth_source IS '认证来源: local, ldap';
COMMENT ON COLUMN users.ldap_dn IS 'LDAP用户的Distinguished Name';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_users_auth_source ON users(auth_source);
CREATE INDEX IF NOT EXISTS idx_users_ldap_dn ON users(ldap_dn);

-- 输出迁移完成信息
DO $$
BEGIN
    RAISE NOTICE '混合认证系统迁移完成！';
    RAISE NOTICE '- 已添加 auth_source 字段（认证来源）';
    RAISE NOTICE '- 已添加 ldap_dn 字段（LDAP DN）';
    RAISE NOTICE '- 已为现有用户设置默认认证源为 local';
    RAISE NOTICE '- 已创建相关索引以提高查询性能';
END
$$;
