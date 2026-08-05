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
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	cloudflareIPsURL  = "https://raw.githubusercontent.com/radioactiveAHM/cf-scanner/refs/heads/main/ipv4.txt"
	cloudflareIPsFile = "cloudflare-ips.txt" // local fallback when the download fails
)

// loadCloudflareRanges downloads the Cloudflare CIDR list (caching it locally),
// falling back to the cached copy. Returns nil if neither is available —
// callers treat nil as "CF separation disabled".
func loadCloudflareRanges() []*net.IPNet {
	var data []byte
	client := &http.Client{Timeout: 30 * time.Second}
	if resp, err := client.Get(cloudflareIPsURL); err == nil {
		if resp.StatusCode == http.StatusOK {
			data, _ = io.ReadAll(resp.Body)
		}
		resp.Body.Close()
	}
	if len(data) > 0 {
		os.WriteFile(cloudflareIPsFile, data, 0644)
	} else {
		data, _ = os.ReadFile(cloudflareIPsFile)
	}
	if len(data) == 0 {
		fmt.Println("Warning: no Cloudflare IP ranges available, skipping CF separation")
		return nil
	}

	var ranges []*net.IPNet
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "/") {
			line += "/32"
		}
		if _, cidr, err := net.ParseCIDR(line); err == nil {
			ranges = append(ranges, cidr)
		}
	}
	return ranges
}

// writeCloudflareFile writes configs whose server address is a literal IP
// inside Cloudflare's ranges to Splitted-By-Protocol/cloudflare.txt
// (base64-encoded, like the other non-vmess protocol files).
// ponytail: domains are not resolved — same behavior as v2ray-tester's cfcheck;
// add a DNS pass here if domain-fronted CF configs turn out to matter.
func writeCloudflareFile(configs []string) {
	ranges := loadCloudflareRanges()
	if len(ranges) == 0 {
		return
	}

	var cf []string
	for _, c := range configs {
		proto, _, _ := strings.Cut(c, "://")
		host, _ := getHostPort(c, proto)
		ip := net.ParseIP(strings.Trim(host, "[]"))
		if ip == nil {
			continue
		}
		for _, cidr := range ranges {
			if cidr.Contains(ip) {
				cf = append(cf, c)
				break
			}
		}
	}

	path := filepath.Join("Splitted-By-Protocol", "cloudflare.txt")
	content := base64.StdEncoding.EncodeToString([]byte(strings.Join(cf, "\n")))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing cloudflare file: %v\n", err)
		return
	}
	fmt.Printf("  cloudflare: %d configs behind Cloudflare IPs\n", len(cf))
}
