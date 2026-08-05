package tester

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	xcore "github.com/xtls/xray-core/core"
	xserial "github.com/xtls/xray-core/infra/conf/serial"

	"v2ray-config-aggregator/internal/converter"
)

func loadLinks(tb testing.TB, n int) []string {
	f, err := os.Open("../../AllConfigsSub.txt")
	if err != nil {
		tb.Skip("no AllConfigsSub.txt")
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1<<20), 1<<20)
	var out []string
	for s.Scan() && len(out) < n {
		l := strings.TrimSpace(s.Text())
		if strings.Contains(l, "://") && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

func fullConfigJSON(link string) ([]byte, error) {
	outbound, err := converter.ConvertLink(link)
	if err != nil {
		return nil, err
	}
	return json.Marshal(converter.M{
		"log": converter.M{"loglevel": "none"},
		"outbounds": []converter.M{
			outbound,
			{"protocol": "freedom", "tag": "direct"},
		},
	})
}

// BenchmarkConvert: URI string -> outbound JSON. Pure CPU, no xray.
func BenchmarkConvert(b *testing.B) {
	links := loadLinks(b, 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fullConfigJSON(links[i%len(links)])
	}
}

// BenchmarkLoadJSONConfig: JSON -> xray protobuf config. Pure CPU, no instance.
func BenchmarkLoadJSONConfig(b *testing.B) {
	links := loadLinks(b, 500)
	var raws [][]byte
	for _, l := range links {
		if r, err := fullConfigJSON(l); err == nil {
			raws = append(raws, r)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xserial.LoadJSONConfig(bytes.NewReader(raws[i%len(raws)]))
	}
}

// BenchmarkInstanceLifecycle: New + Start + Close, no traffic at all.
// This is the per-config fixed cost we pay 1,000,000 times.
func BenchmarkInstanceLifecycle(b *testing.B) {
	links := loadLinks(b, 500)
	var cfgs []*xcore.Config
	for _, l := range links {
		raw, err := fullConfigJSON(l)
		if err != nil {
			continue
		}
		c, err := xserial.LoadJSONConfig(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		cfgs = append(cfgs, c)
	}
	if len(cfgs) == 0 {
		b.Skip("no loadable configs")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inst, err := xcore.New(cfgs[i%len(cfgs)])
		if err != nil {
			continue
		}
		inst.Start()
		inst.Close()
	}
}
