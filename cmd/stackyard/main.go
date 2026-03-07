package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/server"
)

const (
	defaultPort            = 4566
	defaultHTTP2Port       = 4567
	defaultShutdownTimeout = 10 * time.Second
	startupWaitTimeout     = 5 * time.Second
	supportedProvidersText = "aws, gcp, azure, oci"
)

var supportedProviders = map[string]struct{}{
	"aws":   {},
	"gcp":   {},
	"azure": {},
	"oci":   {},
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	if len(args) == 0 {
		// No command is foreground mode to preserve existing Docker/entrypoint behavior.
		return runServeCommand(nil, stderr, getenv)
	}

	switch args[0] {
	case "start":
		return runStartCommand(args[1:], stdout, stderr, getenv)
	case "stop":
		return runStopCommand(args[1:], stdout, stderr, getenv)
	case "serve":
		return runServeCommand(args[1:], stderr, getenv)
	case "help", "-h", "--help":
		return runHelp(args[1:], stdout)
	default:
		// Backward compatibility for legacy flag-only invocation.
		if strings.HasPrefix(args[0], "-") {
			return runServeCommand(args, stderr, getenv)
		}
		return fmt.Errorf("unknown command %q\n\n%s", args[0], rootUsageText())
	}
}

func runHelp(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		writeRootUsage(stdout)
		return nil
	}
	if len(args) == 1 {
		switch args[0] {
		case "start":
			fmt.Fprint(stdout, startUsageText())
			return nil
		case "stop":
			fmt.Fprint(stdout, stopUsageText())
			return nil
		}
	}
	return fmt.Errorf("unknown help topic %q\n\n%s", strings.Join(args, " "), rootUsageText())
}

func runStartCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	opts, err := parseStartOptions(args, stderr, getenv)
	if err != nil {
		return err
	}

	serveOpts := serveOptions{
		providers:            opts.providers,
		addr:                 opts.addr,
		h2Addr:               opts.h2Addr,
		accessKey:            opts.accessKey,
		secretKey:            opts.secretKey,
		logLevel:             opts.logLevel,
		gcpAuthMode:          opts.gcpAuthMode,
		azureAuthMode:        opts.azureAuthMode,
		ociAuthMode:          opts.ociAuthMode,
		pidFile:              opts.pidFile,
		lambdaExecutionMode:  opts.lambdaExecutionMode,
		lambdaWorkDir:        opts.lambdaWorkDir,
		persistenceEnabled:   opts.persistenceEnabled,
		stateDir:             opts.stateDir,
		snapshotLoadStrategy: opts.snapshotLoadStrategy,
		snapshotSaveStrategy: opts.snapshotSaveStrategy,
	}

	if opts.foreground {
		return runServe(serveOpts)
	}
	return startInBackground(serveOpts, opts.logFile, stdout)
}

func runStopCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	opts, err := parseStopOptions(args, stderr, getenv)
	if err != nil {
		return err
	}

	pid, err := readPID(opts.pidFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stackyard is not running (pid file not found: %s)", opts.pidFile)
		}
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to signal process %d: %w", pid, err)
	}

	if waitForFileRemoval(opts.pidFile, opts.timeout) {
		fmt.Fprintf(stdout, "stackyard stopped (pid %d)\n", pid)
		return nil
	}

	// Fallback if process ignores interrupt.
	_ = proc.Kill()
	_ = os.Remove(opts.pidFile)
	fmt.Fprintf(stdout, "stackyard stopped (pid %d, forced)\n", pid)
	return nil
}

func runServeCommand(args []string, stderr io.Writer, getenv func(string) string) error {
	opts, err := parseServeOptions(args, stderr, getenv)
	if err != nil {
		return err
	}
	return runServe(opts)
}

