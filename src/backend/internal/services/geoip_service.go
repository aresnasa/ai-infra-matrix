package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// GeoIPInfo 地理位置信息
type GeoIPInfo struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	ASN         string  `json:"asn"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
	IsProxy     bool    `json:"is_proxy"`
	IsVPN       bool    `json:"is_vpn"`
	IsTor       bool    `json:"is_tor"`
	IsDatacenter bool   `json:"is_datacenter"`
	RiskLevel   string  `json:"risk_level"` // low, medium, high
	Source      string  `json:"source"`     // 数据来源
}

// GeoIPService GeoIP 查询服务
type GeoIPService struct {
	cache       sync.Map // 缓存 IP 查询结果
	cacheTTL    time.Duration
	httpClient  *http.Client
	rateLimiter chan struct{}
}

// NewGeoIPService 创建 GeoIP 服务实例
func NewGeoIPService() *GeoIPService {
	return &GeoIPService{
		cacheTTL: 24 * time.Hour, // 缓存24小时
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimiter: make(chan struct{}, 10), // 限制并发请求数
	}
}

// cacheEntry 缓存条目
type cacheEntry struct {
	info      *GeoIPInfo
	expiresAt time.Time
}

// Lookup 查询 IP 地理位置信息
func (s *GeoIPService) Lookup(ip string) (*GeoIPInfo, error) {
	// 验证 IP 格式
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// 检查是否为内网 IP
	if s.isPrivateIP(parsedIP) {
		return &GeoIPInfo{
			IP:          ip,
			Country:     "Private Network",
			CountryCode: "XX",
			Region:      "Internal",
			City:        "Internal",
			ISP:         "Private Network",
			RiskLevel:   "low",
			Source:      "internal",
		}, nil
	}

	// 检查缓存
	if cached, ok := s.cache.Load(ip); ok {
		entry := cached.(*cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.info, nil
		}
		// 缓存过期，删除
		s.cache.Delete(ip)
	}

	// 查询 GeoIP 信息
	info, err := s.queryGeoIP(ip)
	if err != nil {
		logrus.WithError(err).WithField("ip", ip).Warn("Failed to query GeoIP")
		// 返回基本信息
		return &GeoIPInfo{
			IP:        ip,
			Country:   "Unknown",
			RiskLevel: "unknown",
			Source:    "error",
		}, nil
	}

	// 缓存结果
	s.cache.Store(ip, &cacheEntry{
		info:      info,
		expiresAt: time.Now().Add(s.cacheTTL),
	})

	return info, nil
}

// queryGeoIP 从外部 API 查询 GeoIP 信息
func (s *GeoIPService) queryGeoIP(ip string) (*GeoIPInfo, error) {
	// 使用多个数据源进行查询，提高可靠性
	// 优先使用 ip-api.com (免费，每分钟45次请求限制)
	info, err := s.queryIPAPI(ip)
	if err == nil && info != nil {
		return info, nil
	}

	// 备用: ipinfo.io
	info, err = s.queryIPInfo(ip)
	if err == nil && info != nil {
		return info, nil
	}

	// 备用: ip.sb (国内友好)
	info, err = s.queryIPSB(ip)
	if err == nil && info != nil {
		return info, nil
	}

	return nil, fmt.Errorf("all GeoIP sources failed for IP: %s", ip)
}

// queryIPAPI 查询 ip-api.com
func (s *GeoIPService) queryIPAPI(ip string) (*GeoIPInfo, error) {
	// 限流
	select {
	case s.rateLimiter <- struct{}{}:
		defer func() { <-s.rateLimiter }()
	default:
		return nil, fmt.Errorf("rate limit exceeded")
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,proxy,hosting,query", ip)
	
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Status      string  `json:"status"`
		Message     string  `json:"message"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"region"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		AS          string  `json:"as"`
		Proxy       bool    `json:"proxy"`
		Hosting     bool    `json:"hosting"`
		Query       string  `json:"query"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("ip-api error: %s", result.Message)
	}

	info := &GeoIPInfo{
		IP:           ip,
		Country:      result.Country,
		CountryCode:  result.CountryCode,
		Region:       result.RegionName,
		City:         result.City,
		ISP:          result.ISP,
		Org:          result.Org,
		ASN:          result.AS,
		Latitude:     result.Lat,
		Longitude:    result.Lon,
		Timezone:     result.Timezone,
		IsProxy:      result.Proxy,
		IsDatacenter: result.Hosting,
		Source:       "ip-api.com",
	}

	// 计算风险等级
	info.RiskLevel = s.calculateRiskLevel(info)

	return info, nil
}

// queryIPInfo 查询 ipinfo.io
func (s *GeoIPService) queryIPInfo(ip string) (*GeoIPInfo, error) {
	select {
	case s.rateLimiter <- struct{}{}:
		defer func() { <-s.rateLimiter }()
	default:
		return nil, fmt.Errorf("rate limit exceeded")
	}

	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		IP       string `json:"ip"`
		City     string `json:"city"`
		Region   string `json:"region"`
		Country  string `json:"country"`
		Loc      string `json:"loc"`
		Org      string `json:"org"`
		Timezone string `json:"timezone"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	info := &GeoIPInfo{
		IP:          ip,
		Country:     result.Country,
		CountryCode: result.Country,
		Region:      result.Region,
		City:        result.City,
		Org:         result.Org,
		Timezone:    result.Timezone,
		Source:      "ipinfo.io",
	}

	// 解析经纬度
	if result.Loc != "" {
		parts := strings.Split(result.Loc, ",")
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%f", &info.Latitude)
			fmt.Sscanf(parts[1], "%f", &info.Longitude)
		}
	}

	info.RiskLevel = s.calculateRiskLevel(info)
	return info, nil
}

