package server

// Amazon Location Service Maps V2 (geo-maps): style descriptors, tiles,
// sprites, glyphs and static maps. Everything is a small, valid placeholder
// generated in-process: empty-but-valid vector tiles, solid raster tiles and
// static maps, a one-icon sprite sheet, and glyph PBFs with no glyphs.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const gmapMaxZoom = 22

var gmapService = &geosaService{
	signingName: "geo-maps",
	hosts:       []string{"geo-maps.", "geo-maps-fips.", "maps.geo.", "maps.geo-fips."},
	model:       awsmodels.GeoMapsModel,
	ops:         geomapsOperations,
	handlers: map[string]func(geosaCall) (*geosaReply, *geosaError){
		"GetGlyphs":          gmapGetGlyphs,
		"GetSprites":         gmapGetSprites,
		"GetStaticMap":       gmapGetStaticMap,
		"GetStyleDescriptor": gmapGetStyleDescriptor,
		"GetTile":            gmapGetTile,
	},
	write: gmapWrite,
}

func (s *Server) handleGeoMapsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.geosaServe(w, r, gmapService)
}

func gmapGetStyleDescriptor(c geosaCall) (*geosaReply, *geosaError) {
	style := geosaStr(c.in, "Style")
	scheme := geosaStr(c.in, "ColorScheme")
	if scheme == "" {
		scheme = "Light"
	}
	background := "#f2efe9"
	if scheme == "Dark" {
		background = "#1f2125"
	}
	sources := map[string]any{
		"omv": map[string]any{"type": "vector", "tiles": []string{c.base + "/v2/tiles/vector.basemap/{z}/{x}/{y}"}, "minzoom": 0, "maxzoom": 15},
	}
	layers := []any{map[string]any{"id": "background", "type": "background", "paint": map[string]any{"background-color": background}}}
	if style == "Satellite" || style == "Hybrid" {
		sources["satellite"] = map[string]any{"type": "raster", "tiles": []string{c.base + "/v2/tiles/raster.satellite/{z}/{x}/{y}"}, "tileSize": 256}
		layers = append(layers, map[string]any{"id": "satellite", "type": "raster", "source": "satellite"})
	}
	doc := map[string]any{
		"version":  8,
		"name":     style,
		"metadata": map[string]any{"stackyard:colorScheme": scheme, "stackyard:politicalView": geosaStr(c.in, "PoliticalView")},
		"sources":  sources,
		"sprite":   fmt.Sprintf("%s/v2/styles/%s/%s/Default/sprites/sprites", c.base, style, scheme),
		"glyphs":   c.base + "/v2/glyphs/{fontstack}/{range}.pbf",
		"layers":   layers,
	}
	raw, _ := json.Marshal(doc)
	return &geosaReply{blob: raw, kind: geosaBlobJSON, maxAge: 86400}, nil
}

func gmapGetTile(c geosaCall) (*geosaReply, *geosaError) {
	tileset := geosaStr(c.in, "Tileset")
	z, err := gmapTileCoord(c.in, "Z", gmapMaxZoom)
	if err != nil {
		return nil, err
	}
	limit := int64(1)<<z - 1
	for _, field := range []string{"X", "Y"} {
		if _, err := gmapTileCoord(c.in, field, limit); err != nil {
			return nil, err
		}
	}
	switch {
	case strings.HasPrefix(tileset, "vector."):
		return &geosaReply{blob: gmapVectorTile, kind: geosaBlobVectorTile, pricing: true, maxAge: 86400}, nil
	case tileset == "raster.dem":
		return &geosaReply{blob: gmapDEMTile(), kind: geosaBlobPNG, pricing: true, maxAge: 86400}, nil
	case strings.HasPrefix(tileset, "raster."):
		return &geosaReply{blob: gmapSatelliteTile(), kind: geosaBlobJPEG, pricing: true, maxAge: 86400}, nil
	}
	return nil, &geosaError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: fmt.Sprintf("tileset %q does not exist", tileset)}
}

// gmapTileCoord parses a tile coordinate path member within [0, max].
func gmapTileCoord(in map[string]any, field string, max int64) (int64, *geosaError) {
	n, err := strconv.ParseInt(geosaStr(in, field), 10, 32)
	if err != nil || n < 0 || n > max {
		return 0, geosaValidation("FieldValidationFailed", field, "%s must be an integer between 0 and %d", field, max)
	}
	return n, nil
}

