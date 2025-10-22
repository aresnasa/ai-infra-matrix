package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
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

	// 读取KubeConfig文件
	kubeConfigContent, err := ioutil.ReadFile("docker-desktop-kubeconfig.yaml")
	if err != nil {
		log.Fatalf("Failed to read kubeconfig file: %v", err)
	}

	// 获取集群ID
	clusterID := "13"
	if len(os.Args) > 1 {
		clusterID = os.Args[1]
	}

	// 查询并更新集群信息
	var cluster models.KubernetesCluster
	err = database.DB.First(&cluster, clusterID).Error
	if err != nil {
		log.Fatalf("Failed to find cluster: %v", err)
	}

	log.Printf("Found cluster: %s", cluster.Name)
	log.Printf("Current KubeConfig length: %d", len(cluster.KubeConfig))

	// 更新KubeConfig
	cluster.KubeConfig = string(kubeConfigContent)

	err = database.DB.Save(&cluster).Error
	if err != nil {
		log.Fatalf("Failed to update cluster: %v", err)
	}

	log.Printf("Successfully updated cluster %s with new KubeConfig (length: %d)", cluster.Name, len(cluster.KubeConfig))
}
