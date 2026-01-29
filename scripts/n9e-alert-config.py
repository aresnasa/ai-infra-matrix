#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Nightingale (N9E) 全链路监控和告警配置工具

此脚本用于自动化配置 Nightingale 的监控和告警规则，支持：
1. 创建/管理业务组 (Busi Groups)
2. 创建/管理监控目标 (Targets)
3. 创建/管理告警规则 (Alert Rules)
4. 创建/管理告警屏蔽规则 (Alert Mutes)
5. 创建/管理告警订阅 (Alert Subscribes)
6. 创建/管理仪表盘 (Dashboards)
7. 管理通知渠道 (Notify Channels)

Usage:
    python scripts/n9e-alert-config.py [command] [options]

Commands:
    init                初始化监控配置（创建业务组、导入预设规则等）
    add-rule            添加告警规则
    add-mute            添加告警屏蔽规则
    add-subscribe       添加告警订阅
    list-rules          列出告警规则
    list-groups         列出业务组
    import-rules        从YAML文件导入告警规则
    export-rules        导出告警规则到YAML文件
    test-notify         测试通知渠道
    setup-categraf      配置 Categraf 采集器监控

Examples:
    # 初始化监控配置
    python scripts/n9e-alert-config.py init

    # 添加告警规则
    python scripts/n9e-alert-config.py add-rule --name "CPU使用率告警" --prom-ql 'cpu_usage_active > 80'

    # 从YAML导入告警规则
    python scripts/n9e-alert-config.py import-rules --file rules.yaml

    # 配置 Categraf 监控
    python scripts/n9e-alert-config.py setup-categraf --targets host1,host2,host3
