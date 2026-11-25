package server

// Amazon Location Service Routes V2 (geo-routes). There is no road network:
// routes run in straight great-circle segments through their waypoints at a
// fixed speed per travel mode, isolines are circles, matrices are pairwise
// distances, snapping keeps each trace point where it is, and waypoint
// optimization is nearest-neighbour ordering that honours Before constraints.
// Avoidance, exclusion, traffic and vehicle options are validated, not applied.

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	grteMaxWaypoints       = 25  // CalculateRoutes intermediate waypoints
	grteMaxOptimizeStops   = 50  // OptimizeWaypoints waypoints
	grteMaxMatrixSide      = 100 // CalculateRouteMatrix origins or destinations
	grteDefaultRingPoints  = 64
	grteTimestampLayout    = "2006-01-02T15:04:05Z07:00"
	grteSnapConfidence     = 1.0
	grteStraightLineFactor = 1.0
)

// grteSpeeds are metres per second for each travel mode.
var grteSpeeds = map[string]float64{"Car": 13.9, "Truck": 11.1, "Scooter": 8.3, "Pedestrian": 1.4}

var grteService = &geosaService{
	signingName: "geo-routes",
	hosts:       []string{"geo-routes.", "geo-routes-fips.", "routes.geo.", "routes.geo-fips."},
	model:       awsmodels.GeoRoutesModel,
	ops:         georoutesOperations,
	handlers: map[string]func(geosaCall) (*geosaReply, *geosaError){
		"CalculateIsolines":    grteCalculateIsolines,
		"CalculateRouteMatrix": grteCalculateRouteMatrix,
		"CalculateRoutes":      grteCalculateRoutes,
		"OptimizeWaypoints":    grteOptimizeWaypoints,
		"SnapToRoads":          grteSnapToRoads,
	},
	write: grteWrite,
}

// grteWrite sends a JSON reply and its pricing-bucket header.
func grteWrite(w http.ResponseWriter, rep *geosaReply) {
	raw, _ := json.Marshal(rep.body)
	w.Header().Set("X-Amz-Geo-Pricing-Bucket", geosaPricingBucket)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) handleGeoRoutesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.geosaServe(w, r, grteService)
}

func grteSpeed(mode string) float64 {
	if v, ok := grteSpeeds[mode]; ok {
		return v
	}
	return grteSpeeds["Pedestrian"] // Transit and Intermodal fall back to walking
}

// grteLeg measures a polyline: metres and seconds at the mode's speed.
func grteMeasure(points [][2]float64, mode string) (int64, int64) {
	d := 0.0
	for i := 1; i < len(points); i++ {
		d += geosaDistance(points[i-1], points[i]) * grteStraightLineFactor
	}
	return int64(math.Round(d)), int64(math.Round(d / grteSpeed(mode)))
}

func grteTime(in map[string]any, field string) (*time.Time, *geosaError) {
	raw := geosaStr(in, field)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, geosaValidation("CannotParse", field, "%s must be an ISO 8601 timestamp with an offset", field)
	}
	return &t, nil
}

func grteFormat(t time.Time) string { return t.Format(grteTimestampLayout) }

// grteStop is a point the route passes, with what happens there.
type grteStop struct {
	pos         [2]float64
	passThrough bool
	stop        int64 // StopDuration seconds
}

