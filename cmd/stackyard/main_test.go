package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseStartOptions_Defaults(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(nil, &bytes.Buffer{}, envFromMap(nil))
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}

	if opts.addr != ":4566" {
		t.Fatalf("expected default addr :4566, got %q", opts.addr)
	}
	if opts.logLevel != "info" {
		t.Fatalf("expected default log level info, got %q", opts.logLevel)
	}
	if opts.lambdaExecutionMode != "mock" {
		t.Fatalf("expected default lambda execution mode mock, got %q", opts.lambdaExecutionMode)
	}
	if len(opts.providers) != 1 || opts.providers[0] != "aws" {
		t.Fatalf("expected providers [aws], got %#v", opts.providers)
	}
	if opts.foreground {
		t.Fatalf("expected default foreground false")
	}
	if strings.TrimSpace(opts.pidFile) == "" {
		t.Fatalf("expected default pid file")
	}
	if strings.TrimSpace(opts.logFile) == "" {
		t.Fatalf("expected default log file")
	}
}

func TestParseStartOptions_Examples(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		addr string
		log  string
	}{
		{
			name: "providers only",
			args: []string{"--providers", "aws"},
			addr: ":4566",
			log:  "info",
		},
		{
			name: "providers and port",
			args: []string{"--providers", "aws", "--port", "4566"},
			addr: ":4566",
			log:  "info",
		},
		{
			name: "providers and debug logging",
			args: []string{"--providers", "aws", "--log-level", "debug"},
			addr: ":4566",
			log:  "debug",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts, err := parseStartOptions(tc.args, &bytes.Buffer{}, envFromMap(nil))
			if err != nil {
				t.Fatalf("parseStartOptions returned error: %v", err)
			}
			if opts.addr != tc.addr {
				t.Fatalf("expected addr %q, got %q", tc.addr, opts.addr)
			}
			if opts.logLevel != tc.log {
				t.Fatalf("expected log level %q, got %q", tc.log, opts.logLevel)
			}
			if len(opts.providers) != 1 || opts.providers[0] != "aws" {
				t.Fatalf("expected providers [aws], got %#v", opts.providers)
			}
		})
	}
}

func TestParseStartOptions_PortOverridesEnvAddr(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{"--providers", "aws", "--port", "4570"},
		&bytes.Buffer{},
		envFromMap(map[string]string{"STACKYARD_ADDR": ":9999"}),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	if opts.addr != ":4570" {
		t.Fatalf("expected explicit --port to win over env addr, got %q", opts.addr)
	}
}

func TestParseStartOptions_RejectsUnsupportedProvider(t *testing.T) {
	t.Parallel()

	_, err := parseStartOptions(
		[]string{"--providers", "aws,gcp"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected error for unsupported provider")
	}
}

func TestRunHelpIncludesFlagOptions(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"help"}, &stdout, &stderr, envFromMap(nil))
	if err != nil {
		t.Fatalf("run(help) returned error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"--providers string",
		"Options: aws",
		"--port int",
		"1-65535",
		"--log-level string",
		"debug, info, warn, error",
		"--aws-access-key string",
		"--aws-secret-key string",
		"--lambda-execution-mode",
		"mock, local",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected help output to contain %q, got: %s", want, out)
		}
	}
}

func TestRunHelpStartShowsStartUsage(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"help", "start"}, &stdout, &stderr, envFromMap(nil))
	if err != nil {
		t.Fatalf("run(help start) returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "stackyard start [flags]") {
		t.Fatalf("expected start usage in help output, got: %s", out)
	}
	if !strings.Contains(out, "Options: aws") {
		t.Fatalf("expected provider options in start help output, got: %s", out)
	}
	if !strings.Contains(out, "--aws-access-key string") {
		t.Fatalf("expected aws access key flag in start help output, got: %s", out)
	}
}

func TestParseStartOptions_AWSCredentialFlags(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{"--providers", "aws", "--aws-access-key", "abc", "--aws-secret-key", "def"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	if opts.accessKey != "abc" {
		t.Fatalf("expected aws access key abc, got %q", opts.accessKey)
	}
	if opts.secretKey != "def" {
		t.Fatalf("expected aws secret key def, got %q", opts.secretKey)
	}
}

func TestParseStartOptions_LambdaFlags(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{
			"--providers", "aws",
			"--lambda-execution-mode", "local",
			"--lambda-work-dir", "/tmp/stackyard-lambda-work",
		},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	if opts.lambdaExecutionMode != "local" {
		t.Fatalf("expected lambda execution mode local, got %q", opts.lambdaExecutionMode)
	}
	if opts.lambdaWorkDir != "/tmp/stackyard-lambda-work" {
		t.Fatalf("expected lambda work dir /tmp/stackyard-lambda-work, got %q", opts.lambdaWorkDir)
	}
}

func TestParseStartOptions_RejectsInvalidLambdaExecutionMode(t *testing.T) {
	t.Parallel()

	_, err := parseStartOptions(
		[]string{"--providers", "aws", "--lambda-execution-mode", "remote"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected invalid lambda execution mode error")
	}
}

func TestParseStopOptions_Defaults(t *testing.T) {
	t.Parallel()

	opts, err := parseStopOptions(nil, &bytes.Buffer{}, envFromMap(nil))
	if err != nil {
		t.Fatalf("parseStopOptions returned error: %v", err)
	}
	if strings.TrimSpace(opts.pidFile) == "" {
		t.Fatalf("expected default pid file")
	}
	if opts.timeout != 10*time.Second {
		t.Fatalf("expected default timeout 10s, got %s", opts.timeout)
	}
}

func TestRunHelpStopShowsStopUsage(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"help", "stop"}, &stdout, &stderr, envFromMap(nil))
	if err != nil {
		t.Fatalf("run(help stop) returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "stackyard stop [flags]") {
		t.Fatalf("expected stop usage in help output, got: %s", out)
	}
	if !strings.Contains(out, "--pid-file string") {
		t.Fatalf("expected pid-file flag in stop help output, got: %s", out)
	}
}

func TestWaitHelpers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "wait-helper.txt")
	if waitForFile(path, 100*time.Millisecond) {
		t.Fatalf("expected waitForFile to fail when file does not exist")
	}
	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if !waitForFile(path, 200*time.Millisecond) {
		t.Fatalf("expected waitForFile to succeed when file exists")
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("failed to remove temp file: %v", err)
	}
	if !waitForFileRemoval(path, 200*time.Millisecond) {
		t.Fatalf("expected waitForFileRemoval to succeed when file removed")
	}
}

func envFromMap(values map[string]string) func(string) string {
	return func(key string) string {
		if values == nil {
			return ""
		}
		return values[key]
	}
}
