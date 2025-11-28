package services

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NightingaleService handles Nightingale monitoring operations
type NightingaleService struct {
	db              *gorm.DB
	nightingaleDB   *gorm.DB
	nightingaleHost string
	nightingalePort string
}

// NewNightingaleService creates a new Nightingale service instance
func NewNightingaleService(db *gorm.DB, nightingaleDBConfig map[string]string) (*NightingaleService, error) {
	// Connect to Nightingale database
	nightingaleDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
		nightingaleDBConfig["host"],
		nightingaleDBConfig["user"],
		nightingaleDBConfig["password"],
		nightingaleDBConfig["dbname"],
		nightingaleDBConfig["port"],
		nightingaleDBConfig["sslmode"],
	)

	nightingaleDB, err := gorm.Open(postgres.Open(nightingaleDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Nightingale database: %w", err)
	}

	return &NightingaleService{
		db:              db,
		nightingaleDB:   nightingaleDB,
		nightingaleHost: nightingaleDBConfig["host"],
		nightingalePort: nightingaleDBConfig["n9e_port"],
	}, nil
}

// RegisterMonitoringTarget registers a new monitoring target (host) in Nightingale
func (s *NightingaleService) RegisterMonitoringTarget(hostname, ip string, tags []string) error {
	log.Printf("Registering monitoring target in Nightingale: %s (%s)", hostname, ip)

	// Use hostname or IP as identifier
	ident := hostname
	if ident == "" {
		ident = ip
	}

	// Check if target already exists
	var existingTarget models.NightingaleTarget
	result := s.nightingaleDB.Where("ident = ?", ident).First(&existingTarget)

	// Build tags string
	tagsStr := ""
	if len(tags) > 0 {
		for i, tag := range tags {
			if i > 0 {
				tagsStr += ","
			}
			tagsStr += tag
		}
	}

	currentTime := time.Now().Unix()

	if result.Error == gorm.ErrRecordNotFound {
		// Create new target
		newTarget := &models.NightingaleTarget{
			Ident:    ident,
			Note:     fmt.Sprintf("Host: %s, IP: %s", hostname, ip),
			Tags:     tagsStr,
			UpdateAt: currentTime,
		}

		if err := s.nightingaleDB.Create(newTarget).Error; err != nil {
			return fmt.Errorf("failed to create monitoring target: %w", err)
		}

		log.Printf("✓ Monitoring target '%s' registered in Nightingale", ident)
	} else if result.Error != nil {
		return fmt.Errorf("failed to query monitoring target: %w", result.Error)
	} else {
		// Update existing target
		existingTarget.Note = fmt.Sprintf("Host: %s, IP: %s", hostname, ip)
		existingTarget.Tags = tagsStr
		existingTarget.UpdateAt = currentTime

		if err := s.nightingaleDB.Save(&existingTarget).Error; err != nil {
			return fmt.Errorf("failed to update monitoring target: %w", err)
		}

		log.Printf("✓ Monitoring target '%s' updated in Nightingale", ident)
	}

	return nil
}

// UnregisterMonitoringTarget removes a monitoring target from Nightingale
func (s *NightingaleService) UnregisterMonitoringTarget(hostname string) error {
	log.Printf("Unregistering monitoring target from Nightingale: %s", hostname)

	if err := s.nightingaleDB.Where("ident = ?", hostname).Delete(&models.NightingaleTarget{}).Error; err != nil {
		return fmt.Errorf("failed to delete monitoring target: %w", err)
	}

	log.Printf("✓ Monitoring target '%s' removed from Nightingale", hostname)
	return nil
}

// GetMonitoringTargets retrieves all registered monitoring targets
func (s *NightingaleService) GetMonitoringTargets() ([]models.NightingaleTarget, error) {
	var targets []models.NightingaleTarget
	if err := s.nightingaleDB.Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to get monitoring targets: %w", err)
	}
	return targets, nil
}

// SyncHostToMonitoring syncs a host to Nightingale monitoring system
func (s *NightingaleService) SyncHostToMonitoring(hostname, ip string, tags []string) error {
	return s.RegisterMonitoringTarget(hostname, ip, tags)
}

// CategrafInstallParams Categraf 安装参数
type CategrafInstallParams struct {
	Hostname        string
	HostIP          string
	N9EHost         string
	N9EPort         string
	AppHubURL       string
	GitHubMirror    string
	CategrafVersion string
}

// GetMonitoringAgentInstallScript returns the installation command for monitoring agent (Categraf).
// Uses external template from scripts/templates/categraf-install.sh.tmpl
func (s *NightingaleService) GetMonitoringAgentInstallScript(hostname, ip string) string {
	// Build AppHub URL from environment variables
	apphubURL := os.Getenv("APPHUB_URL")
	if apphubURL == "" {
		externalHost := os.Getenv("EXTERNAL_HOST")
		apphubPort := os.Getenv("APPHUB_PORT")
		if externalHost != "" && apphubPort != "" {
			apphubURL = fmt.Sprintf("http://%s:%s", externalHost, apphubPort)
		}
	}

	// Get configuration from environment
	params := CategrafInstallParams{
		Hostname:        hostname,
		HostIP:          ip,
		N9EHost:         s.nightingaleHost,
		N9EPort:         s.nightingalePort,
		AppHubURL:       apphubURL,
		GitHubMirror:    os.Getenv("GITHUB_MIRROR"),
		CategrafVersion: os.Getenv("CATEGRAF_VERSION"),
	}

	// Generate wrapper script that sets environment and calls the template
	// The actual installation logic is in the template file
	script := fmt.Sprintf(`#!/bin/bash
# Categraf Installation Wrapper
# Generated by AI-Infra-Matrix NightingaleService

set -e

# Export configuration from environment
export HOSTNAME="%s"
export HOST_IP="%s"
export N9E_HOST="%s"
export N9E_PORT="%s"
export APPHUB_URL="%s"
export GITHUB_MIRROR="%s"
export CATEGRAF_VERSION="${CATEGRAF_VERSION:-%s}"

# Download and execute the installation script
SCRIPT_URL="${APPHUB_URL}/scripts/categraf/install-categraf.sh"

if [ -n "${APPHUB_URL}" ]; then
    echo "Downloading installation script from AppHub..."
    curl -fsSL "${SCRIPT_URL}" | bash
else
    echo "ERROR: APPHUB_URL is not configured"
    echo "Please set APPHUB_URL environment variable or configure AppHub"
    exit 1
fi
`, params.Hostname, params.HostIP, params.N9EHost, params.N9EPort,
		params.AppHubURL, params.GitHubMirror, params.CategrafVersion)

	return script
}

// GetMonitoringAgentStatus checks if monitoring agent is running on a target
func (s *NightingaleService) GetMonitoringAgentStatus(hostname string) (bool, error) {
	// Check if target exists and has recent heartbeat
	var target models.NightingaleTarget
	if err := s.nightingaleDB.Where("ident = ?", hostname).First(&target).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to query target: %w", err)
	}

	// Check if last update is within 2 minutes (recent heartbeat)
	currentTime := time.Now().Unix()
	if currentTime-target.UpdateAt < 120 {
		return true, nil
	}

	return false, nil
}
