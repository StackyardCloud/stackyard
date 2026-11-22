package server

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// eicTestKey builds a well-formed ssh-ed25519 public key line.
func eicTestKey(seed byte) string {
	blob := binary.BigEndian.AppendUint32(nil, 11)
	blob = append(blob, "ssh-ed25519"...)
	blob = binary.BigEndian.AppendUint32(blob, 32)
	for i := 0; i < 32; i++ {
		blob = append(blob, seed+byte(i))
	}
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob) + " test@stackyard"
}

func eicRunInstance(t *testing.T, srv *Server) (id, az string) {
	t.Helper()
	res, err := srv.ec2.RunInstances("ami-12345678", "t3.micro", "", "", "", nil, 1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res.Instances[0].ID, res.Instances[0].AvailabilityZone
}

func TestEC2InstanceConnectStage1SendSSHPublicKey(t *testing.T) {
	srv, ts := newEC2InstanceConnectTestServer(t)
	id, az := eicRunInstance(t, srv)
	key := eicTestKey(1)
	send := func(body string) (int, map[string]any) {
		resp := eicRequest(t, ts, "SendSSHPublicKey", body)
		out := map[string]any{}
		_ = json.Unmarshal(mustBody(t, resp), &out)
		return resp.StatusCode, out
	}

	status, out := send(fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q,"AvailabilityZone":%q}`, id, key, az))
	if status != http.StatusOK || out["Success"] != true || out["RequestId"] == "" {
		t.Fatalf("send: %d %v", status, out)
	}

	for name, tc := range map[string]struct{ body, code string }{
		"missing instance": {fmt.Sprintf(`{"InstanceId":"i-0123456789abcdef0","InstanceOSUser":"ec2-user","SSHPublicKey":%q}`, key), "EC2InstanceNotFoundException"},
		"wrong az":         {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q,"AvailabilityZone":"eu-west-1a"}`, id, key), "EC2InstanceNotFoundException"},
		"bad instance id":  {fmt.Sprintf(`{"InstanceId":"vm-1","InstanceOSUser":"ec2-user","SSHPublicKey":%q}`, key), "InvalidArgsException"},
		"bad user":         {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"-root","SSHPublicKey":%q}`, id, key), "InvalidArgsException"},
		"missing user":     {fmt.Sprintf(`{"InstanceId":%q,"SSHPublicKey":%q}`, id, key), "InvalidArgsException"},
		"short key":        {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":"ssh-rsa AAAA"}`, id), "InvalidArgsException"},
		"unknown key type": {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q}`, id, strings.Replace(key, "ssh-ed25519 ", "ssh-dss ", 1)), "InvalidArgsException"},
		"mismatched type":  {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q}`, id, strings.Replace(key, "ssh-ed25519 ", "ssh-rsa ", 1)), "InvalidArgsException"},
		"serial port":      {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q,"SerialPort":0}`, id, key), "InvalidArgsException"},
		"unknown member":   {fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q,"Expires":"never"}`, id, key), "SerializationException"},
	} {
		if status, out := send(tc.body); status != http.StatusBadRequest || out["__type"] != tc.code {
			t.Fatalf("%s: %d %v", name, status, out)
		}
	}
	// Numeric user names are allowed when they contain a letter.
	if status, out := send(fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"1001x","SSHPublicKey":%q}`, id, key)); status != http.StatusOK {
		t.Fatalf("numeric user: %d %v", status, out)
	}

	if _, err := srv.ec2.StopInstances([]string{id}); err != nil {
		t.Fatal(err)
	}
	if status, out := send(fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q}`, id, key)); status != http.StatusBadRequest || out["__type"] != "EC2InstanceStateInvalidException" {
		t.Fatalf("stopped instance: %d %v", status, out)
	}
}

func TestEC2InstanceConnectStage1SerialConsole(t *testing.T) {
	srv, ts := newEC2InstanceConnectTestServer(t)
	id, _ := eicRunInstance(t, srv)
	body := fmt.Sprintf(`{"InstanceId":%q,"SSHPublicKey":%q,"SerialPort":0}`, id, eicTestKey(2))
	resp := eicRequest(t, ts, "SendSerialConsoleSSHPublicKey", body)
	assertStatus(t, resp, http.StatusBadRequest)
	if got := string(mustBody(t, resp)); !strings.Contains(got, "SerialConsoleAccessDisabledException") {
		t.Fatalf("disabled account: %s", got)
	}
	srv.ec2.EnableSerialConsoleAccess()
	resp = eicRequest(t, ts, "SendSerialConsoleSSHPublicKey", body)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	resp = eicRequest(t, ts, "SendSerialConsoleSSHPublicKey", strings.Replace(body, `"SerialPort":0`, `"SerialPort":1`, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestEC2InstanceConnectStage1KeysExpireAndAreBounded(t *testing.T) {
	srv, ts := newEC2InstanceConnectTestServer(t)
	id, _ := eicRunInstance(t, srv)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	srv.ec2InstanceConnect.now = func() time.Time { return now }
	resp := eicRequest(t, ts, "SendSSHPublicKey", fmt.Sprintf(`{"InstanceId":%q,"InstanceOSUser":"ec2-user","SSHPublicKey":%q}`, id, eicTestKey(3)))
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	keys := func() []any {
		resp := signedRequestWithService(t, http.MethodGet, ts.URL+eicKeysPathPrefix+id+"/keys", nil, nil, eicSigningName)
		assertStatus(t, resp, http.StatusOK)
		var out struct{ Keys []any }
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatal(err)
		}
		return out.Keys
	}
	if got := keys(); len(got) != 1 || got[0].(map[string]any)["InstanceOSUser"] != "ec2-user" {
		t.Fatalf("pushed keys: %v", got)
	}
	now = now.Add(61 * time.Second)
	if got := keys(); len(got) != 0 {
		t.Fatalf("keys after 60s: %v", got)
	}

	for i := 0; i < eicMaxKeysPerTarget; i++ {
		if err := srv.ec2InstanceConnect.push(id, eicKey{OSUser: "u"}); err != nil {
			t.Fatalf("push %d: %v", i, err.msg)
		}
	}
	if err := srv.ec2InstanceConnect.push(id, eicKey{OSUser: "u"}); err == nil || err.code != "ThrottlingException" {
		t.Fatalf("over the per-instance cap: %v", err)
	}
}
