package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func csSetupDomain(t *testing.T, ts *httptest.Server, name string) {
	t.Helper()
	csOK(t, ts, "CreateDomain", map[string]string{"DomainName": name})
	for _, f := range []map[string]string{
		{"IndexField.IndexFieldName": "title", "IndexField.IndexFieldType": "text"},
		{"IndexField.IndexFieldName": "genre", "IndexField.IndexFieldType": "literal", "IndexField.LiteralOptions.FacetEnabled": "true"},
		{"IndexField.IndexFieldName": "year", "IndexField.IndexFieldType": "int"},
		{"IndexField.IndexFieldName": "tag_*", "IndexField.IndexFieldType": "literal-array"},
	} {
		f["DomainName"] = name
		csOK(t, ts, "DefineIndexField", f)
	}
	csOK(t, ts, "DefineSuggester", map[string]string{"DomainName": name, "Suggester.SuggesterName": "titles",
		"Suggester.DocumentSuggesterOptions.SourceField": "title", "Suggester.DocumentSuggesterOptions.FuzzyMatching": "low"})
	csOK(t, ts, "IndexDocuments", map[string]string{"DomainName": name})
}

func TestCloudSearchStage1Configuration(t *testing.T) {
	ts := newCSTestServer(t)
	csOK(t, ts, "CreateDomain", map[string]string{"DomainName": "movies"})
	if code := csErrCode(t, ts, "CreateDomain", map[string]string{"DomainName": "movies"}, http.StatusConflict); code != "ResourceAlreadyExists" {
		t.Fatalf("duplicate domain: %s", code)
	}
	if code := csErrCode(t, ts, "CreateDomain", map[string]string{"DomainName": "Bad_Name"}, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad name: %s", code)
	}

	// Defining a field needs re-indexing; options must match the type.
	body := csOK(t, ts, "DefineIndexField", map[string]string{"DomainName": "movies",
		"IndexField.IndexFieldName": "year", "IndexField.IndexFieldType": "int", "IndexField.IntOptions.DefaultValue": "1900"})
	if !strings.Contains(body, "<State>RequiresIndexDocuments</State>") || !strings.Contains(body, "<DefaultValue>1900</DefaultValue>") {
		t.Fatalf("DefineIndexField: %s", body)
	}
	if code := csErrCode(t, ts, "DefineIndexField", map[string]string{"DomainName": "movies",
		"IndexField.IndexFieldName": "x", "IndexField.IndexFieldType": "int", "IndexField.TextOptions.ReturnEnabled": "true"}, http.StatusConflict); code != "InvalidType" {
		t.Fatalf("mismatched options: %s", code)
	}
	if code := csErrCode(t, ts, "DefineIndexField", map[string]string{"DomainName": "movies",
		"IndexField.IndexFieldName": "body", "IndexField.IndexFieldType": "text", "IndexField.TextOptions.AnalysisScheme": "nope"}, http.StatusConflict); code != "InvalidType" {
		t.Fatalf("unknown analysis scheme: %s", code)
	}
	if !strings.Contains(csOK(t, ts, "DescribeDomains", map[string]string{"DomainNames.member.1": "movies"}), "<RequiresIndexDocuments>true</RequiresIndexDocuments>") {
		t.Fatal("domain should require indexing")
	}
	// Nothing is deployed until IndexDocuments.
	if body := csOK(t, ts, "DescribeIndexFields", map[string]string{"DomainName": "movies", "Deployed": "true"}); strings.Contains(body, "year") {
		t.Fatalf("deployed before indexing: %s", body)
	}
	if body := csOK(t, ts, "IndexDocuments", map[string]string{"DomainName": "movies"}); !strings.Contains(body, "<member>year</member>") {
		t.Fatalf("IndexDocuments: %s", body)
	}
	if body := csOK(t, ts, "DescribeIndexFields", map[string]string{"DomainName": "movies", "Deployed": "true"}); !strings.Contains(body, "<State>Active</State>") {
		t.Fatalf("deployed field: %s", body)
	}

	// Delete is pending until the next index, then gone.
	if body := csOK(t, ts, "DeleteIndexField", map[string]string{"DomainName": "movies", "IndexFieldName": "year"}); !strings.Contains(body, "<PendingDeletion>true</PendingDeletion>") {
		t.Fatalf("DeleteIndexField: %s", body)
	}
	csOK(t, ts, "IndexDocuments", map[string]string{"DomainName": "movies"})
	if code := csErrCode(t, ts, "DeleteIndexField", map[string]string{"DomainName": "movies", "IndexFieldName": "year"}, http.StatusConflict); code != "ResourceNotFound" {
		t.Fatalf("delete twice: %s", code)
	}

	// Suggesters need a text or literal source field.
	if code := csErrCode(t, ts, "DefineSuggester", map[string]string{"DomainName": "movies", "Suggester.SuggesterName": "s",
		"Suggester.DocumentSuggesterOptions.SourceField": "missing"}, http.StatusConflict); code != "InvalidType" {
		t.Fatalf("suggester source: %s", code)
	}

	// Expressions, analysis schemes and immediate options round-trip.
	if code := csErrCode(t, ts, "DefineExpression", map[string]string{"DomainName": "movies",
		"Expression.ExpressionName": "bad", "Expression.ExpressionValue": "(year"}, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad expression: %s", code)
	}
	csOK(t, ts, "DefineAnalysisScheme", map[string]string{"DomainName": "movies", "AnalysisScheme.AnalysisSchemeName": "my_en",
		"AnalysisScheme.AnalysisSchemeLanguage": "en", "AnalysisScheme.AnalysisOptions.Stopwords": `["the"]`})
	if body := csOK(t, ts, "DescribeAnalysisSchemes", map[string]string{"DomainName": "movies"}); !strings.Contains(body, "<AnalysisSchemeName>my_en</AnalysisSchemeName>") {
		t.Fatalf("DescribeAnalysisSchemes: %s", body)
	}
	csOK(t, ts, "UpdateAvailabilityOptions", map[string]string{"DomainName": "movies", "MultiAZ": "true"})
	if body := csOK(t, ts, "DescribeAvailabilityOptions", map[string]string{"DomainName": "movies"}); !strings.Contains(body, "<Options>true</Options>") {
		t.Fatalf("availability: %s", body)
	}
	if code := csErrCode(t, ts, "UpdateScalingParameters", map[string]string{"DomainName": "movies",
		"ScalingParameters.DesiredPartitionCount": "2", "ScalingParameters.DesiredInstanceType": "search.small"}, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("partitions on small: %s", code)
	}
	if code := csErrCode(t, ts, "UpdateServiceAccessPolicies", map[string]string{"DomainName": "movies", "AccessPolicies": "{not json"}, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad policy: %s", code)
	}
	if body := csOK(t, ts, "ListDomainNames", nil); !strings.Contains(body, "<entry><key>movies</key><value>2013-01-01</value></entry>") {
		t.Fatalf("ListDomainNames: %s", body)
	}
	csOK(t, ts, "DeleteDomain", map[string]string{"DomainName": "movies"})
	if code := csErrCode(t, ts, "DescribeIndexFields", map[string]string{"DomainName": "movies"}, http.StatusConflict); code != "ResourceNotFound" {
		t.Fatalf("after delete: %s", code)
	}
}
