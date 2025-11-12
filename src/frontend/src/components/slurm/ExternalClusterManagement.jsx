import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { 
  Server, 
  Database, 
  Key, 
  CheckCircle, 
  AlertCircle, 
  Loader,
  Plus,
  Trash2,
  RefreshCw
} from 'lucide-react';
import { api } from '../../services/api';

/**
 * 添加已有 SLURM 集群管理页面
 * 专门用于连接和管理外部已存在的 SLURM 集群
 */
const ExternalClusterManagement = () => {
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(false);
  const [testingConnection, setTestingConnection] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    master_host: '',
    ssh_username: 'root',
    ssh_password: '',
    ssh_port: 22,
    // 复用已有配置
    reuse_config: true,
    reuse_munge: true,
    reuse_database: true,
  });
  const [testResult, setTestResult] = useState(null);
  const [clusterInfo, setClusterInfo] = useState(null);

  useEffect(() => {
    fetchExternalClusters();
  }, []);

  const fetchExternalClusters = async () => {
    try {
      setLoading(true);
      const response = await api.get('/api/slurm/clusters');
      // 只获取外部集群
      const externalClusters = response.data.data.filter(
        (cluster) => cluster.cluster_type === 'external'
      );
      setClusters(externalClusters);
    } catch (error) {
      console.error('获取外部集群失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  // 测试连接
  const testConnection = async () => {
    setTestingConnection(true);
    setTestResult(null);
    setClusterInfo(null);

    try {
      const response = await api.post('/api/slurm/clusters/test-connection', {
        host: formData.master_host,
        ssh: {
          username: formData.ssh_username,
          password: formData.ssh_password,
          port: formData.ssh_port,
        },
      });

      if (response.data.success) {
        setTestResult({
          success: true,
          message: '连接成功！',
        });
        setClusterInfo(response.data.data);
      } else {
        setTestResult({
          success: false,
          message: response.data.error || '连接失败',
        });
      }
    } catch (error) {
      setTestResult({
        success: false,
        message: error.response?.data?.error || '连接测试失败',
      });
    } finally {
      setTestingConnection(false);
    }
  };

  // 添加外部集群
  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      const payload = {
        name: formData.name,
        description: formData.description,
        master_host: formData.master_host,
        master_ssh: {
          host: formData.master_host,
          port: parseInt(formData.ssh_port),
          username: formData.ssh_username,
          auth_type: 'password',
          password: formData.ssh_password,
        },
        config: {
          reuse_existing_config: formData.reuse_config,
          reuse_existing_munge: formData.reuse_munge,
          reuse_existing_database: formData.reuse_database,
        },
      };

      const response = await api.post('/api/slurm/clusters/connect', payload);

      if (response.data.success) {
        alert('外部集群添加成功！');
        setFormData({
          name: '',
          description: '',
          master_host: '',
          ssh_username: 'root',
          ssh_password: '',
          ssh_port: 22,
          reuse_config: true,
          reuse_munge: true,
          reuse_database: true,
        });
        setTestResult(null);
        setClusterInfo(null);
        fetchExternalClusters();
      }
    } catch (error) {
      alert(error.response?.data?.error || '添加集群失败');
    } finally {
      setLoading(false);
    }
  };

  // 删除集群
  const handleDelete = async (clusterId) => {
    if (!confirm('确定要删除此集群吗？')) return;

    try {
      await api.delete(`/api/slurm/clusters/${clusterId}`);
      alert('集群已删除');
      fetchExternalClusters();
    } catch (error) {
      alert(error.response?.data?.error || '删除失败');
    }
  };

  // 刷新集群信息
  const handleRefresh = async (clusterId) => {
    try {
      await api.post(`/api/slurm/clusters/${clusterId}/refresh`);
      alert('刷新成功');
      fetchExternalClusters();
    } catch (error) {
      alert(error.response?.data?.error || '刷新失败');
    }
  };

  return (
    <div className="container mx-auto p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold">外部 SLURM 集群管理</h1>
        <p className="text-muted-foreground mt-2">
          连接和管理已存在的 SLURM 集群，复用现有配置、Munge 密钥和数据库
        </p>
      </div>

      <Tabs defaultValue="add" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="add">
            <Plus className="h-4 w-4 mr-2" />
            添加集群
          </TabsTrigger>
          <TabsTrigger value="list">
            <Server className="h-4 w-4 mr-2" />
            已连接集群 ({clusters.length})
          </TabsTrigger>
        </TabsList>

        {/* 添加集群标签页 */}
        <TabsContent value="add">
          <Card>
            <CardHeader>
              <CardTitle>连接外部 SLURM 集群</CardTitle>
              <CardDescription>
                通过 SSH 连接到已运行的 SLURM 集群，自动发现节点和配置
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-6">
                {/* 基本信息 */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg flex items-center">
                      <Server className="h-5 w-5 mr-2" />
                      基本信息
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div>
                      <Label htmlFor="name">集群名称 *</Label>
                      <Input
                        id="name"
                        name="name"
                        value={formData.name}
                        onChange={handleInputChange}
                        placeholder="例如: 生产集群-01"
                        required
                      />
                    </div>

                    <div>
                      <Label htmlFor="description">描述</Label>
                      <Textarea
                        id="description"
                        name="description"
                        value={formData.description}
                        onChange={handleInputChange}
                        placeholder="集群用途说明..."
                        rows={3}
                      />
                    </div>

                    <div>
                      <Label htmlFor="master_host">Master 节点地址 *</Label>
                      <Input
                        id="master_host"
                        name="master_host"
                        value={formData.master_host}
                        onChange={handleInputChange}
                        placeholder="192.168.1.100 或 slurm-master.example.com"
                        required
                      />
                    </div>
                  </CardContent>
                </Card>

                {/* SSH 连接配置 */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg flex items-center">
                      <Key className="h-5 w-5 mr-2" />
                      SSH 连接配置
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label htmlFor="ssh_username">SSH 用户名 *</Label>
                        <Input
                          id="ssh_username"
                          name="ssh_username"
                          value={formData.ssh_username}
                          onChange={handleInputChange}
                          required
                        />
                      </div>

                      <div>
                        <Label htmlFor="ssh_port">SSH 端口</Label>
                        <Input
                          id="ssh_port"
                          name="ssh_port"
                          type="number"
                          value={formData.ssh_port}
                          onChange={handleInputChange}
                        />
                      </div>
                    </div>

                    <div>
                      <Label htmlFor="ssh_password">SSH 密码 *</Label>
                      <Input
                        id="ssh_password"
                        name="ssh_password"
                        type="password"
                        value={formData.ssh_password}
                        onChange={handleInputChange}
                        placeholder="SSH 登录密码"
                        required
                      />
                    </div>

                    <Button
                      type="button"
                      variant="outline"
                      onClick={testConnection}
                      disabled={!formData.master_host || !formData.ssh_password || testingConnection}
                    >
                      {testingConnection ? (
                        <>
                          <Loader className="h-4 w-4 mr-2 animate-spin" />
                          测试中...
                        </>
                      ) : (
                        '测试连接'
                      )}
                    </Button>

                    {testResult && (
                      <Alert variant={testResult.success ? 'default' : 'destructive'}>
                        {testResult.success ? (
                          <CheckCircle className="h-4 w-4" />
                        ) : (
                          <AlertCircle className="h-4 w-4" />
                        )}
                        <AlertDescription>{testResult.message}</AlertDescription>
                      </Alert>
                    )}

                    {clusterInfo && (
                      <Card className="bg-muted">
                        <CardContent className="pt-4">
                          <div className="space-y-2 text-sm">
                            <div><strong>SLURM 版本:</strong> {clusterInfo.slurm_version}</div>
                            <div><strong>集群名称:</strong> {clusterInfo.cluster_name}</div>
                            <div><strong>节点数量:</strong> {clusterInfo.node_count}</div>
                            <div><strong>控制器:</strong> {clusterInfo.controller_host}</div>
                          </div>
                        </CardContent>
                      </Card>
                    )}
                  </CardContent>
                </Card>

                {/* 配置复用选项 */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg flex items-center">
                      <Database className="h-5 w-5 mr-2" />
                      配置复用选项
                    </CardTitle>
                    <CardDescription>
                      复用集群已有的配置和服务，无需重新配置
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="reuse_config"
                        name="reuse_config"
                        checked={formData.reuse_config}
                        onChange={handleInputChange}
                        className="h-4 w-4"
                      />
                      <Label htmlFor="reuse_config" className="font-normal">
                        复用现有 SLURM 配置文件（slurm.conf）
                      </Label>
                    </div>

                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="reuse_munge"
                        name="reuse_munge"
                        checked={formData.reuse_munge}
                        onChange={handleInputChange}
                        className="h-4 w-4"
                      />
                      <Label htmlFor="reuse_munge" className="font-normal">
                        复用现有 Munge 密钥（/etc/munge/munge.key）
                      </Label>
                    </div>

                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="reuse_database"
                        name="reuse_database"
                        checked={formData.reuse_database}
                        onChange={handleInputChange}
                        className="h-4 w-4"
                      />
                      <Label htmlFor="reuse_database" className="font-normal">
                        复用现有数据库配置（slurmdbd.conf）
                      </Label>
                    </div>
                  </CardContent>
                </Card>

                {/* 提交按钮 */}
                <div className="flex justify-end space-x-4">
                  <Button type="button" variant="outline" onClick={() => {
                    setFormData({
                      name: '',
                      description: '',
                      master_host: '',
                      ssh_username: 'root',
                      ssh_password: '',
                      ssh_port: 22,
                      reuse_config: true,
                      reuse_munge: true,
                      reuse_database: true,
                    });
                    setTestResult(null);
                    setClusterInfo(null);
                  }}>
                    重置
                  </Button>
                  <Button type="submit" disabled={loading || !testResult?.success}>
                    {loading ? (
                      <>
                        <Loader className="h-4 w-4 mr-2 animate-spin" />
                        添加中...
                      </>
                    ) : (
                      '添加集群'
                    )}
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 已连接集群列表标签页 */}
        <TabsContent value="list">
          <div className="space-y-4">
            {loading ? (
              <div className="text-center py-12">
                <Loader className="h-8 w-8 animate-spin mx-auto mb-4" />
                <p className="text-muted-foreground">加载中...</p>
              </div>
            ) : clusters.length === 0 ? (
              <Card>
                <CardContent className="py-12 text-center">
                  <Server className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                  <p className="text-muted-foreground">暂无外部集群</p>
                  <p className="text-sm text-muted-foreground mt-2">
                    点击"添加集群"标签页开始连接外部 SLURM 集群
                  </p>
                </CardContent>
              </Card>
            ) : (
              clusters.map((cluster) => (
                <Card key={cluster.id}>
                  <CardHeader>
                    <div className="flex items-start justify-between">
                      <div>
                        <CardTitle className="flex items-center">
                          {cluster.name}
                          <Badge variant="secondary" className="ml-2">
                            外部集群
                          </Badge>
                        </CardTitle>
                        <CardDescription className="mt-2">
                          {cluster.description}
                        </CardDescription>
                      </div>
                      <div className="flex space-x-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleRefresh(cluster.id)}
                        >
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleDelete(cluster.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <strong>Master 节点:</strong> {cluster.master_host}
                      </div>
                      <div>
                        <strong>节点数量:</strong> {cluster.nodes?.length || 0}
                      </div>
                      <div>
                        <strong>状态:</strong> <Badge>{cluster.status}</Badge>
                      </div>
                      <div>
                        <strong>创建时间:</strong> {new Date(cluster.created_at).toLocaleDateString()}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default ExternalClusterManagement;
