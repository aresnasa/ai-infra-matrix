package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}
	
	// 连接数据库
	err = database.Connect(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// 获取集群ID
	clusterID := "13"
	if len(os.Args) > 1 {
		clusterID = os.Args[1]
	}

	// 查询集群信息
	var cluster models.KubernetesCluster
	err = database.DB.First(&cluster, clusterID).Error
	if err != nil {
		log.Fatalf("Failed to find cluster %s: %v", clusterID, err)
	}

	fmt.Printf("Testing cluster: %s\n", cluster.Name)
	fmt.Printf("API Server: %s\n", cluster.APIServer)
	fmt.Printf("Namespace: %s\n", cluster.Namespace)

	// 手动调用解密函数（因为GORM hooks可能没有自动触发）
	if err := cluster.AfterFind(database.DB); err != nil {
		log.Fatalf("Failed to decrypt cluster data: %v", err)
	}

	fmt.Println("=== AFTER AfterFind Hook ===")
	fmt.Printf("KubeConfig length after hook: %d\n", len(cluster.KubeConfig))
	if len(cluster.KubeConfig) > 0 {
		if len(cluster.KubeConfig) > 100 {
			fmt.Printf("KubeConfig preview: %s...\n", cluster.KubeConfig[:100])
		} else {
			fmt.Printf("KubeConfig content: %s\n", cluster.KubeConfig)
		}
	}

	// 解密kubeconfig (备用方案)
	if database.CryptoService != nil && database.CryptoService.IsEncrypted(cluster.KubeConfig) {
		decrypted, err := database.CryptoService.Decrypt(cluster.KubeConfig)
		if err != nil {
			log.Fatalf("Failed to decrypt kubeconfig: %v", err)
		}
		cluster.KubeConfig = decrypted
		fmt.Println("✓ KubeConfig decrypted successfully")
		
		// 调试：打印解密后的kubeconfig前200个字符
		configPreview := cluster.KubeConfig
		if len(configPreview) > 200 {
			configPreview = configPreview[:200] + "..."
		}
		fmt.Printf("Decrypted kubeconfig preview: %s\n", configPreview)
	}

	// Print decrypted kubeconfig for debugging
	fmt.Println("=== DEBUG: Decrypted Kubeconfig Content ===")
	fmt.Printf("Length: %d bytes\n", len(cluster.KubeConfig))
	
	// Print more content to see the structure
	fmt.Println("Content (first 1000 chars):")
	if len(cluster.KubeConfig) > 1000 {
		fmt.Println(string(cluster.KubeConfig[:1000]))
	} else {
		fmt.Println(string(cluster.KubeConfig))
	}
	
	// Check for specific characters that might cause issues
	content := string(cluster.KubeConfig)
	fmt.Printf("Contains newlines: %v\n", strings.Contains(content, "\n"))
	fmt.Printf("Contains carriage returns: %v\n", strings.Contains(content, "\r"))
	fmt.Printf("Line count: %d\n", strings.Count(content, "\n"))
	
	// Check for null bytes or other binary data
	hasNullBytes := false
	for i, b := range []byte(cluster.KubeConfig) {
		if b == 0 {
			fmt.Printf("Found null byte at position %d\n", i)
			hasNullBytes = true
			break
		}
	}
	if !hasNullBytes {
		fmt.Printf("No null bytes found in kubeconfig\n")
	}
	
	fmt.Printf("First 20 bytes (hex): %x\n", []byte(cluster.KubeConfig[:min(20, len(cluster.KubeConfig))]))
	fmt.Printf("Last 20 bytes (hex): %x\n", []byte(cluster.KubeConfig[max(0, len(cluster.KubeConfig)-20):]))
	fmt.Println("=== END DEBUG ===")

	// 测试连接
	kubernetesService := services.NewKubernetesService()
	clientset, err := kubernetesService.ConnectToCluster(cluster.KubeConfig)
	if err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		// 尝试分析具体错误
		if kubernetesService.IsConnectionError(err) {
			fmt.Println("This appears to be a network connectivity issue.")
			fmt.Println("Possible solutions:")
			fmt.Println("1. Check if the API server address is reachable from inside the container")
			fmt.Println("2. Verify firewall settings")
			fmt.Println("3. Check if the cluster is running")
		}
		os.Exit(1)
	}

	// 测试获取版本信息（使用新的函数签名）
	ctx := context.Background()
	versionInfo, err := kubernetesService.GetClusterVersion(ctx, cluster.KubeConfig)
	if err != nil {
		fmt.Printf("❌ Failed to get cluster version: %v\n", err)
		os.Exit(1)
	}

	// 测试获取节点信息
	nodes, err := kubernetesService.GetNodes(clientset)
	if err != nil {
		fmt.Printf("❌ Failed to get nodes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully connected to cluster!\n")
	fmt.Printf("Cluster version: %s (Major: %s, Minor: %s)\n", 
		versionInfo.GitVersion, versionInfo.Major, versionInfo.Minor)
	fmt.Printf("Platform: %s\n", versionInfo.Platform)
	fmt.Printf("Number of nodes: %d\n", len(nodes.Items))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
