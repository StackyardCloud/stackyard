package server

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Vector from aws-sdk-go-v2 internal/v4a TestDeriveECDSAKeyPairFromSecret.
func TestSigV4APublicKeyDerivationMatchesSDKVector(t *testing.T) {
	pub, err := sigV4APublicKey("AKISORANDOMAASORANDOM", "q+jcrXGc+0zWN6uzclKVhvMmUsIfRPa4rlRandom")
	if err != nil {
		t.Fatal(err)
	}
	wantX, _ := new(big.Int).SetString("15D242CEEBF8D8169FD6A8B5A746C41140414C3B07579038DA06AF89190FFFCB", 16)
	wantY, _ := new(big.Int).SetString("515242CEDD82E94799482E4C0514B505AFCCF2C0C98D6A553BF539F424C5EC0", 16)
	if pub.X.Cmp(wantX) != 0 || pub.Y.Cmp(wantY) != 0 {
		t.Fatalf("derived key mismatch: X=%X Y=%X", pub.X, pub.Y)
	}
}

// sigV4ATestSign signs req the way the SDK/CRT SigV4A signer does, using the
// private key re-derived for the test credentials.
func sigV4ATestSign(t *testing.T, req *http.Request, body []byte, service string) {
	t.Helper()
	priv := sigV4ATestPrivateKey(t, testAccessKey, testSecretKey)
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Region-Set", "*")
	payloadHash := sha256Hex(body)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	signed := "host;x-amz-content-sha256;x-amz-date;x-amz-region-set"
	canonical, err := buildCanonicalRequest(req, signed, payloadHash, false)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(sigV4AStringToSign(amzDate, service, canonical)))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s Credential=%s/%s/%s/aws4_request, SignedHeaders=%s, Signature=%s",
		sigV4AAlgorithm, testAccessKey, amzDate[:8], service, signed, hex.EncodeToString(sig)))
}

func sigV4ATestPrivateKey(t *testing.T, accessKey, secretKey string) *ecdsa.PrivateKey {
	t.Helper()
	pub, err := sigV4APublicKey(accessKey, secretKey)
	if err != nil {
		t.Fatal(err)
	}
	// Re-run the derivation to recover d; the server only keeps the public key.
	inputKey := []byte("AWS4A" + secretKey)
	nMinusTwo := new(big.Int).Sub(pub.Curve.Params().N, big.NewInt(2))
	for counter := 1; counter <= 0xFF; counter++ {
		d := new(big.Int).SetBytes(sigV4AKDF(inputKey, append([]byte(accessKey), byte(counter))))
		if d.Cmp(nMinusTwo) < 0 {
			priv, err := ecdsa.ParseRawPrivateKey(pub.Curve, d.Add(d, big.NewInt(1)).FillBytes(make([]byte, 32)))
			if err != nil {
				t.Fatal(err)
			}
			return priv
		}
	}
	t.Fatal("no key")
	return nil
}

func TestSigV4AValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	newReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "http://123456789012.localhost/key-value-stores/arn%3Aaws%3Acloudfront%3A%3A123456789012%3Akey-value-store%2Fx", nil)
		r.RequestURI = ""
		return r
	}
	r := newReq()
	sigV4ATestSign(t, r, nil, "cloudfront-keyvaluestore")
	if ok, status, code, msg, _ := srv.validateSigV4WithServiceCore(r, "cloudfront-keyvaluestore"); !ok {
		t.Fatalf("valid SigV4A rejected: %d %s %s", status, code, msg)
	}
	if got := sigV4ServiceHint(r); got != "cloudfront-keyvaluestore" {
		t.Fatalf("service hint = %q", got)
	}
	if ok, status, _, _, _ := srv.validateSigV4WithServiceCore(r, "cloudfront"); ok || status != http.StatusForbidden {
		t.Fatalf("wrong service accepted: ok=%v status=%d", ok, status)
	}

	tampered := newReq()
	sigV4ATestSign(t, tampered, nil, "cloudfront-keyvaluestore")
	tampered.URL.Path += "y"
	tampered.URL.RawPath = ""
	if ok, status, _, _, _ := srv.validateSigV4WithServiceCore(tampered, "cloudfront-keyvaluestore"); ok || status != http.StatusForbidden {
		t.Fatalf("tampered request accepted: ok=%v status=%d", ok, status)
	}

	wrongKey := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: "other-secret", LogLevel: "error"})
	r2 := newReq()
	sigV4ATestSign(t, r2, nil, "cloudfront-keyvaluestore")
	if ok, _, _, _, _ := wrongKey.validateSigV4WithServiceCore(r2, "cloudfront-keyvaluestore"); ok {
		t.Fatal("signature from a different secret accepted")
	}
	if _, err := parseSigV4AAuthorization(strings.Replace(r2.Header.Get("Authorization"), "/aws4_request", "/us-east-1/aws4_request", 1)); err == nil {
		t.Fatal("regional scope accepted as SigV4A")
	}
}

func TestRawRequestPathAbsoluteForm(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://a.example:80/x/y%2Fz?q=1", nil)
	r.RequestURI = "http://a.example:80/x/y%2Fz?q=1"
	if got := rawRequestPath(r); got != "/x/y%2Fz" {
		t.Fatalf("rawRequestPath = %q", got)
	}
	r.RequestURI = "http://a.example"
	if got := rawRequestPath(r); got != "/" {
		t.Fatalf("rawRequestPath = %q", got)
	}
}