func runServe(opts serveOptions) error {
	providers := strings.Join(opts.providers, ",")
	log.Printf(
		"stackyard starting providers=%s addr=%s h2-addr=%s log-level=%s lambda-execution-mode=%s",
		providers,
		opts.addr,
		opts.h2Addr,
		opts.logLevel,
		opts.lambdaExecutionMode,
	)

	cfg := server.Config{
		Addr:                 opts.addr,
		H2Addr:               opts.h2Addr,
		Providers:            opts.providers,
		AccessKey:            opts.accessKey,
		SecretKey:            opts.secretKey,
		LogLevel:             opts.logLevel,
		GCPAuthMode:          opts.gcpAuthMode,
		AzureAuthMode:        opts.azureAuthMode,
		OCIAuthMode:          opts.ociAuthMode,
		LambdaExecutionMode:  opts.lambdaExecutionMode,
		LambdaWorkDir:        opts.lambdaWorkDir,
		PersistenceEnabled:   opts.persistenceEnabled,
		StateDir:             opts.stateDir,
		SnapshotLoadStrategy: opts.snapshotLoadStrategy,
		SnapshotSaveStrategy: opts.snapshotSaveStrategy,
	}
	srv := server.New(cfg)

	if strings.TrimSpace(opts.pidFile) != "" {
		if err := writePID(opts.pidFile, os.Getpid()); err != nil {
			return err
		}
		defer func() {
			_ = os.Remove(opts.pidFile)
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	log.Printf("stackyard listening on http/1.1=%s http/2=%s", opts.addr, opts.h2Addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("server failed: %w", err)
			}
			return nil
		case <-sigCh:
			log.Printf("stackyard received interrupt; exiting")
			if err := srv.Close(); err != nil {
				return fmt.Errorf("server shutdown failed: %w", err)
			}
			return nil
		}
	}
}

func startInBackground(opts serveOptions, logFile string, stdout io.Writer) error {
	if strings.TrimSpace(opts.pidFile) == "" {
		return errors.New("pid file path is required for background start")
	}

	if _, err := os.Stat(opts.pidFile); err == nil {
		return fmt.Errorf("stackyard appears to be running (pid file exists: %s). Run `stackyard stop` first", opts.pidFile)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check pid file %s: %w", opts.pidFile, err)
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	outFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			fmt.Fprintf(stdout, "warning: failed to close log file %s: %v\n", logFile, closeErr)
		}
	}()

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	argv := []string{
		"serve",
		"--providers", strings.Join(opts.providers, ","),
		"--addr", opts.addr,
		"--h2-addr", opts.h2Addr,
		"--log-level", opts.logLevel,
		"--aws-access-key", opts.accessKey,
		"--aws-secret-key", opts.secretKey,
		"--gcp-auth-mode", opts.gcpAuthMode,
		"--azure-auth-mode", opts.azureAuthMode,
		"--oci-auth-mode", opts.ociAuthMode,
		"--lambda-execution-mode", opts.lambdaExecutionMode,
		"--lambda-work-dir", opts.lambdaWorkDir,
		"--persist-state", strconv.FormatBool(opts.persistenceEnabled),
		"--state-dir", opts.stateDir,
		"--snapshot-load-strategy", opts.snapshotLoadStrategy,
		"--snapshot-save-strategy", opts.snapshotSaveStrategy,
		"--pid-file", opts.pidFile,
	}
	cmd := exec.Command(exe, argv...)
	cmd.Stdout = outFile
	cmd.Stderr = outFile
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start background server: %w", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()

	if !waitForFile(opts.pidFile, startupWaitTimeout) {
		return fmt.Errorf("stackyard failed to initialize within %s (see log: %s)", startupWaitTimeout, logFile)
	}

	fmt.Fprintf(stdout, "stackyard started in background (pid %d)\n", pid)
	fmt.Fprintf(stdout, "pid file: %s\n", opts.pidFile)
	fmt.Fprintf(stdout, "log file: %s\n", logFile)
	return nil
}

type serveOptions struct {
	providers            []string
	addr                 string
	h2Addr               string
	accessKey            string
	secretKey            string
	logLevel             string
	gcpAuthMode          string
	azureAuthMode        string
	ociAuthMode          string
	pidFile              string
	lambdaExecutionMode  string
	lambdaWorkDir        string
	persistenceEnabled   bool
	stateDir             string
	snapshotLoadStrategy string
	snapshotSaveStrategy string
}

type startOptions struct {
	providers            []string
	addr                 string
	h2Addr               string
	accessKey            string
	secretKey            string
	logLevel             string
	gcpAuthMode          string
	azureAuthMode        string
	ociAuthMode          string
	foreground           bool
	pidFile              string
	logFile              string
	lambdaExecutionMode  string
	lambdaWorkDir        string
	persistenceEnabled   bool
	stateDir             string
	snapshotLoadStrategy string
	snapshotSaveStrategy string
}

