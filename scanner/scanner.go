package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ConfigInfo represents a parsed configuration
type ConfigInfo struct {
	Link    string
	Remark  string
	Latency int
	Host    string
	Port    int
}

// Scanner handles the scanning of configuration files
type Scanner struct {
	timeout       time.Duration
	results       map[string][]ConfigInfo
	mu            sync.RWMutex
	enableLatency bool
}

// NewScanner creates a new scanner instance
func NewScanner(timeout time.Duration) *Scanner {
	return &Scanner{
		timeout:       timeout,
		results:       make(map[string][]ConfigInfo),
		enableLatency: false, // Default to false for speed
	}
}

// SetLatencyMeasurement enables or disables latency measurement
func (s *Scanner) SetLatencyMeasurement(enable bool) {
	s.enableLatency = enable
}

// ScanDirectory scans all sub*.txt files in the given directory
func (s *Scanner) ScanDirectory(dirPath string) error {
	return s.ScanDirectoryInteractive(dirPath, false)
}

// ScanDirectoryInteractive scans all sub*.txt files with interactive progress
func (s *Scanner) ScanDirectoryInteractive(dirPath string, quiet bool) error {
	if !quiet {
		fmt.Printf("🔍 Scanning directory: %s\n", dirPath)
	}

	// Find all sub*.txt and Sub*.txt files
	patterns := []string{
		filepath.Join(dirPath, "sub*.txt"),
		filepath.Join(dirPath, "Sub*.txt"),
	}

	var allMatches []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("error finding files: %v", err)
		}
		allMatches = append(allMatches, matches...)
	}

	// If no local files found, try fallback to GitHub repository
	if len(allMatches) == 0 {
		if !quiet {
			fmt.Println("📡 No local files found, attempting fallback to GitHub repository...")
		}
		return s.scanFromGitHub(quiet)
	}

	matches := allMatches

	if !quiet {
		fmt.Printf("📄 Found %d files to scan\n", len(matches))
	}

	// Process each file with progress indication
	for i, filePath := range matches {
		if !quiet {
			// Show progress with spinner
			spinner := s.getSpinnerChar(i)
			fmt.Printf("\r%s📁 Scanning file: %s (%d/%d)", spinner, filepath.Base(filePath), i+1, len(matches))
		}

		if err := s.scanFile(filePath); err != nil {
			if !quiet {
				fmt.Printf("\n⚠️ Error scanning %s: %v\n", filePath, err)
			}
			continue
		}
	}

	if !quiet {
		fmt.Println() // New line after progress
	}

	return nil
}

// getSpinnerChar returns a spinner character for progress indication
func (s *Scanner) getSpinnerChar(index int) string {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return spinnerChars[index%len(spinnerChars)]
}

