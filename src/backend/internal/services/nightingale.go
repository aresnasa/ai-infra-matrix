package services

import (
	"fmt"
	"log"
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

// GetMonitoringAgentInstallScript returns the installation command for monitoring agent.
// The actual script is served from the Nightingale container at port 17002 (script server).
// Environment variables are passed to configure the installation.
func (s *NightingaleService) GetMonitoringAgentInstallScript(hostname, ip string) string {
	// Script server runs on port 17002 inside Nightingale container
	scriptServerPort := "17002"

	// Return a simple curl command that fetches and executes the script from Nightingale container
	// The script supports environment variables for configuration
	script := fmt.Sprintf(`#!/bin/bash
# Install Nightingale Monitoring Agent (Categraf)
# This script downloads and runs the installation script from Nightingale server

set -e

# Configuration
export HOSTNAME="%s"
export HOST_IP="%s"
export N9E_HOST="%s"
export N9E_PORT="%s"

# Optional: Set mirror configuration via environment variables before running
# export GITHUB_MIRROR="https://ghfast.top"
# export APPHUB_URL="http://apphub:80"

# Download and execute the installation script from script server (port %s)
echo "Downloading installation script from Nightingale..."
curl -fsSL "http://${N9E_HOST}:%s/install-categraf.sh" | bash

echo "Done!"
`, hostname, ip, s.nightingaleHost, s.nightingalePort, scriptServerPort, scriptServerPort)

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