type stopOptions struct {
	pidFile string
	timeout time.Duration
}

type serveFlagValues struct {
	providersValue       string
	addr                 string
	port                 int
	h2Addr               string
	h2Port               int
	accessKey            string
	secretKey            string
	logLevel             string
	gcpAuthMode          string
	azureAuthMode        string
	ociAuthMode          string
	lambdaExecutionMode  string
	lambdaWorkDir        string
	persistenceEnabled   bool
	stateDir             string
	snapshotLoadStrategy string
	snapshotSaveStrategy string
}

func parseStartOptions(args []string, stderr io.Writer, getenv func(string) string) (startOptions, error) {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, startUsageText())
	}

	values := bindServeFlags(fs, getenv)
	defaultPID := envOrDefault(getenv, "STACKYARD_PID_FILE", defaultPIDFile())
	defaultLog := envOrDefault(getenv, "STACKYARD_LOG_FILE", defaultLogFile())

	var foreground bool
	var pidFile string
	var logFile string
	fs.BoolVar(&foreground, "foreground", false, "Run in foreground instead of daemon mode")
	fs.StringVar(&pidFile, "pid-file", defaultPID, "PID file path for daemon lifecycle commands")
	fs.StringVar(&logFile, "log-file", defaultLog, "Log file path used by background mode")

	if err := fs.Parse(args); err != nil {
		return startOptions{}, err
	}
	if fs.NArg() > 0 {
		return startOptions{}, fmt.Errorf("unexpected arguments for start: %s", strings.Join(fs.Args(), " "))
	}

	serveOpts, err := finalizeServeOptions(fs, values)
	if err != nil {
		return startOptions{}, err
	}

	pidFile = strings.TrimSpace(pidFile)
	if pidFile == "" {
		return startOptions{}, errors.New("--pid-file must not be empty")
	}
	logFile = strings.TrimSpace(logFile)
	if !foreground && logFile == "" {
		return startOptions{}, errors.New("--log-file must not be empty in background mode")
	}

	return startOptions{
		providers:            serveOpts.providers,
		addr:                 serveOpts.addr,
		h2Addr:               serveOpts.h2Addr,
		accessKey:            serveOpts.accessKey,
		secretKey:            serveOpts.secretKey,
		logLevel:             serveOpts.logLevel,
		gcpAuthMode:          serveOpts.gcpAuthMode,
		azureAuthMode:        serveOpts.azureAuthMode,
		ociAuthMode:          serveOpts.ociAuthMode,
		foreground:           foreground,
		pidFile:              pidFile,
		logFile:              logFile,
		lambdaExecutionMode:  serveOpts.lambdaExecutionMode,
		lambdaWorkDir:        serveOpts.lambdaWorkDir,
		persistenceEnabled:   serveOpts.persistenceEnabled,
		stateDir:             serveOpts.stateDir,
		snapshotLoadStrategy: serveOpts.snapshotLoadStrategy,
		snapshotSaveStrategy: serveOpts.snapshotSaveStrategy,
	}, nil
}

func parseServeOptions(args []string, stderr io.Writer, getenv func(string) string) (serveOptions, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, startUsageText())
	}

	values := bindServeFlags(fs, getenv)
	var pidFile string
	fs.StringVar(&pidFile, "pid-file", "", "Internal: PID file path")

	if err := fs.Parse(args); err != nil {
		return serveOptions{}, err
	}
	if fs.NArg() > 0 {
		return serveOptions{}, fmt.Errorf("unexpected arguments for serve: %s", strings.Join(fs.Args(), " "))
	}

	opts, err := finalizeServeOptions(fs, values)
	if err != nil {
		return serveOptions{}, err
	}
	opts.pidFile = strings.TrimSpace(pidFile)
	return opts, nil
}

