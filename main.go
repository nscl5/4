package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	timeout         = 20 * time.Second
	maxWorkers      = 10
	maxLinesPerFile = 500
)

var fixedText = `#profile-title: base64:8J+GkyBHaXRodWIgfCBEYW5pYWwgU2FtYWRpIPCfkI0=
#profile-update-interval: 1
#support-url: https://github.com/Danialsamadi/v2go
#profile-web-page-url: https://github.com/Danialsamadi/v2go
`

var protocols = []string{"vmess", "vless", "trojan", "ss", "ssr", "hy2", "tuic", "warp://"}

var links = []string{
	"https://raw.githubusercontent.com/ALIILAPRO/v2rayNG-Config/main/sub.txt",
	"https://raw.githubusercontent.com/mfuu/v2ray/master/v2ray",
	"https://raw.githubusercontent.com/ts-sf/fly/main/v2",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mci/sub_1.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mci/sub_2.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mci/sub_3.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/app/sub.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mtn/sub_1.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mtn/sub_2.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mtn/sub_3.txt",
	"https://raw.githubusercontent.com/mahsanet/MahsaFreeConfig/refs/heads/main/mtn/sub_4.txt",
	"https://raw.githubusercontent.com/yebekhe/vpn-fail/refs/heads/main/sub-link",
}

var dirLinks = []string{
	"https://v2.alicivil.workers.dev",
	"https://raw.githubusercontent.com/Surfboardv2ray/TGParse/main/splitted/mixed",
	"https://raw.githubusercontent.com/itsyebekhe/PSG/main/lite/subscriptions/xray/normal/mix",
	"https://raw.githubusercontent.com/HosseinKoofi/GO_V2rayCollector/main/mixed_iran.txt",
	"https://raw.githubusercontent.com/arshiacomplus/v2rayExtractor/refs/heads/main/mix/sub.html",
	"https://raw.githubusercontent.com/Rayan-Config/C-Sub/refs/heads/main/configs/proxy.txt",
	"https://raw.githubusercontent.com/mahdibland/ShadowsocksAggregator/master/Eternity.txt",
	"https://raw.githubusercontent.com/Everyday-VPN/Everyday-VPN/main/subscription/main.txt",
	"https://raw.githubusercontent.com/MahsaNetConfigTopic/config/refs/heads/main/xray_final.txt",
	"https://github.com/Epodonios/v2ray-configs/raw/main/All_Configs_Sub.txt",
	"https://raw.githubusercontent.com/V2RayRoot/V2RayConfig/refs/heads/main/Config/vless.txt",
	"https://raw.githubusercontent.com/V2RayRoot/V2RayConfig/refs/heads/main/Config/vmess.txt",
	"https://raw.githubusercontent.com/ebrasha/free-v2ray-public-list/refs/heads/main/all_extracted_configs.txt",
	"https://raw.githubusercontent.com/miladtahanian/V2RayScrapeByCountry/refs/heads/main/output_configs/Vless.txt",
	"https://raw.githubusercontent.com/miladtahanian/V2RayScrapeByCountry/refs/heads/main/output_configs/Vmess.txt",
	"https://raw.githubusercontent.com/miladtahanian/V2ray-Config/main/All_Configs_Sub.txt",
}

type Result struct {
	URL        string
	Content    string
	IsBase64   bool
	StatusCode int
	Error      error
}

func main() {
	start := time.Now()
	fmt.Println("Starting V2Ray config aggregator...")

	// Ensure directories exist
	base64Folder, err := ensureDirectoriesExist()
	if err != nil {
		fmt.Printf("Error creating directories: %v\n", err)
		return
	}

	// Create HTTP client with connection pooling
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	// Fetch all URLs concurrently
	fmt.Println("Fetching configurations from sources...")
	allConfigs, failedLinks := fetchAllConfigs(client, links, dirLinks)

	// Filter for protocols
	fmt.Println("Filtering configurations and removing duplicates...")
	originalCount := len(allConfigs)
	filteredConfigs := filterForProtocols(allConfigs, protocols)

	fmt.Printf("Found %d unique valid configurations\n", len(filteredConfigs))
	fmt.Printf("Removed %d duplicates\n", originalCount-len(filteredConfigs))

	// Clean existing files
	cleanExistingFiles(base64Folder)

	// Write main config file (in current directory)
	mainOutputFile := "All_Configs_Sub.txt"
	err = writeMainConfigFile(mainOutputFile, filteredConfigs)
	if err != nil {
		fmt.Printf("Error writing main config file: %v\n", err)
		return
	}

	// Split into smaller files
	fmt.Println("Splitting into smaller files...")
	err = splitIntoFiles(base64Folder, filteredConfigs)
	if err != nil {
		fmt.Printf("Error splitting files: %v\n", err)
		return
	}

	// Calculate protocol statistics
	stats := calculateStats(filteredConfigs)

	// Write summary to UPDATE_SUMMARY.md
	processingTime := time.Since(start).Seconds()
	writeUpdateSummary(len(filteredConfigs), stats, processingTime, originalCount, failedLinks)

	fmt.Println("Configuration aggregation completed successfully!")

	// Now sort configurations by protocol
	sortConfigs()
}

