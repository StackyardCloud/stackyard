package s3

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"
)

func etagFor(body []byte) string {
	sum := md5.Sum(body)
	return hex.EncodeToString(sum[:])
}

func multipartETag(partETags []string) string {
	if len(partETags) == 0 {
		return etagFor(nil) + "-0"
	}
	var combined []byte
	for _, et := range partETags {
		et = strings.Trim(et, "\"")
		bytes, _ := hex.DecodeString(et)
		combined = append(combined, bytes...)
	}
	sum := md5.Sum(combined)
	return hex.EncodeToString(sum[:]) + "-" + strconv.Itoa(len(partETags))
}
