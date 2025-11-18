-- 添加集群类型和 SSH 配置字段到 slurm_clusters 表
-- 迁移时间: 2024-01-01
-- 用途: 支持多集群管理，区分托管集群和外部集群

-- 添加 cluster_type 字段（默认为 managed）
ALTER TABLE slurm_clusters 
ADD COLUMN IF NOT EXISTS cluster_type VARCHAR(50) DEFAULT 'managed';

-- 添加 master_ssh 字段（存储 SSH 连接配置）
ALTER TABLE slurm_clusters 
ADD COLUMN IF NOT EXISTS master_ssh JSONB;

-- 为 cluster_type 添加索引，提高查询性能
CREATE INDEX IF NOT EXISTS idx_slurm_clusters_cluster_type 
ON slurm_clusters(cluster_type);

-- 添加约束，确保 cluster_type 只能是 managed 或 external
ALTER TABLE slurm_clusters
ADD CONSTRAINT IF NOT EXISTS check_cluster_type 
CHECK (cluster_type IN ('managed', 'external'));

-- 更新现有记录为 managed 类型（如果存在）
UPDATE slurm_clusters 
SET cluster_type = 'managed' 
WHERE cluster_type IS NULL;

-- 添加注释
COMMENT ON COLUMN slurm_clusters.cluster_type IS '集群类型: managed(托管集群) 或 external(外部集群)';
COMMENT ON COLUMN slurm_clusters.master_ssh IS 'SSH连接配置 (JSON格式): {host, port, username, auth_type, password, key_path}';
