-- ====================================================================
-- SLURM Tasks 表结构修复脚本
-- ====================================================================
-- 问题: task_id 字段类型为 bigint，但代码使用 UUID 字符串
-- 解决: 修改 task_id 为 VARCHAR(36) 以支持 UUID
-- ====================================================================

-- 开始事务
BEGIN;

-- 1. 备份现有数据到临时表
CREATE TABLE IF NOT EXISTS slurm_tasks_backup AS 
SELECT * FROM slurm_tasks;

SELECT format('✓ 已备份 %s 条记录到 slurm_tasks_backup', COUNT(*)) 
FROM slurm_tasks_backup;

-- 2. 清空相关表（因为有外键约束）
TRUNCATE TABLE slurm_task_events CASCADE;
TRUNCATE TABLE slurm_tasks CASCADE;

-- 3. 删除旧的唯一索引
DROP INDEX IF EXISTS idx_slurm_tasks_task_id;

-- 4. 修改 task_id 字段类型为 VARCHAR(36)
ALTER TABLE slurm_tasks 
ALTER COLUMN task_id TYPE VARCHAR(36);

-- 5. 重新创建唯一索引
CREATE UNIQUE INDEX idx_slurm_tasks_task_id ON slurm_tasks(task_id);

-- 6. 验证修改
SELECT 
    column_name,
    data_type,
    character_maximum_length,
    is_nullable
FROM information_schema.columns
WHERE table_name = 'slurm_tasks' 
  AND column_name = 'task_id';

-- 提交事务
COMMIT;

-- 显示修复结果
\echo '====================================================================';
\echo '✓ 表结构修复完成';
\echo '✓ task_id 字段已从 bigint 改为 varchar(36)';
\echo '✓ 唯一索引已重建';
\echo '====================================================================';