func grteCalculateRoutes(c geosaCall) (*geosaReply, *geosaError) {
	in := c.in
	origin, err := geosaPos(in["Origin"], "Origin")
	if err != nil {
		return nil, err
	}
	dest, err := geosaPos(in["Destination"], "Destination")
	if err != nil {
		return nil, err
	}
	wps := geosaList(in, "Waypoints")
	if len(wps) > grteMaxWaypoints {
		return nil, geosaValidation("FieldValidationFailed", "Waypoints", "Waypoints must have at most %d items", grteMaxWaypoints)
	}
	stops := []grteStop{{pos: origin}}
	for i, raw := range wps {
		wp, _ := raw.(map[string]any)
		p, err := geosaPos(wp["Position"], fmt.Sprintf("Waypoints[%d].Position", i))
		if err != nil {
			return nil, err
		}
		pass, _ := wp["PassThrough"].(bool)
		stops = append(stops, grteStop{pos: p, passThrough: pass, stop: geosaInt(wp, "StopDuration", 0)})
	}
	stops = append(stops, grteStop{pos: dest})
	if geosaStr(in, "DepartureTime") != "" && geosaStr(in, "ArrivalTime") != "" {
		return nil, geosaValidation("FieldValidationFailed", "ArrivalTime", "ArrivalTime and DepartureTime are mutually exclusive")
	}
	depart, err := grteTime(in, "DepartureTime")
	if err != nil {
		return nil, err
	}
	arrive, err := grteTime(in, "ArrivalTime")
	if err != nil {
		return nil, err
	}
	mode := geosaStr(in, "TravelMode")
	if mode == "" {
		mode = "Car"
	}
	format := geosaStr(in, "LegGeometryFormat")
	if format == "" {
		format = "FlexiblePolyline"
	}
	notices := []any{}
	if mode == "Transit" || mode == "Intermodal" {
		notices = append(notices, map[string]any{"Code": "TransitRouteUnavailable", "Impact": "High"})
	}
	legs, distance, duration := grteLegs(stops, mode, format)
	if arrive != nil {
		t := arrive.Add(-time.Duration(duration) * time.Second)
		depart = &t
	}
	if depart != nil {
		grteStampLegs(legs, *depart)
	}
	route := map[string]any{
		"Legs":            grteLegMaps(legs),
		"MajorRoadLabels": []any{},
		"Summary":         map[string]any{"Distance": distance, "Duration": duration},
	}
	return &geosaReply{pricing: true, body: map[string]any{"LegGeometryFormat": format, "Notices": notices, "Routes": []any{route}}}, nil
}

// grteLegData is one leg before rendering.
type grteLegData struct {
	points             [][2]float64
	passOffsets        []int
	distance, duration int64
	stopAfter          int64
	mode, format       string
	startIdx, endIdx   int // waypoint indexes of the endpoints (-1: origin/destination)
	departAt, arriveAt *time.Time
}

func grteLegs(stops []grteStop, mode, format string) ([]*grteLegData, int64, int64) {
	var legs []*grteLegData
	cur := &grteLegData{points: [][2]float64{stops[0].pos}, mode: mode, format: format, startIdx: -1}
	var total, totalDur int64
	for i := 1; i < len(stops); i++ {
		cur.points = append(cur.points, stops[i].pos)
		if stops[i].passThrough && i < len(stops)-1 {
			cur.passOffsets = append(cur.passOffsets, len(cur.points)-1)
			continue
		}
		cur.distance, cur.duration = grteMeasure(cur.points, mode)
		cur.endIdx = i - 1
		if i == len(stops)-1 {
			cur.endIdx = -1
		} else {
			cur.stopAfter = stops[i].stop
		}
		total += cur.distance
		totalDur += cur.duration + cur.stopAfter
		legs = append(legs, cur)
		cur = &grteLegData{points: [][2]float64{stops[i].pos}, mode: mode, format: format, startIdx: i - 1}
	}
	return legs, total, totalDur
}

func grteStampLegs(legs []*grteLegData, start time.Time) {
	t := start
	for _, l := range legs {
		d, a := t, t.Add(time.Duration(l.duration)*time.Second)
		l.departAt, l.arriveAt = &d, &a
		t = a.Add(time.Duration(l.stopAfter) * time.Second)
	}
}

func grtePlace(p [2]float64, waypoint int, at *time.Time) map[string]any {
	place := map[string]any{"Position": geosaPosJSON(p), "OriginalPosition": geosaPosJSON(p)}
	if waypoint >= 0 {
		place["WaypointIndex"] = waypoint
	}
	out := map[string]any{"Place": place}
	if at != nil {
		out["Time"] = grteFormat(*at)
	}
	return out
}

