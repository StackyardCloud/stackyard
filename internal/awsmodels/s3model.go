package awsmodels

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed s3/2006-03-01/service-2.json
var s3ModelJSON []byte

var (
	s3ModelOnce sync.Once
	s3Model     *serviceModel
)

type serviceModel struct {
	Shapes map[string]shape `json:"shapes"`
}

type shape struct {
	Type string   `json:"type"`
	Enum []string `json:"enum"`
}

func loadS3Model() *serviceModel {
	s3ModelOnce.Do(func() {
		var model serviceModel
		if err := json.Unmarshal(s3ModelJSON, &model); err != nil {
			model.Shapes = map[string]shape{}
		}
		s3Model = &model
	})
	return s3Model
}

func s3EnumSet(shapeName string) map[string]struct{} {
	model := loadS3Model()
	shapeDef, ok := model.Shapes[shapeName]
	if !ok || len(shapeDef.Enum) == 0 {
		return map[string]struct{}{}
	}
	set := make(map[string]struct{}, len(shapeDef.Enum))
	for _, value := range shapeDef.Enum {
		set[strings.ToLower(value)] = struct{}{}
	}
	return set
}

var (
	s3StorageClassOnce sync.Once
	s3StorageClasses   map[string]struct{}

	s3BucketAclOnce sync.Once
	s3BucketAcls    map[string]struct{}

	s3ObjectAclOnce sync.Once
	s3ObjectAcls    map[string]struct{}

	s3MetadataDirectiveOnce sync.Once
	s3MetadataDirectives    map[string]struct{}

	s3ServerSideEncOnce sync.Once
	s3ServerSideEnc     map[string]struct{}
)

func IsValidS3StorageClass(value string) bool {
	if value == "" {
		return true
	}
	s3StorageClassOnce.Do(func() {
		s3StorageClasses = s3EnumSet("StorageClass")
	})
	_, ok := s3StorageClasses[strings.ToLower(value)]
	return ok
}

func IsValidS3BucketCannedACL(value string) bool {
	if value == "" {
		return true
	}
	s3BucketAclOnce.Do(func() {
		s3BucketAcls = s3EnumSet("BucketCannedACL")
	})
	_, ok := s3BucketAcls[strings.ToLower(value)]
	return ok
}

func IsValidS3ObjectCannedACL(value string) bool {
	if value == "" {
		return true
	}
	s3ObjectAclOnce.Do(func() {
		s3ObjectAcls = s3EnumSet("ObjectCannedACL")
	})
	_, ok := s3ObjectAcls[strings.ToLower(value)]
	return ok
}

func IsValidS3MetadataDirective(value string) bool {
	if value == "" {
		return true
	}
	s3MetadataDirectiveOnce.Do(func() {
		s3MetadataDirectives = s3EnumSet("MetadataDirective")
	})
	_, ok := s3MetadataDirectives[strings.ToLower(value)]
	return ok
}

func IsValidS3ServerSideEncryption(value string) bool {
	if value == "" {
		return true
	}
	s3ServerSideEncOnce.Do(func() {
		s3ServerSideEnc = s3EnumSet("ServerSideEncryption")
	})
	_, ok := s3ServerSideEnc[strings.ToLower(value)]
	return ok
}
