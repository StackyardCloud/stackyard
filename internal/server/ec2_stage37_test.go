package server

import (
	"reflect"
	"sort"
	"testing"

	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func TestEC2Stage37OperationCatalogMatchesSDK(t *testing.T) {
	sdkOperationByName := map[string]struct{}{}
	clientType := reflect.TypeOf(&awsec2.Client{})
	for idx := 0; idx < clientType.NumMethod(); idx++ {
		name := clientType.Method(idx).Name
		if name == "Options" {
			// Options is a helper method, not an EC2 API operation.
			continue
		}
		sdkOperationByName[name] = struct{}{}
	}

	invalidInCatalog := make([]string, 0)
	for _, operation := range ec2Operations {
		if _, ok := sdkOperationByName[operation.Name]; !ok {
			invalidInCatalog = append(invalidInCatalog, operation.Name)
		}
	}
	sort.Strings(invalidInCatalog)

	missingFromCatalog := make([]string, 0)
	for operationName := range sdkOperationByName {
		if _, ok := ec2OperationByName[operationName]; !ok {
			missingFromCatalog = append(missingFromCatalog, operationName)
		}
	}
	sort.Strings(missingFromCatalog)

	if len(invalidInCatalog) > 0 || len(missingFromCatalog) > 0 {
		t.Fatalf(
			"EC2 operation catalog mismatch with SDK\\ninvalid_in_catalog=%v\\nmissing_from_catalog=%v",
			invalidInCatalog,
			missingFromCatalog,
		)
	}
}
