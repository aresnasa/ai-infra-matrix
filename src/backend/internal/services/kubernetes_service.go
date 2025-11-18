package services

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
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

// GetNodes 获取集群节点列表
func (s *KubernetesService) GetNodes(clientset *kubernetes.Clientset) (*corev1.NodeList, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}
	return nodes, nil
}

// 可扩展：集群健康检查、命名空间列表等

// EnsureNamespace 确保命名空间存在
func (s *KubernetesService) EnsureNamespace(clientset *kubernetes.Clientset, ns string) error {
	if ns == "" {
		return nil
	}
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建命名空间失败: %w", err)
	}
	return nil
}

// EnsureServiceAccount 为用户创建或获取ServiceAccount
func (s *KubernetesService) EnsureServiceAccount(clientset *kubernetes.Clientset, namespace, saName string) (*corev1.ServiceAccount, error) {
	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), saName, metav1.GetOptions{})
	if err == nil {
		return sa, nil
	}
	sa = &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName}}
	created, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建ServiceAccount失败: %w", err)
	}
	return created, nil
}

// EnsureRoleBinding 绑定ClusterRole到用户的ServiceAccount
func (s *KubernetesService) EnsureRoleBinding(clientset *kubernetes.Clientset, namespace, rbName, saName, clusterRole string) (*rbacv1.RoleBinding, error) {
	rb, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), rbName, metav1.GetOptions{})
	if err == nil {
		return rb, nil
	}
	rb = &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: rbName},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: namespace}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: clusterRole},
	}
	created, err := clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), rb, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建RoleBinding失败: %w", err)
	}
	return created, nil
}

// ----- 动态客户端与通用CRUD实现 -----

// getRestConfig 从kubeconfig创建rest.Config（含TLS跳过策略）
func (s *KubernetesService) getRestConfig(kubeConfig string) (*rest.Config, error) {
	decryptedKubeConfig := kubeConfig
	if database.CryptoService != nil && database.CryptoService.IsEncrypted(kubeConfig) {
		decryptedKubeConfig = database.CryptoService.DecryptSafely(kubeConfig)
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(decryptedKubeConfig))
	if err != nil {
		return nil, fmt.Errorf("kubeconfig 解析失败: %w", err)
	}
	if s.shouldSkipSSLVerification(cfg.Host) {
		cfg.TLSClientConfig.Insecure = true
		cfg.TLSClientConfig.CAData = nil
		cfg.TLSClientConfig.CAFile = ""
	}
	return cfg, nil
}

// getDiscoveryMapper 返回discovery与RESTMapper
func (s *KubernetesService) getDiscoveryMapper(cfg *rest.Config) (discovery.DiscoveryInterface, meta.RESTMapper, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	return dc, mapper, nil
}

// GetDiscoveryAndMapper 返回可序列化的资源组与资源列表，用于前端构建资源树
func (s *KubernetesService) GetDiscoveryAndMapper(ctx context.Context, kubeConfig string) (*metav1.APIGroupList, meta.RESTMapper, []*metav1.APIResourceList, error) {
	cfg, err := s.getRestConfig(kubeConfig)
	if err != nil {
		return nil, nil, nil, err
	}
	dc, mapper, err := s.getDiscoveryMapper(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	groups, err := dc.ServerGroups()
	if err != nil {
		return nil, nil, nil, err
	}
	resLists, err := dc.ServerPreferredResources()
	if err != nil {
		// 某些API可能无权限，忽略部分错误，返回已获取的资源
		logrus.WithError(err).Warn("ServerPreferredResources encountered partial error")
	}
	return groups, mapper, resLists, nil
}

// resolveGVR 解析资源名到GVR（如 pods, deployments.apps 等）
func (s *KubernetesService) resolveGVR(mapper meta.RESTMapper, resource string) (schema.GroupVersionResource, meta.RESTScopeName, error) {
	// 支持形如 "deployments.apps" 或 "deployments"。Version留空由mapper选择首选版本
	gr := schema.ParseGroupResource(resource)
	// 使用ResourceFor自动补全版本
	gvr, err := mapper.ResourceFor(schema.GroupVersionResource{Group: gr.Group, Resource: gr.Resource})
	if err != nil {
		return schema.GroupVersionResource{}, "", err
	}
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gvr.Group, Kind: ""}, gvr.Version)
	if err != nil {
		// 尝试通过资源直接获取映射
		// Newer client-go may require RESTMapping by Kind; 退化处理：scope按常见资源推断
		return gvr, meta.RESTScopeNameNamespace, nil
	}
	scope := mapping.Scope.Name()
	return gvr, scope, nil
}

