package tester

import (
	"context"
	"testing"
	"time"
)

// A run can take many minutes. Ctrl-C must stop it promptly and hand back what
// has already been verified, rather than testing every remaining config or
// discarding the work done so far.
func TestTestAllStopsWhenContextIsCancelled(t *testing.T) {
	// Unroutable addresses: every one of these would sit until it times out.
	links := make([]string, 2000)
	for i := range links {
		links[i] = "trojan://pw@192.0.2.1:443?security=tls&sni=example.com#x"
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	results := TestAll(ctx, links, "http://gstatic.com/generate_204", 10, 5, nil)
	elapsed := time.Since(start)

	if elapsed > 8*time.Second {
		t.Fatalf("TestAll ran for %v after cancellation; it should stop promptly", elapsed)
	}
	if len(results) != len(links) {
		t.Fatalf("got %d results, want one per link (%d)", len(results), len(links))
	}
}
