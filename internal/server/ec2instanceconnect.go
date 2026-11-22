package server

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AWS EC2 Instance Connect: awsJson 1.1, X-Amz-Target
// AWSEC2InstanceConnectService.*, signed ec2-instance-connect. Keys pushed to
// an instance are kept for 60 seconds, as on a real instance, and can be read
// back through a Stackyard-only endpoint so tests can see what was pushed.

const (
	eicTargetPrefix     = "AWSEC2InstanceConnectService."
	eicSigningName      = "ec2-instance-connect"
	eicMaxBody          = 64 << 10
	eicKeyTTL           = 60 * time.Second
	eicMaxKeysPerTarget = 100
	eicSweepThreshold   = 1024
	eicKeysPathPrefix   = "/_stackyard/ec2instanceconnect/instances/"
)

var (
	eicInstanceIDPattern = regexp.MustCompile(`^i-[a-f0-9]+$`)
	eicAZPattern         = regexp.MustCompile(`^(\w+-){2,3}\d+\w+$`)
	// InstanceOSUser in the model is two alternatives; the second uses a
	// lookahead (Go has none), so its 2..32 length bound is checked by hand.
	eicOSUserNamePattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9@._-]{0,30}[A-Za-z0-9$_-]?$`)
	eicOSUserNumericPattern = regexp.MustCompile(`^[0-9]{1,32}[A-Za-z@._-][A-Za-z0-9@._-]*[A-Za-z0-9$_-]?$`)
	eicKeyTypes             = map[string]bool{"ssh-rsa": true, "ssh-ed25519": true, "ecdsa-sha2-nistp256": true, "ecdsa-sha2-nistp384": true, "ecdsa-sha2-nistp521": true}
)

type eicKey struct {
	OSUser        string
	PublicKey     string
	SerialConsole bool
	Expires       time.Time
}

type eicStore struct {
	mu   sync.Mutex
	now  func() time.Time
	keys map[string][]eicKey // by instance id
}

func newEC2InstanceConnectStore() *eicStore {
	return &eicStore{now: time.Now, keys: map[string][]eicKey{}}
}

// liveLocked drops expired keys for an instance and returns the rest.
func (st *eicStore) liveLocked(instanceID string) []eicKey {
	now := st.now()
	live := st.keys[instanceID][:0]
	for _, k := range st.keys[instanceID] {
		if now.Before(k.Expires) {
			live = append(live, k)
		}
	}
	if len(live) == 0 {
		delete(st.keys, instanceID)
		return nil
	}
	st.keys[instanceID] = live
	return live
}

func (st *eicStore) push(instanceID string, k eicKey) *eicError {
	st.mu.Lock()
	defer st.mu.Unlock()
	// Expired keys are otherwise only dropped when their instance is touched
	// again; sweep everything once enough instances have keys on record.
	if len(st.keys) > eicSweepThreshold {
		for id := range st.keys {
			st.liveLocked(id)
		}
	}
	live := st.liveLocked(instanceID)
	if len(live) >= eicMaxKeysPerTarget {
		return &eicError{"ThrottlingException", "too many keys are pending on this instance"}
	}
	k.Expires = st.now().Add(eicKeyTTL)
	st.keys[instanceID] = append(live, k)
	return nil
}

type eicError struct{ code, msg string }

func eicInvalid(msg string) *eicError { return &eicError{"InvalidArgsException", msg} }

type eicSendInput struct {
	InstanceId       string `json:"InstanceId"`
	InstanceOSUser   string `json:"InstanceOSUser"`
	SSHPublicKey     string `json:"SSHPublicKey"`
	AvailabilityZone string `json:"AvailabilityZone"`
	SerialPort       *int64 `json:"SerialPort"`
}

func (s *Server) handleEC2InstanceConnectJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet && strings.HasPrefix(rawRequestPath(r), eicKeysPathPrefix) {
		s.eicListKeys(w, r)
		return true
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if r.Method != http.MethodPost || !strings.HasPrefix(target, eicTargetPrefix) {
		return false
	}
	// A declared length bounds what net/http reads, so require one; r.Body is
	// never reassigned.
	if r.ContentLength < 0 || r.ContentLength > eicMaxBody {
		respondEICError(w, eicInvalid("request body must declare a Content-Length of at most 64 KiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, eicSigningName); !ok {
		respondEICStatus(w, status, code, msg)
		return true
	}
	action := strings.TrimPrefix(target, eicTargetPrefix)
	if _, ok := ec2instanceconnectOperationByName[action]; !ok {
		respondEICStatus(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+action)
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondEICError(w, eicInvalid("unable to read request body"))
		return true
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	var in eicSendInput
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		respondEICStatus(w, http.StatusBadRequest, "SerializationException", "invalid request body: "+err.Error())
		return true
	}
	if eerr := s.eicSend(action == "SendSerialConsoleSSHPublicKey", in); eerr != nil {
		respondEICError(w, eerr)
		return true
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	out, _ := json.Marshal(map[string]any{"RequestId": uuid.NewString(), "Success": true})
	_, _ = w.Write(out)
	return true
}

func (s *Server) eicSend(serial bool, in eicSendInput) *eicError {
	if in.InstanceId == "" {
		return eicInvalid("InstanceId is required")
	}
	if len(in.InstanceId) < 10 || len(in.InstanceId) > 32 || !eicInstanceIDPattern.MatchString(in.InstanceId) {
		return eicInvalid("InstanceId " + in.InstanceId + " is not a valid instance id")
	}
	if err := eicCheckPublicKey(in.SSHPublicKey); err != nil {
		return err
	}
	if serial {
		if in.InstanceOSUser != "" || in.AvailabilityZone != "" {
			return eicInvalid("SendSerialConsoleSSHPublicKey takes no InstanceOSUser or AvailabilityZone")
		}
		if in.SerialPort != nil && *in.SerialPort != 0 {
			return eicInvalid("only serial port 0 is supported")
		}
	} else {
		if in.SerialPort != nil {
			return eicInvalid("SendSSHPublicKey takes no SerialPort")
		}
		if !eicValidOSUser(in.InstanceOSUser) {
			return eicInvalid("InstanceOSUser " + in.InstanceOSUser + " is not a valid user name")
		}
		if in.AvailabilityZone != "" && (len(in.AvailabilityZone) < 6 || len(in.AvailabilityZone) > 32 || !eicAZPattern.MatchString(in.AvailabilityZone)) {
			return eicInvalid("AvailabilityZone " + in.AvailabilityZone + " is not valid")
		}
	}

	// The EC2 store has a lock of its own; read it before taking ours.
	var az, state string
	found := false
	for _, res := range s.ec2.DescribeInstances([]string{in.InstanceId}) {
		for _, inst := range res.Instances {
			if inst.ID == in.InstanceId {
				az, state, found = inst.AvailabilityZone, inst.StateName, true
			}
		}
	}
	if !found {
		return &eicError{"EC2InstanceNotFoundException", "instance " + in.InstanceId + " not found"}
	}
	if state != "running" {
		return &eicError{"EC2InstanceStateInvalidException", "instance " + in.InstanceId + " is " + state + ", not running"}
	}
	if in.AvailabilityZone != "" && in.AvailabilityZone != az {
		return &eicError{"EC2InstanceNotFoundException", "instance " + in.InstanceId + " is not in " + in.AvailabilityZone}
	}
	if serial && !s.ec2.GetSerialConsoleAccessStatus().SerialConsoleAccessEnabled {
		return &eicError{"SerialConsoleAccessDisabledException", "serial console access is disabled for this account"}
	}
	// ponytail: instance types aren't checked for serial console support
	// (Nitro), and session limits aren't modelled.
	user := in.InstanceOSUser
	if serial {
		user = ""
	}
	return s.ec2InstanceConnect.push(in.InstanceId, eicKey{OSUser: user, PublicKey: in.SSHPublicKey, SerialConsole: serial})
}

func eicValidOSUser(u string) bool {
	if len(u) < 1 || len(u) > 32 {
		return false
	}
	return eicOSUserNamePattern.MatchString(u) || (len(u) >= 2 && eicOSUserNumericPattern.MatchString(u))
}

// eicCheckPublicKey accepts OpenSSH public keys ("<type> <base64> [comment]")
// whose wire-format body names the same key type.
func eicCheckPublicKey(key string) *eicError {
	if len(key) < 80 || len(key) > 4096 {
		return eicInvalid("SSHPublicKey must be 80 to 4096 characters")
	}
	fields := strings.Fields(key)
	if len(fields) < 2 || !eicKeyTypes[fields[0]] {
		return eicInvalid("SSHPublicKey must be an OpenSSH ssh-rsa, ssh-ed25519 or ecdsa-sha2-nistp* public key")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || len(blob) < 4 {
		return eicInvalid("SSHPublicKey key data is not valid base64")
	}
	n := binary.BigEndian.Uint32(blob[:4])
	if uint64(n) > uint64(len(blob)-4) || string(blob[4:4+n]) != fields[0] {
		return eicInvalid("SSHPublicKey key data does not match its " + fields[0] + " type")
	}
	return nil
}

// eicListKeys serves GET /_stackyard/ec2instanceconnect/instances/{id}/keys:
// the unexpired keys pushed to an instance. Signed like the API.
func (s *Server) eicListKeys(w http.ResponseWriter, r *http.Request) {
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, eicSigningName); !ok {
		respondEICStatus(w, status, code, msg)
		return
	}
	id, ok := strings.CutSuffix(strings.TrimPrefix(rawRequestPath(r), eicKeysPathPrefix), "/keys")
	if !ok || !eicInstanceIDPattern.MatchString(id) {
		respondEICStatus(w, http.StatusNotFound, "ResourceNotFoundException", "no such route")
		return
	}
	st := s.ec2InstanceConnect
	st.mu.Lock()
	keys := []map[string]any{}
	for _, k := range st.liveLocked(id) {
		keys = append(keys, map[string]any{"InstanceOSUser": k.OSUser, "SSHPublicKey": k.PublicKey, "SerialConsole": k.SerialConsole, "ExpiresAt": k.Expires.UTC().Format(time.RFC3339)})
	}
	out, _ := json.Marshal(map[string]any{"InstanceId": id, "Keys": keys})
	st.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func respondEICError(w http.ResponseWriter, e *eicError) {
	respondEICStatus(w, http.StatusBadRequest, e.code, e.msg)
}

func respondEICStatus(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	out, _ := json.Marshal(map[string]string{"__type": code, "Message": msg})
	_, _ = w.Write(out)
}
