package server

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// The public raster data collections and a deterministic scene catalog:
// one scene per revisit period over the requested time range, footprints
// covering the area of interest, and property values hashed from the scene
// ID, so searches page and filter like the real service.

const (
	sgeoRegion   = "us-west-2" // the only Region the service runs in
	sgeoPageSize = 20
	sgeoMaxScan  = 10000 // scenes examined per search
)

type sgeoCollection struct {
	id, name, desc, page, kind string
	bands                      []string
	revisit                    time.Duration // 0: one static scene
	sceneID                    func(i int, t time.Time) string
	platform                   func(i int) string
}

func (c *sgeoCollection) arn() string {
	return "arn:aws:sagemaker-geospatial:" + sgeoRegion + ":378778860802:raster-data-collection/public/" + c.id
}

var sgeoCollections = []*sgeoCollection{
	{id: "nmqj48dcu3g7ayw8", name: "Sentinel 2 L2A COGs", kind: "PUBLIC",
		desc:    "Sentinel-2a and Sentinel-2b imagery, processed to Level 2A (Surface Reflectance) and converted to Cloud-Optimized GeoTIFFs",
		page:    "https://registry.opendata.aws/sentinel-2-l2a-cogs",
		bands:   []string{"coastal", "blue", "green", "red", "rededge1", "rededge2", "rededge3", "nir", "nir08", "nir09", "swir16", "swir22", "aot", "wvp", "scl"},
		revisit: 5 * 24 * time.Hour,
		sceneID: func(i int, t time.Time) string {
			return fmt.Sprintf("S2%s_10TET_%s_0_L2A", string(rune('A'+i%2)), t.Format("20060102"))
		},
		platform: func(i int) string { return "sentinel-2" + string(rune('a'+i%2)) }},
	{id: "gmqa64dcu2g9ayx1", name: "Landsat Collection 2 Level-2 Science Products", kind: "PUBLIC",
		desc:    "Landsat Collection 2 Level-2 Surface Reflectance and Surface Temperature science products",
		page:    "https://registry.opendata.aws/usgs-landsat",
		bands:   []string{"coastal", "blue", "green", "red", "nir08", "swir16", "swir22", "lwir11", "qa_aerosol", "qa_pixel", "qa_radsat"},
		revisit: 8 * 24 * time.Hour,
		sceneID: func(i int, t time.Time) string {
			return fmt.Sprintf("LC0%d_L2SP_046027_%s_02_T1", 8+i%2, t.Format("20060102"))
		},
		platform: func(i int) string { return "landsat-" + strconv.Itoa(8+i%2) }},
	{id: "dxxbpqwvu9041ny8", name: "Copernicus DEM", kind: "PUBLIC",
		desc:     "The Copernicus DEM is a Digital Surface Model which represents the surface of the Earth including buildings, infrastructure and vegetation",
		page:     "https://registry.opendata.aws/copernicus-dem",
		bands:    []string{"elevation"},
		sceneID:  func(int, time.Time) string { return "Copernicus_DSM_COG_10_N47_00_W123_00_DEM" },
		platform: func(int) string { return "copernicus-dem" }},
}

func sgeoCollectionByArn(arn string) *sgeoCollection {
	for _, c := range sgeoCollections {
		if c.arn() == arn {
			return c
		}
	}
	return nil
}

func (c *sgeoCollection) summary() map[string]any {
	filters := []any{
		map[string]any{"Name": "EoCloudCover", "Type": "number", "Minimum": 0, "Maximum": 100},
		map[string]any{"Name": "ViewOffNadir", "Type": "number", "Minimum": 0, "Maximum": 90},
		map[string]any{"Name": "Platform", "Type": "string"},
	}
	return map[string]any{"Arn": c.arn(), "Name": c.name, "Description": c.desc, "DescriptionPageUrl": c.page,
		"Type": c.kind, "SupportedFilters": filters, "ImageSourceBands": fcAnyList(c.bands), "Tags": map[string]any{}}
}

func (st *sgeoStore) getCollection(in map[string]any) (map[string]any, *fcErr) {
	c := sgeoCollectionByArn(fcStr(in, "Arn"))
	if c == nil {
		return nil, fcNotFound("Raster data collection %s does not exist", fcStr(in, "Arn"))
	}
	return c.summary(), nil
}

func (st *sgeoStore) listCollections(in map[string]any) (map[string]any, *fcErr) {
	items := make([]map[string]any, 0, len(sgeoCollections))
	for _, c := range sgeoCollections {
		items = append(items, c.summary())
	}
	return sgeoPage(items, in, "RasterDataCollectionSummaries")
}

// sgeoPage returns one page of items in their given order.
func sgeoPage(items []map[string]any, in map[string]any, listKey string) (map[string]any, *fcErr) {
	start := 0
	if tok := fcStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, sgeoInvalid("invalid NextToken")
		}
		start = int(n)
	}
	limit := sgeoPageSize
	if v, ok := fcNum(in, "MaxResults"); ok {
		limit = int(v) // the model bounds MaxResults to 1..20
	}
	end := min(start+limit, len(items))
	page := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		page = append(page, item)
	}
	out := map[string]any{listKey: page}
	if end < len(items) {
		out["NextToken"] = strconv.Itoa(end)
	}
	return out, nil
}

