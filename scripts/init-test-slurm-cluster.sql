-- 初始化测试 SLURM 集群和节点数据

-- 1. 创建测试集群
INSERT INTO slurm_clusters (id, name, description, master_host, master_port, config, status, created_at, updated_at)
VALUES (
    1,
    'test-cluster',
    '测试 SLURM 集群',
    'slurm-master',
    6817,
    '{}',
    'active',
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    master_host = EXCLUDED.master_host,
    master_port = EXCLUDED.master_port,
    status = EXCLUDED.status,
    updated_at = NOW();

-- 2. 创建测试节点（test-ssh 系列）
INSERT INTO slurm_nodes (cluster_id, node_name, node_type, host, port, username, auth_type, password, status, cpus, memory, created_at, updated_at)
VALUES
    (1, 'test-ssh01', 'compute', 'test-ssh01', 22, 'root', 'password', 'test', 'pending', 2, 2048, NOW(), NOW()),
    (1, 'test-ssh02', 'compute', 'test-ssh02', 22, 'root', 'password', 'test', 'pending', 2, 2048, NOW(), NOW()),
    (1, 'test-ssh03', 'compute', 'test-ssh03', 22, 'root', 'password', 'test', 'pending', 2, 2048, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- 3. 创建测试节点（test-rocky 系列）
INSERT INTO slurm_nodes (cluster_id, node_name, node_type, host, port, username, auth_type, password, status, cpus, memory, created_at, updated_at)
VALUES
    (1, 'test-rocky01', 'compute', 'test-rocky01', 22, 'root', 'password', 'test', 'pending', 2, 2048, NOW(), NOW()),
    (1, 'test-rocky02', 'compute', 'test-rocky02', 22, 'root', 'password', 'test', 'pending', 2, 2048, NOW(), NOW()),
    (1, 'test-rocky03', 'compute', 'test-rocky03', 22, 'root', 'password', 'test', 'pending', 2, 2048, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- 4. 显示创建结果
SELECT 'SLURM 集群:' as type, id, name, status FROM slurm_clusters WHERE id = 1
UNION ALL
SELECT '节点总数:' as type, NULL as id, CAST(COUNT(*) AS VARCHAR) as name, NULL as status FROM slurm_nodes WHERE cluster_id = 1;

-- 5. 显示所有节点
SELECT id, cluster_id, node_name, node_type, host, status, salt_minion_id 
FROM slurm_nodes 
WHERE cluster_id = 1 
ORDER BY node_name;
