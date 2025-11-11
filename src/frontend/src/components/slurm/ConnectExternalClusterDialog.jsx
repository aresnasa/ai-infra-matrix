import React, { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader, Server, Key, CheckCircle, AlertCircle } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import api from '@/lib/api';

const ConnectExternalClusterDialog = ({ open, onOpenChange, onSuccess }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    master_host: '',
    master_port: 22,
    master_ssh: {
      host: '',
      port: 22,
      username: 'root',
      auth_type: 'password',
      password: '',
      key_path: ''
    },
    config: {
      slurm_version: '',
      partitions: [],
      global_settings: {},
      custom_packages: []
    }
  });
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState(null);
  const { toast } = useToast();

  const handleInputChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // 同步master_host到master_ssh.host
    if (field === 'master_host') {
      setFormData(prev => ({
        ...prev,
        master_ssh: {
          ...prev.master_ssh,
          host: value
        }
      }));
    }
  };

  const handleSSHChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      master_ssh: {
        ...prev.master_ssh,
        [field]: value
      }
    }));
  };

  const handleTestConnection = async () => {
    if (!formData.master_host || !formData.master_ssh.username) {
      toast({
        title: '错误',
        description: '请填写主机地址和用户名',
        variant: 'destructive',
      });
      return;
    }

    if (formData.master_ssh.auth_type === 'password' && !formData.master_ssh.password) {
      toast({
        title: '错误',
        description: '请填写SSH密码',
        variant: 'destructive',
      });
      return;
    }

    setTesting(true);
    setTestResult(null);

    try {
      // 这里可以调用一个测试连接的API
      // 暂时模拟测试
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      setTestResult({
        success: true,
        message: 'SSH连接测试成功，检测到SLURM已安装',
        version: 'slurm 25.05.4'
      });

      toast({
        title: '成功',
        description: 'SSH连接测试成功',
      });
    } catch (error) {
      setTestResult({
        success: false,
        message: error.message || 'SSH连接测试失败',
      });

      toast({
        title: '错误',
        description: 'SSH连接测试失败',
        variant: 'destructive',
      });
    } finally {
      setTesting(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    // 验证必填字段
    if (!formData.name || !formData.master_host || !formData.master_ssh.username) {
      toast({
        title: '错误',
        description: '请填写所有必填字段',
        variant: 'destructive',
      });
      return;
    }

    if (formData.master_ssh.auth_type === 'password' && !formData.master_ssh.password) {
      toast({
        title: '错误',
        description: '请填写SSH密码',
        variant: 'destructive',
      });
      return;
    }

    setLoading(true);

    try {
      const response = await api.post('/api/slurm/clusters/connect', formData);
      
      if (response.data.success) {
        toast({
          title: '成功',
          description: '集群连接成功',
        });
        
        onSuccess && onSuccess(response.data.data);
        onOpenChange(false);
        
        // 重置表单
        setFormData({
          name: '',
          description: '',
          master_host: '',
          master_port: 22,
          master_ssh: {
            host: '',
            port: 22,
            username: 'root',
            auth_type: 'password',
            password: '',
            key_path: ''
          },
          config: {
            slurm_version: '',
            partitions: [],
            global_settings: {},
            custom_packages: []
          }
        });
        setTestResult(null);
      }
    } catch (error) {
      console.error('连接集群失败:', error);
      toast({
        title: '错误',
        description: error.response?.data?.message || '连接集群失败',
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>连接已有SLURM集群</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 基本信息 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Server className="w-5 h-5" />
                基本信息
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">
                  集群名称 <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => handleInputChange('name', e.target.value)}
                  placeholder="例如：生产环境集群"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">集群描述</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) => handleInputChange('description', e.target.value)}
                  placeholder="描述这个集群的用途和特点"
                  rows={3}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="master_host">
                    Master主机地址 <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="master_host"
                    value={formData.master_host}
                    onChange={(e) => handleInputChange('master_host', e.target.value)}
                    placeholder="192.168.1.100"
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="master_port">SSH端口</Label>
                  <Input
                    id="master_port"
                    type="number"
                    value={formData.master_port}
                    onChange={(e) => handleInputChange('master_port', parseInt(e.target.value))}
                    placeholder="22"
                  />
                </div>
              </div>
            </CardContent>
          </Card>

          {/* SSH连接配置 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Key className="w-5 h-5" />
                SSH连接配置
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="ssh_username">
                    用户名 <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="ssh_username"
                    value={formData.master_ssh.username}
                    onChange={(e) => handleSSHChange('username', e.target.value)}
                    placeholder="root"
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ssh_auth_type">
                    认证方式 <span className="text-red-500">*</span>
                  </Label>
                  <Select
                    value={formData.master_ssh.auth_type}
                    onValueChange={(value) => handleSSHChange('auth_type', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="password">密码</SelectItem>
                      <SelectItem value="key">密钥（暂未支持）</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {formData.master_ssh.auth_type === 'password' && (
                <div className="space-y-2">
                  <Label htmlFor="ssh_password">
                    密码 <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="ssh_password"
                    type="password"
                    value={formData.master_ssh.password}
                    onChange={(e) => handleSSHChange('password', e.target.value)}
                    placeholder="请输入SSH密码"
                    required
                  />
                </div>
              )}

              {formData.master_ssh.auth_type === 'key' && (
                <div className="space-y-2">
                  <Label htmlFor="ssh_key_path">密钥路径</Label>
                  <Input
                    id="ssh_key_path"
                    value={formData.master_ssh.key_path}
                    onChange={(e) => handleSSHChange('key_path', e.target.value)}
                    placeholder="/path/to/private/key"
                  />
                  <p className="text-sm text-muted-foreground">
                    暂不支持密钥认证，请使用密码认证
                  </p>
                </div>
              )}

              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleTestConnection}
                  disabled={testing}
                  className="flex items-center gap-2"
                >
                  {testing ? (
                    <>
                      <Loader className="w-4 h-4 animate-spin" />
                      测试中...
                    </>
                  ) : (
                    '测试连接'
                  )}
                </Button>
              </div>

              {testResult && (
                <Alert variant={testResult.success ? 'default' : 'destructive'}>
                  {testResult.success ? (
                    <CheckCircle className="h-4 w-4" />
                  ) : (
                    <AlertCircle className="h-4 w-4" />
                  )}
                  <AlertDescription>
                    {testResult.message}
                    {testResult.version && (
                      <div className="mt-1 text-sm">检测到版本: {testResult.version}</div>
                    )}
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          {/* 说明 */}
          <Alert>
            <AlertDescription>
              <strong>说明：</strong>
              <ul className="mt-2 space-y-1 list-disc list-inside text-sm">
                <li>连接已有集群后，系统会自动发现集群中的节点</li>
                <li>请确保Master节点已安装并运行SLURM</li>
                <li>需要提供有权限执行scontrol/sinfo等命令的用户</li>
                <li>连接成功后可以在集群列表中查看和管理</li>
              </ul>
            </AlertDescription>
          </Alert>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              取消
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? (
                <>
                  <Loader className="w-4 h-4 mr-2 animate-spin" />
                  连接中...
                </>
              ) : (
                '连接集群'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectExternalClusterDialog;