func parseStopOptions(args []string, stderr io.Writer, getenv func(string) string) (stopOptions, error) {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, stopUsageText())
	}

	var pidFile string
	var timeout time.Duration
	fs.StringVar(&pidFile, "pid-file", envOrDefault(getenv, "STACKYARD_PID_FILE", defaultPIDFile()), "PID file path used by stackyard start")
	fs.DurationVar(&timeout, "timeout", defaultShutdownTimeout, "How long to wait for graceful shutdown before forcing")

	if err := fs.Parse(args); err != nil {
		return stopOptions{}, err
	}
	if fs.NArg() > 0 {
		return stopOptions{}, fmt.Errorf("unexpected arguments for stop: %s", strings.Join(fs.Args(), " "))
	}
	if timeout <= 0 {
		return stopOptions{}, errors.New("--timeout must be greater than 0")
	}
	pidFile = strings.TrimSpace(pidFile)
	if pidFile == "" {
		return stopOptions{}, errors.New("--pid-file must not be empty")
	}
	return stopOptions{pidFile: pidFile, timeout: timeout}, nil
}

func bindServeFlags(fs *flag.FlagSet, getenv func(string) string) *serveFlagValues {
	values := &serveFlagValues{
		providersValue:       envOrDefault(getenv, "STACKYARD_PROVIDERS", "aws"),
		addr:                 strings.TrimSpace(getenv("STACKYARD_ADDR")),
		port:                 envInt(getenv, "STACKYARD_PORT", defaultPort),
		h2Addr:               strings.TrimSpace(getenv("STACKYARD_H2_ADDR")),
		h2Port:               envInt(getenv, "STACKYARD_H2_PORT", defaultHTTP2Port),
		accessKey:            envOrDefault(getenv, "STACKYARD_ACCESS_KEY", "stackyard"),
		secretKey:            envOrDefault(getenv, "STACKYARD_SECRET_KEY", "stackyard"),
		logLevel:             envOrDefault(getenv, "STACKYARD_LOG_LEVEL", "info"),
		gcpAuthMode:          envOrDefault(getenv, "STACKYARD_GCP_AUTH_MODE", "emulator"),
		azureAuthMode:        envOrDefault(getenv, "STACKYARD_AZURE_AUTH_MODE", "shared_key_or_sas"),
		ociAuthMode:          envOrDefault(getenv, "STACKYARD_OCI_AUTH_MODE", "signature"),
		lambdaExecutionMode:  envOrDefault(getenv, "STACKYARD_LAMBDA_EXECUTION_MODE", "mock"),
		lambdaWorkDir:        strings.TrimSpace(getenv("STACKYARD_LAMBDA_WORK_DIR")),
		persistenceEnabled:   envBool(getenv, "STACKYARD_PERSISTENCE", false),
		stateDir:             strings.TrimSpace(getenv("STACKYARD_STATE_DIR")),
		snapshotLoadStrategy: envOrDefault(getenv, "STACKYARD_SNAPSHOT_LOAD_STRATEGY", "on_startup"),
		snapshotSaveStrategy: envOrDefault(getenv, "STACKYARD_SNAPSHOT_SAVE_STRATEGY", "on_request"),
	}
	fs.StringVar(&values.providersValue, "providers", values.providersValue, "Comma-separated providers to enable (currently: "+supportedProvidersText+")")
	fs.StringVar(&values.addr, "addr", values.addr, "HTTP listen address (example: :4566)")
	fs.IntVar(&values.port, "port", values.port, "HTTP listen port (used when --addr is not provided)")
	fs.StringVar(&values.h2Addr, "h2-addr", values.h2Addr, "HTTP/2 listen address for gRPC clients (example: :4567)")
	fs.IntVar(&values.h2Port, "h2-port", values.h2Port, "HTTP/2 listen port (used when --h2-addr is not provided)")
	fs.StringVar(&values.accessKey, "aws-access-key", values.accessKey, "Access key expected by SigV4 validation")
	fs.StringVar(&values.secretKey, "aws-secret-key", values.secretKey, "Secret key expected by SigV4 validation")
	fs.StringVar(&values.logLevel, "log-level", values.logLevel, "Log level (debug, info, warn, error)")
	fs.StringVar(&values.gcpAuthMode, "gcp-auth-mode", values.gcpAuthMode, "GCP request auth mode (emulator, bearer_tolerant, bearer_required)")
	fs.StringVar(&values.azureAuthMode, "azure-auth-mode", values.azureAuthMode, "Azure request auth mode (shared_key_or_sas, shared_key, sas, disabled)")
	fs.StringVar(&values.ociAuthMode, "oci-auth-mode", values.ociAuthMode, "OCI request auth mode (signature, disabled)")
	fs.StringVar(&values.lambdaExecutionMode, "lambda-execution-mode", values.lambdaExecutionMode, "Lambda invoke behavior (mock, local)")
	fs.StringVar(&values.lambdaWorkDir, "lambda-work-dir", values.lambdaWorkDir, "Lambda local execution workspace directory")
	fs.BoolVar(&values.persistenceEnabled, "persist-state", values.persistenceEnabled, "Enable persistent state journal and startup replay")
	fs.StringVar(&values.stateDir, "state-dir", values.stateDir, "Persistent state directory")
	fs.StringVar(&values.snapshotLoadStrategy, "snapshot-load-strategy", values.snapshotLoadStrategy, "Snapshot load strategy (on_startup, manual)")
	fs.StringVar(&values.snapshotSaveStrategy, "snapshot-save-strategy", values.snapshotSaveStrategy, "Snapshot save strategy (on_request, on_shutdown, manual)")
	return values
}