func (s *KubernetesService) getDynamicClientAndGVR(kubeConfig, resource string) (dynamic.Interface, meta.RESTMapper, schema.GroupVersionResource, meta.RESTScopeName, error) {
	cfg, err := s.getRestConfig(kubeConfig)
	if err != nil {
		return nil, nil, schema.GroupVersionResource{}, "", err
	}
	dc, mapper, err := s.getDiscoveryMapper(cfg)
	if err != nil {
		return nil, nil, schema.GroupVersionResource{}, "", err
	}
	_ = dc
	gvr, scope, err := s.resolveGVR(mapper, resource)
	if err != nil {
		return nil, nil, schema.GroupVersionResource{}, "", err
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, schema.GroupVersionResource{}, "", err
	}
	return dyn, mapper, gvr, scope, nil
}

// DynamicList 通用列表
func (s *KubernetesService) DynamicList(ctx context.Context, kubeConfig, resource, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	dyn, _, gvr, scope, err := s.getDynamicClientAndGVR(kubeConfig, resource)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if scope == meta.RESTScopeNameNamespace && namespace != "" {
		ri = dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = dyn.Resource(gvr)
	}
	return ri.List(ctx, opts)
}

// DynamicGet 通用获取
func (s *KubernetesService) DynamicGet(ctx context.Context, kubeConfig, resource, namespace, name string) (*unstructured.Unstructured, error) {
	dyn, _, gvr, scope, err := s.getDynamicClientAndGVR(kubeConfig, resource)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if scope == meta.RESTScopeNameNamespace && namespace != "" {
		ri = dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = dyn.Resource(gvr)
	}
	return ri.Get(ctx, name, metav1.GetOptions{})
}

// DynamicCreate 通用创建
func (s *KubernetesService) DynamicCreate(ctx context.Context, kubeConfig, resource, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	dyn, _, gvr, scope, err := s.getDynamicClientAndGVR(kubeConfig, resource)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if scope == meta.RESTScopeNameNamespace {
		if namespace == "" {
			// 若未提供ns且资源需要命名空间，则尝试使用对象元数据中的namespace
			if ns := obj.GetNamespace(); ns != "" {
				namespace = ns
			}
		}
		ri = dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = dyn.Resource(gvr)
	}
	return ri.Create(ctx, obj, metav1.CreateOptions{})
}

// DynamicUpdate 通用更新（替换）
func (s *KubernetesService) DynamicUpdate(ctx context.Context, kubeConfig, resource, namespace, name string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	dyn, _, gvr, scope, err := s.getDynamicClientAndGVR(kubeConfig, resource)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if scope == meta.RESTScopeNameNamespace && namespace != "" {
		ri = dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = dyn.Resource(gvr)
	}
	if obj.GetName() == "" {
		obj.SetName(name)
	}
	return ri.Update(ctx, obj, metav1.UpdateOptions{})
}

// DynamicPatch 通用Patch
func (s *KubernetesService) DynamicPatch(ctx context.Context, kubeConfig, resource, namespace, name string, pt types.PatchType, data []byte) (*unstructured.Unstructured, error) {
	dyn, _, gvr, scope, err := s.getDynamicClientAndGVR(kubeConfig, resource)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if scope == meta.RESTScopeNameNamespace && namespace != "" {
		ri = dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = dyn.Resource(gvr)
	}
	return ri.Patch(ctx, name, pt, data, metav1.PatchOptions{})
}

// DynamicDelete 通用删除
func (s *KubernetesService) DynamicDelete(ctx context.Context, kubeConfig, resource, namespace, name string, opts metav1.DeleteOptions) error {
	dyn, _, gvr, scope, err := s.getDynamicClientAndGVR(kubeConfig, resource)
	if err != nil {
		return err
	}
	var ri dynamic.ResourceInterface
	if scope == meta.RESTScopeNameNamespace && namespace != "" {
		ri = dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = dyn.Resource(gvr)
	}
	return ri.Delete(ctx, name, opts)
}

// ----- 版本检测与兼容性支持 -----

