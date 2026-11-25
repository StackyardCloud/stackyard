package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func TestGeoPlacesStage0CatalogMatchesModel(t *testing.T) {
	geoCatalogMatchesModel(t, awsmodels.GeoPlacesModel(), geoplacesOperationByName, gplcService.handlers)
}

func TestGeoPlacesStage0Routing(t *testing.T) {
	ts := newGeoTestServer(t)
	geoReject(t, ts, "geo-places", http.MethodPut, "/v2/geocode", "{}", "UnknownOperation")
	// Location's place-index search (signing name geo) keeps its traffic.
	status, _, data := geoCall(t, ts, "geo", http.MethodPost, "/places/v0/indexes/idx/search/text", `{"Text":"x"}`)
	if status != http.StatusOK || strings.Contains(string(data), "PricingBucket") || strings.Contains(string(data), "ResultItems") {
		t.Fatalf("location SearchPlaceIndexForText: %d %s", status, data)
	}
	// Routes and Maps signing names don't reach Places.
	geoReject(t, ts, "geo-routes", http.MethodPost, "/v2/geocode", `{"QueryText":"Seattle"}`, "UnknownOperation")
}

func gplcTitles(out map[string]any) []string {
	var titles []string
	for _, raw := range out["ResultItems"].([]any) {
		item := raw.(map[string]any)
		title, _ := item["Title"].(string)
		titles = append(titles, title)
	}
	return titles
}