func finalizeServeOptions(fs *flag.FlagSet, values *serveFlagValues) (serveOptions, error) {
	if values.port < 1 || values.port > 65535 {
		return serveOptions{}, fmt.Errorf("invalid --port %d: must be between 1 and 65535", values.port)
	}
	if values.h2Port < 1 || values.h2Port > 65535 {
		return serveOptions{}, fmt.Errorf("invalid --h2-port %d: must be between 1 and 65535", values.h2Port)
	}

	var addrSet, portSet, h2AddrSet, h2PortSet bool
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			addrSet = true
		}
		if f.Name == "port" {
			portSet = true
		}
		if f.Name == "h2-addr" {
			h2AddrSet = true
		}
		if f.Name == "h2-port" {
			h2PortSet = true
		}
	})

	resolvedAddr := strings.TrimSpace(values.addr)
	if !addrSet {
		switch {
		case portSet:
			resolvedAddr = ":" + strconv.Itoa(values.port)
		case resolvedAddr == "":
			resolvedAddr = ":" + strconv.Itoa(values.port)
		}
	}
	resolvedH2Addr := strings.TrimSpace(values.h2Addr)
	if !h2AddrSet {
		switch {
		case h2PortSet:
			resolvedH2Addr = resolveSiblingPortAddr(resolvedAddr, values.h2Port)
		case resolvedH2Addr == "":
			resolvedH2Addr = resolveSiblingPortAddr(resolvedAddr, values.h2Port)
		}
	}
	if h2AddrSet && resolvedH2Addr == "" {
		return serveOptions{}, errors.New("--h2-addr must not be empty")
	}
	if resolvedH2Addr == resolvedAddr {
		return serveOptions{}, errors.New("--h2-addr must differ from --addr")
	}

	providers, err := parseProviders(values.providersValue)
	if err != nil {
		return serveOptions{}, err
	}
	gcpAuthMode := strings.ToLower(strings.TrimSpace(values.gcpAuthMode))
	switch gcpAuthMode {
	case "", "emulator":
		gcpAuthMode = "emulator"
	case "bearer_tolerant", "bearer_required":
	default:
		return serveOptions{}, fmt.Errorf("unsupported --gcp-auth-mode %q (supported: emulator, bearer_tolerant, bearer_required)", values.gcpAuthMode)
	}
	azureAuthMode := strings.ToLower(strings.TrimSpace(values.azureAuthMode))
	switch azureAuthMode {
	case "", "shared_key_or_sas":
		azureAuthMode = "shared_key_or_sas"
	case "shared_key", "sas", "disabled":
	default:
		return serveOptions{}, fmt.Errorf("unsupported --azure-auth-mode %q (supported: shared_key_or_sas, shared_key, sas, disabled)", values.azureAuthMode)
	}
	ociAuthMode := strings.ToLower(strings.TrimSpace(values.ociAuthMode))
	switch ociAuthMode {
	case "", "signature":
		ociAuthMode = "signature"
	case "disabled":
	default:
		return serveOptions{}, fmt.Errorf("unsupported --oci-auth-mode %q (supported: signature, disabled)", values.ociAuthMode)
	}
	lambdaExecutionMode := strings.ToLower(strings.TrimSpace(values.lambdaExecutionMode))
	switch lambdaExecutionMode {
	case "", "mock":
		lambdaExecutionMode = "mock"
	case "local":
	default:
		return serveOptions{}, fmt.Errorf("unsupported --lambda-execution-mode %q (supported: mock, local)", values.lambdaExecutionMode)
	}
	snapshotLoadStrategy := strings.ToLower(strings.TrimSpace(values.snapshotLoadStrategy))
	switch snapshotLoadStrategy {
	case "", "on_startup":
		snapshotLoadStrategy = "on_startup"
	case "manual":
	default:
		return serveOptions{}, fmt.Errorf("unsupported --snapshot-load-strategy %q (supported: on_startup, manual)", values.snapshotLoadStrategy)
	}
	snapshotSaveStrategy := strings.ToLower(strings.TrimSpace(values.snapshotSaveStrategy))
	switch snapshotSaveStrategy {
	case "", "on_request":
		snapshotSaveStrategy = "on_request"
	case "on_shutdown", "manual":
	default:
		return serveOptions{}, fmt.Errorf("unsupported --snapshot-save-strategy %q (supported: on_request, on_shutdown, manual)", values.snapshotSaveStrategy)
	}

	return serveOptions{
		providers:            providers,
		addr:                 resolvedAddr,
		h2Addr:               resolvedH2Addr,
		accessKey:            values.accessKey,
		secretKey:            values.secretKey,
		logLevel:             values.logLevel,
		gcpAuthMode:          gcpAuthMode,
		azureAuthMode:        azureAuthMode,
		ociAuthMode:          ociAuthMode,
		lambdaExecutionMode:  lambdaExecutionMode,
		lambdaWorkDir:        strings.TrimSpace(values.lambdaWorkDir),
		persistenceEnabled:   values.persistenceEnabled,
		stateDir:             strings.TrimSpace(values.stateDir),
		snapshotLoadStrategy: snapshotLoadStrategy,
		snapshotSaveStrategy: snapshotSaveStrategy,
	}, nil
}

