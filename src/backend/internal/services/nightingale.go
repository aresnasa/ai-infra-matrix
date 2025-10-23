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

// GetMonitoringAgentInstallScript generates the installation script for monitoring agent
func (s *NightingaleService) GetMonitoringAgentInstallScript(hostname, ip string) string {
	// Categraf is the monitoring agent used by Nightingale
	script := fmt.Sprintf(`#!/bin/bash
# Install Nightingale Monitoring Agent (Categraf)

set -e

echo "=== Installing Nightingale Monitoring Agent ==="

# Download and install Categraf
CATEGRAF_VERSION="v0.3.80"
CATEGRAF_URL="https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/categraf-${CATEGRAF_VERSION}-linux-amd64.tar.gz"

# Create installation directory
sudo mkdir -p /opt/categraf
cd /opt/categraf

# Download Categraf
echo "Downloading Categraf ${CATEGRAF_VERSION}..."
sudo wget -q ${CATEGRAF_URL} -O categraf.tar.gz

# Extract
echo "Extracting Categraf..."
sudo tar -xzf categraf.tar.gz --strip-components=1
sudo rm -f categraf.tar.gz

# Configure Categraf
echo "Configuring Categraf..."
sudo cat > /opt/categraf/conf/config.toml <<EOF
[global]
hostname = "%s"
labels = { ip="%s" }

[heartbeat]
enable = true
url = "http://%s:%s/v1/n9e/heartbeat"
interval = 10

[writer_opt]
batch = 2000
chan_size = 10000

[[writers]]
url = "http://%s:%s/prometheus/v1/write"
EOF

# Enable system collectors
sudo mkdir -p /opt/categraf/conf/input.cpu
sudo cat > /opt/categraf/conf/input.cpu/cpu.toml <<EOF
[[instances]]
collect_per_cpu = true
report_active = true
EOF

sudo mkdir -p /opt/categraf/conf/input.mem
sudo cat > /opt/categraf/conf/input.mem/mem.toml <<EOF
[[instances]]
# collect memory stats
EOF

sudo mkdir -p /opt/categraf/conf/input.disk
sudo cat > /opt/categraf/conf/input.disk/disk.toml <<EOF
[[instances]]
mount_points = ["/"]
ignore_fs = ["tmpfs", "devtmpfs", "devfs", "iso9660", "overlay", "aufs", "squashfs"]
EOF

sudo mkdir -p /opt/categraf/conf/input.net
sudo cat > /opt/categraf/conf/input.net/net.toml <<EOF
[[instances]]
interfaces = ["eth*", "en*"]
EOF

sudo mkdir -p /opt/categraf/conf/input.netstat
sudo cat > /opt/categraf/conf/input.netstat/netstat.toml <<EOF
[[instances]]
# collect netstat
EOF

# Create systemd service
echo "Creating systemd service..."
sudo cat > /etc/systemd/system/categraf.service <<'SVCEOF'
[Unit]
Description=Categraf Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/opt/categraf/categraf --configs /opt/categraf/conf
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
SVCEOF

# Start service
echo "Starting Categraf service..."
sudo systemctl daemon-reload
sudo systemctl enable categraf
sudo systemctl start categraf

# Check status
echo "Checking service status..."
sudo systemctl status categraf --no-pager

echo "✓ Nightingale Monitoring Agent installed successfully!"
echo "  Hostname: %s"
echo "  IP: %s"
echo "  Reporting to: http://%s:%s"
`, hostname, ip, s.nightingaleHost, s.nightingalePort, s.nightingaleHost, s.nightingalePort, hostname, ip, s.nightingaleHost, s.nightingalePort)

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