// scanFromGitHub fetches configurations from GitHub repository as fallback
func (s *Scanner) scanFromGitHub(quiet bool) error {
	if !quiet {
		fmt.Println("🌐 Fetching configurations from GitHub repository...")
	}

	// GitHub raw URL for the AllConfigsSub.txt file
	githubURL := "https://raw.githubusercontent.com/Danialsamadi/v2go/main/AllConfigsSub.txt"

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make HTTP request
	resp, err := client.Get(githubURL)
	if err != nil {
		return fmt.Errorf("failed to fetch from GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read GitHub response: %v", err)
	}

	if !quiet {
		fmt.Printf("📡 Downloaded %d bytes from GitHub\n", len(body))
		fmt.Println("🔄 Processing configurations...")
	}

	// Process the content as if it were a file
	content := string(body)
	lines := strings.Split(content, "\n")

	configCount := 0
	linkRegex := regexp.MustCompile(`(vmess://[^\s]+|vless://[^\s]+|trojan://[^\s]+|ss://[^\s]+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract links from the line
		matches := linkRegex.FindAllString(line, -1)
		for _, link := range matches {
			s.processLink(link)
			configCount++
		}
	}

	if !quiet {
		fmt.Printf("✅ Processed %d configurations from GitHub\n", configCount)
	}

	return nil
}

// scanFile processes a single configuration file
func (s *Scanner) scanFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	linkRegex := regexp.MustCompile(`(vmess://[^\s]+|vless://[^\s]+|trojan://[^\s]+|ss://[^\s]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Extract links from the line
		matches := linkRegex.FindAllString(line, -1)
		for _, link := range matches {
			s.processLink(link)
		}
	}

	return scanner.Err()
}

// processLink processes a single configuration link
func (s *Scanner) processLink(link string) {
	host, port, remark, protocol := s.decodeLink(link)
	if host == "" || port == 0 {
		return
	}

	latency := 0
	if s.enableLatency {
		// Measure latency only if enabled
		latency = s.measureLatency(host, port)
		if latency == -1 || latency > 800 {
			return // Skip if latency measurement failed or too slow
		}
	}

	config := ConfigInfo{
		Link:    link,
		Remark:  remark,
		Latency: latency,
		Host:    host,
		Port:    port,
	}

	s.mu.Lock()
	s.results[protocol] = append(s.results[protocol], config)
	s.mu.Unlock()
}

// decodeLink decodes a configuration link and returns host, port, remark, and protocol
func (s *Scanner) decodeLink(link string) (host string, port int, remark string, protocol string) {
	switch {
	case strings.HasPrefix(link, "vmess://"):
		return s.decodeVMess(link)
	case strings.HasPrefix(link, "vless://"):
		return s.decodeVLess(link)
	case strings.HasPrefix(link, "trojan://"):
		return s.decodeTrojan(link)
	case strings.HasPrefix(link, "ss://"):
		return s.decodeSS(link)
	default:
		return "", 0, "", ""
	}
}

// decodeVMess decodes a VMess configuration
func (s *Scanner) decodeVMess(link string) (host string, port int, remark string, protocol string) {
	protocol = "vmess"

	// Remove vmess:// prefix
	encoded := strings.TrimPrefix(link, "vmess://")

	// Add padding if necessary
	if len(encoded)%4 != 0 {
		encoded += strings.Repeat("=", 4-len(encoded)%4)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", 0, "", protocol
	}

	var config map[string]interface{}
	if err := json.Unmarshal(decoded, &config); err != nil {
		return "", 0, "", protocol
	}

	host, _ = config["add"].(string)

	// Handle port as both string and number
	if portFloat, ok := config["port"].(float64); ok {
		port = int(portFloat)
	} else if portStr, ok := config["port"].(string); ok {
		port, _ = strconv.Atoi(portStr)
	}

	remark, _ = config["ps"].(string)

	return host, port, remark, protocol
}

// decodeVLess decodes a VLess configuration
func (s *Scanner) decodeVLess(link string) (host string, port int, remark string, protocol string) {
	protocol = "vless"
	return s.decodeGeneric(link, "vless://")
}

// decodeTrojan decodes a Trojan configuration
func (s *Scanner) decodeTrojan(link string) (host string, port int, remark string, protocol string) {
	protocol = "trojan"
	return s.decodeGeneric(link, "trojan://")
}

// decodeSS decodes a Shadowsocks configuration
func (s *Scanner) decodeSS(link string) (host string, port int, remark string, protocol string) {
	protocol = "ss"
	return s.decodeGeneric(link, "ss://")
}

// decodeGeneric decodes generic URL-based configurations
func (s *Scanner) decodeGeneric(link, prefix string) (host string, port int, remark string, protocol string) {
	// Extract protocol from prefix
	protocol = strings.TrimSuffix(prefix, "://")

	parsedURL, err := url.Parse(link)
	if err != nil {
		return "", 0, "", protocol
	}

	host = parsedURL.Hostname()
	portStr := parsedURL.Port()
	if portStr != "" {
		port, _ = strconv.Atoi(portStr)
	}

	remark = parsedURL.Fragment
	if remark == "" {
		remark = "NoRemark"
	}

	return host, port, remark, protocol
}

// measureLatency measures the latency to a host:port
func (s *Scanner) measureLatency(host string, port int) int {
	address := fmt.Sprintf("%s:%d", host, port)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, s.timeout)
	if err != nil {
		return -1
	}
	defer conn.Close()

	latency := int(time.Since(start).Milliseconds())
	return latency
}

