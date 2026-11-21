package server

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CloudFront control-plane KeyValueStore operations. The stores they create
// are the ones the cloudfront-keyvaluestore data plane reads and writes.

const cloudFrontXMLNS = "http://cloudfront.amazonaws.com/doc/2020-05-31/"

var cloudFrontKVSOperations = map[string]bool{
	"CreateKeyValueStore": true, "DescribeKeyValueStore": true, "ListKeyValueStores": true,
	"UpdateKeyValueStore": true, "DeleteKeyValueStore": true,
}

type cloudFrontKVSXML struct {
	XMLName          xml.Name `xml:"KeyValueStore"`
	Xmlns            string   `xml:"xmlns,attr,omitempty"`
	Name             string   `xml:"Name"`
	ID               string   `xml:"Id"`
	Comment          string   `xml:"Comment"`
	ARN              string   `xml:"ARN"`
	Status           string   `xml:"Status"`
	LastModifiedTime string   `xml:"LastModifiedTime"`
}

type cloudFrontKVSListXML struct {
	XMLName    xml.Name           `xml:"KeyValueStoreList"`
	Xmlns      string             `xml:"xmlns,attr"`
	NextMarker string             `xml:"NextMarker,omitempty"`
	MaxItems   int                `xml:"MaxItems"`
	Quantity   int                `xml:"Quantity"`
	Items      []cloudFrontKVSXML `xml:"Items>KeyValueStore"`
}

// cloudFrontKVSRequestBody is the XML body of Create/UpdateKeyValueStore.
type cloudFrontKVSRequestBody struct {
	Name         string    `xml:"Name"`
	Comment      *string   `xml:"Comment"`
	ImportSource *struct{} `xml:"ImportSource"`
}

// cloudFrontKVSRoute maps the REST routes of the KeyValueStore operations.
func cloudFrontKVSRoute(method, path string) (action, name string) {
	path = strings.TrimSuffix(path, "/")
	if path == "/2020-05-31/key-value-store" {
		switch method {
		case http.MethodGet:
			return "ListKeyValueStores", ""
		case http.MethodPost:
			return "CreateKeyValueStore", ""
		}
		return "", ""
	}
	rest, ok := strings.CutPrefix(path, "/2020-05-31/key-value-store/")
	if !ok || rest == "" || strings.Contains(rest, "/") {
		return "", ""
	}
	name, err := url.PathUnescape(rest)
	if err != nil {
		return "", ""
	}
	switch method {
	case http.MethodGet:
		return "DescribeKeyValueStore", name
	case http.MethodPut:
		return "UpdateKeyValueStore", name
	case http.MethodDelete:
		return "DeleteKeyValueStore", name
	}
	return "", ""
}

func (s *Server) handleCloudFrontKVSOperation(w http.ResponseWriter, r *http.Request, action string, payload map[string]any) {
	_, pathName := cloudFrontKVSRoute(r.Method, strings.SplitN(rawRequestPath(r), "?", 2)[0])
	var body cloudFrontKVSRequestBody
	if raw, ok := payload["RawBody"].(string); ok && strings.TrimSpace(raw) != "" {
		if err := xml.Unmarshal([]byte(raw), &body); err != nil {
			respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", "invalid XML request body")
			return
		}
	}
	// The JSON form (X-Stackyard-Operation) carries every member in the body.
	field := func(key string) string { return strings.TrimSpace(cloudFrontPayloadString(payload, key, "")) }
	name := firstNonEmpty(pathName, body.Name, field("Name"))
	comment := field("Comment")
	if body.Comment != nil {
		comment = *body.Comment
	}
	ifMatch := firstNonEmpty(strings.TrimSpace(r.Header.Get("If-Match")), field("IfMatch"))

	switch action {
	case "CreateKeyValueStore":
		if body.ImportSource != nil || payload["ImportSource"] != nil {
			respondCloudFrontError(w, http.StatusBadRequest, "UnsupportedOperation", "ImportSource is not supported by Stackyard")
			return
		}
		kvs, err := s.cloudfrontKVS.Create(name, comment)
		if err != nil {
			respondCloudFrontKVSErr(w, err)
			return
		}
		w.Header().Set("Location", "https://cloudfront.amazonaws.com/2020-05-31/key-value-store/"+kvs.Name)
		respondCloudFrontKVS(w, http.StatusCreated, kvs)
	case "DescribeKeyValueStore":
		kvs, err := s.cloudfrontKVS.Describe(name)
		if err != nil {
			respondCloudFrontKVSErr(w, err)
			return
		}
		respondCloudFrontKVS(w, http.StatusOK, kvs)
	case "UpdateKeyValueStore":
		if body.Comment == nil && payload["Comment"] == nil {
			respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", "Comment is required")
			return
		}
		kvs, err := s.cloudfrontKVS.Update(name, comment, ifMatch)
		if err != nil {
			respondCloudFrontKVSErr(w, err)
			return
		}
		respondCloudFrontKVS(w, http.StatusOK, kvs)
	case "DeleteKeyValueStore":
		if err := s.cloudfrontKVS.Delete(name, ifMatch); err != nil {
			respondCloudFrontKVSErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "ListKeyValueStores":
		query := r.URL.Query()
		maxItems := 100
		if raw := firstNonEmpty(query.Get("MaxItems"), field("MaxItems")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 {
				respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", "MaxItems must be between 1 and 100")
				return
			}
			maxItems = n
		}
		stores, next := s.cloudfrontKVS.List(firstNonEmpty(query.Get("Marker"), field("Marker")), maxItems, firstNonEmpty(query.Get("Status"), field("Status")))
		list := cloudFrontKVSListXML{Xmlns: cloudFrontXMLNS, NextMarker: next, MaxItems: maxItems, Quantity: len(stores)}
		for _, kvs := range stores {
			list.Items = append(list.Items, cloudFrontKVSToXML(kvs, ""))
		}
		respondCloudFrontXMLBody(w, http.StatusOK, list)
	}
}

func cloudFrontKVSToXML(kvs cfKVStore, xmlns string) cloudFrontKVSXML {
	return cloudFrontKVSXML{
		Xmlns: xmlns, Name: kvs.Name, ID: kvs.ID, Comment: kvs.Comment, ARN: kvs.ARN,
		Status: kvs.Status, LastModifiedTime: kvs.LastModified.Format(time.RFC3339),
	}
}

func respondCloudFrontKVS(w http.ResponseWriter, status int, kvs cfKVStore) {
	w.Header().Set("ETag", kvs.StoreETag)
	respondCloudFrontXMLBody(w, status, cloudFrontKVSToXML(kvs, cloudFrontXMLNS))
}

func respondCloudFrontXMLBody(w http.ResponseWriter, status int, body any) {
	out, err := xml.Marshal(body)
	if err != nil {
		respondCloudFrontError(w, http.StatusInternalServerError, "InternalError", "unable to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
}

func respondCloudFrontKVSErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCFKVSNotFound):
		respondCloudFrontError(w, http.StatusNotFound, "EntityNotFound", err.Error())
	case errors.Is(err, errCFKVSExists):
		respondCloudFrontError(w, http.StatusConflict, "EntityAlreadyExists", err.Error())
	case errors.Is(err, errCFKVSETagMismatch):
		respondCloudFrontError(w, http.StatusPreconditionFailed, "PreconditionFailed", err.Error())
	case errors.Is(err, errCFKVSMissingIfMatch):
		respondCloudFrontError(w, http.StatusBadRequest, "InvalidIfMatchVersion", err.Error())
	default:
		respondCloudFrontError(w, http.StatusBadRequest, "InvalidArgument", err.Error())
	}
}
