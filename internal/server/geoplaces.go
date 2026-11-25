package server

// Amazon Location Service Places V2 (geo-places): search, geocoding and
// place lookup answered from the built-in gazetteer (geoplaces_gazetteer.go).
// Text matching is token-prefix based; ordering is by great-circle distance
// from the bias or query position when one is given, else gazetteer order.

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	gplcIDPrefix    = "stackyard-place-"
	gplcQueryPrefix = "stackyard-query:"
	gplcTokenPrefix = "stackyard-offset:"
)

var gplcService = &geosaService{
	signingName: "geo-places",
	hosts:       []string{"geo-places.", "geo-places-fips.", "places.geo.", "places.geo-fips."},
	model:       awsmodels.GeoPlacesModel,
	ops:         geoplacesOperations,
	handlers: map[string]func(geosaCall) (*geosaReply, *geosaError){
		"Autocomplete":   gplcAutocomplete,
		"Geocode":        gplcGeocode,
		"GetPlace":       gplcGetPlace,
		"ReverseGeocode": gplcReverseGeocode,
		"SearchNearby":   gplcSearchNearby,
		"SearchText":     gplcSearchText,
		"Suggest":        gplcSuggest,
	},
	write: gplcWrite,
}

// gplcWrite sends a JSON reply and its pricing-bucket header.
func gplcWrite(w http.ResponseWriter, rep *geosaReply) {
	raw, _ := json.Marshal(rep.body)
	w.Header().Set("X-Amz-Geo-Pricing-Bucket", geosaPricingBucket)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) handleGeoPlacesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.geosaServe(w, r, gplcService)
}

var gplcByID = func() map[string]*gplcPlace {
	out := make(map[string]*gplcPlace, len(gplcGazetteer))
	for i := range gplcGazetteer {
		out[gplcIDPrefix+gplcGazetteer[i].id] = &gplcGazetteer[i]
	}
	return out
}()

// gplcScope is the parsed bias/filter part of a request.
type gplcScope struct {
	bias      *[2]float64
	bbox      *[4]float64
	circle    *[2]float64
	radius    float64
	countries []any
	types     []any
	features  []any
}

// gplcParseScope reads BiasPosition and Filter. need is how many of
// BiasPosition, Filter.BoundingBox and Filter.Circle must be given: -1 for
// "at most one", 1 for "exactly one".
func gplcParseScope(in map[string]any, need int) (*gplcScope, *geosaError) {
	sc := &gplcScope{features: geosaList(in, "AdditionalFeatures")}
	filter := geosaObj(in, "Filter")
	sc.countries = geosaList(filter, "IncludeCountries")
	sc.types = geosaList(filter, "IncludePlaceTypes")
	given := 0
	if v, ok := in["BiasPosition"]; ok {
		p, err := geosaPos(v, "BiasPosition")
		if err != nil {
			return nil, err
		}
		sc.bias, given = &p, given+1
	}
	if v, ok := filter["BoundingBox"]; ok {
		b, err := geosaBBox(v, "Filter.BoundingBox")
		if err != nil {
			return nil, err
		}
		sc.bbox, given = &b, given+1
	}
	if c := geosaObj(filter, "Circle"); c != nil {
		p, err := geosaPos(c["Center"], "Filter.Circle.Center")
		if err != nil {
			return nil, err
		}
		sc.radius, _ = c["Radius"].(float64)
		sc.circle, given = &p, given+1
	}
	if given > 1 {
		return nil, geosaValidation("FieldValidationFailed", "BiasPosition", "BiasPosition, Filter.BoundingBox and Filter.Circle are mutually exclusive")
	}
	if need == 1 && given == 0 {
		return nil, geosaValidation("Missing", "BiasPosition", "one of BiasPosition, Filter.BoundingBox or Filter.Circle is required")
	}
	return sc, nil
}

// origin is the point distances are measured from.
func (sc *gplcScope) origin() *[2]float64 {
	if sc.bias != nil {
		return sc.bias
	}
	return sc.circle
}