// SaveResults saves the scan results to files
func (s *Scanner) SaveResults() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	savedFiles := 0
	totalConfigs := 0

	for protocol, configs := range s.results {
		if len(configs) == 0 {
			continue
		}

		// Sort by latency
		sort.Slice(configs, func(i, j int) bool {
			return configs[i].Latency < configs[j].Latency
		})

		// Split into fast and normal (or all if no latency measured)
		var fast, normal []string
		for _, config := range configs {
			line := fmt.Sprintf("%s # %s", config.Link, config.Remark)
			if config.Latency > 0 {
				line = fmt.Sprintf("%s - %d ms", line, config.Latency)
				if config.Latency < 200 {
					fast = append(fast, line)
				} else {
					normal = append(normal, line)
				}
			} else {
				// No latency measured, put in normal category
				normal = append(normal, line)
			}
		}

		// Save fast configs
		if len(fast) > 0 {
			filename := s.getSafeFilename(fmt.Sprintf("fast_%s.txt", protocol))
			if err := s.writeFile(filename, fast); err != nil {
				return fmt.Errorf("error writing fast %s file: %v", protocol, err)
			}
			fmt.Printf("✅ Saved %d fast %s configs to %s\n", len(fast), protocol, filename)
			savedFiles++
			totalConfigs += len(fast)
		}

		// Save normal configs
		if len(normal) > 0 {
			filename := s.getSafeFilename(fmt.Sprintf("%s.txt", protocol))
			if err := s.writeFile(filename, normal); err != nil {
				return fmt.Errorf("error writing %s file: %v", protocol, err)
			}
			fmt.Printf("✅ Saved %d normal %s configs to %s\n", len(normal), protocol, filename)
			savedFiles++
			totalConfigs += len(normal)
		}
	}

	if savedFiles > 0 {
		fmt.Printf("\n💾 Summary: %d files created with %d total configurations\n", savedFiles, totalConfigs)
	} else {
		fmt.Println("⚠️ No configurations found to save")
	}

	return nil
}

// getSafeFilename ensures the filename is safe for the current platform
func (s *Scanner) getSafeFilename(filename string) string {
	// Replace any problematic characters for cross-platform compatibility
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")

	// Ensure the filename is not too long
	if len(filename) > 200 {
		filename = filename[:200]
	}

	return filename
}

// writeFile writes a slice of strings to a file
func (s *Scanner) writeFile(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// GetResults returns the current scan results
func (s *Scanner) GetResults() map[string][]ConfigInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string][]ConfigInfo)
	for protocol, configs := range s.results {
		result[protocol] = make([]ConfigInfo, len(configs))
		copy(result[protocol], configs)
	}

	return result
}

// PrintSummary prints a summary of the scan results
func (s *Scanner) PrintSummary() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fmt.Println("\n📊 Scan Summary:")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	total := 0
	protocols := []string{"vmess", "vless", "trojan", "ss"}

	for _, protocol := range protocols {
		configs, exists := s.results[protocol]
		count := 0
		if exists {
			count = len(configs)
		}
		total += count

		// Create a visual bar for the count
		bar := s.createProgressBar(count, 20)
		fmt.Printf("  %-8s: %3d configs %s\n", strings.ToUpper(protocol), count, bar)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("  %-8s: %3d configs\n", "TOTAL", total)

	// Show platform info
	fmt.Printf("\n🖥️  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// createProgressBar creates a visual progress bar
func (s *Scanner) createProgressBar(count, maxWidth int) string {
	if count == 0 {
		return strings.Repeat("░", maxWidth)
	}

	// Simple bar representation
	filled := count / 5 // Scale down for display
	if filled > maxWidth {
		filled = maxWidth
	}

	bar := strings.Repeat("█", filled)
	bar += strings.Repeat("░", maxWidth-filled)

	return "[" + bar + "]"
}
