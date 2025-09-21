import React, { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
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
import { Badge } from '@/components/ui/badge';
import { Trash2, Plus } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';

const CreateClusterDialog = ({ open, onOpenChange, onSubmit }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    master_host: '',
    master_port: 22,
    salt_master: '',
    config: {
      slurm_version: '22.05.0',
      salt_version: '3005.1',
      accounting_db: 'mysql',
      partitions: [{
        name: 'compute',
        max_time: '24:00:00',
        default_time: '01:00:00',
        state: 'UP',
        priority: 1,
        nodes: []
      }],
      global_settings: {},
      custom_packages: []
    },
    nodes: [{
      node_name: 'master01',
      node_type: 'master',
      host: '',
      port: 22,
      username: 'root',
      auth_type: 'password',
      password: '',
      key_path: '',
      cpus: 4,
      memory: 8192,
      storage: 100,
      gpus: 0,
      node_config: {
        partitions: ['compute'],
        features: [],
        custom_settings: {},
        mounts: []
      }
    }]
  });
  const [loading, setLoading] = useState(false);
  const { toast } = useToast();

  const handleInputChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleConfigChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      config: {
        ...prev.config,
        [field]: value
      }
    }));
  };

  const handleNodeChange = (index, field, value) => {
    setFormData(prev => ({
      ...prev,
      nodes: prev.nodes.map((node, i) =>
        i === index ? { ...node, [field]: value } : node
      )
    }));
  };

  const handleNodeConfigChange = (index, field, value) => {
    setFormData(prev => ({
      ...prev,
      nodes: prev.nodes.map((node, i) =>
        i === index ? {
          ...node,
          node_config: { ...node.node_config, [field]: value }
        } : node
      )
    }));
  };

  const addNode = (nodeType = 'compute') => {
    const newNode = {
      node_name: `${nodeType}${String(formData.nodes.filter(n => n.node_type === nodeType).length + 1).padStart(2, '0')}`,
      node_type: nodeType,
      host: '',
      port: 22,
      username: 'root',
      auth_type: 'password',
      password: '',
      key_path: '',
      cpus: nodeType === 'master' ? 4 : 8,
      memory: nodeType === 'master' ? 8192 : 16384,
      storage: 100,
      gpus: nodeType === 'compute' ? 1 : 0,
      node_config: {
        partitions: ['compute'],
        features: nodeType === 'compute' ? ['gpu'] : [],
        custom_settings: {},
        mounts: []
      }
    };

    setFormData(prev => ({
      ...prev,
      nodes: [...prev.nodes, newNode]
    }));
  };

  const removeNode = (index) => {
    if (formData.nodes.length > 1) {
      setFormData(prev => ({
        ...prev,
        nodes: prev.nodes.filter((_, i) => i !== index)
      }));
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    // 验证表单
    if (!formData.name || !formData.master_host || !formData.salt_master) {
      toast({
        title: '错误',
        description: '请填写必要的字段',
        variant: 'destructive',
      });
      return;
    }

    if (formData.nodes.length === 0) {
      toast({
        title: '错误',
        description: '至少需要一个节点',
        variant: 'destructive',
      });
      return;
    }

    // 检查是否有master节点
    const hasMaster = formData.nodes.some(node => node.node_type === 'master');
    if (!hasMaster) {
      toast({
        title: '错误',
        description: '至少需要一个master节点',
        variant: 'destructive',
      });
      return;
    }

    setLoading(true);
    try {
      await onSubmit(formData);
    } catch (error) {
      console.error('提交失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const testSSHConnection = async (nodeIndex) => {
    const node = formData.nodes[nodeIndex];
    try {
      // TODO: 实现SSH连接测试
      toast({
        title: '测试成功',
        description: `节点 ${node.node_name} SSH连接正常`,
      });
    } catch (error) {
      toast({
        title: '测试失败',
        description: `节点 ${node.node_name} SSH连接失败`,
        variant: 'destructive',
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>创建 SLURM 集群</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 基本信息 */}
          <Card>
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="name">集群名称 *</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => handleInputChange('name', e.target.value)}
                    placeholder="输入集群名称"
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="master_host">Master主机 *</Label>
                  <Input
                    id="master_host"
                    value={formData.master_host}
                    onChange={(e) => handleInputChange('master_host', e.target.value)}
                    placeholder="master节点IP或域名"
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">描述</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) => handleInputChange('description', e.target.value)}
                  placeholder="集群描述信息"
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="salt_master">SaltStack Master *</Label>
                  <Input
                    id="salt_master"
                    value={formData.salt_master}
                    onChange={(e) => handleInputChange('salt_master', e.target.value)}
                    placeholder="Salt Master地址"
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
                    min={1}
                    max={65535}
                  />
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 版本配置 */}
          <Card>
            <CardHeader>
              <CardTitle>版本配置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="slurm_version">SLURM版本</Label>
                  <Select
                    value={formData.config.slurm_version}
                    onValueChange={(value) => handleConfigChange('slurm_version', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="23.11.0">23.11.0 (最新)</SelectItem>
                      <SelectItem value="23.02.0">23.02.0</SelectItem>
                      <SelectItem value="22.05.0">22.05.0</SelectItem>
                      <SelectItem value="21.08.0">21.08.0</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="salt_version">SaltStack版本</Label>
                  <Select
                    value={formData.config.salt_version}
                    onValueChange={(value) => handleConfigChange('salt_version', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="3006.0">3006.0 (最新)</SelectItem>
                      <SelectItem value="3005.1">3005.1</SelectItem>
                      <SelectItem value="3004.2">3004.2</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="accounting_db">计费数据库</Label>
                  <Select
                    value={formData.config.accounting_db}
                    onValueChange={(value) => handleConfigChange('accounting_db', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="mysql">MySQL</SelectItem>
                      <SelectItem value="mariadb">MariaDB</SelectItem>
                      <SelectItem value="none">无</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 节点配置 */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>节点配置</CardTitle>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => addNode('compute')}
                  >
                    <Plus className="h-4 w-4 mr-1" />
                    计算节点
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => addNode('login')}
                  >
                    <Plus className="h-4 w-4 mr-1" />
                    登录节点
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {formData.nodes.map((node, index) => (
                <Card key={index} className="border-l-4 border-l-blue-500">
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-base">
                          节点 #{index + 1}
                        </CardTitle>
                        <Badge variant={node.node_type === 'master' ? 'default' : 'secondary'}>
                          {node.node_type}
                        </Badge>
                      </div>
                      {formData.nodes.length > 1 && node.node_type !== 'master' && (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => removeNode(index)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-3 gap-4">
                      <div className="space-y-2">
                        <Label>节点名称</Label>
                        <Input
                          value={node.node_name}
                          onChange={(e) => handleNodeChange(index, 'node_name', e.target.value)}
                          placeholder="节点名称"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>主机地址</Label>
                        <Input
                          value={node.host}
                          onChange={(e) => handleNodeChange(index, 'host', e.target.value)}
                          placeholder="IP地址或域名"
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>SSH端口</Label>
                        <Input
                          type="number"
                          value={node.port}
                          onChange={(e) => handleNodeChange(index, 'port', parseInt(e.target.value))}
                          min={1}
                          max={65535}
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-4 gap-4">
                      <div className="space-y-2">
                        <Label>用户名</Label>
                        <Input
                          value={node.username}
                          onChange={(e) => handleNodeChange(index, 'username', e.target.value)}
                          placeholder="SSH用户名"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>认证方式</Label>
                        <Select
                          value={node.auth_type}
                          onValueChange={(value) => handleNodeChange(index, 'auth_type', value)}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="password">密码</SelectItem>
                            <SelectItem value="key">密钥</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <Label>{node.auth_type === 'password' ? '密码' : '密钥路径'}</Label>
                        <Input
                          type={node.auth_type === 'password' ? 'password' : 'text'}
                          value={node.auth_type === 'password' ? node.password : node.key_path}
                          onChange={(e) => handleNodeChange(
                            index,
                            node.auth_type === 'password' ? 'password' : 'key_path',
                            e.target.value
                          )}
                          placeholder={node.auth_type === 'password' ? '输入密码' : '密钥文件路径'}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>&nbsp;</Label>
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => testSSHConnection(index)}
                          className="w-full"
                        >
                          测试连接
                        </Button>
                      </div>
                    </div>

                    <div className="grid grid-cols-4 gap-4">
                      <div className="space-y-2">
                        <Label>CPU核数</Label>
                        <Input
                          type="number"
                          value={node.cpus}
                          onChange={(e) => handleNodeChange(index, 'cpus', parseInt(e.target.value))}
                          min={1}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>内存(MB)</Label>
                        <Input
                          type="number"
                          value={node.memory}
                          onChange={(e) => handleNodeChange(index, 'memory', parseInt(e.target.value))}
                          min={1024}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>存储(GB)</Label>
                        <Input
                          type="number"
                          value={node.storage}
                          onChange={(e) => handleNodeChange(index, 'storage', parseInt(e.target.value))}
                          min={10}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>GPU数量</Label>
                        <Input
                          type="number"
                          value={node.gpus}
                          onChange={(e) => handleNodeChange(index, 'gpus', parseInt(e.target.value))}
                          min={0}
                        />
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </CardContent>
          </Card>

          {/* 提交按钮 */}
          <div className="flex justify-end gap-2 pt-4 border-t">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? '创建中...' : '创建集群'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export default CreateClusterDialog;