"""

import os
import sys
import json
import yaml
import argparse
import requests
from typing import Optional, List, Dict, Any
from dataclasses import dataclass, field, asdict
from datetime import datetime
import logging
from pathlib import Path
from dotenv import load_dotenv

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


# ============================================
# 配置类
# ============================================

@dataclass
class N9EConfig:
    """Nightingale 配置
    
    支持两种认证模式：
    1. JWT 模式（默认）：通过 /api/n9e/auth/login 登录获取 token
       - 适用于 Web 访问
       - 配置：api_mode = "web"
    2. Service API 模式：使用 Basic Auth 直接访问 /v1/n9e/*
       - 适用于脚本自动化
       - 配置：api_mode = "service"
       - 需要在 Nightingale config.toml 中启用 [HTTP.APIForService]
    """
    host: str = "localhost"
    port: int = 80  # 通过 nginx 访问
    username: str = "n9e-api"  # Service API 默认用户名
    password: str = "123456"   # Service API 默认密码
    timeout: int = 30
    ssl: bool = False
    api_mode: str = "service"  # "web" 或 "service"
    
    @property
    def base_url(self) -> str:
        """获取 API 基础 URL"""
        protocol = "https" if self.ssl else "http"
        if self.api_mode == "service":
            # Service API 端点（Basic Auth）
            return f"{protocol}://{self.host}:{self.port}/v1/n9e"
        else:
            # Web API 端点（JWT Auth）
            return f"{protocol}://{self.host}:{self.port}/api/n9e"
    
    @classmethod
    def from_env(cls) -> 'N9EConfig':
        """从环境变量加载配置"""
        # 尝试从项目根目录加载 .env 文件
        env_path = Path(__file__).parent.parent / '.env'
        if env_path.exists():
            load_dotenv(env_path)
        
        return cls(
            host=os.getenv('NIGHTINGALE_HOST', os.getenv('N9E_HOST', 'localhost')),
            port=int(os.getenv('NIGHTINGALE_PORT', os.getenv('N9E_PORT', '80'))),
            username=os.getenv('N9E_API_USER', os.getenv('N9E_USERNAME', 'n9e-api')),
            password=os.getenv('N9E_API_PASSWORD', os.getenv('N9E_PASSWORD', '123456')),
            timeout=int(os.getenv('N9E_TIMEOUT', '30')),
            ssl=os.getenv('N9E_SSL', 'false').lower() == 'true',
            api_mode=os.getenv('N9E_API_MODE', 'service')  # 默认使用 service API
        )


@dataclass
class AlertRule:
    """告警规则"""
    name: str
    note: str = ""
    prod: str = ""
    cate: str = "prometheus"
    algorithm: str = ""
    prom_ql: str = ""
    severity: int = 2  # 1:紧急 2:警告 3:通知
    disabled: int = 0  # 0:启用 1:禁用
    prom_eval_interval: int = 15
    recover_duration: int = 60
    notify_recovered: int = 1
    notify_repeat_step: int = 60
    notify_max_number: int = 0
    append_tags: List[str] = field(default_factory=list)
    annotations: Dict[str, str] = field(default_factory=dict)
    datasource_queries: List[Dict] = field(default_factory=lambda: [{"name": "$all"}])
    enable_stime: str = "00:00"
    enable_etime: str = "23:59"
    enable_days_of_week: List[int] = field(default_factory=lambda: [0, 1, 2, 3, 4, 5, 6])
    rule_config: Dict = field(default_factory=dict)
    
    def to_api_dict(self) -> Dict[str, Any]:
        """转换为API请求格式"""
        result = {
            "name": self.name,
            "note": self.note,
            "prod": self.prod,
            "cate": self.cate,
            "algorithm": self.algorithm,
            "prom_ql": self.prom_ql,
            "severity": self.severity,
            "disabled": self.disabled,
            "prom_eval_interval": self.prom_eval_interval,
            "recover_duration": self.recover_duration,
            "notify_recovered": self.notify_recovered,
            "notify_repeat_step": self.notify_repeat_step,
            "notify_max_number": self.notify_max_number,
            "append_tags": self.append_tags,
            "annotations": self.annotations,
            "datasource_queries": self.datasource_queries,
            "enable_stimes": [self.enable_stime],
            "enable_etimes": [self.enable_etime],
            "enable_days_of_weeks": [self.enable_days_of_week],
            "notify_version": 1,
            "notify_channels": [],
            "notify_groups": [],
            "notify_rule_ids": [],
        }
        
        # 添加 rule_config
        if self.rule_config:
            result["rule_config"] = self.rule_config
        else:
            # 默认 rule_config
            result["rule_config"] = {
                "queries": [{"prom_ql": self.prom_ql, "severity": self.severity}],
                "triggers": [],
                "inhibit": False
            }
        
        return result


@dataclass
class BusiGroup:
    """业务组"""
    name: str
    label_enable: int = 0
    label_value: str = ""
    
    def to_api_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "label_enable": self.label_enable,
            "label_value": self.label_value,
            "user_group_ids": [],
            "members": []
        }


@dataclass
class AlertMute:
    """告警屏蔽规则"""
    note: str
    prod: str = ""
    cate: str = "prometheus"
    datasource_ids: List[int] = field(default_factory=list)
    cluster: str = ""
    severity: int = 0  # 0表示所有级别
    disabled: int = 0
    mute_time_type: int = 0  # 0:固定时间 1:周期时间
    btime: int = 0  # 开始时间戳
    etime: int = 0  # 结束时间戳
    tags: List[Dict] = field(default_factory=list)
    
    def to_api_dict(self) -> Dict[str, Any]:
        return asdict(self)


# ============================================
# API 客户端
# ============================================

class N9EClient:
    """Nightingale API 客户端
    
    支持两种认证模式：
    1. JWT 模式（api_mode="web"）：通过登录获取 token
    2. Service API 模式（api_mode="service"）：使用 Basic Auth
    """
    
    def __init__(self, config: N9EConfig):
        self.config = config
        self.session = requests.Session()
        self.token: Optional[str] = None
        self._setup_session()
    
    def _setup_session(self):
        """设置会话"""
        self.session.headers.update({
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            'X-Language': 'zh_CN'
        })
        self.session.timeout = self.config.timeout
        
        # Service API 模式使用 Basic Auth
        if self.config.api_mode == "service":
            from requests.auth import HTTPBasicAuth
            self.session.auth = HTTPBasicAuth(
                self.config.username, 
                self.config.password
            )
            logger.debug(f"使用 Service API 模式 (Basic Auth), 用户: {self.config.username}")
    
    def login(self) -> bool:
        """登录获取token（仅 Web API 模式需要）"""
        # Service API 模式不需要登录，直接返回成功
        if self.config.api_mode == "service":
            logger.info(f"Service API 模式，使用 Basic Auth，用户: {self.config.username}")
            # 测试连接
            try:
                result = self.get('/busi-groups')
                if result.get('err', '') == '':
                    logger.info("Service API 连接成功")
                    return True
                else:
                    logger.error(f"Service API 连接失败: {result.get('err')}")
                    return False
            except Exception as e:
                logger.error(f"Service API 连接异常: {e}")
                return False
        
        # Web API 模式使用 JWT 登录
        try:
            url = f"{self.config.base_url}/auth/login"
            data = {
                "username": self.config.username,
                "password": self.config.password
            }
            response = self.session.post(url, json=data)
            result = response.json()
            
            if result.get('err') == '' and result.get('dat'):
                self.token = result['dat'].get('access_token')
                if self.token:
                    self.session.headers['Authorization'] = f'Bearer {self.token}'
                    logger.info(f"JWT 登录成功，用户: {self.config.username}")
                    return True
            
            logger.error(f"登录失败: {result.get('err', 'Unknown error')}")
            return False
        except Exception as e:
            logger.error(f"登录异常: {e}")
            return False
    
    def _request(self, method: str, endpoint: str, **kwargs) -> Dict:
        """发送HTTP请求"""
        url = f"{self.config.base_url}{endpoint}"
        try:
            response = self.session.request(method, url, **kwargs)
            
            # 检查 HTTP 状态码
            if response.status_code == 401:
                logger.error(f"认证失败 [{endpoint}]: 请检查用户名密码")
                return {'err': 'Authentication failed', 'dat': None}
            
            result = response.json()
            
            if result.get('err', '') != '':
                logger.error(f"API错误 [{endpoint}]: {result.get('err')}")
            
            return result
        except requests.exceptions.RequestException as e:
            logger.error(f"请求异常 [{endpoint}]: {e}")
            return {'err': str(e), 'dat': None}
    
    def get(self, endpoint: str, **kwargs) -> Dict:
        return self._request('GET', endpoint, **kwargs)
    
    def post(self, endpoint: str, **kwargs) -> Dict:
        return self._request('POST', endpoint, **kwargs)
    
    def put(self, endpoint: str, **kwargs) -> Dict:
        return self._request('PUT', endpoint, **kwargs)
    
    def delete(self, endpoint: str, **kwargs) -> Dict:
        return self._request('DELETE', endpoint, **kwargs)
    
    # ========================================
    # 业务组 API
    # ========================================
    
    def list_busi_groups(self) -> List[Dict]:
        """获取所有业务组"""
        result = self.get('/busi-groups')
        return result.get('dat', []) if result.get('err') == '' else []
    
    def get_busi_group(self, group_id: int) -> Optional[Dict]:
        """获取单个业务组"""
        result = self.get(f'/busi-group/{group_id}')
        return result.get('dat') if result.get('err') == '' else None
    
    def create_busi_group(self, group: BusiGroup) -> Optional[int]:
        """创建业务组"""
        result = self.post('/busi-groups', json=group.to_api_dict())
        if result.get('err') == '':
            logger.info(f"业务组创建成功: {group.name}")
            return result.get('dat', {}).get('id')
        return None
    
    def get_or_create_busi_group(self, name: str) -> Optional[int]:
        """获取或创建业务组"""
        groups = self.list_busi_groups()
        for g in groups:
            if g.get('name') == name:
                return g.get('id')
        return self.create_busi_group(BusiGroup(name=name))
    
    # ========================================
    # 告警规则 API
    # ========================================
    
    def list_alert_rules(self, group_id: int) -> List[Dict]:
        """获取业务组下的告警规则"""
        result = self.get(f'/busi-group/{group_id}/alert-rules')
        return result.get('dat', []) if result.get('err') == '' else []
    
    def get_alert_rule(self, rule_id: int) -> Optional[Dict]:
        """获取单个告警规则"""
        result = self.get(f'/alert-rule/{rule_id}')
        return result.get('dat') if result.get('err') == '' else None
    
    def create_alert_rules(self, group_id: int, rules: List[AlertRule]) -> Dict[str, str]:
        """创建告警规则（批量）"""
        data = [rule.to_api_dict() for rule in rules]
        result = self.post(f'/busi-group/{group_id}/alert-rules', json=data)
        if result.get('err') == '':
            logger.info(f"成功创建 {len(rules)} 条告警规则")
            return result.get('dat', {})
        return {'error': result.get('err')}
    
    def update_alert_rule(self, group_id: int, rule_id: int, rule: AlertRule) -> bool:
        """更新告警规则"""
        data = rule.to_api_dict()
        data['id'] = rule_id
        result = self.put(f'/busi-group/{group_id}/alert-rule/{rule_id}', json=data)
        return result.get('err') == ''
    
    def delete_alert_rules(self, group_id: int, rule_ids: List[int]) -> bool:
        """删除告警规则"""
        result = self.delete(f'/busi-group/{group_id}/alert-rules', json={"ids": rule_ids})
        return result.get('err') == ''
    
    def enable_alert_rules(self, group_id: int, rule_ids: List[int]) -> bool:
        """启用告警规则"""
        result = self.put(f'/busi-group/{group_id}/alert-rules/fields', 
                         json={"ids": rule_ids, "fields": {"disabled": 0}})
        return result.get('err') == ''
    
    def disable_alert_rules(self, group_id: int, rule_ids: List[int]) -> bool:
        """禁用告警规则"""
        result = self.put(f'/busi-group/{group_id}/alert-rules/fields', 
                         json={"ids": rule_ids, "fields": {"disabled": 1}})
        return result.get('err') == ''
    
    # ========================================
    # 告警屏蔽 API
    # ========================================
    
    def list_alert_mutes(self, group_id: int) -> List[Dict]:
        """获取告警屏蔽规则"""
        result = self.get(f'/busi-group/{group_id}/alert-mutes')
        return result.get('dat', []) if result.get('err') == '' else []
    
    def create_alert_mute(self, group_id: int, mute: AlertMute) -> Optional[int]:
        """创建告警屏蔽规则"""
        result = self.post(f'/busi-group/{group_id}/alert-mutes', json=mute.to_api_dict())
        if result.get('err') == '':
            logger.info(f"告警屏蔽规则创建成功: {mute.note}")
            return result.get('dat')
        return None
    
    def delete_alert_mute(self, group_id: int, mute_ids: List[int]) -> bool:
        """删除告警屏蔽规则"""
        result = self.delete(f'/busi-group/{group_id}/alert-mutes', json={"ids": mute_ids})
        return result.get('err') == ''
    
    # ========================================
    # 告警订阅 API
    # ========================================
    
    def list_alert_subscribes(self, group_id: int) -> List[Dict]:
        """获取告警订阅"""
        result = self.get(f'/busi-group/{group_id}/alert-subscribes')
        return result.get('dat', []) if result.get('err') == '' else []
    
    def create_alert_subscribe(self, group_id: int, subscribe: Dict) -> Optional[int]:
        """创建告警订阅"""
        result = self.post(f'/busi-group/{group_id}/alert-subscribes', json=[subscribe])
        if result.get('err') == '':
            logger.info("告警订阅创建成功")
            return result.get('dat')
        return None
    
    # ========================================
    # 仪表盘 API
    # ========================================
    
    def list_dashboards(self, group_id: int) -> List[Dict]:
        """获取仪表盘列表"""
        result = self.get(f'/busi-group/{group_id}/boards')
        return result.get('dat', []) if result.get('err') == '' else []
    
    def get_dashboard(self, board_id: int) -> Optional[Dict]:
        """获取仪表盘详情"""
        result = self.get(f'/board/{board_id}')
        return result.get('dat') if result.get('err') == '' else None
    
    def create_dashboard(self, group_id: int, name: str, tags: List[str] = None, 
                        configs: Dict = None) -> Optional[int]:
        """创建仪表盘"""
        data = {
            "name": name,
            "tags": tags or [],
            "configs": json.dumps(configs or {}),
            "ident": ""
        }
        result = self.post(f'/busi-group/{group_id}/boards', json=data)
        if result.get('err') == '':
            logger.info(f"仪表盘创建成功: {name}")
            return result.get('dat')
        return None
    
    # ========================================
    # 监控目标 API
    # ========================================
    
    def list_targets(self, query: str = "", limit: int = 100) -> List[Dict]:
        """获取监控目标列表"""
        result = self.get('/targets', params={"query": query, "limit": limit})
        return result.get('dat', {}).get('list', []) if result.get('err') == '' else []
    
    def update_target_tags(self, idents: List[str], tags: List[str]) -> bool:
        """更新目标标签"""
        result = self.post('/targets/tags', json={"idents": idents, "tags": tags})
        return result.get('err') == ''
    
    def update_target_busi_group(self, idents: List[str], group_id: int) -> bool:
        """更新目标所属业务组"""
        result = self.put('/targets/bgids', json={"idents": idents, "bgids": [group_id]})
        return result.get('err') == ''
    
    # ========================================
    # 数据源 API
    # ========================================
    
    def list_datasources(self) -> List[Dict]:
        """获取数据源列表"""
        result = self.post('/datasource/list', json={})
        return result.get('dat', []) if result.get('err') == '' else []
    
    # ========================================
    # 通知规则 API
    # ========================================
    
    def list_notify_rules(self) -> List[Dict]:
        """获取通知规则列表"""
        result = self.get('/notify-rules')
        return result.get('dat', []) if result.get('err') == '' else []
    
    def list_notify_channels(self) -> List[Dict]:
        """获取通知渠道列表"""
        result = self.get('/notify-channel-configs')
        return result.get('dat', []) if result.get('err') == '' else []
    
    # ========================================
    # 用户组 API
    # ========================================
    
    def list_user_groups(self) -> List[Dict]:
        """获取用户组列表"""
        result = self.get('/user-groups')
        return result.get('dat', []) if result.get('err') == '' else []


# ============================================
# 预定义告警规则模板
# ============================================

class AlertRuleTemplates:
    """预定义告警规则模板"""
    
    @staticmethod
    def cpu_high_usage(threshold: int = 80) -> AlertRule:
        """CPU使用率过高告警"""
        return AlertRule(
            name=f"CPU使用率超过{threshold}%",
            note=f"主机CPU使用率超过{threshold}%，请检查是否有异常进程",
            prom_ql=f'cpu_usage_active > {threshold}',
            severity=2,
            append_tags=["type=cpu", "level=warning"],
            annotations={"summary": "CPU使用率告警", "description": "CPU使用率超过阈值"}
        )
    
    @staticmethod
    def memory_high_usage(threshold: int = 80) -> AlertRule:
        """内存使用率过高告警"""
        return AlertRule(
            name=f"内存使用率超过{threshold}%",
            note=f"主机内存使用率超过{threshold}%，请检查内存占用情况",
            prom_ql=f'mem_used_percent > {threshold}',
            severity=2,
            append_tags=["type=memory", "level=warning"],
            annotations={"summary": "内存使用率告警", "description": "内存使用率超过阈值"}
        )
    
    @staticmethod
    def disk_high_usage(threshold: int = 85) -> AlertRule:
        """磁盘使用率过高告警"""
        return AlertRule(
            name=f"磁盘使用率超过{threshold}%",
            note=f"主机磁盘使用率超过{threshold}%，请及时清理或扩容",
            prom_ql=f'disk_used_percent{{path="/"}} > {threshold}',
            severity=1,
            append_tags=["type=disk", "level=critical"],
            annotations={"summary": "磁盘使用率告警", "description": "磁盘使用率超过阈值"}
        )
    
    @staticmethod
    def host_down() -> AlertRule:
        """主机宕机告警"""
        return AlertRule(
            name="主机宕机告警",
            note="主机心跳丢失超过3分钟，可能已宕机",
            prom_ql='up == 0',
            severity=1,
            recover_duration=180,
            append_tags=["type=host", "level=critical"],
            annotations={"summary": "主机宕机", "description": "主机已离线"}
        )
    
    @staticmethod
    def network_error() -> AlertRule:
        """网络错误告警"""
        return AlertRule(
            name="网络接口错误告警",
            note="网络接口出现错误，请检查网络状态",
            prom_ql='rate(net_errs[5m]) > 0',
            severity=2,
            append_tags=["type=network", "level=warning"],
            annotations={"summary": "网络错误", "description": "网络接口出现错误"}
        )
    
    @staticmethod
    def load_high(threshold: int = 10) -> AlertRule:
        """负载过高告警"""
        return AlertRule(
            name=f"系统负载超过{threshold}",
            note=f"系统负载超过{threshold}，请检查系统状态",
            prom_ql=f'system_load1 > {threshold}',
            severity=2,
            append_tags=["type=load", "level=warning"],
            annotations={"summary": "负载过高", "description": "系统负载超过阈值"}
        )
    
    @staticmethod
    def disk_io_high(threshold: int = 80) -> AlertRule:
        """磁盘IO过高告警"""
        return AlertRule(
            name=f"磁盘IO使用率超过{threshold}%",
            note=f"磁盘IO使用率超过{threshold}%，可能影响系统性能",
            prom_ql=f'diskio_io_time_percent > {threshold}',
            severity=2,
            append_tags=["type=diskio", "level=warning"],
            annotations={"summary": "磁盘IO告警", "description": "磁盘IO使用率超过阈值"}
        )
    
    @staticmethod
    def docker_container_down() -> AlertRule:
        """Docker容器停止告警"""
        return AlertRule(
            name="Docker容器停止运行",
            note="Docker容器已停止运行，请检查容器状态",
            prom_ql='docker_container_state_running == 0',
            severity=2,
            append_tags=["type=docker", "level=warning"],
            annotations={"summary": "容器停止", "description": "Docker容器已停止运行"}
        )
    
    @staticmethod
    def kubernetes_pod_not_ready() -> AlertRule:
        """Kubernetes Pod未就绪告警"""
        return AlertRule(
            name="Kubernetes Pod未就绪",
            note="Kubernetes Pod状态异常，未处于Ready状态",
            prom_ql='kube_pod_status_ready{condition="true"} == 0',
            severity=2,
            append_tags=["type=kubernetes", "level=warning"],
            annotations={"summary": "Pod未就绪", "description": "Kubernetes Pod状态异常"}
        )
    
    @staticmethod
    def mysql_connection_high(threshold: int = 80) -> AlertRule:
        """MySQL连接数过高告警"""
        return AlertRule(
            name=f"MySQL连接数使用率超过{threshold}%",
            note=f"MySQL连接数使用率超过{threshold}%，可能影响服务",
            prom_ql=f'(mysql_global_status_threads_connected / mysql_global_variables_max_connections) * 100 > {threshold}',
            severity=2,
            append_tags=["type=mysql", "level=warning"],
            annotations={"summary": "MySQL连接数告警", "description": "MySQL连接数使用率过高"}
        )
    
    @staticmethod
    def redis_memory_high(threshold: int = 80) -> AlertRule:
        """Redis内存使用率过高告警"""
        return AlertRule(
            name=f"Redis内存使用率超过{threshold}%",
            note=f"Redis内存使用率超过{threshold}%，请检查缓存策略",
            prom_ql=f'(redis_memory_used_bytes / redis_memory_max_bytes) * 100 > {threshold}',
            severity=2,
            append_tags=["type=redis", "level=warning"],
            annotations={"summary": "Redis内存告警", "description": "Redis内存使用率过高"}
        )
    
    @staticmethod
    def get_all_templates() -> List[AlertRule]:
        """获取所有预定义模板"""
        return [
            AlertRuleTemplates.cpu_high_usage(),
            AlertRuleTemplates.memory_high_usage(),
            AlertRuleTemplates.disk_high_usage(),
            AlertRuleTemplates.host_down(),
            AlertRuleTemplates.network_error(),
            AlertRuleTemplates.load_high(),
            AlertRuleTemplates.disk_io_high(),
            AlertRuleTemplates.docker_container_down(),
        ]


# ============================================
# 主要功能函数
# ============================================

def init_monitoring(client: N9EClient, group_name: str = "Default BusiGroup"):
    """初始化监控配置"""
    logger.info("开始初始化监控配置...")
    
    # 1. 创建或获取业务组
    group_id = client.get_or_create_busi_group(group_name)
    if not group_id:
        logger.error("无法创建业务组")
        return False
    
    logger.info(f"业务组ID: {group_id}")
    
    # 2. 获取现有告警规则
    existing_rules = client.list_alert_rules(group_id)
    existing_names = {r['name'] for r in existing_rules}
    
    # 3. 添加预定义告警规则
    templates = AlertRuleTemplates.get_all_templates()
    new_rules = [r for r in templates if r.name not in existing_names]
    
    if new_rules:
        result = client.create_alert_rules(group_id, new_rules)
        logger.info(f"创建了 {len(new_rules)} 条告警规则")
    else:
        logger.info("所有预定义规则已存在")
    
    # 4. 显示统计信息
    rules = client.list_alert_rules(group_id)
    logger.info(f"当前业务组共有 {len(rules)} 条告警规则")
    
    return True


def import_rules_from_yaml(client: N9EClient, file_path: str, group_id: int):
    """从YAML文件导入告警规则"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = yaml.safe_load(f)
        
        rules = []
        for rule_data in data.get('rules', []):
            rule = AlertRule(
                name=rule_data['name'],
                note=rule_data.get('note', ''),
                prom_ql=rule_data.get('prom_ql', rule_data.get('expr', '')),
                severity=rule_data.get('severity', 2),
                prom_eval_interval=rule_data.get('interval', 15),
                append_tags=rule_data.get('labels', []),
                annotations=rule_data.get('annotations', {})
            )
            rules.append(rule)
        
        if rules:
            result = client.create_alert_rules(group_id, rules)
            logger.info(f"成功导入 {len(rules)} 条告警规则")
            return result
        else:
            logger.warning("YAML文件中没有找到有效的规则")
            return {}
    
    except Exception as e:
        logger.error(f"导入规则失败: {e}")
        return {'error': str(e)}


def export_rules_to_yaml(client: N9EClient, group_id: int, file_path: str):
    """导出告警规则到YAML文件"""
    try:
        rules = client.list_alert_rules(group_id)
        
        export_data = {
            'rules': []
        }
        
        for rule in rules:
            export_data['rules'].append({
                'name': rule['name'],
                'note': rule.get('note', ''),
                'prom_ql': rule.get('prom_ql', ''),
                'severity': rule.get('severity', 2),
                'interval': rule.get('prom_eval_interval', 15),
                'labels': rule.get('append_tags', []),
                'annotations': rule.get('annotations', {})
            })
        
        with open(file_path, 'w', encoding='utf-8') as f:
            yaml.dump(export_data, f, default_flow_style=False, allow_unicode=True)
        
        logger.info(f"成功导出 {len(rules)} 条告警规则到 {file_path}")
        return True
    
    except Exception as e:
        logger.error(f"导出规则失败: {e}")
        return False


def add_custom_rule(client: N9EClient, group_id: int, name: str, prom_ql: str,
                   severity: int = 2, note: str = ""):
    """添加自定义告警规则"""
    rule = AlertRule(
        name=name,
        note=note,
        prom_ql=prom_ql,
        severity=severity
    )
    
    result = client.create_alert_rules(group_id, [rule])
    if 'error' not in result:
        logger.info(f"告警规则 '{name}' 创建成功")
        return True
    return False


def print_status(client: N9EClient):
    """打印当前状态"""
    print("\n" + "=" * 60)
    print("Nightingale 监控系统状态")
    print(f"API 模式: {client.config.api_mode}")
    print(f"API 地址: {client.config.base_url}")
    print("=" * 60)
    
    # 业务组
    groups = client.list_busi_groups()
    print(f"\n📁 业务组: {len(groups)} 个")
    for g in groups[:5]:
        print(f"   - {g['name']} (ID: {g['id']})")
    if len(groups) > 5:
        print(f"   ... 还有 {len(groups) - 5} 个")
    
    # 数据源
    datasources = client.list_datasources()
    print(f"\n💾 数据源: {len(datasources)} 个")
    for ds in datasources:
        print(f"   - {ds['name']} ({ds.get('plugin_type', 'unknown')})")
    
    # 监控目标
    targets = client.list_targets(limit=10)
    print(f"\n🖥️  监控目标: {len(targets)} 个 (显示前10个)")
    for t in targets[:5]:
        print(f"   - {t['ident']}")
    
    # 通知规则
    notify_rules = client.list_notify_rules()
    print(f"\n📢 通知规则: {len(notify_rules)} 个")
    
    # 通知渠道
    channels = client.list_notify_channels()
    print(f"\n📣 通知渠道: {len(channels)} 个")
    for ch in channels:
        print(f"   - {ch['name']} ({ch.get('ident', '')})")
    
    print("\n" + "=" * 60)


# ============================================
# CLI 入口
# ============================================

def main():
    parser = argparse.ArgumentParser(
        description='Nightingale (N9E) 全链路监控和告警配置工具',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  初始化监控:        python %(prog)s init
  添加告警规则:      python %(prog)s add-rule --name "CPU告警" --prom-ql 'cpu_usage > 80'
  导入告警规则:      python %(prog)s import-rules --file rules.yaml --group-id 1
  导出告警规则:      python %(prog)s export-rules --group-id 1 --output rules.yaml
  列出业务组:        python %(prog)s list-groups
  列出告警规则:      python %(prog)s list-rules --group-id 1
  查看系统状态:      python %(prog)s status

认证模式:
  Service API (默认): 使用 Basic Auth，适合脚本自动化
    --api-mode service --username n9e-api --password 123456
  
  Web API: 使用 JWT 登录，适合交互式访问
    --api-mode web --username root --password root.2020
        """
    )
    
    # 通用参数
    parser.add_argument('--host', default=None, help='N9E 主机地址 (默认从环境变量 N9E_HOST)')
    parser.add_argument('--port', type=int, default=None, help='N9E 端口 (默认: 80 通过 nginx)')
    parser.add_argument('--username', default=None, help='用户名 (Service API: n9e-api, Web API: root)')
    parser.add_argument('--password', default=None, help='密码')
    parser.add_argument('--api-mode', choices=['service', 'web'], default=None,
                       help='API 模式: service(Basic Auth) 或 web(JWT)')
    parser.add_argument('-v', '--verbose', action='store_true', help='显示详细信息')
    
    subparsers = parser.add_subparsers(dest='command', help='可用命令')
    
    # init 命令
    init_parser = subparsers.add_parser('init', help='初始化监控配置')
    init_parser.add_argument('--group-name', default='Default BusiGroup', help='业务组名称')
    
    # add-rule 命令
    add_rule_parser = subparsers.add_parser('add-rule', help='添加告警规则')
    add_rule_parser.add_argument('--name', required=True, help='规则名称')
    add_rule_parser.add_argument('--prom-ql', required=True, help='PromQL表达式')
    add_rule_parser.add_argument('--severity', type=int, default=2, choices=[1, 2, 3], 
                                 help='严重程度 (1:紧急 2:警告 3:通知)')
    add_rule_parser.add_argument('--note', default='', help='规则说明')
    add_rule_parser.add_argument('--group-id', type=int, required=True, help='业务组ID')
    
    # import-rules 命令
    import_parser = subparsers.add_parser('import-rules', help='从YAML文件导入告警规则')
    import_parser.add_argument('--file', required=True, help='YAML文件路径')
    import_parser.add_argument('--group-id', type=int, required=True, help='业务组ID')
    
    # export-rules 命令
    export_parser = subparsers.add_parser('export-rules', help='导出告警规则到YAML文件')
    export_parser.add_argument('--group-id', type=int, required=True, help='业务组ID')
    export_parser.add_argument('--output', required=True, help='输出文件路径')
    
    # list-groups 命令
    subparsers.add_parser('list-groups', help='列出所有业务组')
    
    # list-rules 命令
    list_rules_parser = subparsers.add_parser('list-rules', help='列出告警规则')
    list_rules_parser.add_argument('--group-id', type=int, required=True, help='业务组ID')
    
    # status 命令
    subparsers.add_parser('status', help='查看系统状态')
    
    # 添加预设规则命令
    preset_parser = subparsers.add_parser('add-preset', help='添加预设告警规则')
    preset_parser.add_argument('--group-id', type=int, required=True, help='业务组ID')
    preset_parser.add_argument('--type', choices=['cpu', 'memory', 'disk', 'host', 'network', 
                                                   'load', 'diskio', 'docker', 'all'],
                              default='all', help='预设类型')
    preset_parser.add_argument('--threshold', type=int, help='阈值 (适用于某些规则)')
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    if not args.command:
        parser.print_help()
        return 1
    
    # 加载配置
    config = N9EConfig.from_env()
    if args.host:
        config.host = args.host
    if args.port:
        config.port = args.port
    if args.username:
        config.username = args.username
    if args.password:
        config.password = args.password
    if hasattr(args, 'api_mode') and args.api_mode:
        config.api_mode = args.api_mode
    
    # 创建客户端
    client = N9EClient(config)
    
    # 登录/连接测试
    if not client.login():
        logger.error("连接失败，请检查配置")
        logger.error(f"  API 地址: {config.base_url}")
        logger.error(f"  API 模式: {config.api_mode}")
        logger.error(f"  用户名: {config.username}")
        return 1
    
    # 执行命令
    if args.command == 'init':
        init_monitoring(client, args.group_name)
    
    elif args.command == 'add-rule':
        add_custom_rule(client, args.group_id, args.name, args.prom_ql, 
                       args.severity, args.note)
    
    elif args.command == 'import-rules':
        import_rules_from_yaml(client, args.file, args.group_id)
    
    elif args.command == 'export-rules':
        export_rules_to_yaml(client, args.group_id, args.output)
    
    elif args.command == 'list-groups':
        groups = client.list_busi_groups()
        print(f"\n共 {len(groups)} 个业务组:")
        for g in groups:
            print(f"  ID: {g['id']:4d} | 名称: {g['name']}")
    
    elif args.command == 'list-rules':
        rules = client.list_alert_rules(args.group_id)
        print(f"\n共 {len(rules)} 条告警规则:")
        for r in rules:
            status = "✅" if r['disabled'] == 0 else "❌"
            severity_map = {1: "🔴紧急", 2: "🟡警告", 3: "🔵通知"}
            severity = severity_map.get(r['severity'], "⚪未知")
            print(f"  {status} ID: {r['id']:4d} | {severity} | {r['name']}")
    
    elif args.command == 'status':
        print_status(client)
    
    elif args.command == 'add-preset':
        templates = []
        if args.type == 'all':
            templates = AlertRuleTemplates.get_all_templates()
        elif args.type == 'cpu':
            templates = [AlertRuleTemplates.cpu_high_usage(args.threshold or 80)]
        elif args.type == 'memory':
            templates = [AlertRuleTemplates.memory_high_usage(args.threshold or 80)]
        elif args.type == 'disk':
            templates = [AlertRuleTemplates.disk_high_usage(args.threshold or 85)]
        elif args.type == 'host':
            templates = [AlertRuleTemplates.host_down()]
        elif args.type == 'network':
            templates = [AlertRuleTemplates.network_error()]
        elif args.type == 'load':
            templates = [AlertRuleTemplates.load_high(args.threshold or 10)]
        elif args.type == 'diskio':
            templates = [AlertRuleTemplates.disk_io_high(args.threshold or 80)]
        elif args.type == 'docker':
            templates = [AlertRuleTemplates.docker_container_down()]
        
        if templates:
            result = client.create_alert_rules(args.group_id, templates)
            print(f"成功添加 {len(templates)} 条预设规则")
    
    return 0


if __name__ == '__main__':
    sys.exit(main())
