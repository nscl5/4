package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func sortConfigs() {
	fmt.Println("Starting protocol-based config sorting...")

	// Setup paths for new directory structure in current directory
	protocolDir := "Splitted-By-Protocol"

	// Create directory if it doesn't exist
	if err := os.MkdirAll(protocolDir, 0755); err != nil {
		fmt.Printf("Error creating protocol directory: %v\n", err)
		return
	}

	// Define file paths
	files := map[string]string{
		"vmess":  filepath.Join(protocolDir, "vmess.txt"),
		"vless":  filepath.Join(protocolDir, "vless.txt"),
		"trojan": filepath.Join(protocolDir, "trojan.txt"),
		"ss":     filepath.Join(protocolDir, "ss.txt"),
		"ssr":    filepath.Join(protocolDir, "ssr.txt"),
		"hy2":    filepath.Join(protocolDir, "hy2.txt"),
		"tuic":   filepath.Join(protocolDir, "tuic.txt"),
		"warp":   filepath.Join(protocolDir, "warp.txt"),
	}

	// Clear existing files
	for protocol, filePath := range files {
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			fmt.Printf("Error clearing %s file: %v\n", protocol, err)
			return
		}
	}

	// Process local file
	fmt.Println("Processing local AllConfigsSub.txt...")
	localFile, err := os.Open("AllConfigsSub.txt")
	if err != nil {
		fmt.Printf("Error opening local config file: %v\n", err)
		return
	}
	defer localFile.Close()

	// Process the file line by line for memory efficiency
	scanner := bufio.NewScanner(localFile)

	// Collect configs by protocol
	protocolConfigs := make(map[string][]string)
	// Track duplicates for each protocol
	seenConfigs := make(map[string]map[string]bool)
	for protocol := range files {
		seenConfigs[protocol] = make(map[string]bool)
	}

	vmessFile, err := os.OpenFile(files["vmess"], os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening vmess file: %v\n", err)
		return
	}
	defer vmessFile.Close()

	vmessWriter := bufio.NewWriter(vmessFile)
	defer vmessWriter.Flush()

	configCount := make(map[string]int)
	duplicateCount := make(map[string]int)

	fmt.Println("Processing configurations...")
	unknownCount := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check protocol and categorize
		matched := false
		for protocol := range files {
			prefix := protocol + "://"
			if protocol == "warp" {
				prefix = "warp://"
			}

			if strings.HasPrefix(line, prefix) {
				matched = true
				if seenConfigs[protocol][line] {
					duplicateCount[protocol]++
					break
				}
				seenConfigs[protocol][line] = true
				configCount[protocol]++

				if protocol == "vmess" {
					if _, err := vmessWriter.WriteString(line + "\n"); err != nil {
						fmt.Printf("Error writing vmess config: %v\n", err)
						return
					}
				} else {
					protocolConfigs[protocol] = append(protocolConfigs[protocol], line)
				}
				break
			}
		}

		if !matched {
			unknownCount++
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	// Flush vmess writer
	vmessWriter.Flush()

	// Write other protocols as base64-encoded content
	for protocol, configs := range protocolConfigs {
		if len(configs) == 0 {
			continue
		}

		// Join all configs for this protocol
		content := strings.Join(configs, "\n")

		// Base64 encode the content
		encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

		// Write to file
		if err := os.WriteFile(files[protocol], []byte(encodedContent), 0644); err != nil {
			fmt.Printf("Error writing %s file: %v\n", protocol, err)
			return
		}
	}

	// Sort protocols for consistent output
	protocols := []string{"vmess", "vless", "trojan", "ss", "ssr", "hy2", "tuic", "warp"}

	// Print summary
	fmt.Println("\nProtocol sorting completed!")
	fmt.Println("Configuration counts (after removing duplicates):")
	for _, protocol := range protocols {
		count := configCount[protocol]
		fmt.Printf("  %s: %d configs\n", protocol, count)
	}
	if unknownCount > 0 {
		fmt.Printf("  Unknown/Other: %d configs\n", unknownCount)
	}

	total := 0
	totalDuplicates := 0
	for _, count := range configCount {
		total += count
	}
	for _, count := range duplicateCount {
		totalDuplicates += count
	}
	fmt.Printf("  Total unique identified: %d configs\n", total)

	if totalDuplicates > 0 {
		fmt.Println("\nDuplicates removed during sorting:")
		for _, protocol := range protocols {
			count := duplicateCount[protocol]
			if count > 0 {
				fmt.Printf("  %s: %d duplicates\n", protocol, count)
			}
		}
		fmt.Printf("  Total duplicates removed: %d\n", totalDuplicates)
		fmt.Printf("  Main file total lines: %d\n", total+totalDuplicates+unknownCount)
	}
}
