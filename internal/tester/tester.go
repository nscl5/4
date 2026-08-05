package tester

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	xnet "github.com/xtls/xray-core/common/net"
	xcore "github.com/xtls/xray-core/core"
	xserial "github.com/xtls/xray-core/infra/conf/serial"

	_ "github.com/xtls/xray-core/main/distro/all"

	"v2ray-config-aggregator/internal/converter"
)

type Result struct {
	Link    string
	DelayMs int
}

func TestAll(links []string, testURL string, timeoutSec, concurrent int) []Result {
	timeout := time.Duration(timeoutSec) * time.Second

	results := make([]Result, len(links))
	for i, l := range links {
		results[i] = Result{Link: l, DelayMs: -1}
	}

	sem := make(chan struct{}, concurrent)
	var wg sync.WaitGroup
	var done, working atomic.Int64
	total := int64(len(links))

	log.Printf("Testing %d config(s) with %d concurrent workers...", total, concurrent)

	for i, link := range links {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, l string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					results[idx].DelayMs = -1
				}
			}()

			delay := testOne(l, testURL, timeout)
			results[idx].DelayMs = delay

			d := done.Add(1)
			if delay > 0 {
				working.Add(1)
			}
			if d%500 == 0 || d == total {
				log.Printf("  Progress: %d/%d tested, %d working", d, total, working.Load())
			}
		}(i, link)
	}

	wg.Wait()
	return results
}

func testOne(link, testURL string, timeout time.Duration) int {
	outbound, err := converter.ConvertLink(link)
	if err != nil {
		return -1
	}

	fullConfig := converter.M{
		"log": converter.M{"loglevel": "none"},
		"outbounds": []converter.M{
			outbound,
			{"protocol": "freedom", "tag": "direct"},
		},
	}

	jsonBytes, err := json.Marshal(fullConfig)
	if err != nil {
		return -1
	}

	cfg, err := xserial.LoadJSONConfig(bytes.NewReader(jsonBytes))
	if err != nil {
		return -1
	}

	instance, err := xcore.New(cfg)
	if err != nil {
		return -1
	}
	if err := instance.Start(); err != nil {
		return -1
	}
	defer instance.Close()

	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		port, err := net.LookupPort(network, portStr)
		if err != nil {
			return nil, err
		}
		dest := xnet.TCPDestination(xnet.ParseAddress(host), xnet.Port(port))
		return xcore.Dial(ctx, instance, dest)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext:       dialer,
			DisableKeepAlives: true,
		},
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return -1
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return int(time.Since(start).Milliseconds())
	}
	return -1
}
