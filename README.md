[![GPLv3 license](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0.html) [![Update Configs](https://github.com/Danialsamadi/v2go/actions/workflows/update-configs.yml/badge.svg)](https://github.com/Danialsamadi/v2go/actions/workflows/update-configs.yml) ![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg) ![GitHub Stars](https://img.shields.io/github/stars/Danialsamadi/v2go?style=flat&logo=github&color=yellow) ![Last Commit](https://img.shields.io/github/last-commit/Danialsamadi/v2go?style=flat&logo=github&color=green)

# v2go — V2Ray Config Aggregator and Live Tester

A high-performance Go rewrite of [Epodonios/v2ray-configs](https://github.com/Epodonios/v2ray-configs). v2go collects V2Ray configurations from public subscription sources, verifies that each one actually carries traffic through an embedded Xray-core instance, and publishes only the working configs as subscription files.

## How It Works

Every configuration passes through a five-stage pipeline:

1. **Fetch** — Roughly 35 public subscription sources are downloaded concurrently (base64 and plain-text formats).
2. **Screen** — A fast TCP connectivity check filters out unreachable servers, invalid parameters that crash clients are rejected, and identity-based deduplication (protocol, credentials, host, port) removes duplicates even when names differ.
3. **Live test** — Each surviving config is converted into an outbound-only Xray configuration and run in an embedded [Xray-core](https://github.com/XTLS/Xray-core) instance in memory. An HTTP request to a 204 test endpoint is made through the proxy. Only configs that successfully pass real traffic are kept. No external binaries, no subprocesses, no open ports.
4. **Publish** — Working configs are renamed to a consistent format (`v2go | DE | VLESS | 1`), tagged by GeoIP country, and written into the subscription files listed below. Configs whose server address is a Cloudflare IP are additionally collected into a dedicated file.
5. **Clean up** — Subscription files that have not been refreshed within 24 hours are removed automatically.

The pipeline runs hourly via GitHub Actions. A typical run fetches around 50,000 raw configs, deduplicates them to roughly 10,000-15,000 candidates, live-tests all of them in about two minutes, and publishes the several hundred that demonstrably work.

### Why Live Testing Matters

A TCP port check only proves that something answers on the server's port; it says nothing about whether the proxy behind it works. Testing through a real Xray instance eliminates dead configs that would otherwise pass a port scan, so subscription lists contain only servers that carried actual traffic at test time.

## Supported Protocols

- VLESS (primary)
- VMess
- Trojan
- Shadowsocks (SS)
- Hysteria2 (HY2)
- TUIC
- ShadowsocksR (SSR)

The live-test stage covers VLESS, VMess, Trojan, Shadowsocks, and Hysteria2, including TLS, REALITY, WebSocket, and gRPC transports.

## Quick Start

### Prerequisites

- Go 1.26 or higher
- Git

### Build and Run

```bash
git clone --depth=1 https://github.com/Danialsamadi/v2go.git
cd v2go

go build -o aggregator .

ulimit -n 65535
./aggregator
```

The GeoIP database is downloaded automatically on first run.

### Configuration

The live-test stage can be tuned through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `V2GO_TEST_CONCURRENCY` | 500 | Number of configs tested in parallel. Values above ~500 saturate the network and produce worse results, not faster ones. |
| `V2GO_TEST_TIMEOUT` | 5 | Per-config test timeout in seconds. |

### Tests

```bash
go test ./...
```

## Output Structure

```
v2go/
├── AllConfigsSub.txt              # All working configs (plain text)
├── Sub1.txt ... SubN.txt          # Split into 500-config chunks
├── Base64/                        # Base64-encoded variants of the Sub files
├── Splitted-By-Protocol/          # Organized by protocol
│   ├── vless.txt
│   ├── vmess.txt
│   ├── ss.txt
│   ├── trojan.txt
│   ├── hy2.txt
│   ├── tuic.txt
│   └── cloudflare.txt             # Configs behind Cloudflare IPs (any protocol)
└── Splitted-By-Country/           # Organized by GeoIP location (US.txt, DE.txt, ...)
```

Note on encoding: `vmess.txt` is plain text; every other file in `Splitted-By-Protocol/` is base64-encoded.

## Subscription Links

### Main subscription (recommended)

```
https://raw.githubusercontent.com/Danialsamadi/v2go/main/AllConfigsSub.txt
```

### Country-specific subscriptions

Replace `XX` with any two-letter country code (US, DE, GB, ...):

```
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Country/XX.txt
```

### Protocol-specific subscriptions

```
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/vless.txt
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/vmess.txt
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/ss.txt
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/trojan.txt
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/hy2.txt
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Splitted-By-Protocol/cloudflare.txt
```

### Split subscriptions (500 configs each)

```
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Sub1.txt
https://raw.githubusercontent.com/Danialsamadi/v2go/main/Sub2.txt
...
```

Higher-numbered files exist when enough working configs are available; stale files are removed automatically after 24 hours.

## Compatible Clients

| Platform | Clients |
|----------|---------|
| Android | v2rayNG (recommended), Clash for Android |
| iOS | Streisand, Shadowrocket, Fair VPN |
| Windows / Linux | Hiddify Next (recommended), Nekoray, v2rayN, Clash Verge |
| macOS | V2rayU, ClashX |

### Usage

1. Copy one of the subscription links above.
2. Open your client's subscription settings and paste the link.
3. Update the subscription regularly; the lists are refreshed hourly.

## Architecture

| Component | Purpose |
|-----------|---------|
| `main.go` | Pipeline orchestration: fetch, screen, deduplicate, rename, GeoIP tagging, output |
| `internal/converter` | Converts share links (`vless://`, `vmess://`, `ss://`, `trojan://`, `hysteria2://`) into Xray outbound JSON, including TLS, REALITY, and transport settings |
| `internal/tester` | Runs each config in an embedded in-memory Xray-core instance and probes a 204 endpoint through it |
| `cloudflare.go` | Matches server IPs against Cloudflare CIDR ranges and writes `cloudflare.txt` |
| `sort.go` | Splits the working set into per-protocol files |
| `.github/workflows/update-configs.yml` | Hourly automated pipeline runs and 24-hour stale-file cleanup |

### Design Notes

- Xray-core is embedded as a Go library. Testing a config requires no subprocess, no temporary files, and no listening ports; cleanup is a single in-memory close.
- Live-test concurrency defaults to 500. Measurements on real config sets showed accuracy degrades above that point: network saturation causes working configs to time out and be reported dead.
- Per-config CPU cost is approximately 40 microseconds; the test stage is bounded almost entirely by network round-trips.
- CI uses shallow and blob-filtered checkouts. Because generated output is committed hourly, repository history is large; partial clones keep workflow runs to a few minutes.

## Contributing

Contributions are welcome. Please open an issue first for major changes.

## License

This project is licensed under the GNU General Public License v3.0. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Epodonios/v2ray-configs](https://github.com/Epodonios/v2ray-configs) — the original concept and Python implementation
- [XTLS/Xray-core](https://github.com/XTLS/Xray-core) — the proxy core used for live testing
- The V2Ray community for protocol specifications and documentation

---

## Star History

<a href="https://www.star-history.com/?repos=Danialsamadi%2Fv2go&type=timeline&logscale=&legend=bottom-right">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=Danialsamadi/v2go&type=timeline&theme=dark&logscale&legend=bottom-right&sealed_token=KgAALd0q2tbc6r9QJE69DWvs8y_WKaOYEpmrWVgGo31_C7YVwki4kX560bOtfpIsuigTP8jdgGhPZieAJtWcVdL9TZdrf-FweVuizeY9cK32vRXyghnG30kVYD1nvkvxrxxJ5PHtS5VBrNZRqAiAqUtzNO2fTtybBxo2_U82-3oovbb6thocZ3axbgK3" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=Danialsamadi/v2go&type=timeline&logscale&legend=bottom-right&sealed_token=KgAALd0q2tbc6r9QJE69DWvs8y_WKaOYEpmrWVgGo31_C7YVwki4kX560bOtfpIsuigTP8jdgGhPZieAJtWcVdL9TZdrf-FweVuizeY9cK32vRXyghnG30kVYD1nvkvxrxxJ5PHtS5VBrNZRqAiAqUtzNO2fTtybBxo2_U82-3oovbb6thocZ3axbgK3" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=Danialsamadi/v2go&type=timeline&logscale&legend=bottom-right&sealed_token=KgAALd0q2tbc6r9QJE69DWvs8y_WKaOYEpmrWVgGo31_C7YVwki4kX560bOtfpIsuigTP8jdgGhPZieAJtWcVdL9TZdrf-FweVuizeY9cK32vRXyghnG30kVYD1nvkvxrxxJ5PHtS5VBrNZRqAiAqUtzNO2fTtybBxo2_U82-3oovbb6thocZ3axbgK3" />
 </picture>
</a>

---

**Dani Samadi** — If you find this project useful, consider giving it a star on GitHub.