func parseProviders(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		if _, ok := supportedProviders[p]; !ok {
			return nil, fmt.Errorf("unsupported provider %q (supported: %s)", p, supportedProvidersText)
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one provider must be set (supported: %s)", supportedProvidersText)
	}
	return out, nil
}

func resolveSiblingPortAddr(addr string, port int) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return ":" + strconv.Itoa(port)
	}
	host, _, err := net.SplitHostPort(trimmed)
	if err != nil {
		return ":" + strconv.Itoa(port)
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func readPID(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid file %s", path)
	}
	return pid, nil
}

func writePID(path string, pid int) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create pid directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write pid file %s: %w", path, err)
	}
	return nil
}

func waitForFile(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func waitForFileRemoval(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func defaultPIDFile() string {
	return filepath.Join(os.TempDir(), "stackyard.pid")
}

func defaultLogFile() string {
	return filepath.Join(os.TempDir(), "stackyard.log")
}

func envOrDefault(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(getenv func(string) string, key string, fallback int) int {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(getenv func(string) string, key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(getenv(key)))
	switch raw {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func writeRootUsage(w io.Writer) {
	fmt.Fprint(w, rootUsageText())
}

func rootUsageText() string {
	return `Stackyard CLI

Usage:
  stackyard start [flags]
  stackyard stop [flags]
  stackyard help [command]

Commands:
  start    Start Stackyard in background by default
  stop     Stop a running Stackyard daemon
  help     Show help for a command

Start Flags:
  --providers string       Comma-separated providers to enable.
                           Options: aws, gcp, azure, oci
  --port int               HTTP listen port (used when --addr is not provided).
                           Options: 1-65535 (default 4566)
  --addr string            HTTP listen address.
                           Options: host:port or :port; when set, overrides --port
  --h2-port int            HTTP/2 listen port (used when --h2-addr is not provided).
                           Options: 1-65535 (default 4567)
  --h2-addr string         HTTP/2 listen address for gRPC clients.
                           Options: host:port or :port; when set, overrides --h2-port
  --log-level string       Server log verbosity level.
                           Options: debug, info, warn, error (default info)
  --aws-access-key string  Access key expected by SigV4 validation.
                           Options: any non-empty string (default stackyard)
  --aws-secret-key string  Secret key expected by SigV4 validation.
                           Options: any non-empty string (default stackyard)
  --gcp-auth-mode string   GCP request auth mode.
                           Options: emulator, bearer_tolerant, bearer_required (default emulator)
  --azure-auth-mode string Azure request auth mode.
                           Options: shared_key_or_sas, shared_key, sas, disabled (default shared_key_or_sas)
  --oci-auth-mode string   OCI request auth mode.
                           Options: signature, disabled (default signature)
  --lambda-execution-mode string
                           Lambda invoke behavior.
                           Options: mock, local (default mock)
  --lambda-work-dir string Lambda local execution workspace directory.
                           Options: absolute or relative path
  --persist-state          Enable persistent state journal and startup replay
  --state-dir string       Persistent state directory (default OS temp dir)
  --snapshot-load-strategy string
                           Snapshot load strategy.
                           Options: on_startup, manual (default on_startup)
  --snapshot-save-strategy string
                           Snapshot save strategy.
                           Options: on_request, on_shutdown, manual (default on_request)
  --foreground             Run in foreground instead of background mode
  --pid-file string        PID file used by start/stop lifecycle (default OS temp dir)
  --log-file string        Daemon log file path (default OS temp dir)

Stop Flags:
  --pid-file string        PID file used to find the daemon process
  --timeout duration       Graceful shutdown wait time before force kill

Examples:
  stackyard start --providers aws
  stackyard start --providers aws,gcp,azure,oci
  stackyard start --providers aws --port 4566
  stackyard start --providers aws --h2-port 4567
  stackyard start --providers aws --log-level debug
  stackyard start --providers aws --lambda-execution-mode local
  stackyard start --providers aws --persist-state --state-dir /tmp/stackyard-state
  stackyard stop
  stackyard help start
`
}

func startUsageText() string {
	return `Usage:
  stackyard start [flags]

Behavior:
  Starts in background by default. Use --foreground to run attached.

Flags:
  --providers string       Comma-separated providers to enable.
                           Options: aws, gcp, azure, oci
  --port int               HTTP listen port (used when --addr is not provided).
                           Options: 1-65535 (default 4566)
  --addr string            HTTP listen address (example: :4566).
                           Options: host:port or :port; when set, overrides --port
  --h2-port int            HTTP/2 listen port (used when --h2-addr is not provided).
                           Options: 1-65535 (default 4567)
  --h2-addr string         HTTP/2 listen address for gRPC clients (example: :4567).
                           Options: host:port or :port; when set, overrides --h2-port
  --log-level string       Log level.
                           Options: debug, info, warn, error
  --aws-access-key string  Access key expected by SigV4 validation.
                           Options: any non-empty string
  --aws-secret-key string  Secret key expected by SigV4 validation.
                           Options: any non-empty string
  --gcp-auth-mode string   GCP request auth mode (emulator, bearer_tolerant, bearer_required)
  --azure-auth-mode string Azure request auth mode (shared_key_or_sas, shared_key, sas, disabled)
  --oci-auth-mode string   OCI request auth mode (signature, disabled)
  --lambda-execution-mode string
                           Lambda invoke behavior.
                           Options: mock, local
  --lambda-work-dir string Lambda local execution workspace directory
  --persist-state          Enable persistent state journal and startup replay
  --state-dir string       Persistent state directory
  --snapshot-load-strategy string
                           Snapshot load strategy (on_startup, manual)
  --snapshot-save-strategy string
                           Snapshot save strategy (on_request, on_shutdown, manual)
  --foreground             Run in foreground instead of daemon mode
  --pid-file string        PID file path used for stop command
  --log-file string        Background log file path

Examples:
  stackyard start --providers aws
  stackyard start --providers aws,gcp,azure,oci
  stackyard start --providers aws --port 4566
  stackyard start --providers aws --h2-port 4567
  stackyard start --providers aws --log-level debug
  stackyard start --providers aws --lambda-execution-mode local
  stackyard start --providers aws --persist-state --state-dir /tmp/stackyard-state
  stackyard start --providers aws --foreground
`
}

func stopUsageText() string {
	return `Usage:
  stackyard stop [flags]

Flags:
  --pid-file string   PID file path used by stackyard start
  --timeout duration  Graceful shutdown wait time (default 10s)

Examples:
  stackyard stop
  stackyard stop --pid-file /tmp/stackyard.pid
`
}