func grteLegMaps(legs []*grteLegData) []any {
	out := make([]any, 0, len(legs))
	for _, l := range legs {
		first, last := l.points[0], l.points[len(l.points)-1]
		pass := []any{}
		for _, off := range l.passOffsets {
			pass = append(pass, map[string]any{"GeometryOffset": off, "Place": map[string]any{"Position": geosaPosJSON(l.points[off])}})
		}
		steps := []any{
			map[string]any{"Type": "Depart", "Distance": l.distance, "Duration": l.duration, "GeometryOffset": 0,
				"Instruction": fmt.Sprintf("Travel %d m to the destination.", l.distance)},
			map[string]any{"Type": "Arrive", "Distance": 0, "Duration": 0, "GeometryOffset": len(l.points) - 1, "Instruction": "Arrive at the destination."},
		}
		details := map[string]any{
			"Departure": grtePlace(first, l.startIdx, l.departAt), "Arrival": grtePlace(last, l.endIdx, l.arriveAt),
			"AfterTravelSteps": []any{}, "Notices": []any{}, "PassThroughWaypoints": pass, "Spans": []any{}, "TravelSteps": steps,
			"Summary": map[string]any{
				"Overview":   map[string]any{"Distance": l.distance, "Duration": l.duration},
				"TravelOnly": map[string]any{"Duration": l.duration},
			},
		}
		leg := map[string]any{"Geometry": grteGeometry(l.points, l.format)}
		if _, vehicle := grteSpeeds[l.mode]; vehicle && l.mode != "Pedestrian" {
			for _, k := range []string{"Incidents", "Tolls", "TollSystems", "TruckRoadTypes", "Zones"} {
				details[k] = []any{}
			}
			leg["Type"], leg["TravelMode"], leg["VehicleLegDetails"] = "Vehicle", l.mode, details
		} else {
			leg["Type"], leg["TravelMode"], leg["PedestrianLegDetails"] = "Pedestrian", "Pedestrian", details
		}
		out = append(out, leg)
	}
	return out
}

func grteCalculateRouteMatrix(c geosaCall) (*geosaReply, *geosaError) {
	in := c.in
	origins, err := grteMatrixSide(in, "Origins")
	if err != nil {
		return nil, err
	}
	dests, err := grteMatrixSide(in, "Destinations")
	if err != nil {
		return nil, err
	}
	all := append(append([][2]float64{}, origins...), dests...)
	inside, boundary, err := grteBoundary(geosaObj(in, "RoutingBoundary"), all)
	if err != nil {
		return nil, err
	}
	mode := geosaStr(in, "TravelMode")
	if mode == "" {
		mode = "Car"
	}
	errorCount := 0
	matrix := make([]any, 0, len(origins))
	for _, o := range origins {
		row := make([]any, 0, len(dests))
		for _, d := range dests {
			entry := map[string]any{"Distance": 0, "Duration": 0}
			switch {
			case !inside(o) && !inside(d):
				entry["Error"] = "OutOfBounds"
			case !inside(o):
				entry["Error"] = "OutOfBoundsOrigin"
			case !inside(d):
				entry["Error"] = "OutOfBoundsDestination"
			default:
				entry["Distance"], entry["Duration"] = grteMeasure([][2]float64{o, d}, mode)
			}
			if entry["Error"] != nil {
				errorCount++
			}
			row = append(row, entry)
		}
		matrix = append(matrix, row)
	}
	return &geosaReply{pricing: true, body: map[string]any{"ErrorCount": errorCount, "RouteMatrix": matrix, "RoutingBoundary": boundary}}, nil
}

