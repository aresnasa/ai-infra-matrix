import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import {
  CheckCircle,
  AlertCircle,
  Clock,
  Loader,
  Terminal,
  Play,
  XCircle,
  Activity,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';

const DeploymentProgressDialog = ({ open, onOpenChange, deploymentId, deploymentStatus }) => {
  const [logs, setLogs] = useState([]);
  const [showDetailedLogs, setShowDetailedLogs] = useState(false);

  useEffect(() => {
    if (deploymentId && open) {
      // 这里可以获取详细的部署日志
      fetchDeploymentLogs();
    }
  }, [deploymentId, open]);

  const fetchDeploymentLogs = async () => {
    try {
      // TODO: 实现获取详细部署日志的API调用
      // const response = await api.get(`/api/slurm/deployments/${deploymentId}/logs`);
      // setLogs(response.data.ssh_logs || []);
    } catch (error) {
      console.error('获取部署日志失败:', error);
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'failed':
        return <XCircle className="h-5 w-5 text-red-500" />;
      case 'running':
        return <Loader className="h-5 w-5 text-blue-500 animate-spin" />;
      case 'pending':
        return <Clock className="h-5 w-5 text-gray-500" />;
      default:
        return <Activity className="h-5 w-5 text-gray-500" />;
    }
  };

  const getNodeStatusBadge = (status) => {
    const statusConfig = {
      pending: { variant: 'secondary', text: '等待中' },
      running: { variant: 'default', text: '执行中' },
      completed: { variant: 'success', text: '已完成' },
      failed: { variant: 'destructive', text: '失败' },
    };

    const config = statusConfig[status] || statusConfig.pending;

    return (
      <Badge variant={config.variant}>
        {config.text}
      </Badge>
    );
  };

  const formatDuration = (seconds) => {
    if (!seconds) return '0s';

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    } else {
      return `${secs}s`;
    }
  };

  if (!deploymentStatus) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>部署进度</DialogTitle>
          </DialogHeader>
          <div className="flex items-center justify-center h-32">
            <Loader className="h-8 w-8 animate-spin" />
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {getStatusIcon(deploymentStatus.status)}
            部署进度 - {deploymentStatus.deployment_id}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-6">
          {/* 总体进度 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">总体进度</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span>进度: {deploymentStatus.progress || 0}%</span>
                  <span className="text-muted-foreground">
                    状态: {deploymentStatus.status}
                  </span>
                </div>
                <Progress value={deploymentStatus.progress || 0} className="h-2" />
              </div>

              <div className="text-sm text-muted-foreground">
                当前步骤: {deploymentStatus.current_step || '等待开始'}
              </div>

              {deploymentStatus.status === 'failed' && (
                <Alert variant="destructive">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    部署失败，请检查日志获取详细错误信息
                  </AlertDescription>
                </Alert>
              )}

              {deploymentStatus.status === 'completed' && (
                <Alert>
                  <CheckCircle className="h-4 w-4" />
                  <AlertDescription>
                    集群部署成功完成！
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          {/* 节点进度 */}
          {deploymentStatus.node_tasks && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">节点进度</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4">
                  {Object.entries(deploymentStatus.node_tasks).map(([nodeId, nodeTask]) => (
                    <Card key={nodeId} className="border-l-4 border-l-blue-500">
                      <CardContent className="pt-4">
                        <div className="flex items-center justify-between mb-2">
                          <div className="font-medium">节点 #{nodeId}</div>
                          {getNodeStatusBadge(nodeTask.status)}
                        </div>

                        <div className="space-y-2">
                          <div className="flex items-center justify-between text-sm">
                            <span>进度: {nodeTask.progress || 0}%</span>
                            <span className="text-muted-foreground">
                              {nodeTask.current_step || '等待中'}
                            </span>
                          </div>
                          <Progress value={nodeTask.progress || 0} className="h-1" />
                        </div>

                        {nodeTask.error && (
                          <div className="mt-2 text-sm text-red-600 bg-red-50 p-2 rounded">
                            错误: {nodeTask.error}
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* 详细日志 */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">部署日志</CardTitle>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowDetailedLogs(!showDetailedLogs)}
                >
                  <Terminal className="h-4 w-4 mr-1" />
                  {showDetailedLogs ? '隐藏' : '显示'}详细日志
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {showDetailedLogs ? (
                <ScrollArea className="h-64 w-full border rounded p-2 bg-gray-50">
                  <div className="space-y-2 text-sm font-mono">
                    {logs.length > 0 ? (
                      logs.map((log, index) => (
                        <div key={index} className="flex gap-2">
                          <span className="text-muted-foreground whitespace-nowrap">
                            {new Date(log.started_at).toLocaleTimeString()}
                          </span>
                          <span className={log.success ? 'text-green-600' : 'text-red-600'}>
                            [{log.host}]
                          </span>
                          <span className="break-all">
                            {log.command.slice(0, 100)}
                            {log.command.length > 100 && '...'}
                          </span>
                        </div>
                      ))
                    ) : (
                      <div className="text-center text-muted-foreground py-8">
                        暂无详细日志
                      </div>
                    )}
                  </div>
                </ScrollArea>
              ) : (
                <div className="text-center text-muted-foreground py-4">
                  点击"显示详细日志"查看SSH执行记录
                </div>
              )}
            </CardContent>
          </Card>

          {/* 统计信息 */}
          {deploymentStatus.started_at && (
            <div className="grid grid-cols-3 gap-4 text-sm">
              <div className="text-center">
                <div className="font-medium">开始时间</div>
                <div className="text-muted-foreground">
                  {new Date(deploymentStatus.started_at).toLocaleString()}
                </div>
              </div>
              <div className="text-center">
                <div className="font-medium">更新时间</div>
                <div className="text-muted-foreground">
                  {new Date(deploymentStatus.updated_at).toLocaleString()}
                </div>
              </div>
              <div className="text-center">
                <div className="font-medium">运行时长</div>
                <div className="text-muted-foreground">
                  {formatDuration(
                    Math.floor((new Date(deploymentStatus.updated_at) - new Date(deploymentStatus.started_at)) / 1000)
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        <div className="flex justify-end pt-4 border-t">
          <Button onClick={() => onOpenChange(false)}>
            关闭
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default DeploymentProgressDialog;
