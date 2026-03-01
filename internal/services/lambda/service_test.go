package lambda

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestServiceLifecycle(t *testing.T) {
	svc := NewService()

	fn, err := svc.CreateFunction(
		"demo-fn",
		"arn:aws:iam::123456789012:role/lambda-role",
		"bootstrap",
		"provided.al2",
		"demo",
		3,
		128,
		[]byte("zip-content"),
		map[string]string{"env": "test"},
		nil,
	)
	if err != nil {
		t.Fatalf("create function: %v", err)
	}
	if fn.ARN == "" {
		t.Fatalf("expected function arn")
	}

	functions, _ := svc.ListFunctions(10, 0)
	if len(functions) != 1 {
		t.Fatalf("expected one function, got %d", len(functions))
	}

	gotFn, tags, err := svc.GetFunction(fn.Name, "")
	if err != nil {
		t.Fatalf("get function: %v", err)
	}
	if gotFn.Name != fn.Name {
		t.Fatalf("unexpected function name %q", gotFn.Name)
	}
	if tags["env"] != "test" {
		t.Fatalf("expected env tag")
	}

	desc := "updated"
	timeout := int32(10)
	updatedCfg, err := svc.UpdateFunctionConfiguration(fn.Name, "", nil, nil, nil, &desc, &timeout, nil, nil)
	if err != nil {
		t.Fatalf("update function configuration: %v", err)
	}
	if updatedCfg.Description != "updated" || updatedCfg.Timeout != 10 {
		t.Fatalf("expected updated function configuration")
	}

	updatedCode, err := svc.UpdateFunctionCode(fn.Name, "", []byte("new-code"), false)
	if err != nil {
		t.Fatalf("update function code: %v", err)
	}
	if updatedCode.CodeSHA256 == fn.CodeSHA256 {
		t.Fatalf("expected code hash to change")
	}

	version, err := svc.PublishVersion(fn.Name, "published")
	if err != nil {
		t.Fatalf("publish version: %v", err)
	}
	if version.Version != "1" {
		t.Fatalf("expected first published version to be 1, got %q", version.Version)
	}

	versions, err := svc.ListVersionsByFunction(fn.Name)
	if err != nil {
		t.Fatalf("list versions by function: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected two versions ($LATEST + published), got %d", len(versions))
	}

	alias, err := svc.CreateAlias(fn.Name, "live", "1", "live alias")
	if err != nil {
		t.Fatalf("create alias: %v", err)
	}
	if alias.ARN == "" {
		t.Fatalf("expected alias arn")
	}

	gotAlias, err := svc.GetAlias(fn.Name, "live")
	if err != nil {
		t.Fatalf("get alias: %v", err)
	}
	if gotAlias.FunctionVersion != "1" {
		t.Fatalf("expected alias version 1")
	}

	aliases, err := svc.ListAliases(fn.Name)
	if err != nil {
		t.Fatalf("list aliases: %v", err)
	}
	if len(aliases) != 1 {
		t.Fatalf("expected one alias, got %d", len(aliases))
	}

	aliasDesc := "prod"
	updatedAlias, err := svc.UpdateAlias(fn.Name, "live", "1", &aliasDesc)
	if err != nil {
		t.Fatalf("update alias: %v", err)
	}
	if updatedAlias.Description != "prod" {
		t.Fatalf("expected alias description update")
	}

	statement, err := svc.AddPermission(fn.Name, "", "stmt-1", "lambda:InvokeFunction", "123456789012", "", "")
	if err != nil {
		t.Fatalf("add permission: %v", err)
	}
	if statement == "" {
		t.Fatalf("expected statement payload")
	}

	policy, revision, err := svc.GetPolicy(fn.Name, "")
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if policy == "" || revision == "" {
		t.Fatalf("expected policy and revision")
	}

	if err := svc.RemovePermission(fn.Name, "", "stmt-1"); err != nil {
		t.Fatalf("remove permission: %v", err)
	}

	if err := svc.TagResource(fn.ARN, map[string]string{"team": "platform"}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	tagMap, err := svc.ListTags(fn.ARN)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if tagMap["team"] != "platform" {
		t.Fatalf("expected team tag")
	}
	if err := svc.UntagResource(fn.ARN, []string{"team"}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	invokeResp, err := svc.Invoke(fn.Name, "", "RequestResponse", []byte(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("invoke function: %v", err)
	}
	if invokeResp.StatusCode != 200 {
		t.Fatalf("expected invoke status 200, got %d", invokeResp.StatusCode)
	}

	if err := svc.DeleteAlias(fn.Name, "live"); err != nil {
		t.Fatalf("delete alias: %v", err)
	}
	if err := svc.DeleteFunction(fn.Name, ""); err != nil {
		t.Fatalf("delete function: %v", err)
	}
}

func TestServiceErrors(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateFunction("", "", "", "", "", 0, 0, nil, nil, nil); err != ErrInvalidParameter {
		t.Fatalf("expected invalid parameter for create function, got %v", err)
	}

	if err := svc.DeleteFunction("missing", ""); err != ErrNotFound {
		t.Fatalf("expected not found for delete function, got %v", err)
	}

	if _, _, err := svc.GetFunction("missing", ""); err != ErrNotFound {
		t.Fatalf("expected not found for get function, got %v", err)
	}

	if _, err := svc.CreateFunction(
		"demo-fn",
		"arn:aws:iam::123456789012:role/lambda-role",
		"bootstrap",
		"provided.al2",
		"demo",
		3,
		128,
		[]byte("zip-content"),
		nil,
		nil,
	); err != nil {
		t.Fatalf("create function: %v", err)
	}

	if _, err := svc.CreateAlias("demo-fn", "live", "9", "bad"); err != ErrNotFound {
		t.Fatalf("expected not found for alias version, got %v", err)
	}
	if _, err := svc.AddPermission("demo-fn", "", "", "", "", "", ""); err != ErrInvalidParameter {
		t.Fatalf("expected invalid parameter for add permission, got %v", err)
	}
	if _, err := svc.ListTags("arn:aws:lambda:us-east-1:123456789012:function:missing"); err != ErrNotFound {
		t.Fatalf("expected not found for list tags, got %v", err)
	}
}

func TestServiceInvokeLocalProvidedRuntime(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithConfig(ServiceConfig{
		ExecutionMode: ExecutionModeLocal,
		WorkDir:       t.TempDir(),
	})

	code := zipBootstrapCode(t, `#!/bin/sh
set -eu
payload="$(cat)"
if [ -z "${payload}" ]; then
  payload='{}'
fi
printf '{"ok":true,"payload":%s}' "$payload"
`)

	if _, err := svc.CreateFunction(
		"demo-local",
		"arn:aws:iam::123456789012:role/lambda-role",
		"bootstrap",
		"provided.al2",
		"demo local function",
		3,
		128,
		code,
		nil,
		nil,
	); err != nil {
		t.Fatalf("create function: %v", err)
	}

	resp, err := svc.Invoke("demo-local", "", "RequestResponse", []byte(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("invoke function: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status code 200, got %d", resp.StatusCode)
	}
	if resp.FunctionError != "" {
		t.Fatalf("expected no function error, got %q", resp.FunctionError)
	}
	if !bytes.Contains(resp.Payload, []byte(`"ok":true`)) {
		t.Fatalf("expected success payload, got %s", string(resp.Payload))
	}
	if !bytes.Contains(resp.Payload, []byte(`"hello":"world"`)) {
		t.Fatalf("expected invocation payload echoed back, got %s", string(resp.Payload))
	}
}

func TestServiceInvokeLocalPublishedVersionUsesVersionedCode(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithConfig(ServiceConfig{
		ExecutionMode: ExecutionModeLocal,
		WorkDir:       t.TempDir(),
	})

	if _, err := svc.CreateFunction(
		"demo-versioned",
		"arn:aws:iam::123456789012:role/lambda-role",
		"bootstrap",
		"provided.al2",
		"demo versioned function",
		3,
		128,
		zipBootstrapCode(t, `#!/bin/sh
printf '{"version":"v1"}'
`),
		nil,
		nil,
	); err != nil {
		t.Fatalf("create function: %v", err)
	}

	if _, err := svc.PublishVersion("demo-versioned", "snapshot v1"); err != nil {
		t.Fatalf("publish version: %v", err)
	}
	if _, err := svc.UpdateFunctionCode("demo-versioned", "", zipBootstrapCode(t, `#!/bin/sh
printf '{"version":"v2"}'
`), false); err != nil {
		t.Fatalf("update function code: %v", err)
	}

	v1Resp, err := svc.Invoke("demo-versioned", "1", "RequestResponse", nil)
	if err != nil {
		t.Fatalf("invoke version 1: %v", err)
	}
	if string(v1Resp.Payload) != `{"version":"v1"}` {
		t.Fatalf("expected version 1 payload, got %s", string(v1Resp.Payload))
	}
	if v1Resp.ExecutedVersion != "1" {
		t.Fatalf("expected executed version 1, got %q", v1Resp.ExecutedVersion)
	}

	latestResp, err := svc.Invoke("demo-versioned", "", "RequestResponse", nil)
	if err != nil {
		t.Fatalf("invoke latest: %v", err)
	}
	if string(latestResp.Payload) != `{"version":"v2"}` {
		t.Fatalf("expected latest payload to use updated code, got %s", string(latestResp.Payload))
	}
	if latestResp.ExecutedVersion != "$LATEST" {
		t.Fatalf("expected executed version $LATEST, got %q", latestResp.ExecutedVersion)
	}
}

func TestServiceInvokeLocalInvalidArchiveReturnsFunctionError(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithConfig(ServiceConfig{
		ExecutionMode: ExecutionModeLocal,
		WorkDir:       t.TempDir(),
	})

	if _, err := svc.CreateFunction(
		"bad-archive",
		"arn:aws:iam::123456789012:role/lambda-role",
		"bootstrap",
		"provided.al2",
		"invalid archive",
		3,
		128,
		[]byte("not-a-zip"),
		nil,
		nil,
	); err != nil {
		t.Fatalf("create function: %v", err)
	}

	resp, err := svc.Invoke("bad-archive", "", "RequestResponse", nil)
	if err != nil {
		t.Fatalf("invoke function: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status code 200, got %d", resp.StatusCode)
	}
	if resp.FunctionError != "Unhandled" {
		t.Fatalf("expected FunctionError Unhandled, got %q", resp.FunctionError)
	}
}

func zipBootstrapCode(t *testing.T, script string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	header := &zip.FileHeader{
		Name:   "bootstrap",
		Method: zip.Deflate,
	}
	header.SetMode(0o755)
	w, err := zw.CreateHeader(header)
	if err != nil {
		t.Fatalf("create bootstrap header: %v", err)
	}
	if _, err := w.Write([]byte(script)); err != nil {
		t.Fatalf("write bootstrap script: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}