func gmapGetSprites(c geosaCall) (*geosaReply, *geosaError) {
	name := geosaStr(c.in, "FileName")
	ratio := 1
	if strings.Contains(name, "@2x") {
		ratio = 2
	}
	if strings.HasSuffix(name, ".json") {
		size := 16 * ratio
		raw, _ := json.Marshal(map[string]any{
			"stackyard-marker": map[string]any{"x": 0, "y": 0, "width": size, "height": size, "pixelRatio": ratio},
		})
		return &geosaReply{blob: raw, kind: geosaBlobJSON, maxAge: 86400}, nil
	}
	return &geosaReply{blob: gmapSprite(ratio), kind: geosaBlobPNG, maxAge: 86400}, nil
}

func gmapGetGlyphs(c geosaCall) (*geosaReply, *geosaError) {
	rng := strings.TrimSuffix(geosaStr(c.in, "FontUnicodeRange"), ".pbf")
	lo, hi, _ := strings.Cut(rng, "-")
	start, err1 := strconv.ParseInt(lo, 10, 32)
	end, err2 := strconv.ParseInt(hi, 10, 32)
	if err1 != nil || err2 != nil || start%256 != 0 || end != start+255 || end > 65535 {
		return nil, geosaValidation("FieldValidationFailed", "FontUnicodeRange", "FontUnicodeRange must be a 256-codepoint block such as 0-255.pbf")
	}
	font := geosaStr(c.in, "FontStack")
	if strings.TrimSpace(font) == "" {
		return nil, geosaValidation("FieldValidationFailed", "FontStack", "FontStack must name a font")
	}
	return &geosaReply{blob: gmapGlyphPBF(font, rng), kind: geosaBlobProtobuf, maxAge: 86400}, nil
}

func gmapGetStaticMap(c geosaCall) (*geosaReply, *geosaError) {
	in := c.in
	anchors := 0
	for _, k := range []string{"Center", "BoundingBox", "BoundedPositions"} {
		if geosaStr(in, k) != "" {
			anchors++
		}
	}
	overlay := geosaStr(in, "GeoJsonOverlay") != "" || geosaStr(in, "CompactOverlay") != ""
	if anchors > 1 {
		return nil, geosaValidation("FieldValidationFailed", "Center", "Center, BoundingBox and BoundedPositions are mutually exclusive")
	}
	if anchors == 0 && !overlay {
		return nil, geosaValidation("Missing", "Center", "one of Center, BoundingBox, BoundedPositions, GeoJsonOverlay or CompactOverlay is required")
	}
	_, hasZoom := in["Zoom"]
	_, hasRadius := in["Radius"]
	if hasZoom && hasRadius {
		return nil, geosaValidation("FieldValidationFailed", "Zoom", "Zoom and Radius are mutually exclusive")
	}
	if geosaStr(in, "Center") == "" && (hasZoom || hasRadius) {
		return nil, geosaValidation("FieldValidationFailed", "Zoom", "Zoom and Radius apply only with Center")
	}
	for _, k := range []string{"Center", "BoundingBox", "BoundedPositions"} {
		if err := gmapCheckPositions(in, k); err != nil {
			return nil, err
		}
	}
	if raw := geosaStr(in, "GeoJsonOverlay"); raw != "" {
		var doc map[string]any
		if json.Unmarshal([]byte(raw), &doc) != nil || geosaStr(doc, "type") == "" {
			return nil, geosaValidation("CannotParse", "GeoJsonOverlay", "GeoJsonOverlay must be a GeoJSON object")
		}
	}
	scale := int64(1)
	if strings.HasSuffix(geosaStr(in, "FileName"), "@2x") {
		scale = 2
	}
	width, height := geosaInt(in, "Width", 64)*scale, geosaInt(in, "Height", 64)*scale
	fill := color.RGBA{0xf2, 0xef, 0xe9, 0xff}
	switch {
	case geosaStr(in, "Style") == "Satellite":
		fill = color.RGBA{0x2f, 0x4a, 0x2c, 0xff}
	case geosaStr(in, "ColorScheme") == "Dark":
		fill = color.RGBA{0x1f, 0x21, 0x25, 0xff}
	}
	return &geosaReply{blob: gmapJPEG(int(width), int(height), fill), kind: geosaBlobJPEG, pricing: true, maxAge: 3600}, nil
}

