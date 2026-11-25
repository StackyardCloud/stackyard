package server

import (
	"math"
	"strings"
)

// grteFlexAlphabet is the flexible polyline URL-safe alphabet
// (https://github.com/heremaps/flexible-polyline).
const grteFlexAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// grteFlexPrecision is the number of decimal places kept for each coordinate.
const grteFlexPrecision = 6

// grteFlexEncode encodes [longitude, latitude] points as a 2D flexible
// polyline (which stores latitude first).
func grteFlexEncode(points [][2]float64, precision int) string {
	var b strings.Builder
	grteFlexUnsigned(&b, 1)                 // format version
	grteFlexUnsigned(&b, uint64(precision)) // header: precision, no third dimension
	mult := math.Pow(10, float64(precision))
	var lastLat, lastLon int64
	for _, p := range points {
		lat, lon := int64(math.Round(p[1]*mult)), int64(math.Round(p[0]*mult))
		grteFlexSigned(&b, lat-lastLat)
		grteFlexSigned(&b, lon-lastLon)
		lastLat, lastLon = lat, lon
	}
	return b.String()
}

func grteFlexSigned(b *strings.Builder, v int64) {
	u := uint64(v) << 1
	if v < 0 {
		u = ^u
	}
	grteFlexUnsigned(b, u)
}

func grteFlexUnsigned(b *strings.Builder, v uint64) {
	for v > 0x1f {
		b.WriteByte(grteFlexAlphabet[(v&0x1f)|0x20])
		v >>= 5
	}
	b.WriteByte(grteFlexAlphabet[v])
}

// grteCircle is a closed ring of n vertices radius metres around c.
func grteCircle(c [2]float64, radius float64, n int) [][2]float64 {
	ring := make([][2]float64, 0, n+1)
	for i := 0; i < n; i++ {
		ring = append(ring, geosaOffset(c, float64(i)*360/float64(n), radius))
	}
	return append(ring, ring[0])
}

func grteLineJSON(points [][2]float64) []any {
	out := make([]any, len(points))
	for i, p := range points {
		out[i] = geosaPosJSON(p)
	}
	return out
}

// grteGeometry renders a line as LineString or a flexible Polyline.
func grteGeometry(points [][2]float64, format string) map[string]any {
	if format == "Simple" {
		return map[string]any{"LineString": grteLineJSON(points)}
	}
	return map[string]any{"Polyline": grteFlexEncode(points, grteFlexPrecision)}
}
