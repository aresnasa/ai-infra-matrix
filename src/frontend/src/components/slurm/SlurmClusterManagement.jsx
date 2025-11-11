import React, { useState, useEffect } from 'react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Plus,
  Play,
  Pause,
  Settings,
  Monitor,
  AlertCircle,
  CheckCircle,
  Clock,
  Loader,
  Trash2,
  Scale,
  Terminal,
} from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import api from '@/lib/api';
import CreateClusterDialog from './CreateClusterDialog';
import DeploymentProgressDialog from './DeploymentProgressDialog';
import ClusterDetailsDialog from './ClusterDetailsDialog';
import ScaleClusterDialog from './ScaleClusterDialog';
import ConnectExternalClusterDialog from './ConnectExternalClusterDialog';

const SlurmClusterManagement = () => {
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [deploymentStatus, setDeploymentStatus] = useState({});
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showConnectDialog, setShowConnectDialog] = useState(false);
  const [showProgressDialog, setShowProgressDialog] = useState(false);
  const [showDetailsDialog, setShowDetailsDialog] = useState(false);
  const [showScaleDialog, setShowScaleDialog] = useState(false);
  const [currentDeploymentId, setCurrentDeploymentId] = useState(null);
  const { toast } = useToast();

  useEffect(() => {
    fetchClusters();
  }, []);

  const fetchClusters = async () => {
    try {
      setLoading(true);
      const response = await api.get('/api/slurm/clusters');
      setClusters(response.data.clusters || []);
    } catch (error) {
      console.error('获取集群列表失败:', error);
      toast({
        title: '错误',
        description: '获取集群列表失败',
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateCluster = async (clusterData) => {
    try {
      const response = await api.post('/api/slurm/clusters', clusterData);
      if (response.data.success) {
        toast({
          title: '成功',
          description: '集群创建成功',
        });
        setShowCreateDialog(false);
        fetchClusters();
      }
    } catch (error) {
      console.error('创建集群失败:', error);
      toast({
        title: '错误',
        description: error.response?.data?.message || '创建集群失败',
        variant: 'destructive',
      });
    }
  };

  const handleDeployCluster = async (clusterId, action = 'deploy') => {
    try {
      const response = await api.post('/api/slurm/clusters/deploy', {
        cluster_id: clusterId,
        action: action,
        config: {
          parallel: 3,
          timeout: 3600,
          retry_count: 3,
        },
      });

      if (response.data.success) {
        const deploymentId = response.data.deployment_id;
        setCurrentDeploymentId(deploymentId);
        setShowProgressDialog(true);
        toast({
          title: '成功',
          description: `${action === 'deploy' ? '部署' : '操作'}已开始`,
        });

        // 开始轮询部署状态
        startDeploymentPolling(deploymentId);
      }
    } catch (error) {
      console.error('部署失败:', error);
      toast({
        title: '错误',
        description: error.response?.data?.message || '部署失败',
        variant: 'destructive',
      });
    }
  };

  const startDeploymentPolling = (deploymentId) => {
    const interval = setInterval(async () => {
      try {
        const response = await api.get(`/api/slurm/deployments/${deploymentId}/status`);
        const status = response.data;

        setDeploymentStatus(prev => ({
          ...prev,
          [deploymentId]: status,
        }));

        // 如果部署完成或失败，停止轮询
        if (status.status === 'completed' || status.status === 'failed') {
          clearInterval(interval);
          fetchClusters(); // 刷新集群列表

          if (status.status === 'completed') {
            toast({
              title: '成功',
              description: '部署完成',
            });
          } else {
            toast({
              title: '失败',
              description: '部署失败',
              variant: 'destructive',
            });
          }
        }
      } catch (error) {
        console.error('获取部署状态失败:', error);
        clearInterval(interval);
      }
    }, 2000);

    return interval;
  };

  const handleScaleCluster = async (scaleData) => {
    try {
      const response = await api.post('/api/slurm/clusters/scale', scaleData);

      if (response.data.success) {
        const deploymentId = response.data.deployment_id;
        setCurrentDeploymentId(deploymentId);
        setShowScaleDialog(false);
        setShowProgressDialog(true);

        toast({
          title: '成功',
          description: '扩缩容操作已开始',
        });

        startDeploymentPolling(deploymentId);
      }
    } catch (error) {
      console.error('扩缩容失败:', error);
      toast({
        title: '错误',
        description: error.response?.data?.message || '扩缩容失败',
        variant: 'destructive',
      });
    }
  };

  const getStatusBadge = (status) => {
    const statusConfig = {
      pending: { variant: 'secondary', icon: Clock, text: '待部署' },
      deploying: { variant: 'default', icon: Loader, text: '部署中' },
      running: { variant: 'success', icon: CheckCircle, text: '运行中' },
      scaling: { variant: 'default', icon: Scale, text: '扩缩容中' },
      failed: { variant: 'destructive', icon: AlertCircle, text: '失败' },
      stopped: { variant: 'secondary', icon: Pause, text: '已停止' },
    };

    const config = statusConfig[status] || statusConfig.pending;
    const Icon = config.icon;

    return (
      <Badge variant={config.variant} className="flex items-center gap-1">
        <Icon className="h-3 w-3" />
        {config.text}
      </Badge>
    );
  };

  const openClusterDetails = (cluster) => {
    setSelectedCluster(cluster);
    setShowDetailsDialog(true);
  };

  const openScaleDialog = (cluster) => {
    setSelectedCluster(cluster);
    setShowScaleDialog(true);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Monitor className="h-5 w-5" />
                SLURM 集群管理
              </CardTitle>
              <p className="text-sm text-muted-foreground mt-1">
                管理和监控 SLURM 高性能计算集群
              </p>
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => setShowConnectDialog(true)}
                className="flex items-center gap-2"
              >
                <Plus className="h-4 w-4" />
                连接已有集群
              </Button>
              <Button
                onClick={() => setShowCreateDialog(true)}
                className="flex items-center gap-2"
              >
                <Plus className="h-4 w-4" />
                创建新集群
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {clusters.length === 0 ? (
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                暂无集群，点击"创建新集群"或"连接已有集群"开始使用
              </AlertDescription>
            </Alert>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>集群名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>Master节点</TableHead>
                  <TableHead>节点数量</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {clusters.map((cluster) => (
                  <TableRow key={cluster.id}>
                    <TableCell>
                      <div>
                        <div className="font-medium">{cluster.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {cluster.description}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={cluster.cluster_type === 'external' ? 'secondary' : 'default'}>
                        {cluster.cluster_type === 'external' ? '外部集群' : '托管集群'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {getStatusBadge(cluster.status)}
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <div>{cluster.master_host}</div>
                        <div className="text-muted-foreground">
                          Salt Master: {cluster.salt_master}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {cluster.nodes?.length || 0} 个节点
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(cluster.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2 justify-end">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openClusterDetails(cluster)}
                        >
                          <Settings className="h-4 w-4" />
                        </Button>

                        {/* 托管集群显示部署和扩容按钮 */}
                        {cluster.cluster_type !== 'external' && cluster.status === 'pending' && (
                          <Button
                            size="sm"
                            onClick={() => handleDeployCluster(cluster.id)}
                          >
                            <Play className="h-4 w-4" />
                          </Button>
                        )}

                        {cluster.cluster_type !== 'external' && cluster.status === 'running' && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openScaleDialog(cluster)}
                          >
                            <Scale className="h-4 w-4" />
                          </Button>
                        )}

                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {/* TODO: 实现日志查看 */}}
                        >
                          <Terminal className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 创建集群对话框 */}
      <CreateClusterDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSubmit={handleCreateCluster}
      />

      {/* 连接外部集群对话框 */}
      <ConnectExternalClusterDialog
        open={showConnectDialog}
        onOpenChange={setShowConnectDialog}
        onSuccess={() => {
          fetchClusters();
        }}
      />

      {/* 部署进度对话框 */}
      <DeploymentProgressDialog
        open={showProgressDialog}
        onOpenChange={setShowProgressDialog}
        deploymentId={currentDeploymentId}
        deploymentStatus={deploymentStatus[currentDeploymentId]}
      />

      {/* 集群详情对话框 */}
      <ClusterDetailsDialog
        open={showDetailsDialog}
        onOpenChange={setShowDetailsDialog}
        cluster={selectedCluster}
        onDeploy={(clusterId, action) => {
          setShowDetailsDialog(false);
          handleDeployCluster(clusterId, action);
        }}
      />

      {/* 扩缩容对话框 */}
      <ScaleClusterDialog
        open={showScaleDialog}
        onOpenChange={setShowScaleDialog}
        cluster={selectedCluster}
        onSubmit={handleScaleCluster}
      />
    </div>
  );
};

export default SlurmClusterManagement;