func ensureDirectoriesExist() (string, error) {
	// Create Base64 directory in current directory
	base64Folder := "Base64"
	if err := os.MkdirAll(base64Folder, 0755); err != nil {
		return "", err
	}

	return base64Folder, nil
}

func fetchAllConfigs(client *http.Client, base64Links, textLinks []string) ([]string, []string) {
	var wg sync.WaitGroup
	resultChan := make(chan Result, len(base64Links)+len(textLinks))
	var failedLinks []string

	// Worker pool for concurrent requests
	semaphore := make(chan struct{}, maxWorkers)

	// Fetch base64-encoded links
	for _, link := range base64Links {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			res := fetchAndDecodeBase64(client, url)
			resultChan <- res
		}(link)
	}

	// Fetch text links
	for _, link := range textLinks {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			res := fetchText(client, url)
			resultChan <- res
		}(link)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allConfigs []string
	for result := range resultChan {
		if result.StatusCode != http.StatusOK || result.Error != nil {
			status := "Error"
			if result.StatusCode > 0 {
				status = fmt.Sprintf("HTTP %d", result.StatusCode)
			}
			failedLinks = append(failedLinks, fmt.Sprintf("%s (%s)", result.URL, status))
			continue
		}

		lines := strings.Split(strings.TrimSpace(result.Content), "\n")
		allConfigs = append(allConfigs, lines...)
	}

	return allConfigs, failedLinks
}

func fetchAndDecodeBase64(client *http.Client, url string) Result {
	res := Result{URL: url, IsBase64: true}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		res.Error = err
		return res
	}

	resp, err := client.Do(req)
	if err != nil {
		res.Error = err
		return res
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		return res
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = err
		return res
	}

	// Try to decode base64
	decoded, err := decodeBase64(body)
	if err != nil {
		res.Error = err
		return res
	}

	res.Content = decoded
	return res
}

func fetchText(client *http.Client, url string) Result {
	res := Result{URL: url, IsBase64: false}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		res.Error = err
		return res
	}

	resp, err := client.Do(req)
	if err != nil {
		res.Error = err
		return res
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		return res
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = err
		return res
	}

	res.Content = string(body)
	return res
}