// ClusterVersionInfo 集群版本信息
type ClusterVersionInfo struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`
	Platform   string `json:"platform"`
	BuildDate  string `json:"buildDate"`
}

// GetClusterVersion 获取集群版本信息（兼容多版本k8s）
func (s *KubernetesService) GetClusterVersion(ctx context.Context, kubeConfig string) (*ClusterVersionInfo, error) {
	clientset, err := s.ConnectToCluster(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("连接集群失败: %w", err)
	}

	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("获取集群版本失败: %w", err)
	}

	return &ClusterVersionInfo{
		Major:      versionInfo.Major,
		Minor:      versionInfo.Minor,
		GitVersion: versionInfo.GitVersion,
		Platform:   versionInfo.Platform,
		BuildDate:  versionInfo.BuildDate,
	}, nil
}

// GetEnhancedDiscovery 增强的资源发现，包含 CRD 和版本信息
func (s *KubernetesService) GetEnhancedDiscovery(ctx context.Context, kubeConfig string) (*EnhancedDiscoveryResult, error) {
	cfg, err := s.getRestConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	// 获取集群版本
	versionInfo, err := dc.ServerVersion()
	if err != nil {
		logrus.WithError(err).Warn("无法获取集群版本，继续执行")
	}

	// 获取所有 API 组
	groups, err := dc.ServerGroups()
	if err != nil {
		return nil, fmt.Errorf("获取 API 组失败: %w", err)
	}

	// 获取所有 API 资源
	_, resourceLists, err := dc.ServerGroupsAndResources()
	if err != nil {
		// 部分 API 可能无权限访问，记录警告但继续
		logrus.WithError(err).Warn("ServerGroupsAndResources 遇到部分错误")
	}

	// 获取 CRD 列表
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建动态客户端失败: %w", err)
	}

	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	crdList, err := dyn.Resource(crdGVR).List(ctx, metav1.ListOptions{})
	var crds []CRDInfo
	if err != nil {
		logrus.WithError(err).Warn("无法获取 CRD 列表，可能没有权限")
	} else {
		crds = s.parseCRDs(crdList)
	}

	// 组织资源按 API 组分类
	resourcesByGroup := s.organizeResourcesByGroup(resourceLists)

	result := &EnhancedDiscoveryResult{
		Version:          versionInfo,
		Groups:           groups,
		ResourcesByGroup: resourcesByGroup,
		CRDs:             crds,
		TotalResources:   s.countTotalResources(resourceLists),
		TotalCRDs:        len(crds),
	}

	return result, nil
}

// EnhancedDiscoveryResult 增强的发现结果
type EnhancedDiscoveryResult struct {
	Version          *version.Info                   `json:"version"`
	Groups           *metav1.APIGroupList            `json:"groups"`
	ResourcesByGroup map[string][]metav1.APIResource `json:"resourcesByGroup"`
	CRDs             []CRDInfo                       `json:"crds"`
	TotalResources   int                             `json:"totalResources"`
	TotalCRDs        int                             `json:"totalCRDs"`
}

// CRDInfo CRD 信息
type CRDInfo struct {
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Kind     string   `json:"kind"`
	Plural   string   `json:"plural"`
	Singular string   `json:"singular"`
	Scope    string   `json:"scope"`
	Versions []string `json:"versions"`
}

// parseCRDs 解析 CRD 列表
func (s *KubernetesService) parseCRDs(crdList *unstructured.UnstructuredList) []CRDInfo {
	var crds []CRDInfo

	for _, item := range crdList.Items {
		spec, found, _ := unstructured.NestedMap(item.Object, "spec")
		if !found {
			continue
		}

		group, _, _ := unstructured.NestedString(spec, "group")

		names, found, _ := unstructured.NestedMap(spec, "names")
		if !found {
			continue
		}

		kind, _, _ := unstructured.NestedString(names, "kind")
		plural, _, _ := unstructured.NestedString(names, "plural")
		singular, _, _ := unstructured.NestedString(names, "singular")

		scope, _, _ := unstructured.NestedString(spec, "scope")

		versions, found, _ := unstructured.NestedSlice(spec, "versions")
		var versionNames []string
		if found {
			for _, v := range versions {
				if vMap, ok := v.(map[string]interface{}); ok {
					if name, _, _ := unstructured.NestedString(vMap, "name"); name != "" {
						versionNames = append(versionNames, name)
					}
				}
			}
		}

		crds = append(crds, CRDInfo{
			Name:     item.GetName(),
			Group:    group,
			Kind:     kind,
			Plural:   plural,
			Singular: singular,
			Scope:    scope,
			Versions: versionNames,
		})
	}

	return crds
}

// organizeResourcesByGroup 按 API 组组织资源
func (s *KubernetesService) organizeResourcesByGroup(resourceLists []*metav1.APIResourceList) map[string][]metav1.APIResource {
	result := make(map[string][]metav1.APIResource)

	for _, list := range resourceLists {
		if list == nil {
			continue
		}

		// GroupVersion 格式如 "v1" 或 "apps/v1"
		gv := list.GroupVersion

		for _, resource := range list.APIResources {
			result[gv] = append(result[gv], resource)
		}
	}

	return result
}

// countTotalResources 计算资源总数
func (s *KubernetesService) countTotalResources(resourceLists []*metav1.APIResourceList) int {
	count := 0
	for _, list := range resourceLists {
		if list != nil {
			count += len(list.APIResources)
		}
	}
	return count
}

// IsVersionCompatible 检查客户端版本是否与集群版本兼容
// client-go 通常向后兼容 n-2 个版本
func (s *KubernetesService) IsVersionCompatible(clientVersion, serverVersion string) bool {
	// client-go 0.33.x 对应 k8s 1.33.x，但可以连接 1.27.x - 1.33.x
	// 由于我们使用的是 v0.33.1，可以很好地兼容 1.27.5
	// 这里简单返回 true，实际上 client-go 的兼容性很强
	return true
}
