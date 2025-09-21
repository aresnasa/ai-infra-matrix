import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Monitor,
  Activity,
  Database,
  Network,
  HardDrive,
  Cpu,
  MemoryStick,
  Play,
  Square,
  RotateCcw,
  Settings,
  Terminal,
  CheckCircle,
  XCircle,
  Clock,
} from 'lucide-react';
import api from '@/lib/api';

const ClusterDetailsDialog = ({ open, onOpenChange, cluster, onDeploy }) => {
  const [clusterDetails, setClusterDetails] = useState(null);
  const [deployments, setDeployments] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (cluster && open) {
      fetchClusterDetails();
      fetchDeployments();
    }
  }, [cluster, open]);

  const fetchClusterDetails = async () => {
    if (!cluster?.id) return;

    try {
      setLoading(true);
      const response = await api.get(`/api/slurm/clusters/${cluster.id}`);
      setClusterDetails(response.data);
    } catch (error) {
      console.error('获取集群详情失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchDeployments = async () => {
    // TODO: 实现获取部署历史的API
    setDeployments([]);
  };

  const getNodeTypeIcon = (nodeType) => {
    switch (nodeType) {
      case 'master':
        return <Database className="h-4 w-4" />;
      case 'compute':
        return <Cpu className="h-4 w-4" />;
      case 'login':
        return <Terminal className="h-4 w-4" />;
      default:
        return <Monitor className="h-4 w-4" />;
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'active':
      case 'running':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'pending':
        return <Clock className="h-4 w-4 text-gray-500" />;
      default:
        return <Activity className="h-4 w-4 text-blue-500" />;
    }
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (!cluster) return null;

  const details = clusterDetails || cluster;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Monitor className="h-5 w-5" />
            {details.name} - 集群详情
          </DialogTitle>
        </DialogHeader>

        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="overview">概览</TabsTrigger>
            <TabsTrigger value="nodes">节点</TabsTrigger>
            <TabsTrigger value="config">配置</TabsTrigger>
            <TabsTrigger value="deployments">部署历史</TabsTrigger>
          </TabsList>

          {/* 概览标签页 */}
          <TabsContent value="overview" className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Activity className="h-5 w-5" />
                    集群状态
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span>运行状态</span>
                    <div className="flex items-center gap-2">
                      {getStatusIcon(details.status)}
                      <Badge variant={details.status === 'running' ? 'success' : 'secondary'}>
                        {details.status}
                      </Badge>
                    </div>
                  </div>

                  <div className="flex items-center justify-between">
                    <span>节点总数</span>
                    <Badge variant="outline">
                      {details.nodes?.length || 0} 个节点
                    </Badge>
                  </div>

                  <div className="flex items-center justify-between">
                    <span>创建时间</span>
                    <span className="text-sm text-muted-foreground">
                      {new Date(details.created_at).toLocaleString()}
                    </span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span>更新时间</span>
                    <span className="text-sm text-muted-foreground">
                      {new Date(details.updated_at).toLocaleString()}
                    </span>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Settings className="h-5 w-5" />
                    配置信息
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span>Master主机</span>
                    <span className="font-mono text-sm">{details.master_host}</span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span>Salt Master</span>
                    <span className="font-mono text-sm">{details.salt_master}</span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span>SLURM版本</span>
                    <Badge variant="outline">
                      {details.config?.slurm_version || 'N/A'}
                    </Badge>
                  </div>

                  <div className="flex items-center justify-between">
                    <span>Salt版本</span>
                    <Badge variant="outline">
                      {details.config?.salt_version || 'N/A'}
                    </Badge>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* 资源统计 */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">资源统计</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-4 gap-4">
                  <div className="text-center p-4 bg-blue-50 rounded-lg">
                    <Cpu className="h-8 w-8 text-blue-600 mx-auto mb-2" />
                    <div className="font-semibold text-lg">
                      {details.nodes?.reduce((sum, node) => sum + (node.cpus || 0), 0) || 0}
                    </div>
                    <div className="text-sm text-muted-foreground">CPU核心</div>
                  </div>

                  <div className="text-center p-4 bg-green-50 rounded-lg">
                    <MemoryStick className="h-8 w-8 text-green-600 mx-auto mb-2" />
                    <div className="font-semibold text-lg">
                      {formatBytes((details.nodes?.reduce((sum, node) => sum + (node.memory || 0), 0) || 0) * 1024 * 1024)}
                    </div>
                    <div className="text-sm text-muted-foreground">总内存</div>
                  </div>

                  <div className="text-center p-4 bg-purple-50 rounded-lg">
                    <HardDrive className="h-8 w-8 text-purple-600 mx-auto mb-2" />
                    <div className="font-semibold text-lg">
                      {details.nodes?.reduce((sum, node) => sum + (node.storage || 0), 0) || 0} GB
                    </div>
                    <div className="text-sm text-muted-foreground">总存储</div>
                  </div>

                  <div className="text-center p-4 bg-orange-50 rounded-lg">
                    <Monitor className="h-8 w-8 text-orange-600 mx-auto mb-2" />
                    <div className="font-semibold text-lg">
                      {details.nodes?.reduce((sum, node) => sum + (node.gpus || 0), 0) || 0}
                    </div>
                    <div className="text-sm text-muted-foreground">GPU数量</div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* 操作按钮 */}
            <div className="flex justify-center gap-4 pt-4 border-t">
              {details.status === 'pending' && (
                <Button
                  onClick={() => onDeploy(details.id, 'deploy')}
                  className="flex items-center gap-2"
                >
                  <Play className="h-4 w-4" />
                  部署集群
                </Button>
              )}

              {details.status === 'running' && (
                <>
                  <Button
                    variant="outline"
                    onClick={() => onDeploy(details.id, 'restart')}
                    className="flex items-center gap-2"
                  >
                    <RotateCcw className="h-4 w-4" />
                    重启服务
                  </Button>

                  <Button
                    variant="outline"
                    onClick={() => onDeploy(details.id, 'stop')}
                    className="flex items-center gap-2"
                  >
                    <Square className="h-4 w-4" />
                    停止集群
                  </Button>
                </>
              )}

              {details.status === 'failed' && (
                <Button
                  onClick={() => onDeploy(details.id, 'deploy')}
                  className="flex items-center gap-2"
                >
                  <Play className="h-4 w-4" />
                  重新部署
                </Button>
              )}
            </div>
          </TabsContent>

          {/* 节点标签页 */}
          <TabsContent value="nodes">
            <Card>
              <CardHeader>
                <CardTitle>节点列表</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>节点名称</TableHead>
                      <TableHead>类型</TableHead>
                      <TableHead>主机地址</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>资源配置</TableHead>
                      <TableHead>Salt Minion ID</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {details.nodes?.map((node) => (
                      <TableRow key={node.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getNodeTypeIcon(node.node_type)}
                            <span className="font-medium">{node.node_name}</span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={node.node_type === 'master' ? 'default' : 'secondary'}>
                            {node.node_type}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="font-mono text-sm">
                            <div>{node.host}:{node.port}</div>
                            <div className="text-muted-foreground">
                              {node.username}@{node.host}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getStatusIcon(node.status)}
                            <span className="text-sm">{node.status}</span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm space-y-1">
                            <div>CPU: {node.cpus}核</div>
                            <div>内存: {formatBytes(node.memory * 1024 * 1024)}</div>
                            <div>存储: {node.storage}GB</div>
                            {node.gpus > 0 && <div>GPU: {node.gpus}块</div>}
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="font-mono text-sm">{node.salt_minion_id}</span>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 配置标签页 */}
          <TabsContent value="config">
            <Card>
              <CardHeader>
                <CardTitle>集群配置</CardTitle>
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-96">
                  <pre className="text-sm bg-gray-50 p-4 rounded-lg">
                    {JSON.stringify(details.config, null, 2)}
                  </pre>
                </ScrollArea>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 部署历史标签页 */}
          <TabsContent value="deployments">
            <Card>
              <CardHeader>
                <CardTitle>部署历史</CardTitle>
              </CardHeader>
              <CardContent>
                {deployments.length === 0 ? (
                  <div className="text-center text-muted-foreground py-8">
                    暂无部署记录
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>部署ID</TableHead>
                        <TableHead>操作类型</TableHead>
                        <TableHead>状态</TableHead>
                        <TableHead>开始时间</TableHead>
                        <TableHead>完成时间</TableHead>
                        <TableHead>操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {deployments.map((deployment) => (
                        <TableRow key={deployment.id}>
                          <TableCell className="font-mono text-sm">
                            {deployment.deployment_id}
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">{deployment.action}</Badge>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              {getStatusIcon(deployment.status)}
                              <span>{deployment.status}</span>
                            </div>
                          </TableCell>
                          <TableCell className="text-sm">
                            {deployment.started_at && new Date(deployment.started_at).toLocaleString()}
                          </TableCell>
                          <TableCell className="text-sm">
                            {deployment.completed_at && new Date(deployment.completed_at).toLocaleString()}
                          </TableCell>
                          <TableCell>
                            <Button variant="outline" size="sm">
                              <Terminal className="h-4 w-4 mr-1" />
                              查看日志
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        <div className="flex justify-end pt-4 border-t">
          <Button onClick={() => onOpenChange(false)}>关闭</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ClusterDetailsDialog;