func (sc *gplcScope) keep(p *gplcPlace) bool {
	if sc.bbox != nil && !geosaInBBox(*sc.bbox, p.pos) {
		return false
	}
	if sc.circle != nil && geosaDistance(*sc.circle, p.pos) > sc.radius {
		return false
	}
	if len(sc.countries) > 0 && !geosaHas(sc.countries, p.country[0]) && !geosaHas(sc.countries, p.country[1]) {
		return false
	}
	return len(sc.types) == 0 || geosaHas(sc.types, p.placeType)
}

// gplcSelect filters the gazetteer and orders it by distance from origin.
func gplcSelect(keep func(*gplcPlace) bool, origin *[2]float64) []*gplcPlace {
	var out []*gplcPlace
	for i := range gplcGazetteer {
		if p := &gplcGazetteer[i]; keep(p) {
			out = append(out, p)
		}
	}
	if origin != nil {
		sort.SliceStable(out, func(i, j int) bool {
			return geosaDistance(*origin, out[i].pos) < geosaDistance(*origin, out[j].pos)
		})
	}
	return out
}

var gplcFold = strings.NewReplacer("é", "e", "è", "e", "ê", "e", "î", "i", "ï", "i", "ô", "o", "ü", "u", "ö", "o", "ä", "a", "ß", "ss")

