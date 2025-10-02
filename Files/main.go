package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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

var fixedText = `#profile-title: base64:RnJlZWRvbSB0byBEcmVhbQ==
#profile-update-interval: 6
#support-url: https://github.com/NiREvil/vless
#profile-web-page-url: https://github.com/NiREvil/vless
`

var protocols = []string{"vmess", "vless", "trojan", "ss", "ssr", "hy2", "tuic", "warp://"}

var links = []string{
	"https://v2.alicivil.workers.dev",
	"https://raw.githubusercontent.com/Epodonios/v2ray-configs/refs/heads/main/Splitted-By-Protocol/ss.txt",
	"https://raw.githubusercontent.com/Surfboardv2ray/Proxy-sorter/refs/heads/main/submerge/US.txt",
	"https://raw.githubusercontent.com/Surfboardv2ray/TGParse/main/splitted/mixed",
}

var dirLinks = []string{
    "https://raw.githubusercontent.com/DiDiten/HiN-VPN/main/subscription/normal/mix",
	"https://raw.githubusercontent.com/darkvpnapp/CloudflarePlus/refs/heads/main/proxy",
	"https://raw.githubusercontent.com/arshiacomplus/robinhood-v1-v2-v3ray/refs/heads/main/conf.txt",
	"https://raw.githubusercontent.com/nscl5/5/refs/heads/main/configs/shadowsocks.txt",
    "https://raw.githubusercontent.com/Surfboardv2ray/Proxy-sorter/refs/heads/main/custom/mahsa.txt",
    "https://raw.githubusercontent.com/Surfboardv2ray/Proxy-sorter/main/output/US.txt",
    "https://raw.githubusercontent.com/Surfboardv2ray/Proxy-sorter/refs/heads/main/output/US.txt",
	"https://raw.githubusercontent.com/arshiacomplus/v2rayExtractor/refs/heads/main/mix/sub.html",
	"https://raw.githubusercontent.com/Rayan-Config/C-Sub/refs/heads/main/configs/proxy.txt",
	"https://raw.githubusercontent.com/mahdibland/ShadowsocksAggregator/master/Eternity.txt",
	"https://raw.githubusercontent.com/MahsaNetConfigTopic/config/refs/heads/main/xray_final.txt",
}


type Result struct {
	Content  string
	IsBase64 bool
}

func main() {
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
	allConfigs := fetchAllConfigs(client, links, dirLinks)

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

func fetchAllConfigs(client *http.Client, base64Links, textLinks []string) []string {
	var wg sync.WaitGroup
	resultChan := make(chan Result, len(base64Links)+len(textLinks))

	// Worker pool for concurrent requests
	semaphore := make(chan struct{}, maxWorkers)

	// Fetch base64-encoded links
	for _, link := range base64Links {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			content := fetchAndDecodeBase64(client, url)
			if content != "" {
				resultChan <- Result{Content: content, IsBase64: true}
			}
		}(link)
	}

	// Fetch text links
	for _, link := range textLinks {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			content := fetchText(client, url)
			if content != "" {
				resultChan <- Result{Content: content, IsBase64: false}
			}
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
		lines := strings.Split(strings.TrimSpace(result.Content), "\n")
		allConfigs = append(allConfigs, lines...)
	}

	return allConfigs
}

func fetchAndDecodeBase64(client *http.Client, url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Try to decode base64
	decoded, err := decodeBase64(body)
	if err != nil {
		return ""
	}

	return decoded
}

func fetchText(client *http.Client, url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(body)
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
	seen := make(map[string]bool) // Track duplicates

	for _, line := range data {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip if we've already seen this config
		if seen[line] {
			continue
		}

		for _, protocol := range protocols {
			if strings.HasPrefix(line, protocol) {
				filtered = append(filtered, line)
				seen[line] = true // Mark as seen
				break
			}
		}
	}
	return filtered
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
		profileTitle := fmt.Sprintf("Freedom to Dream | Sub%d", i+1)
		encodedTitle := base64.StdEncoding.EncodeToString([]byte(profileTitle))
		customFixedText := fmt.Sprintf(`#profile-title: base64:%s
#profile-update-interval: 6
#support-url: https://github.com/NiREvil
#profile-web-page-url: https://github.com/NiREvil/vless
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