func decodeBase64(encoded []byte) (string, error) {
	// Add padding if necessary
	encodedStr := string(encoded)
	if len(encodedStr)%4 != 0 {
		encodedStr += strings.Repeat("=", 4-len(encodedStr)%4)
	}

	decoded, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func filterForProtocols(data []string, protocols []string) []string {
	var filtered []string
	seen := make(map[string]bool) // Track unique server identity (Protocol + Host + Port)

	for _, line := range data {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Identify protocol
		var currentProtocol string
		for _, protocol := range protocols {
			prefix := protocol
			if !strings.HasSuffix(prefix, "://") && protocol != "warp://" {
				prefix += "://"
			}
			if strings.HasPrefix(line, prefix) {
				currentProtocol = protocol
				break
			}
		}

		if currentProtocol == "" {
			continue
		}

		// Smart Deduplication: Parse core identity (Address + Port)
		identity := parseCoreIdentity(line, currentProtocol)
		if seen[identity] {
			continue
		}

		// Clean Namer: Standardize the name
		cleanLine := standardizeName(line, currentProtocol, len(filtered)+1)
		filtered = append(filtered, cleanLine)
		seen[identity] = true
	}
	return filtered
}

// standardizeName renames a configuration to a professional format: v2go | Protocol | ID
func standardizeName(config string, protocol string, index int) string {
	newName := fmt.Sprintf("v2go | %s | %d", strings.ToUpper(protocol), index)

	switch protocol {
	case "vmess":
		trimmed := strings.TrimPrefix(config, "vmess://")
		decoded, err := decodeBase64([]byte(trimmed))
		if err != nil {
			return config
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(decoded), &data); err != nil {
			return config
		}
		data["ps"] = newName
		updated, _ := json.Marshal(data)
		return "vmess://" + base64.StdEncoding.EncodeToString(updated)

	case "ssr":
		trimmed := strings.TrimPrefix(config, "ssr://")
		decoded, err := decodeBase64([]byte(trimmed))
		if err != nil {
			return config
		}
		// SSR format: host:port:protocol:method:obfs:base64pass/?obfsparam=...&remarks=base64remarks&...
		parts := strings.Split(decoded, "/?")
		if len(parts) < 1 {
			return config
		}

		mainInfo := parts[0]
		params := ""
		if len(parts) > 1 {
			params = parts[1]
		}

		// Handle remarks in params
		paramList := strings.Split(params, "&")
		newParamList := []string{}
		remarksFound := false
		encodedName := strings.ReplaceAll(base64.StdEncoding.EncodeToString([]byte(newName)), "=", "")

		for _, p := range paramList {
			if strings.HasPrefix(p, "remarks=") {
				newParamList = append(newParamList, "remarks="+encodedName)
				remarksFound = true
			} else if p != "" {
				newParamList = append(newParamList, p)
			}
		}
		if !remarksFound {
			newParamList = append(newParamList, "remarks="+encodedName)
		}

		updatedDecoded := mainInfo + "/?" + strings.Join(newParamList, "&")
		return "ssr://" + strings.ReplaceAll(base64.StdEncoding.EncodeToString([]byte(updatedDecoded)), "=", "")

	default:
		// Standard URL protocols: vless, trojan, ss, hy2, tuic
		u, err := url.Parse(config)
		if err != nil {
			// Fallback: if url.Parse fails, try to replace existing fragment
			if strings.Contains(config, "#") {
				return config[:strings.Index(config, "#")] + "#" + url.PathEscape(newName)
			}
			return config + "#" + url.PathEscape(newName)
		}
		u.Fragment = newName
		return u.String()
	}
}

// parseCoreIdentity extracts the Protocol + Host + Port from a config line.
// This allows us to find duplicates that have different names or parameters but point to the same server.
func parseCoreIdentity(config string, protocol string) string {
	config = strings.TrimSpace(config)

	switch protocol {
	case "vmess":
		trimmed := strings.TrimPrefix(config, "vmess://")
		decoded, err := decodeBase64([]byte(trimmed))
		if err != nil {
			return config // Fallback to full string if decoding fails
		}
		var data struct {
			Add  string      `json:"add"`
			Port interface{} `json:"port"` // Use interface because port can be string or int
		}
		if err := json.Unmarshal([]byte(decoded), &data); err != nil {
			return config
		}
		return fmt.Sprintf("vmess://%s:%v", data.Add, data.Port)

	case "ssr":
		trimmed := strings.TrimPrefix(config, "ssr://")
		decoded, err := decodeBase64([]byte(trimmed))
		if err != nil {
			// SSR padding is often weird, try simple trim if padding fails
			return config
		}
		// SSR format: host:port:protocol:method:obfs:base64pass/?obfsparam=...
		parts := strings.Split(decoded, ":")
		if len(parts) >= 2 {
			return fmt.Sprintf("ssr://%s:%s", parts[0], parts[1])
		}
		return config

	default:
		// Works for vless, trojan, ss, hy2, tuic
		u, err := url.Parse(config)
		if err != nil {
			return config
		}
		host := u.Hostname()
		port := u.Port()
		if host == "" {
			return config
		}
		return fmt.Sprintf("%s://%s:%s", protocol, host, port)
	}
}

func cleanExistingFiles(base64Folder string) {
	// Remove main files
	os.Remove("All_Configs_Sub.txt")
	os.Remove("All_Configs_base64_Sub.txt")

	// Remove split files
	for i := 0; i < 20; i++ {
		os.Remove(fmt.Sprintf("Sub%d.txt", i))
		os.Remove(filepath.Join(base64Folder, fmt.Sprintf("Sub%d_base64.txt", i)))
	}
}

func writeMainConfigFile(filename string, configs []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write fixed text
	if _, err := writer.WriteString(fixedText); err != nil {
		return err
	}

	// Write configs
	for _, config := range configs {
		if _, err := writer.WriteString(config + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func splitIntoFiles(base64Folder string, configs []string) error {
	numFiles := (len(configs) + maxLinesPerFile - 1) / maxLinesPerFile

	// Reverse configs so newest go into Sub1, Sub2, etc.
	reversedConfigs := make([]string, len(configs))
	for i, config := range configs {
		reversedConfigs[len(configs)-1-i] = config
	}

	for i := 0; i < numFiles; i++ {
		// Create custom header for this file
		profileTitle := fmt.Sprintf("🆓 Git:DanialSamadi | Sub%d 🔥", i+1)
		encodedTitle := base64.StdEncoding.EncodeToString([]byte(profileTitle))
		customFixedText := fmt.Sprintf(`#profile-title: base64:%s
#profile-update-interval: 1
#support-url: https://github.com/Danialsamadi/v2go
#profile-web-page-url: https://github.com/Danialsamadi/v2go
`, encodedTitle)

		// Calculate slice bounds (using reversed configs)
		start := i * maxLinesPerFile
		end := start + maxLinesPerFile
		if end > len(reversedConfigs) {
			end = len(reversedConfigs)
		}

		// Write regular file (in current directory)
		filename := fmt.Sprintf("Sub%d.txt", i+1)
		if err := writeSubFile(filename, customFixedText, reversedConfigs[start:end]); err != nil {
			return err
		}

		// Read the file and create base64 version
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		base64Filename := filepath.Join(base64Folder, fmt.Sprintf("Sub%d_base64.txt", i+1))
		encodedContent := base64.StdEncoding.EncodeToString(content)
		if err := os.WriteFile(base64Filename, []byte(encodedContent), 0644); err != nil {
			return err
		}
	}

	return nil
}

func writeSubFile(filename, header string, configs []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	if _, err := writer.WriteString(header); err != nil {
		return err
	}

	// Write configs
	for _, config := range configs {
		if _, err := writer.WriteString(config + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func calculateStats(configs []string) map[string]int {
	stats := make(map[string]int)
	for _, config := range configs {
		for _, protocol := range protocols {
			if strings.HasPrefix(config, protocol) {
				stats[protocol]++
				break
			}
		}
	}
	return stats
}

func writeUpdateSummary(total int, stats map[string]int, duration float64, originalTotal int, failedLinks []string) {
	summaryPath := "UPDATE_SUMMARY.md"

	file, err := os.Create(summaryPath)
	if err != nil {
		fmt.Printf("Error creating summary file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	writer.WriteString("# V2Ray Config Update Summary\n")
	writer.WriteString(fmt.Sprintf("Generated on: %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	writer.WriteString("## Configuration Statistics\n")
	writer.WriteString(fmt.Sprintf("- Total unique configurations: %d\n", total))
	writer.WriteString("- Protocol breakdown:\n")

	// Sort protocols for consistent output
	for _, p := range protocols {
		count := stats[p]
		writer.WriteString(fmt.Sprintf("  - %s: %d configs\n", p, count))
	}

	writer.WriteString("\n## Performance\n")
	writer.WriteString(fmt.Sprintf("- Processing time: %.2f seconds\n", duration))
	if originalTotal > 0 {
		reduction := float64(originalTotal-total) / float64(originalTotal) * 100
		writer.WriteString(fmt.Sprintf("- Duplicate removal: %.1f%% reduction (from %d to %d)\n", reduction, originalTotal, total))
	}

	if len(failedLinks) > 0 {
		writer.WriteString("\n## ⚠️ Failed Links (404 or Errors)\n")
		writer.WriteString("The following sources could not be reached or returned no data:\n")
		for _, link := range failedLinks {
			writer.WriteString(fmt.Sprintf("- %s\n", link))
		}
	} else {
		writer.WriteString("\n## ✅ All Sources Successful\n")
		writer.WriteString("All configured sources were reached successfully.\n")
	}
}