func gplcTokens(s string) []string {
	return strings.FieldsFunc(gplcFold.Replace(strings.ToLower(s)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// gplcMatches reports whether every query token prefixes a token of the place.
func gplcMatches(p *gplcPlace, query string) bool {
	q := gplcTokens(query)
	if len(q) == 0 {
		return false
	}
	hay := gplcTokens(strings.Join(append([]string{p.title, p.locality, p.street, p.number, p.postal, p.district}, p.aliases...), " "))
	for _, c := range p.categories {
		hay = append(hay, gplcTokens(c[1])...)
	}
	for _, t := range q {
		found := false
		for _, h := range hay {
			if strings.HasPrefix(h, t) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func gplcAddress(p *gplcPlace) map[string]any {
	addr := map[string]any{
		"Label":   p.title,
		"Country": map[string]any{"Code2": p.country[0], "Code3": p.country[1], "Name": p.country[2]},
		"Region":  map[string]any{"Code": p.region[0], "Name": p.region[1]},
	}
	if p.placeType != "Locality" {
		parts := []string{p.title, strings.TrimSpace(p.number + " " + p.street), p.locality, strings.TrimSpace(p.region[0] + " " + p.postal), p.country[2]}
		if p.placeType == "PointAddress" {
			parts = parts[1:]
		}
		addr["Label"] = strings.Join(gplcNonEmpty(parts), ", ")
	}
	set := func(k, v string) {
		if v != "" {
			addr[k] = v
		}
	}
	if p.subRegion != "" {
		addr["SubRegion"] = map[string]any{"Name": p.subRegion}
	}
	set("Locality", p.locality)
	set("District", p.district)
	set("PostalCode", p.postal)
	set("Street", p.street)
	set("AddressNumber", p.number)
	return addr
}

func gplcNonEmpty(parts []string) []string {
	out := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// gplcItem renders a place; the caller's output shape drops members it lacks.
func gplcItem(p *gplcPlace, features []any, origin *[2]float64) map[string]any {
	span := 0.002
	if p.placeType == "Locality" {
		span = 0.2
	}
	item := map[string]any{
		"PlaceId":   gplcIDPrefix + p.id,
		"PlaceType": p.placeType,
		"Title":     p.title,
		"Address":   gplcAddress(p),
		"Position":  geosaPosJSON(p.pos),
		"MapView":   []any{geosaRound(p.pos[0]-span, 5), geosaRound(p.pos[1]-span, 5), geosaRound(p.pos[0]+span, 5), geosaRound(p.pos[1]+span, 5)},
	}
	if origin != nil {
		item["Distance"] = int64(geosaDistance(*origin, p.pos))
	}
	if len(p.categories) > 0 {
		cats := make([]any, 0, len(p.categories))
		for i, c := range p.categories {
			cats = append(cats, map[string]any{"Id": c[0], "Name": c[1], "Primary": i == 0})
		}
		item["Categories"] = cats
	}
	if geosaHas(features, "TimeZone") {
		tz := map[string]any{"Name": p.timeZone, "Offset": p.tzOffset}
		// The model bounds OffsetSeconds at zero, so west-of-UTC zones carry only Offset.
		if secs := gplcOffsetSeconds(p.tzOffset); secs >= 0 {
			tz["OffsetSeconds"] = secs
		}
		item["TimeZone"] = tz
	}
	if geosaHas(features, "Contact") && p.website != "" {
		item["Contacts"] = map[string]any{"Websites": []any{map[string]any{"Value": p.website}}}
	}
	if geosaHas(features, "Access") && p.placeType != "Locality" {
		item["AccessPoints"] = []any{map[string]any{"Position": geosaPosJSON(p.pos)}}
	}
	return item
}

func gplcOffsetSeconds(off string) int64 {
	if len(off) != 6 {
		return 0
	}
	h, _ := strconv.ParseInt(off[1:3], 10, 32)
	m, _ := strconv.ParseInt(off[4:6], 10, 32)
	secs := h*3600 + m*60
	if off[0] == '-' {
		return -secs
	}
	return secs
}

// gplcHighlight marks the first query token inside text (rune offsets).
func gplcHighlight(text, query string) []any {
	q := gplcTokens(query)
	if len(q) == 0 {
		return nil
	}
	lower := []rune(gplcFold.Replace(strings.ToLower(text)))
	tok := []rune(q[0])
	for i := 0; i+len(tok) <= len(lower); i++ {
		if string(lower[i:i+len(tok)]) == q[0] && (i == 0 || !unicode.IsLetter(lower[i-1])) {
			runes := []rune(text)
			end := min(i+len(tok), len(runes))
			return []any{map[string]any{"StartIndex": i, "EndIndex": end, "Value": string(runes[i:end])}}
		}
	}
	return nil
}

func gplcItems(places []*gplcPlace, max int, sc *gplcScope, each func(*gplcPlace, map[string]any)) []any {
	out := []any{}
	for _, p := range places {
		if len(out) >= max {
			break
		}
		item := gplcItem(p, sc.features, sc.origin())
		if each != nil {
			each(p, item)
		}
		out = append(out, item)
	}
	return out
}

func gplcReply(body map[string]any) (*geosaReply, *geosaError) {
	return &geosaReply{body: body, pricing: true}, nil
}

func gplcAutocomplete(c geosaCall) (*geosaReply, *geosaError) {
	sc, err := gplcParseScope(c.in, -1)
	if err != nil {
		return nil, err
	}
	query := geosaStr(c.in, "QueryText")
	places := gplcSelect(func(p *gplcPlace) bool {
		return p.placeType != "PointOfInterest" && sc.keep(p) && gplcMatches(p, query)
	}, sc.origin())
	items := gplcItems(places, int(geosaInt(c.in, "MaxResults", 5)), sc, func(p *gplcPlace, item map[string]any) {
		if h := gplcHighlight(p.title, query); h != nil {
			item["Highlights"] = map[string]any{"Title": h}
		}
	})
	return gplcReply(map[string]any{"ResultItems": items})
}

func gplcGeocode(c geosaCall) (*geosaReply, *geosaError) {
	query := geosaStr(c.in, "QueryText")
	comps := geosaObj(c.in, "QueryComponents")
	if query == "" && len(comps) == 0 {
		return nil, geosaValidation("Missing", "QueryText", "one of QueryText or QueryComponents is required")
	}
	sc, err := gplcParseScope(c.in, -1)
	if err != nil {
		return nil, err
	}
	places := gplcSelect(func(p *gplcPlace) bool {
		return sc.keep(p) && (query == "" || gplcMatches(p, query)) && gplcComponentsMatch(p, comps)
	}, sc.origin())
	items := gplcItems(places, int(geosaInt(c.in, "MaxResults", 20)), sc, func(p *gplcPlace, item map[string]any) {
		score := 0.8
		if query == "" || strings.EqualFold(strings.TrimSpace(query), p.title) {
			score = 1
		}
		item["MatchScores"] = map[string]any{"Overall": score}
	})
	return gplcReply(map[string]any{"ResultItems": items})
}

// gplcComponentsMatch checks each given component against the address
// (case-insensitive; Country matches its name or either code).
func gplcComponentsMatch(p *gplcPlace, comps map[string]any) bool {
	fields := map[string][]string{
		"Country": {p.country[0], p.country[1], p.country[2]}, "Region": {p.region[0], p.region[1]},
		"SubRegion": {p.subRegion}, "Locality": {p.locality}, "District": {p.district},
		"Street": {p.street}, "AddressNumber": {p.number}, "PostalCode": {p.postal},
	}
	for key, v := range comps {
		want, _ := v.(string)
		ok := false
		for _, have := range fields[key] {
			if have != "" && gplcFold.Replace(strings.ToLower(have)) == gplcFold.Replace(strings.ToLower(strings.TrimSpace(want))) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func gplcGetPlace(c geosaCall) (*geosaReply, *geosaError) {
	p := gplcByID[geosaStr(c.in, "PlaceId")]
	if p == nil {
		return nil, geosaValidation("FieldValidationFailed", "PlaceId", "PlaceId %q does not identify a place", geosaStr(c.in, "PlaceId"))
	}
	return gplcReply(gplcItem(p, geosaList(c.in, "AdditionalFeatures"), nil))
}

func gplcReverseGeocode(c geosaCall) (*geosaReply, *geosaError) {
	pos, err := geosaPos(c.in["QueryPosition"], "QueryPosition")
	if err != nil {
		return nil, err
	}
	radius := float64(geosaInt(c.in, "QueryRadius", 21000000))
	types := geosaList(geosaObj(c.in, "Filter"), "IncludePlaceTypes")
	sc := &gplcScope{bias: &pos, features: geosaList(c.in, "AdditionalFeatures")}
	places := gplcSelect(func(p *gplcPlace) bool {
		return geosaDistance(pos, p.pos) <= radius && (len(types) == 0 || geosaHas(types, p.placeType))
	}, &pos)
	return gplcReply(map[string]any{"ResultItems": gplcItems(places, int(geosaInt(c.in, "MaxResults", 1)), sc, nil)})
}

func gplcSearchNearby(c geosaCall) (*geosaReply, *geosaError) {
	pos, err := geosaPos(c.in["QueryPosition"], "QueryPosition")
	if err != nil {
		return nil, err
	}
	filter := geosaObj(c.in, "Filter")
	sc := &gplcScope{bias: &pos, countries: geosaList(filter, "IncludeCountries"), features: geosaList(c.in, "AdditionalFeatures")}
	if v, ok := filter["BoundingBox"]; ok {
		b, err := geosaBBox(v, "Filter.BoundingBox")
		if err != nil {
			return nil, err
		}
		sc.bbox = &b
	}
	radius := float64(geosaInt(c.in, "QueryRadius", 21000000))
	include, exclude := geosaList(filter, "IncludeCategories"), geosaList(filter, "ExcludeCategories")
	// The gazetteer has no business chains or food types, so including any
	// leaves nothing.
	none := len(geosaList(filter, "IncludeBusinessChains")) > 0 || len(geosaList(filter, "IncludeFoodTypes")) > 0
	places := gplcSelect(func(p *gplcPlace) bool {
		if none || p.placeType != "PointOfInterest" || geosaDistance(pos, p.pos) > radius || !sc.keep(p) {
			return false
		}
		in := len(include) == 0
		for _, cat := range p.categories {
			if geosaHas(exclude, cat[0]) {
				return false
			}
			in = in || geosaHas(include, cat[0])
		}
		return in
	}, &pos)
	return gplcPage(c.in, places, sc)
}

func gplcSearchText(c geosaCall) (*geosaReply, *geosaError) {
	query, id := geosaStr(c.in, "QueryText"), geosaStr(c.in, "QueryId")
	if (query == "") == (id == "") {
		return nil, geosaValidation("FieldValidationFailed", "QueryText", "exactly one of QueryText or QueryId is required")
	}
	if id != "" {
		raw, err := base64.RawURLEncoding.DecodeString(id)
		if err != nil || !strings.HasPrefix(string(raw), gplcQueryPrefix) {
			return nil, geosaValidation("FieldValidationFailed", "QueryId", "QueryId is not a query from Suggest")
		}
		query = strings.TrimPrefix(string(raw), gplcQueryPrefix)
	}
	sc, err := gplcParseScope(c.in, 1)
	if err != nil {
		return nil, err
	}
	places := gplcSelect(func(p *gplcPlace) bool { return sc.keep(p) && gplcMatches(p, query) }, sc.origin())
	return gplcPage(c.in, places, sc)
}

// gplcPage returns one page of places with an offset NextToken.
func gplcPage(in map[string]any, places []*gplcPlace, sc *gplcScope) (*geosaReply, *geosaError) {
	offset := 0
	if tok := geosaStr(in, "NextToken"); tok != "" {
		raw, err := base64.RawURLEncoding.DecodeString(tok)
		n, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), gplcTokenPrefix), 10, 32)
		if err != nil || perr != nil || !strings.HasPrefix(string(raw), gplcTokenPrefix) || n < 0 || int(n) > len(places) {
			return nil, geosaValidation("FieldValidationFailed", "NextToken", "NextToken is invalid")
		}
		offset = int(n)
	}
	max := int(geosaInt(in, "MaxResults", 20))
	body := map[string]any{"ResultItems": gplcItems(places[offset:], max, sc, nil)}
	if next := offset + max; next < len(places) {
		body["NextToken"] = base64.RawURLEncoding.EncodeToString([]byte(gplcTokenPrefix + strconv.Itoa(next)))
	}
	return gplcReply(body)
}

func gplcSuggest(c geosaCall) (*geosaReply, *geosaError) {
	sc, err := gplcParseScope(c.in, 1)
	if err != nil {
		return nil, err
	}
	query := geosaStr(c.in, "QueryText")
	places := gplcSelect(func(p *gplcPlace) bool { return sc.keep(p) && gplcMatches(p, query) }, sc.origin())
	max := int(geosaInt(c.in, "MaxResults", 20))
	items := []any{}
	// A query that names a gazetteer category is offered as a category search.
	if cat := gplcCategoryNamed(query); cat != "" && max > 0 {
		items = append(items, map[string]any{
			"Title": cat, "SuggestResultItemType": "Query",
			"Query": map[string]any{"QueryId": base64.RawURLEncoding.EncodeToString([]byte(gplcQueryPrefix + query)), "QueryType": "Category"},
		})
	}
	for _, p := range places {
		if len(items) >= max {
			break
		}
		item := map[string]any{"Title": p.title, "SuggestResultItemType": "Place", "Place": gplcItem(p, sc.features, sc.origin())}
		if h := gplcHighlight(p.title, query); h != nil {
			item["Highlights"] = map[string]any{"Title": h}
		}
		items = append(items, item)
	}
	return gplcReply(map[string]any{"ResultItems": items, "QueryRefinements": []any{}})
}

func gplcCategoryNamed(query string) string {
	q := strings.Join(gplcTokens(query), " ")
	for _, p := range gplcGazetteer {
		for _, c := range p.categories {
			if q != "" && strings.Join(gplcTokens(c[1]), " ") == q {
				return c[1]
			}
		}
	}
	return ""
}