// queryIPSB 查询 ip.sb (国内友好)
func (s *GeoIPService) queryIPSB(ip string) (*GeoIPInfo, error) {
	select {
	case s.rateLimiter <- struct{}{}:
		defer func() { <-s.rateLimiter }()
	default:
		return nil, fmt.Errorf("rate limit exceeded")
	}

	url := fmt.Sprintf("https://api.ip.sb/geoip/%s", ip)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AI-Infra-Matrix/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		IP          string  `json:"ip"`
		Country     string  `json:"country"`
		CountryCode string  `json:"country_code"`
		Region      string  `json:"region"`
		City        string  `json:"city"`
		ISP         string  `json:"isp"`
		Org         string  `json:"organization"`
		ASN         int     `json:"asn"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Timezone    string  `json:"timezone"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	info := &GeoIPInfo{
		IP:          ip,
		Country:     result.Country,
		CountryCode: result.CountryCode,
		Region:      result.Region,
		City:        result.City,
		ISP:         result.ISP,
		Org:         result.Org,
		ASN:         fmt.Sprintf("AS%d", result.ASN),
		Latitude:    result.Latitude,
		Longitude:   result.Longitude,
		Timezone:    result.Timezone,
		Source:      "ip.sb",
	}

	info.RiskLevel = s.calculateRiskLevel(info)
	return info, nil
}

// calculateRiskLevel 计算风险等级
func (s *GeoIPService) calculateRiskLevel(info *GeoIPInfo) string {
	score := 0

	// 代理/VPN/Tor 检测
	if info.IsProxy || info.IsVPN {
		score += 30
	}
	if info.IsTor {
		score += 50
	}
	if info.IsDatacenter {
		score += 20
	}

	// 高风险国家/地区检测 (可以根据业务需求调整)
	highRiskCountries := map[string]bool{
		// 常见攻击来源国家
	}
	if highRiskCountries[info.CountryCode] {
		score += 20
	}

	// 根据分数返回风险等级
	switch {
	case score >= 50:
		return "high"
	case score >= 30:
		return "medium"
	default:
		return "low"
	}
}

// isPrivateIP 检查是否为内网 IP
func (s *GeoIPService) isPrivateIP(ip net.IP) bool {
	// IPv4 私有地址范围
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"100.64.0.0/10", // CGNAT
		"169.254.0.0/16", // Link-local
	}

	// IPv6 私有地址范围
	privateRangesV6 := []string{
		"::1/128",       // Loopback
		"fc00::/7",      // Unique local
		"fe80::/10",     // Link-local
	}

	ranges := privateRanges
	if ip.To4() == nil {
		ranges = privateRangesV6
	}

	for _, r := range ranges {
		_, network, err := net.ParseCIDR(r)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// BatchLookup 批量查询 IP
func (s *GeoIPService) BatchLookup(ips []string) map[string]*GeoIPInfo {
	results := make(map[string]*GeoIPInfo)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, ip := range ips {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()
			info, _ := s.Lookup(ipAddr)
			mu.Lock()
			results[ipAddr] = info
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	return results
}

// ClearCache 清除缓存
func (s *GeoIPService) ClearCache() {
	s.cache = sync.Map{}
}

// GetCacheStats 获取缓存统计
func (s *GeoIPService) GetCacheStats() map[string]interface{} {
	count := 0
	s.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return map[string]interface{}{
		"cached_entries": count,
		"cache_ttl":      s.cacheTTL.String(),
	}
}
