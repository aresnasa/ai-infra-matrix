package services

import (
	"fmt"
	"net"
	"os"
	"strings"
	"context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesService 封装集群连接与基本操作
// 只做最小实现，后续可扩展

type KubernetesService struct{}

func NewKubernetesService() *KubernetesService {
	return &KubernetesService{}
}

// ConnectToCluster 通过 kubeconfig 内容连接集群
func (s *KubernetesService) ConnectToCluster(kubeConfig string) (*kubernetes.Clientset, error) {
	// 如果kubeconfig是加密的，先解密
	decryptedKubeConfig := kubeConfig
	if database.CryptoService != nil && database.CryptoService.IsEncrypted(kubeConfig) {
		decryptedKubeConfig = database.CryptoService.DecryptSafely(kubeConfig)
		logrus.Debug("KubeConfig decrypted successfully for connection")
	}
	
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(decryptedKubeConfig))
	if err != nil {
		return nil, fmt.Errorf("kubeconfig 解析失败: %w", err)
	}
	
	// 增强的SSL跳过检查
	if s.shouldSkipSSLVerification(config.Host) {
		// 方式1：简单跳过SSL验证（优先使用，兼容性最好）
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
		// 注意：不再设置自定义Transport，避免冲突
	}
	
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("创建 clientset 失败: %w", err)
	}
	return clientset, nil
}

// ConnectToClusterByRestConfig 支持直接传递 rest.Config
func (s *KubernetesService) ConnectToClusterByRestConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// shouldSkipSSLVerification 增强的SSL跳过检查
func (s *KubernetesService) shouldSkipSSLVerification(host string) bool {
	// 1. 检查环境变量强制跳过SSL
	if os.Getenv("SKIP_SSL_VERIFY") == "true" || os.Getenv("K8S_SKIP_TLS_VERIFY") == "true" {
		return true
	}
	
	// 2. 检查是否为Docker Desktop集群
	if s.isDockerDesktopCluster(host) {
		return true
	}
	
	// 3. 检查是否为开发环境
	if s.isDevelopmentEnvironment() {
		return true
	}
	
	// 4. 检查是否为自签名证书常见的地址
	if s.isLikelySelfSignedCert(host) {
		return true
	}
	
	return false
}

// isDockerDesktopCluster 检查是否为Docker Desktop的Kubernetes集群
func (s *KubernetesService) isDockerDesktopCluster(host string) bool {
	dockerPatterns := []string{
		"kubernetes.docker.internal",
		"docker-desktop",
		"docker.for.mac.kubernetes.internal",
		"docker.for.windows.kubernetes.internal",
	}
	
	for _, pattern := range dockerPatterns {
		if strings.Contains(host, pattern) {
			return true
		}
	}
	return false
}

// isLikelySelfSignedCert 检查是否可能是自签名证书
func (s *KubernetesService) isLikelySelfSignedCert(host string) bool {
	selfSignedPatterns := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"local.cluster",
		"k8s.local",
		"kubernetes.local",
		"minikube",
		"kind",
		"k3s",
		"microk8s",
	}
	
	for _, pattern := range selfSignedPatterns {
		if strings.Contains(host, pattern) {
			return true
		}
	}
	
	// 检查私有IP地址段
	if s.isPrivateIP(host) {
		return true
	}
	
	return false
}

// isPrivateIP 检查是否为私有IP地址
func (s *KubernetesService) isPrivateIP(host string) bool {
	// 提取主机名中的IP地址
	hostParts := strings.Split(host, ":")
	if len(hostParts) > 0 {
		ip := net.ParseIP(strings.Replace(hostParts[0], "https://", "", 1))
		if ip != nil {
			return ip.IsPrivate() || ip.IsLoopback()
		}
	}
	return false
}

// isDevelopmentEnvironment 检查是否为开发环境
func (s *KubernetesService) isDevelopmentEnvironment() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	devEnvs := []string{"development", "dev", "local", "test", "testing", ""}
	
	for _, devEnv := range devEnvs {
		if env == devEnv {
			return true
		}
	}
	return false
}

// IsConnectionError 检查是否是连接错误
func (s *KubernetesService) IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"no such host",
		"network is unreachable",
		"timeout",
		"dial tcp",
		"certificate signed by unknown authority",
	}
	
	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errStr), connErr) {
			return true
		}
	}
	return false
}

// GetClusterVersion 获取集群版本
func (s *KubernetesService) GetClusterVersion(clientset *kubernetes.Clientset) (string, error) {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("获取集群版本失败: %w", err)
	}
	return version.GitVersion, nil
}

// GetNodes 获取集群节点列表
func (s *KubernetesService) GetNodes(clientset *kubernetes.Clientset) (*v1.NodeList, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}
	return nodes, nil
}

// 可扩展：集群健康检查、命名空间列表等