func TestGeoPlacesStage1Search(t *testing.T) {
	ts := newGeoTestServer(t)
	md := awsmodels.GeoPlacesModel()
	call := func(op, path, body string) map[string]any {
		t.Helper()
		return geoJSON(t, ts, md, "geo-places", op, http.MethodPost, path, body)
	}

	out := call("Geocode", "/v2/geocode", `{"QueryText":"space needle","AdditionalFeatures":["TimeZone"]}`)
	items := out["ResultItems"].([]any)
	if len(items) != 1 {
		t.Fatalf("geocode: %v", out)
	}
	needle := items[0].(map[string]any)
	if needle["PlaceType"] != "PointOfInterest" || needle["TimeZone"].(map[string]any)["Name"] != "America/Los_Angeles" {
		t.Fatalf("geocode item: %v", needle)
	}
	out = call("Geocode", "/v2/geocode", `{"QueryComponents":{"Country":"FRA","Locality":"Paris"},"Filter":{"IncludePlaceTypes":["PointOfInterest"]}}`)
	if got := gplcTitles(out); len(got) != 2 {
		t.Fatalf("geocode components: %v", got)
	}

	place := geoJSON(t, ts, md, "geo-places", "GetPlace", http.MethodGet, "/v2/place/"+needle["PlaceId"].(string)+"?additional-features=Contact", "")
	if place["Title"] != "Space Needle" || place["Contacts"] == nil {
		t.Fatalf("get place: %v", place)
	}

	out = call("ReverseGeocode", "/v2/reverse-geocode", `{"QueryPosition":[2.2945,48.8583],"QueryRadius":500}`)
	if got := gplcTitles(out); len(got) != 1 || got[0] != "Eiffel Tower" {
		t.Fatalf("reverse geocode: %v", got)
	}
	out = call("ReverseGeocode", "/v2/reverse-geocode", `{"QueryPosition":[0,0],"QueryRadius":1000}`)
	if len(out["ResultItems"].([]any)) != 0 {
		t.Fatalf("reverse geocode in the ocean: %v", out)
	}

	out = call("SearchNearby", "/v2/search-nearby", `{"QueryPosition":[-122.34,47.61],"QueryRadius":5000,"MaxResults":1}`)
	if got := gplcTitles(out); len(got) != 1 || got[0] != "Pike Place Market" || out["NextToken"] == nil {
		t.Fatalf("nearby page 1: %v", out)
	}
	out = call("SearchNearby", "/v2/search-nearby", `{"QueryPosition":[-122.34,47.61],"QueryRadius":5000,"MaxResults":1,"NextToken":"`+out["NextToken"].(string)+`"}`)
	if got := gplcTitles(out); len(got) != 1 || got[0] != "Space Needle" || out["NextToken"] != nil {
		t.Fatalf("nearby page 2: %v", out)
	}
	out = call("SearchNearby", "/v2/search-nearby", `{"QueryPosition":[-122.34,47.61],"QueryRadius":5000,"Filter":{"ExcludeCategories":["market"]}}`)
	if got := gplcTitles(out); len(got) != 1 || got[0] != "Space Needle" {
		t.Fatalf("nearby exclude: %v", got)
	}

	out = call("SearchText", "/v2/search-text", `{"QueryText":"tower","BiasPosition":[2.35,48.85]}`)
	if got := gplcTitles(out); len(got) != 4 || got[0] != "Eiffel Tower" {
		t.Fatalf("search text: %v", got)
	}

	out = call("Suggest", "/v2/suggest", `{"QueryText":"museum","BiasPosition":[2.35,48.85]}`)
	items = out["ResultItems"].([]any)
	query := items[0].(map[string]any)
	if query["SuggestResultItemType"] != "Query" || len(items) != 2 {
		t.Fatalf("suggest: %v", out)
	}
	qid := query["Query"].(map[string]any)["QueryId"].(string)
	out = call("SearchText", "/v2/search-text", `{"QueryId":"`+qid+`","BiasPosition":[2.35,48.85]}`)
	if got := gplcTitles(out); len(got) != 1 || got[0] != "Musée du Louvre" {
		t.Fatalf("search by query id: %v", got)
	}

	out = call("Autocomplete", "/v2/autocomplete", `{"QueryText":"van","Filter":{"IncludeCountries":["CA"]}}`)
	items = out["ResultItems"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["Highlights"] == nil {
		t.Fatalf("autocomplete: %v", out)
	}
}

func TestGeoPlacesStage1Validation(t *testing.T) {
	ts := newGeoTestServer(t)
	cases := []struct{ name, path, body, reason string }{
		{"missing query", "/v2/autocomplete", `{}`, "Missing"},
		{"unknown member", "/v2/autocomplete", `{"QueryText":"x","Bogus":1}`, "UnknownField"},
		{"not json", "/v2/autocomplete", `{`, "CannotParse"},
		{"too many results", "/v2/autocomplete", `{"QueryText":"x","MaxResults":21}`, "FieldValidationFailed"},
		{"bias and box", "/v2/autocomplete", `{"QueryText":"x","BiasPosition":[0,0],"Filter":{"BoundingBox":[0,0,1,1]}}`, "FieldValidationFailed"},
		{"geocode needs a query", "/v2/geocode", `{"MaxResults":3}`, "Missing"},
		{"latitude range", "/v2/reverse-geocode", `{"QueryPosition":[0,91]}`, "FieldValidationFailed"},
		{"longitude range", "/v2/search-nearby", `{"QueryPosition":[181,0]}`, "FieldValidationFailed"},
		{"bad token", "/v2/search-nearby", `{"QueryPosition":[0,0],"NextToken":"bm9wZQ"}`, "FieldValidationFailed"},
		{"text and id", "/v2/search-text", `{"QueryText":"x","QueryId":"y","BiasPosition":[0,0]}`, "FieldValidationFailed"},
		{"search text needs bias", "/v2/search-text", `{"QueryText":"x"}`, "Missing"},
		{"bad query id", "/v2/search-text", `{"QueryId":"nope","BiasPosition":[0,0]}`, "FieldValidationFailed"},
		{"south above north", "/v2/suggest", `{"QueryText":"x","Filter":{"BoundingBox":[0,10,1,5]}}`, "FieldValidationFailed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			geoReject(t, ts, "geo-places", http.MethodPost, tc.path, tc.body, tc.reason)
		})
	}
	geoReject(t, ts, "geo-places", http.MethodGet, "/v2/place/stackyard-place-nowhere", "", "FieldValidationFailed")
	// Unsigned requests to a Places host without an API key are refused;
	// with one they are served.
	for key, want := range map[string]int{"": http.StatusForbidden, "?key=v1.public.test": http.StatusOK} {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v2/geocode"+key, strings.NewReader(`{"QueryText":"Berlin"}`))
		req.Host = "places.geo.eu-central-1.amazonaws.com"
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("unsigned geocode %q: %d, want %d", key, resp.StatusCode, want)
		}
	}
	if status, _, _ := geoCall(t, ts, "geo-places", http.MethodPost, "/v2/geocode", `{"QueryText":""}`); status != http.StatusBadRequest {
		t.Fatalf("empty query text: %d", status)
	}
}
