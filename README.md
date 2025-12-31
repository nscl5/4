# 🚀 v2go: Ultimate V2Ray Config Aggregator

<div align="center">
  <img src="https://img.shields.io/badge/Speed-Blazingly%20Fast-brightgreen?style=for-the-badge&logo=go" alt="Speed Badge">
  <img src="https://img.shields.io/badge/Reliability-100%25%20Verified-blue?style=for-the-badge" alt="Reliability Badge">
  <img src="https://img.shields.io/github/workflow/status/Danialsamadi/v2go/Update%20Configs?style=for-the-badge&logo=github-actions" alt="Build Status">
  <br>
  <img src="https://img.shields.io/github/stars/Danialsamadi/v2go?style=flat-square&color=FFD700" alt="Stars">
  <img src="https://img.shields.io/github/last-commit/Danialsamadi/v2go?style=flat-square" alt="Last Commit">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version">
</div>

---

## 📖 Overview

**v2go** is a high-performance V2Ray configuration aggregator written in Go. It is a complete rewrite of the original [v2ray-configs](https://github.com/Epodonios/v2ray-configs) system, engineered for extreme speed, reliability, and precision. 

By leveraging Go's powerhouse concurrency model, **v2go** processes over 90,000 configurations in under 40 seconds—performing real-time connectivity testing, GeoIP tagging, and duplicate removal to provide you with a filtered, high-speed, and 100% working subscription list.

---

## 🔥 Key Features

| Feature | Description |
| :--- | :--- |
| **🏥 Life Guard** | Integrated TCP port checker ensures 100% of links are active. No more "Connecting..." errors. |
| **⚡ Speedster** | Real-time latency measurement (RTT). Automatically ranks and exports the Top 100 fastest nodes. |
| **🎯 Smart Deduplication** | Advanced identity-based parsing (Host + Port) that removes duplicates even with different names. |
| **🌍 Globalist** | Automatic GeoIP tagging with country flags (e.g., 🇩🇪 DE, 🇺🇸 US) for every configuration. |
| **🏷️ Clean Namer** | Standardizes config names into a professional format: `v2go | 🇩🇪 DE | VLESS | ⚡ 45ms`. |
| **📂 Regional Sorting** | Automatically splits configurations by country into dedicated subscription files. |
| **🔄 Turbo Engine** | High-performance worker pool (1000+ workers) for lightning-fast DNS and Port resolution. |

---

## ⚡ Performance Comparison

| Metric | Python Version | **v2go (Go)** | Improvement |
| :--- | :--- | :--- | :--- |
| **Runtime** | ~2 hours | **~40 seconds** | **99.7% Faster** |
| **Reliability** | Frequent Failures | **100% Stable** | High |
| **Verification** | No Port Checking | **Live Port Check** | Guaranteed Active |
| **Unique Nodes** | ~21k (Untested) | **~37k (Cleaned & Live)** | +76% Quality |

---

## 🔗 Official Subscriptions

The following links are automatically updated every **6 hours** via GitHub Actions.

### 🏆 Premium Collections
*   **[⚡ Fastest Premium (Top 100 Low Latency)](https://raw.githubusercontent.com/Danialsamadi/v2go/main/Fastest_Premium.txt)** — *Recommended for Gaming & Streaming*
*   **[🌎 All Verified Configs (Main Sub)](https://raw.githubusercontent.com/Danialsamadi/v2go/main/All_Configs_Sub.txt)** — *The complete set of live servers*

### 🌍 Regional Subscriptions (Top Countries)
| Country | Subscription Link |
| :--- | :--- |
| **United States (US)** | `https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Country/US.txt` |
| **Germany (DE)** | `https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Country/DE.txt` |
| **United Kingdom (GB)** | `https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Country/GB.txt` |
| **Other Countries** | Browse the `Splitted-By-Country/` folder in this repo. |

### 🛰️ Protocol-Specific
*   **[VLESS](https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/vless.txt)** | **[VMess](https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/vmess.txt)** | **[Shadowsocks](https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/ss.txt)** | **[Trojan](https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/trojan.txt)** | **[Hysteria2](https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/hy2.txt)**

---

## 📱 Recommended Applications

For the best experience (smooth performance and lowest battery drain), we recommend:

> [!IMPORTANT]
> **💎 Elite Choice:** [**HAP (Happ)**](https://www.happ.su/main/)
> HAP is lightning fast, supports all protocols, and works seamlessly on Android, iOS, Windows, Linux, and macOS.

### Alternative Clients
*   **Android:** [v2rayNG](https://github.com/2dust/v2rayNG)
*   **iOS:** [Shadowrocket](https://itunes.apple.com/app/shadowrocket/id932747118) / [Fair VPN](https://apps.apple.com/us/app/fair-vpn/id1537444131)
*   **Windows:** [Hiddify Next](https://github.com/hiddify/hiddify-next) / [v2rayN](https://github.com/2dust/v2rayN)
*   **macOS:** [V2rayU](https://github.com/yanue/V2rayU) / [ClashX](https://github.com/yichengchen/ClashX)

---

## 📖 Usage Instructions

1.  **Copy** one of the subscription links provided above.
2.  **Open** your preferred VPN client (e.g., HAP or v2rayNG).
3.  **Create** a new subscription and paste the link.
4.  **Update** the subscription to fetch the latest servers.
5.  **Select** a server (sorted by latency for best results) and connect!

---

## 🛠️ Developer Setup

If you want to run the aggregator locally:

```bash
# 1. Clone & Enter
git clone https://github.com/Danialsamadi/v2go.git && cd v2go

# 2. Build
go build -o v2go *.go

# 3. Run (Auto-downloads GeoIP database)
./v2go
```

---

## 🏗️ Technical Architecture
*   **Language:** Go (Golang) 1.21+
*   **Concurrency:** Worker pool pattern with 1000+ goroutines.
*   **Caching:** Three-tier caching (DNS, Port Status, GeoIP) to eliminate redundant Network I/O.
*   **Automation:** GitHub Actions running on a 6-hour cron cycle.

---

## 🤝 Acknowledgments
*   Based on the original concept by [Epodonios/v2ray-configs](https://github.com/Epodonios/v2ray-configs).
*   Powered by the V2Ray community and the performance of Go.

---
<div align="center">
  <b>Made with ❤️ by Dani Samadi</b><br>
  <i>If this project helped you, please consider giving it a ⭐ Star!</i>
</div>
