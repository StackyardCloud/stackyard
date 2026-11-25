package server

type cloudSearchDomainOperation struct {
	Name        string
	Method, URI string
}

// Amazon CloudSearch Domain operations sourced from:
// https://docs.aws.amazon.com/cloudsearch/latest/developerguide/search-api.html and
// https://docs.aws.amazon.com/cloudsearch/latest/developerguide/documents-batch-resource.html
var cloudSearchDomainOperations = []cloudSearchDomainOperation{
	{Name: "Search", Method: "GET", URI: "/2013-01-01/search?format=sdk&pretty=true"},
	{Name: "Suggest", Method: "GET", URI: "/2013-01-01/suggest?format=sdk&pretty=true"},
	{Name: "UploadDocuments", Method: "POST", URI: "/2013-01-01/documents/batch?format=sdk"},
}

var cloudSearchDomainOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range cloudSearchDomainOperations {
		out[op.Name] = true
	}
	return out
}()
