package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A failed run must be distinguishable from a successful one by exit code
// alone. CI runs `./aggregator` and treats success as "configs were produced";
// returning 0 after failing to read the input silently publishes nothing.
func TestRunReportsFailureWhenInputFileIsMissing(t *testing.T) {
	inTempDir(t)

	err := run([]string{"-input", "does-not-exist.txt"})

	if err == nil {
		t.Fatal("run succeeded despite an unreadable input file")
	}
	if !strings.Contains(err.Error(), "does-not-exist.txt") {
		t.Errorf("error should name the file it could not read, got: %v", err)
	}
}

func TestRunReportsFailureWhenInputFileIsNotConfigs(t *testing.T) {
	dir := inTempDir(t)
	junk := filepath.Join(dir, "junk.txt")
	if err := os.WriteFile(junk, []byte("this is not a config list\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"-input", junk}); err == nil {
		t.Fatal("run succeeded on a file containing no configs")
	}
}

// inTempDir runs the test in a scratch directory, since the pipeline writes
// its output relative to the working directory.
func inTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

func TestParseFlagsDefaults(t *testing.T) {
	t.Setenv("V2GO_TEST_CONCURRENCY", "")
	t.Setenv("V2GO_TEST_TIMEOUT", "")

	opts, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.concurrency != testConcurrency {
		t.Errorf("concurrency = %d, want %d", opts.concurrency, testConcurrency)
	}
	if opts.timeoutSec != testTimeoutSec {
		t.Errorf("timeout = %d, want %d", opts.timeoutSec, testTimeoutSec)
	}
}

// CI sets these; a flag on the command line must still win over them.
func TestParseFlagsPreferFlagOverEnvironment(t *testing.T) {
	t.Setenv("V2GO_TEST_CONCURRENCY", "111")

	opts, err := parseFlags([]string{"-concurrency", "222"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.concurrency != 222 {
		t.Errorf("concurrency = %d, want the flag value 222", opts.concurrency)
	}
}

func TestParseFlagsReadsEnvironmentWhenFlagAbsent(t *testing.T) {
	t.Setenv("V2GO_TEST_CONCURRENCY", "111")

	opts, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.concurrency != 111 {
		t.Errorf("concurrency = %d, want the environment value 111", opts.concurrency)
	}
}

// A concurrency of zero would hang forever on an empty semaphore rather than
// doing nothing, so it has to be rejected up front.
func TestParseFlagsRejectsNonPositiveValues(t *testing.T) {
	for _, args := range [][]string{
		{"-concurrency", "0"},
		{"-concurrency", "-5"},
		{"-timeout", "0"},
	} {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("parseFlags(%v) accepted an invalid value", args)
		}
	}
}

// A typo like `v2go input.txt` (missing the dash) must not be silently ignored
// and run against the built-in sources instead of the user's file.
func TestParseFlagsRejectsPositionalArguments(t *testing.T) {
	_, err := parseFlags([]string{"my-configs.txt"})
	if err == nil {
		t.Fatal("parseFlags accepted a stray positional argument")
	}
	if !strings.Contains(err.Error(), "my-configs.txt") {
		t.Errorf("error should quote the offending argument, got: %v", err)
	}
}

func TestParseFlagsUnknownFlagIsAnError(t *testing.T) {
	if _, err := parseFlags([]string{"-nope"}); err == nil {
		t.Fatal("parseFlags accepted an unknown flag")
	}
}