// sgeoBBox is the bounding box of an area of interest's first ring, or a
// default footprint over the Pacific Northwest.
func sgeoBBox(aoi map[string]any) [4]float64 {
	geom := fcObj(aoi, "AreaOfInterestGeometry")
	rings, _ := fcObj(geom, "PolygonGeometry")["Coordinates"].([]any)
	if multi, _ := fcObj(geom, "MultiPolygonGeometry")["Coordinates"].([]any); len(rings) == 0 && len(multi) > 0 {
		rings, _ = multi[0].([]any)
	}
	box := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	if len(rings) > 0 {
		ring, _ := rings[0].([]any)
		for _, p := range ring {
			pos, _ := p.([]any)
			if len(pos) != 2 {
				continue
			}
			x, okx := pos[0].(float64)
			y, oky := pos[1].(float64)
			if okx && oky {
				box = [4]float64{min(box[0], x), min(box[1], y), max(box[2], x), max(box[3], y)}
			}
		}
	}
	if math.IsInf(box[0], 0) {
		return [4]float64{-123, 47, -122, 48}
	}
	return box
}

func sgeoFootprint(b [4]float64) map[string]any {
	pt := func(x, y float64) any { return []any{x, y} }
	ring := []any{pt(b[0], b[1]), pt(b[2], b[1]), pt(b[2], b[3]), pt(b[0], b[3]), pt(b[0], b[1])}
	return map[string]any{"Type": "Polygon", "Coordinates": []any{ring}}
}

func sgeoSceneProperties(c *sgeoCollection, i int, id string) map[string]any {
	h := func(k string) float64 { return amlScore(id + "|" + k) }
	props := map[string]any{"Platform": c.platform(i), "EoCloudCover": math.Round(h("cloud")*10000) / 100,
		"ViewOffNadir": math.Round(h("nadir")*1000) / 100, "ViewSunAzimuth": math.Round(h("az")*36000) / 100,
		"ViewSunElevation": math.Round((20+h("el")*50)*100) / 100}
	if strings.HasPrefix(c.platform(i), "landsat") {
		props["LandsatCloudCoverLand"] = props["EoCloudCover"]
	}
	return props
}

// sgeoMatchesFilters applies a query's PropertyFilters (AND) to a scene.
func sgeoMatchesFilters(filters map[string]any, props map[string]any) bool {
	list, _ := filters["Properties"].([]any)
	for _, f := range list {
		for name, cond := range fcObj(asMap(f), "Property") {
			c := asMap(cond)
			if name == "Platform" {
				v, want := fcStr(props, "Platform"), fcStr(c, "Value")
				ok := map[string]bool{"": v == want, "EQUALS": v == want, "NOT_EQUALS": v != want, "STARTS_WITH": strings.HasPrefix(v, want)}[fcStr(c, "ComparisonOperator")]
				if !ok {
					return false
				}
				continue
			}
			v, has := props[name].(float64)
			lo, _ := fcNum(c, "LowerBound")
			hi, _ := fcNum(c, "UpperBound")
			if !has || v < lo || v > hi {
				return false
			}
		}
	}
	return true
}

// scenes walks the collection's scenes in the query's time range from index
// start, calling visit until it returns false; at most sgeoMaxScan scenes.
func sgeoScenes(c *sgeoCollection, query map[string]any, start int, visit func(i int, item map[string]any) bool) {
	tr := fcObj(query, "TimeRangeFilter")
	from, to := amlTime(tr["StartTime"]), amlTime(tr["EndTime"])
	t0 := time.Unix(int64(from), 0).UTC()
	box := sgeoBBox(fcObj(query, "AreaOfInterest"))
	bands := map[string]bool{}
	for _, b := range fcStrs(query, "BandFilter") {
		bands[b] = true
	}
	for i := start; i < start+sgeoMaxScan; i++ {
		t := t0.Add(time.Duration(i) * c.revisit)
		if fcEpoch(t) > to || (c.revisit == 0 && i > 0) {
			return
		}
		id := c.sceneID(i, t)
		props := sgeoSceneProperties(c, i, id)
		if !sgeoMatchesFilters(fcObj(query, "PropertyFilters"), props) {
			continue
		}
		assets := map[string]any{}
		for _, b := range c.bands {
			if len(bands) == 0 || bands[b] {
				assets[b] = map[string]any{"Href": "https://" + c.id + ".s3." + sgeoRegion + ".amazonaws.com/" + id + "/" + b + ".tif"}
			}
		}
		item := map[string]any{"Id": id, "DateTime": fcEpoch(t), "Geometry": sgeoFootprint(box), "Properties": props, "Assets": assets}
		if !visit(i, item) {
			return
		}
	}
}

func (st *sgeoStore) search(in map[string]any) (map[string]any, *fcErr) {
	c := sgeoCollectionByArn(fcStr(in, "Arn"))
	if c == nil {
		return nil, fcNotFound("Raster data collection %s does not exist", fcStr(in, "Arn"))
	}
	query := fcObj(in, "RasterDataCollectionQuery")
	tr := fcObj(query, "TimeRangeFilter")
	if amlTime(tr["EndTime"]) < amlTime(tr["StartTime"]) {
		return nil, sgeoInvalid("TimeRangeFilter EndTime must not be before StartTime")
	}
	for _, b := range fcStrs(query, "BandFilter") {
		if !fcContainsAny(fcAnyList(c.bands), b) {
			return nil, sgeoInvalid("band %s is not in %s", b, c.name)
		}
	}
	start := 0
	if tok := fcStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || n > sgeoMaxScan*100 {
			return nil, sgeoInvalid("invalid NextToken")
		}
		start = int(n)
	}
	total := 0
	sgeoScenes(c, query, 0, func(int, map[string]any) bool { total++; return true })
	items := []any{}
	next := -1
	sgeoScenes(c, query, start, func(i int, item map[string]any) bool {
		if len(items) == sgeoPageSize {
			next = i
			return false
		}
		items = append(items, item)
		return true
	})
	out := map[string]any{"ApproximateResultCount": total, "Items": items}
	if next >= 0 {
		out["NextToken"] = strconv.Itoa(next)
	}
	return out, nil
}
