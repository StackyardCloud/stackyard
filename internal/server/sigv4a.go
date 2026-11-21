package server

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
)

// SigV4A (AWS4-ECDSA-P256-SHA256) is the multi-region signature some AWS
// services require, for example CloudFront KeyValueStore. The client derives
// a P-256 key pair from its secret key and signs with ECDSA, so the server
// derives the same public key and verifies the signature.
const sigV4AAlgorithm = "AWS4-ECDSA-P256-SHA256"

func parseSigV4AAuthorization(value string) (sigV4Request, error) {
	value = strings.TrimSpace(value)
	rest, ok := strings.CutPrefix(value, sigV4AAlgorithm+" ")
	if !ok {
		return sigV4Request{}, errors.New("unsupported authorization scheme")
	}
	fields := map[string]string{}
	for _, part := range strings.Split(rest, ",") {
		if k, v, found := strings.Cut(strings.TrimSpace(part), "="); found {
			fields[k] = strings.TrimSpace(v)
		}
	}
	credential, signedHeaders, signature := fields["Credential"], fields["SignedHeaders"], fields["Signature"]
	if credential == "" || signedHeaders == "" || signature == "" {
		return sigV4Request{}, errors.New("incomplete SigV4A authorization")
	}
	// SigV4A scope has no region: AccessKey/Date/Service/aws4_request.
	parts := strings.Split(credential, "/")
	if len(parts) != 4 || parts[3] != "aws4_request" {
		return sigV4Request{}, errors.New("invalid credential scope")
	}
	return sigV4Request{
		AccessKey:     parts[0],
		ScopeDate:     parts[1],
		Service:       parts[2],
		SignedHeaders: signedHeaders,
		Signature:     signature,
	}, nil
}

// verifySigV4A checks req.Signature against the canonical request and the
// alternate URI encodings the HMAC path also accepts.
func (s *Server) verifySigV4A(r *http.Request, req sigV4Request, payloadHash, canonicalRequest string) bool {
	sig, err := hex.DecodeString(req.Signature)
	if err != nil {
		return false
	}
	pub, err := sigV4APublicKey(req.AccessKey, s.secretKey)
	if err != nil {
		return false
	}
	verify := func(canonical string) bool {
		digest := sha256.Sum256([]byte(sigV4AStringToSign(req.AmzDate, req.Service, canonical)))
		return ecdsa.VerifyASN1(pub, digest[:], sig)
	}
	if verify(canonicalRequest) {
		return true
	}
	for _, uri := range alternateCanonicalURIs(r.URL, rawRequestPath(r), buildCanonicalURI(r.URL)) {
		if candidate, err := buildCanonicalRequestForURI(r, uri, req.SignedHeaders, payloadHash, false); err == nil && verify(candidate) {
			return true
		}
	}
	return false
}

func sigV4AStringToSign(amzDate, service, canonicalRequest string) string {
	date := amzDate
	if len(date) >= 8 {
		date = date[:8]
	}
	hashed := sha256.Sum256([]byte(canonicalRequest))
	return strings.Join([]string{sigV4AAlgorithm, amzDate, date + "/" + service + "/aws4_request", hex.EncodeToString(hashed[:])}, "\n")
}

var sigV4AKeyCache sync.Map // accessKey+"\x00"+secretKey -> *ecdsa.PublicKey

// sigV4APublicKey derives the SigV4A key pair (FIPS 186-4 B.4.2 with an
// SP 800-108 HMAC-SHA256 counter KDF), matching aws-sdk-go-v2 internal/v4a.
func sigV4APublicKey(accessKey, secretKey string) (*ecdsa.PublicKey, error) {
	cacheKey := accessKey + "\x00" + secretKey
	if cached, ok := sigV4AKeyCache.Load(cacheKey); ok {
		return cached.(*ecdsa.PublicKey), nil
	}
	curve := elliptic.P256()
	nMinusTwo := new(big.Int).Sub(curve.Params().N, big.NewInt(2))
	inputKey := []byte("AWS4A" + secretKey)
	for counter := 1; counter <= 0xFF; counter++ {
		context := append([]byte(accessKey), byte(counter))
		candidate := sigV4AKDF(inputKey, context)
		d := new(big.Int).SetBytes(candidate)
		if d.Cmp(nMinusTwo) >= 0 {
			continue
		}
		d.Add(d, big.NewInt(1))
		priv, err := ecdsa.ParseRawPrivateKey(curve, d.FillBytes(make([]byte, 32)))
		if err != nil {
			return nil, err
		}
		sigV4AKeyCache.Store(cacheKey, &priv.PublicKey)
		return &priv.PublicKey, nil
	}
	return nil, errors.New("exhausted SigV4A key derivation counter")
}

// sigV4AKDF returns 256 bits from HMAC(key, i || label || 0x00 || context || L).
// One HMAC-SHA256 block is exactly 256 bits, so i is always 1.
func sigV4AKDF(key, context []byte) []byte {
	var fixed bytes.Buffer
	_ = binary.Write(&fixed, binary.BigEndian, int32(1))
	fixed.WriteString(sigV4AAlgorithm)
	fixed.WriteByte(0x00)
	fixed.Write(context)
	_ = binary.Write(&fixed, binary.BigEndian, int32(256))
	mac := hmac.New(sha256.New, key)
	mac.Write(fixed.Bytes())
	return mac.Sum(nil)
}
