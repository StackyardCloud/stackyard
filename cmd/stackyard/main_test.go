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
	if opts.h2Addr != ":4567" {
		t.Fatalf("expected default h2 addr :4567, got %q", opts.h2Addr)
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
		h2   string
		log  string
	}{
		{
			name: "providers only",
			args: []string{"--providers", "aws"},
			addr: ":4566",
			h2:   ":4567",
			log:  "info",
		},
		{
			name: "providers and port",
			args: []string{"--providers", "aws", "--port", "4566"},
			addr: ":4566",
			h2:   ":4567",
			log:  "info",
		},
		{
			name: "providers and debug logging",
			args: []string{"--providers", "aws", "--log-level", "debug"},
			addr: ":4566",
			h2:   ":4567",
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
			if opts.h2Addr != tc.h2 {
				t.Fatalf("expected h2 addr %q, got %q", tc.h2, opts.h2Addr)
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
	if opts.h2Addr != ":4567" {
		t.Fatalf("expected default h2 addr :4567, got %q", opts.h2Addr)
	}
}

func TestParseStartOptions_HTTP2PortTracksResolvedHost(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{"--providers", "aws", "--addr", "127.0.0.1:4566", "--h2-port", "50051"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	if opts.h2Addr != "127.0.0.1:50051" {
		t.Fatalf("expected h2 addr 127.0.0.1:50051, got %q", opts.h2Addr)
	}
}

func TestParseStartOptions_RejectsSameHTTPAndHTTP2Addr(t *testing.T) {
	t.Parallel()

	_, err := parseStartOptions(
		[]string{"--providers", "aws", "--addr", ":4566", "--h2-addr", ":4566"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected error when --h2-addr matches --addr")
	}
}

func TestParseStartOptions_RejectsUnsupportedProvider(t *testing.T) {
	t.Parallel()

	_, err := parseStartOptions(
		[]string{"--providers", "aws,digitalocean"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected error for unsupported provider")
	}
}

func TestParseStartOptions_AcceptsMultipleProviders(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{"--providers", "aws,gcp,azure,oci"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	want := []string{"aws", "gcp", "azure", "oci"}
	if len(opts.providers) != len(want) {
		t.Fatalf("expected %d providers, got %#v", len(want), opts.providers)
	}
	for i := range want {
		if opts.providers[i] != want[i] {
			t.Fatalf("expected provider %d to be %q, got %q", i, want[i], opts.providers[i])
		}
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
		"Options: aws, gcp, azure, oci",
		"--port int",
		"--h2-port int",
		"1-65535",
		"--log-level string",
		"debug, info, warn, error",
		"--aws-access-key string",
		"--aws-secret-key string",
		"--gcp-auth-mode string",
		"--azure-auth-mode string",
		"--oci-auth-mode string",
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
	if !strings.Contains(out, "Options: aws, gcp, azure, oci") {
		t.Fatalf("expected provider options in start help output, got: %s", out)
	}
	if !strings.Contains(out, "--aws-access-key string") {
		t.Fatalf("expected aws access key flag in start help output, got: %s", out)
	}
	if !strings.Contains(out, "--h2-port int") {
		t.Fatalf("expected h2-port flag in start help output, got: %s", out)
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

func TestParseStartOptions_ProviderAuthModeFlags(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{
			"--providers", "aws,gcp,azure,oci",
			"--gcp-auth-mode", "bearer_required",
			"--azure-auth-mode", "shared_key",
			"--oci-auth-mode", "disabled",
		},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	if opts.gcpAuthMode != "bearer_required" {
		t.Fatalf("expected gcp auth mode bearer_required, got %q", opts.gcpAuthMode)
	}
	if opts.azureAuthMode != "shared_key" {
		t.Fatalf("expected azure auth mode shared_key, got %q", opts.azureAuthMode)
	}
	if opts.ociAuthMode != "disabled" {
		t.Fatalf("expected oci auth mode disabled, got %q", opts.ociAuthMode)
	}
}

func TestParseStartOptions_RejectsInvalidProviderAuthModes(t *testing.T) {
	t.Parallel()

	_, err := parseStartOptions(
		[]string{"--providers", "aws,gcp", "--gcp-auth-mode", "oauth"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected invalid gcp auth mode error")
	}

	_, err = parseStartOptions(
		[]string{"--providers", "aws,azure", "--azure-auth-mode", "aad"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected invalid azure auth mode error")
	}

	_, err = parseStartOptions(
		[]string{"--providers", "aws,oci", "--oci-auth-mode", "instance_principal"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected invalid oci auth mode error")
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

func TestParseStartOptions_PersistenceFlags(t *testing.T) {
	t.Parallel()

	opts, err := parseStartOptions(
		[]string{
			"--providers", "aws",
			"--persist-state",
			"--state-dir", "/tmp/stackyard-state",
			"--snapshot-load-strategy", "manual",
			"--snapshot-save-strategy", "on_shutdown",
		},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err != nil {
		t.Fatalf("parseStartOptions returned error: %v", err)
	}
	if !opts.persistenceEnabled {
		t.Fatalf("expected persist-state true")
	}
	if opts.stateDir != "/tmp/stackyard-state" {
		t.Fatalf("expected state dir /tmp/stackyard-state, got %q", opts.stateDir)
	}
	if opts.snapshotLoadStrategy != "manual" {
		t.Fatalf("expected snapshot load strategy manual, got %q", opts.snapshotLoadStrategy)
	}
	if opts.snapshotSaveStrategy != "on_shutdown" {
		t.Fatalf("expected snapshot save strategy on_shutdown, got %q", opts.snapshotSaveStrategy)
	}
}

func TestParseStartOptions_RejectsInvalidSnapshotStrategies(t *testing.T) {
	t.Parallel()

	_, err := parseStartOptions(
		[]string{"--providers", "aws", "--snapshot-load-strategy", "eager"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected invalid snapshot load strategy error")
	}

	_, err = parseStartOptions(
		[]string{"--providers", "aws", "--snapshot-save-strategy", "never"},
		&bytes.Buffer{},
		envFromMap(nil),
	)
	if err == nil {
		t.Fatalf("expected invalid snapshot save strategy error")
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