// gmapCheckPositions range-checks a "lon,lat,lon,lat..." query member (the
// model pattern already fixed its syntax).
func gmapCheckPositions(in map[string]any, field string) *geosaError {
	raw := geosaStr(in, field)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := 0; i+1 < len(parts); i += 2 {
		lon, err1 := strconv.ParseFloat(parts[i], 64)
		lat, err2 := strconv.ParseFloat(parts[i+1], 64)
		if err1 != nil || err2 != nil {
			return geosaValidation("CannotParse", field, "%s must list longitude,latitude pairs", field)
		}
		if _, err := geosaPos([]any{lon, lat}, field); err != nil {
			return err
		}
	}
	return nil
}

// gmapWrite sends a binary payload with one of a fixed set of content types.
func gmapWrite(w http.ResponseWriter, rep *geosaReply) {
	h := w.Header()
	if rep.pricing {
		h.Set("X-Amz-Geo-Pricing-Bucket", geosaPricingBucket)
	}
	switch rep.kind {
	case geosaBlobJSON:
		h.Set("Content-Type", "application/json")
	case geosaBlobPNG:
		h.Set("Content-Type", "image/png")
	case geosaBlobJPEG:
		h.Set("Content-Type", "image/jpeg")
	case geosaBlobProtobuf:
		h.Set("Content-Type", "application/x-protobuf")
	case geosaBlobVectorTile:
		h.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	default:
		h.Set("Content-Type", "application/octet-stream")
	}
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "public, max-age="+strconv.Itoa(rep.maxAge))
	sum := sha256.Sum256(rep.blob)
	h.Set("ETag", `"`+hex.EncodeToString(sum[:16])+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(rep.blob)
}

// --- placeholder payloads --------------------------------------------------

// gmapVectorTile is a Mapbox Vector Tile with one empty version-2 layer
// named "stackyard" (extent 4096).
var gmapVectorTile = func() []byte {
	layer := []byte{0x78, 0x02, 0x0a, byte(len("stackyard"))}
	layer = append(layer, "stackyard"...)
	layer = append(layer, 0x28, 0x80, 0x20)
	return append([]byte{0x1a, byte(len(layer))}, layer...)
}()

// gmapDEMTile is a 256x256 terrain-RGB tile at sea level:
// height = -10000 + (R*65536 + G*256 + B) * 0.1 = 0.
var gmapDEMTile = sync.OnceValue(func() []byte {
	return gmapPNG(256, color.RGBA{0x01, 0x86, 0xa0, 0xff})
})

var gmapSatelliteTile = sync.OnceValue(func() []byte {
	return gmapJPEG(256, 256, color.RGBA{0x2f, 0x4a, 0x2c, 0xff})
})

var gmapSprites = [2]func() []byte{
	sync.OnceValue(func() []byte { return gmapPNG(16, color.RGBA{0xe7, 0x4c, 0x3c, 0xff}) }),
	sync.OnceValue(func() []byte { return gmapPNG(32, color.RGBA{0xe7, 0x4c, 0x3c, 0xff}) }),
}

func gmapSprite(ratio int) []byte { return gmapSprites[ratio-1]() }

func gmapPNG(size int, fill color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = fill.R, fill.G, fill.B, fill.A
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// gmapJPEG encodes a solid image; sizes are bounded by the model (1400x1400,
// doubled for @2x).
func gmapJPEG(width, height int, fill color.RGBA) []byte {
	img := image.NewUniform(fill)
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, gmapSized{img, image.Rect(0, 0, width, height)}, &jpeg.Options{Quality: 60})
	return buf.Bytes()
}

// gmapSized gives a uniform image finite bounds.
type gmapSized struct {
	*image.Uniform
	rect image.Rectangle
}

func (g gmapSized) Bounds() image.Rectangle { return g.rect }

// gmapGlyphPBF encodes a glyphs message holding one fontstack (name, range)
// and no glyphs, the shape MapLibre expects for a range with nothing to draw.
func gmapGlyphPBF(font, rng string) []byte {
	var stack []byte
	stack = gmapPBFString(stack, 1, font)
	stack = gmapPBFString(stack, 2, rng)
	return gmapPBFBytes(nil, 1, stack)
}

func gmapPBFString(buf []byte, field int, s string) []byte {
	return gmapPBFBytes(buf, field, []byte(s))
}

func gmapPBFBytes(buf []byte, field int, b []byte) []byte {
	buf = gmapVarint(buf, uint64(field)<<3|2)
	buf = gmapVarint(buf, uint64(len(b)))
	return append(buf, b...)
}

func gmapVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}
