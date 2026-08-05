// v2go - High-Performance V2Ray Config Aggregator (Go Edition)
// Copyright (C) 2025  Danialsamadi
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var sortProtocols = []string{"vmess", "vless", "trojan", "ss", "ssr", "hy2", "tuic", "warp"}

// sortConfigs splits the already-deduplicated config list into per-protocol
// files. vmess stays plain text; every other protocol file is base64-encoded
// (that asymmetry is what existing subscribers expect).
func sortConfigs(configs []string) {
	fmt.Println("Starting protocol-based config sorting...")

	protocolDir := "Splitted-By-Protocol"
	if err := os.MkdirAll(protocolDir, 0755); err != nil {
		fmt.Printf("Error creating protocol directory: %v\n", err)
		return
	}

	byProtocol := make(map[string][]string)
	unknownCount := 0
	for _, line := range configs {
		matched := false
		for _, protocol := range sortProtocols {
			if strings.HasPrefix(line, protocol+"://") {
				byProtocol[protocol] = append(byProtocol[protocol], line)
				matched = true
				break
			}
		}
		if !matched {
			unknownCount++
		}
	}

	fmt.Println("\nProtocol sorting completed!")
	fmt.Println("Configuration counts:")
	for _, protocol := range sortProtocols {
		lines := byProtocol[protocol]
		content := strings.Join(lines, "\n")
		if protocol != "vmess" {
			content = base64.StdEncoding.EncodeToString([]byte(content))
		}
		path := filepath.Join(protocolDir, protocol+".txt")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Printf("Error writing %s file: %v\n", protocol, err)
			return
		}
		fmt.Printf("  %s: %d configs\n", protocol, len(lines))
	}
	if unknownCount > 0 {
		fmt.Printf("  Unknown/Other: %d configs\n", unknownCount)
	}
}
