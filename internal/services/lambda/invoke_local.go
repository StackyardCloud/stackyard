package lambda

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func invokeLocal(fn Function, code, payload []byte, configuredWorkDir string) (InvokeResult, error) {
	if len(code) == 0 {
		return InvokeResult{}, ErrInvalidParameter
	}

	workDir, cleanup, err := lambdaInvocationDir(configuredWorkDir)
	if err != nil {
		return InvokeResult{}, fmt.Errorf("%w: %v", ErrInvocationFailed, err)
	}
	defer cleanup()

	if err := unpackLambdaArchive(code, workDir); err != nil {
		return invocationFailureResult("invalid deployment package", err.Error()), nil
	}

	command, args, err := lambdaRuntimeCommand(fn, workDir)
	if err != nil {
		return invocationFailureResult(err.Error(), ""), nil
	}

	timeout := time.Duration(fn.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(),
		"AWS_REGION="+DefaultRegion,
		"AWS_DEFAULT_REGION="+DefaultRegion,
		"AWS_LAMBDA_FUNCTION_NAME="+fn.Name,
		"AWS_LAMBDA_FUNCTION_VERSION="+fn.Version,
		"AWS_EXECUTION_ENV=stackyard-local",
		"LAMBDA_TASK_ROOT="+workDir,
		"_HANDLER="+fn.Handler,
	)

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return invocationFailureResult(
				fmt.Sprintf("function timed out after %s", timeout),
				stderr.String(),
			), nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(err.Error())
		}
		return invocationFailureResult(msg, stderr.String()), nil
	}

	body := bytes.TrimSpace(stdout.Bytes())
	if len(body) == 0 {
		body = []byte("{}")
	}

	result := InvokeResult{
		StatusCode: 200,
		Payload:    append([]byte(nil), body...),
	}
	if stderr.Len() > 0 {
		result.LogResult = base64.StdEncoding.EncodeToString(stderr.Bytes())
	}
	return result, nil
}

func invocationFailureResult(message, stderr string) InvokeResult {
	errBody, _ := json.Marshal(map[string]any{
		"errorType":    "RuntimeError",
		"errorMessage": strings.TrimSpace(message),
	})
	result := InvokeResult{
		StatusCode:    200,
		Payload:       errBody,
		FunctionError: "Unhandled",
	}
	if strings.TrimSpace(stderr) != "" {
		result.LogResult = base64.StdEncoding.EncodeToString([]byte(stderr))
	}
	return result
}

func lambdaInvocationDir(configuredWorkDir string) (string, func(), error) {
	base := strings.TrimSpace(configuredWorkDir)
	if base == "" {
		base = os.TempDir()
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", nil, err
	}
	workDir, err := os.MkdirTemp(base, "stackyard-lambda-*")
	if err != nil {
		return "", nil, err
	}
	return workDir, func() {
		_ = os.RemoveAll(workDir)
	}, nil
}

func unpackLambdaArchive(code []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		name := filepath.Clean(file.Name)
		if name == "." || name == "" {
			continue
		}
		if filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(filepath.Separator)) || name == ".." {
			return fmt.Errorf("invalid archive entry path %q", file.Name)
		}
		target := filepath.Join(dest, name)
		if target != dest && !strings.HasPrefix(target, dest+string(filepath.Separator)) {
			return fmt.Errorf("invalid archive entry path %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			_ = src.Close()
			return err
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := dst.Close()
		_ = src.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func lambdaRuntimeCommand(fn Function, workDir string) (string, []string, error) {
	runtime := strings.ToLower(strings.TrimSpace(fn.Runtime))
	switch {
	case strings.HasPrefix(runtime, "provided"):
		bootstrap := filepath.Join(workDir, "bootstrap")
		info, err := os.Stat(bootstrap)
		if err != nil || info.IsDir() {
			return "", nil, errors.New("missing bootstrap executable")
		}
		if info.Mode()&0o111 == 0 {
			return "/bin/sh", []string{bootstrap}, nil
		}
		return bootstrap, nil, nil
	case strings.HasPrefix(runtime, "python"):
		module, function := parseLambdaHandler(fn.Handler)
		runnerPath := filepath.Join(workDir, "__stackyard_py_runner__.py")
		if err := os.WriteFile(runnerPath, []byte(lambdaPythonRunnerSource), 0o644); err != nil {
			return "", nil, err
		}
		return "python3", []string{runnerPath, module, function}, nil
	case strings.HasPrefix(runtime, "nodejs"):
		module, function := parseLambdaHandler(fn.Handler)
		runnerPath := filepath.Join(workDir, "__stackyard_node_runner__.js")
		if err := os.WriteFile(runnerPath, []byte(lambdaNodeRunnerSource), 0o644); err != nil {
			return "", nil, err
		}
		return "node", []string{runnerPath, module, function}, nil
	default:
		return "", nil, fmt.Errorf("unsupported runtime %q", fn.Runtime)
	}
}

func parseLambdaHandler(handler string) (module string, function string) {
	handler = strings.TrimSpace(handler)
	if handler == "" {
		return "handler", "handler"
	}
	parts := strings.Split(handler, ".")
	if len(parts) < 2 {
		return handler, "handler"
	}
	module = strings.Join(parts[:len(parts)-1], ".")
	function = parts[len(parts)-1]
	if strings.TrimSpace(module) == "" {
		module = "handler"
	}
	if strings.TrimSpace(function) == "" {
		function = "handler"
	}
	return module, function
}

const lambdaPythonRunnerSource = `import importlib
import json
import os
import sys

module_name = sys.argv[1]
function_name = sys.argv[2]
payload_raw = sys.stdin.read()
if payload_raw.strip():
    try:
        payload = json.loads(payload_raw)
    except Exception:
        payload = payload_raw
else:
    payload = {}

mod = importlib.import_module(module_name)
handler = getattr(mod, function_name)
result = handler(payload, {"function_name": os.environ.get("AWS_LAMBDA_FUNCTION_NAME", "")})
if result is None:
    result = {}
sys.stdout.write(json.dumps(result))
`

const lambdaNodeRunnerSource = `const fs = require('fs');

const moduleName = process.argv[2];
const functionName = process.argv[3];

let payloadRaw = fs.readFileSync(0, 'utf8');
let payload = {};
if (payloadRaw && payloadRaw.trim() !== '') {
  try {
    payload = JSON.parse(payloadRaw);
  } catch (err) {
    payload = payloadRaw;
  }
}

async function run() {
  const mod = require('./' + moduleName);
  const handler = mod[functionName];
  if (typeof handler !== 'function') {
    throw new Error('handler not found');
  }
  const result = await handler(payload, { functionName: process.env.AWS_LAMBDA_FUNCTION_NAME || '' });
  process.stdout.write(JSON.stringify(result || {}));
}

run().catch((err) => {
  process.stderr.write(String(err && err.stack ? err.stack : err));
  process.exit(1);
});
`