func grteMatrixSide(in map[string]any, field string) ([][2]float64, *geosaError) {
	items := geosaList(in, field)
	if len(items) > grteMaxMatrixSide {
		return nil, geosaValidation("FieldValidationFailed", field, "%s must have at most %d items", field, grteMaxMatrixSide)
	}
	out := make([][2]float64, 0, len(items))
	for i, raw := range items {
		item, _ := raw.(map[string]any)
		p, err := geosaPos(item["Position"], fmt.Sprintf("%s[%d].Position", field, i))
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// grteBoundary turns a RoutingBoundary into a containment test and the
// boundary to report (AutoCircle is resolved to a Circle).
func grteBoundary(rb map[string]any, points [][2]float64) (func([2]float64) bool, map[string]any, *geosaError) {
	everywhere := func([2]float64) bool { return true }
	geom := geosaObj(rb, "Geometry")
	unbounded, _ := rb["Unbounded"].(bool)
	if unbounded && geom != nil {
		return nil, nil, geosaValidation("FieldValidationFailed", "RoutingBoundary", "RoutingBoundary.Geometry and Unbounded are mutually exclusive")
	}
	if geom == nil {
		return everywhere, map[string]any{"Unbounded": true}, nil
	}
	if len(geom) != 1 {
		return nil, nil, geosaValidation("FieldValidationFailed", "RoutingBoundary.Geometry", "RoutingBoundary.Geometry must set exactly one of AutoCircle, Circle, BoundingBox or Polygon")
	}
	if circle := geosaObj(geom, "Circle"); circle != nil {
		center, err := geosaPos(circle["Center"], "RoutingBoundary.Geometry.Circle.Center")
		if err != nil {
			return nil, nil, err
		}
		radius, _ := circle["Radius"].(float64)
		return grteInCircle(center, radius), map[string]any{"Geometry": map[string]any{"Circle": circle}}, nil
	}
	if auto, ok := geom["AutoCircle"].(map[string]any); ok {
		center, radius := grteAutoCircle(points, auto)
		out := map[string]any{"Center": geosaPosJSON(center), "Radius": radius}
		return grteInCircle(center, radius), map[string]any{"Geometry": map[string]any{"Circle": out}}, nil
	}
	if v, ok := geom["BoundingBox"]; ok {
		b, err := geosaBBox(v, "RoutingBoundary.Geometry.BoundingBox")
		if err != nil {
			return nil, nil, err
		}
		return func(p [2]float64) bool { return geosaInBBox(b, p) }, map[string]any{"Geometry": map[string]any{"BoundingBox": v}}, nil
	}
	rings := geosaList(geom, "Polygon")
	if len(rings) != 1 {
		return nil, nil, geosaValidation("FieldValidationFailed", "RoutingBoundary.Geometry", "RoutingBoundary.Geometry must set exactly one of AutoCircle, Circle, BoundingBox or Polygon")
	}
	ringRaw, _ := rings[0].([]any)
	ring := make([][2]float64, 0, len(ringRaw))
	for i, raw := range ringRaw {
		p, err := geosaPos(raw, fmt.Sprintf("RoutingBoundary.Geometry.Polygon[0][%d]", i))
		if err != nil {
			return nil, nil, err
		}
		ring = append(ring, p)
	}
	if len(ring) < 4 || ring[0] != ring[len(ring)-1] {
		return nil, nil, geosaValidation("FieldValidationFailed", "RoutingBoundary.Geometry.Polygon", "the polygon ring must have at least four positions and be closed (first and last equal)")
	}
	return func(p [2]float64) bool { return grteInRing(ring, p) }, map[string]any{"Geometry": map[string]any{"Polygon": rings}}, nil
}

func grteInCircle(center [2]float64, radius float64) func([2]float64) bool {
	return func(p [2]float64) bool { return geosaDistance(center, p) <= radius }
}

// grteAutoCircle centres on the mean position and covers every point plus
// Margin, capped at MaxRadius.
func grteAutoCircle(points [][2]float64, auto map[string]any) ([2]float64, float64) {
	var c [2]float64
	for _, p := range points {
		c[0] += p[0] / float64(len(points))
		c[1] += p[1] / float64(len(points))
	}
	c = [2]float64{geosaRound(c[0], 7), geosaRound(c[1], 7)}
	radius := 0.0
	for _, p := range points {
		radius = math.Max(radius, geosaDistance(c, p))
	}
	radius += float64(geosaInt(auto, "Margin", 0))
	if max := geosaInt(auto, "MaxRadius", 0); max > 0 {
		radius = math.Min(radius, float64(max))
	}
	return c, math.Round(radius)
}

// grteInRing is an even-odd ray-casting test in plain lon/lat space.
func grteInRing(ring [][2]float64, p [2]float64) bool {
	in := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		a, b := ring[i], ring[j]
		if (a[1] > p[1]) != (b[1] > p[1]) && p[0] < (b[0]-a[0])*(p[1]-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

func grteCalculateIsolines(c geosaCall) (*geosaReply, *geosaError) {
	in := c.in
	_, hasOrigin := in["Origin"]
	_, hasDest := in["Destination"]
	if hasOrigin == hasDest {
		return nil, geosaValidation("FieldValidationFailed", "Origin", "exactly one of Origin or Destination is required")
	}
	field := "Origin"
	if hasDest {
		field = "Destination"
	}
	center, err := geosaPos(in[field], field)
	if err != nil {
		return nil, err
	}
	th := geosaObj(in, "Thresholds")
	distances, times := geosaList(th, "Distance"), geosaList(th, "Time")
	if len(distances) == 0 && len(times) == 0 {
		return nil, geosaValidation("Missing", "Thresholds", "Thresholds must list Distance or Time values")
	}
	mode := geosaStr(in, "TravelMode")
	if mode == "" {
		mode = "Car"
	}
	format := geosaStr(in, "IsolineGeometryFormat")
	if format == "" {
		format = "FlexiblePolyline"
	}
	n := grteDefaultRingPoints
	if mp := geosaInt(geosaObj(in, "IsolineGranularity"), "MaxPoints", 0); mp > 0 && int(mp-1) < n {
		n = int(mp - 1)
	}
	isoline := func(radius float64) map[string]any {
		ring := grteCircle(center, math.Max(radius, 1), n)
		geom := map[string]any{"Polygon": []any{grteLineJSON(ring)}}
		if format != "Simple" {
			geom = map[string]any{"PolylinePolygon": []any{grteFlexEncode(ring, grteFlexPrecision)}}
		}
		return map[string]any{"Connections": []any{}, "Geometries": []any{geom}}
	}
	isolines := []any{}
	for _, v := range distances {
		d, _ := v.(float64)
		iso := isoline(d)
		iso["DistanceThreshold"] = int64(d)
		isolines = append(isolines, iso)
	}
	for _, v := range times {
		t, _ := v.(float64)
		iso := isoline(t * grteSpeed(mode))
		iso["TimeThreshold"] = int64(t)
		isolines = append(isolines, iso)
	}
	body := map[string]any{"IsolineGeometryFormat": format, "Isolines": isolines, "Snapped" + field: geosaPosJSON(center)}
	for _, k := range []string{"DepartureTime", "ArrivalTime"} {
		t, err := grteTime(in, k)
		if err != nil {
			return nil, err
		}
		if t != nil {
			body[k] = grteFormat(*t)
		}
	}
	return &geosaReply{pricing: true, body: body}, nil
}

// grteVisit is a waypoint to order.
type grteVisit struct {
	id      string
	pos     [2]float64
	service int64
	before  []int // waypoints that must come after this one
}

func grteOptimizeWaypoints(c geosaCall) (*geosaReply, *geosaError) {
	in := c.in
	origin, err := geosaPos(in["Origin"], "Origin")
	if err != nil {
		return nil, err
	}
	raw := geosaList(in, "Waypoints")
	if len(raw) > grteMaxOptimizeStops {
		return nil, geosaValidation("FieldValidationFailed", "Waypoints", "Waypoints must have at most %d items", grteMaxOptimizeStops)
	}
	start := time.Now().UTC().Truncate(time.Second)
	if t, err := grteTime(in, "DepartureTime"); err != nil {
		return nil, err
	} else if t != nil {
		start = *t
	}
	originID := grteID(geosaObj(in, "OriginOptions"), "Origin")
	ids := map[string]bool{originID: true}
	visits := make([]grteVisit, 0, len(raw))
	for i, r := range raw {
		wp, _ := r.(map[string]any)
		p, err := geosaPos(wp["Position"], fmt.Sprintf("Waypoints[%d].Position", i))
		if err != nil {
			return nil, err
		}
		v := grteVisit{id: grteID(wp, "Waypoint"+strconv.Itoa(i)), pos: p, service: geosaInt(wp, "ServiceDuration", 0)}
		for _, b := range geosaList(wp, "Before") {
			idx, _ := b.(float64)
			if idx < 0 || int(idx) >= len(raw) || int(idx) == i {
				return nil, geosaValidation("FieldValidationFailed", fmt.Sprintf("Waypoints[%d].Before", i), "Before must list indexes of other waypoints")
			}
			v.before = append(v.before, int(idx))
		}
		if ids[v.id] {
			return nil, geosaValidation("FieldValidationFailed", fmt.Sprintf("Waypoints[%d].Id", i), "waypoint Id %q is not unique", v.id)
		}
		ids[v.id] = true
		visits = append(visits, v)
	}
	var dest *grteVisit
	if d, ok := in["Destination"]; ok {
		p, err := geosaPos(d, "Destination")
		if err != nil {
			return nil, err
		}
		opts := geosaObj(in, "DestinationOptions")
		dest = &grteVisit{id: grteID(opts, "Destination"), pos: p, service: geosaInt(opts, "ServiceDuration", 0)}
	}
	mode := geosaStr(in, "TravelMode")
	if mode == "" {
		mode = "Car"
	}
	order, impeded := grteNearestOrder(origin, visits)
	stops := make([]grteVisit, 0, len(order)+1)
	for _, i := range order {
		stops = append(stops, visits[i])
	}
	if dest != nil {
		stops = append(stops, *dest)
	}
	return &geosaReply{pricing: true, body: grteItinerary(grteVisit{id: originID, pos: origin}, stops, impeded, visits, start, mode)}, nil
}

func grteID(opts map[string]any, def string) string {
	if id := geosaStr(opts, "Id"); id != "" {
		return id
	}
	return def
}

// grteNearestOrder visits the nearest eligible waypoint next. A waypoint is
// eligible once every waypoint listing it in Before has been visited;
// waypoints left over (constraint cycles) are impeding.
func grteNearestOrder(origin [2]float64, visits []grteVisit) ([]int, []int) {
	pending := make([]int, len(visits)) // count of unvisited predecessors
	for _, v := range visits {
		for _, b := range v.before {
			pending[b]++
		}
	}
	done := make([]bool, len(visits))
	var order []int
	cur := origin
	for len(order) < len(visits) {
		best := -1
		for i, v := range visits {
			if !done[i] && pending[i] == 0 && (best < 0 || geosaDistance(cur, v.pos) < geosaDistance(cur, visits[best].pos)) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		done[best], cur = true, visits[best].pos
		order = append(order, best)
		for _, b := range visits[best].before {
			pending[b]--
		}
	}
	var impeded []int
	for i := range visits {
		if !done[i] {
			impeded = append(impeded, i)
		}
	}
	return order, impeded
}

func grteItinerary(origin grteVisit, stops []grteVisit, impeded []int, visits []grteVisit, start time.Time, mode string) map[string]any {
	t := start
	optimized := []any{map[string]any{"Id": origin.id, "Position": geosaPosJSON(origin.pos), "DepartureTime": grteFormat(t)}}
	connections := []any{}
	var distance, travel, service int64
	prev := origin
	for _, s := range stops {
		d, dur := grteMeasure([][2]float64{prev.pos, s.pos}, mode)
		connections = append(connections, map[string]any{
			"From": prev.id, "To": s.id, "Distance": d, "TravelDuration": dur, "RestDuration": 0, "WaitDuration": 0,
		})
		arrive := t.Add(time.Duration(dur) * time.Second)
		t = arrive.Add(time.Duration(s.service) * time.Second)
		optimized = append(optimized, map[string]any{
			"Id": s.id, "Position": geosaPosJSON(s.pos), "ArrivalTime": grteFormat(arrive), "DepartureTime": grteFormat(t),
		})
		distance, travel, service = distance+d, travel+dur, service+s.service
		prev = s
	}
	impeding := []any{}
	for _, i := range impeded {
		impeding = append(impeding, map[string]any{
			"Id": visits[i].id, "Position": geosaPosJSON(visits[i].pos),
			"FailedConstraints": []any{map[string]any{"Constraint": "Before", "Reason": "Before constraints form a cycle."}},
		})
	}
	return map[string]any{
		"OptimizedWaypoints": optimized, "Connections": connections, "ImpedingWaypoints": impeding,
		"Distance": distance, "Duration": travel + service,
		"TimeBreakdown": map[string]any{"TravelDuration": travel, "ServiceDuration": service, "RestDuration": 0, "WaitDuration": 0},
	}
}

func grteSnapToRoads(c geosaCall) (*geosaReply, *geosaError) {
	in := c.in
	format := geosaStr(in, "SnappedGeometryFormat")
	if format == "" {
		format = "FlexiblePolyline"
	}
	traces := geosaList(in, "TracePoints")
	points := make([][2]float64, 0, len(traces))
	snapped := make([]any, 0, len(traces))
	var outOfOrder []any
	var last *time.Time
	for i, raw := range traces {
		tp, _ := raw.(map[string]any)
		field := fmt.Sprintf("TracePoints[%d]", i)
		p, err := geosaPos(tp["Position"], field+".Position")
		if err != nil {
			return nil, err
		}
		ts, err := grteTime(tp, "Timestamp")
		if err != nil {
			return nil, geosaValidation("CannotParse", field+".Timestamp", "%s.Timestamp must be an ISO 8601 timestamp with an offset", field)
		}
		if ts != nil {
			if last != nil && ts.Before(*last) {
				outOfOrder = append(outOfOrder, i)
			}
			last = ts
		}
		points = append(points, p)
		snapped = append(snapped, map[string]any{"OriginalPosition": geosaPosJSON(p), "SnappedPosition": geosaPosJSON(p), "Confidence": grteSnapConfidence})
	}
	notices := []any{}
	if len(outOfOrder) > 0 {
		notices = append(notices, map[string]any{"Code": "TracePointsOutOfSequence", "Title": "Trace points are out of sequence.", "TracePointIndexes": outOfOrder})
	}
	return &geosaReply{pricing: true, body: map[string]any{
		"Notices": notices, "SnappedGeometryFormat": format, "SnappedTracePoints": snapped, "SnappedGeometry": grteGeometry(points, format),
	}}, nil
}
