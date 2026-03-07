package server

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPMapsSolarRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}
	if !isGCPMapsSolarPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.maps.solar.v1.Solar/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPMapsSolarFindClosestBuildingInsights(w, r, path) {
			return true
		}
		if handleGCPMapsSolarGetDataLayers(w, r, path) {
			return true
		}
		if handleGCPMapsSolarGetGeoTiff(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMapsSolarPath(path string) bool {
	normalizedPath := normalizeGCPMapsSolarPath(path)
	if normalizedPath == "/gcp/v1/buildingInsights:findClosest" {
		return true
	}
	if normalizedPath == "/gcp/v1/dataLayers:get" {
		return true
	}
	if normalizedPath == "/gcp/v1/geoTiff:get" {
		return true
	}
	return normalizedPath == "/gcp/google.maps.solar.v1.Solar/FindClosestBuildingInsights" ||
		normalizedPath == "/gcp/google.maps.solar.v1.Solar/GetDataLayers" ||
		normalizedPath == "/gcp/google.maps.solar.v1.Solar/GetGeoTiff"
}

func handleGCPMapsSolarFindClosestBuildingInsights(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPMapsSolarPath(path) != "/gcp/v1/buildingInsights:findClosest" {
		return false
	}

	lat, lng, ok := parseGCPMapsSolarLatLng(w, path, r)
	if !ok {
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"name": "buildingInsights/stackyard-building-1",
		"center": map[string]any{
			"latitude":  lat,
			"longitude": lng,
		},
		"imageryDate": map[string]any{
			"year":  2024,
			"month": 1,
			"day":   15,
		},
		"imageryProcessedDate": map[string]any{
			"year":  2024,
			"month": 2,
			"day":   1,
		},
		"postalCode":         "94105",
		"administrativeArea": "CA",
		"regionCode":         "US",
		"solarPotential": map[string]any{
			"maxArrayPanelsCount":        12,
			"maxArrayAreaMeters2":        24.5,
			"maxSunshineHoursPerYear":    1820.0,
			"carbonOffsetFactorKgPerMwh": 80.0,
			"panelCapacityWatts":         400,
			"panelHeightMeters":          1.7,
			"panelWidthMeters":           1.0,
			"panelLifetimeYears":         20,
		},
	})
	return true
}

func handleGCPMapsSolarGetDataLayers(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPMapsSolarPath(path) != "/gcp/v1/dataLayers:get" {
		return false
	}

	_, _, ok := parseGCPMapsSolarLatLng(w, path, r)
	if !ok {
		return true
	}

	radiusRaw := strings.TrimSpace(r.URL.Query().Get("radiusMeters"))
	if radiusRaw == "" {
		respondGCPMapsSolarInvalidArgument(w, path, "radiusMeters is required")
		return true
	}
	radius, err := strconv.ParseFloat(radiusRaw, 64)
	if err != nil || radius <= 0 {
		respondGCPMapsSolarInvalidArgument(w, path, "radiusMeters must be a positive number")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"imageryDate": map[string]any{
			"year":  2024,
			"month": 1,
			"day":   15,
		},
		"imageryProcessedDate": map[string]any{
			"year":  2024,
			"month": 2,
			"day":   1,
		},
		"dsmUrl":         "https://storage.googleapis.com/stackyard-solar/dsm.tif",
		"rgbUrl":         "https://storage.googleapis.com/stackyard-solar/rgb.tif",
		"maskUrl":        "https://storage.googleapis.com/stackyard-solar/mask.tif",
		"annualFluxUrl":  "https://storage.googleapis.com/stackyard-solar/annual-flux.tif",
		"monthlyFluxUrl": "https://storage.googleapis.com/stackyard-solar/monthly-flux.tif",
		"hourlyShadeUrls": []any{
			"https://storage.googleapis.com/stackyard-solar/hourly-shade-jan.tif",
		},
	})
	return true
}

func handleGCPMapsSolarGetGeoTiff(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPMapsSolarPath(path) != "/gcp/v1/geoTiff:get" {
		return false
	}

	assetID := strings.TrimSpace(r.URL.Query().Get("id"))
	if assetID == "" {
		respondGCPMapsSolarInvalidArgument(w, path, "id is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"contentType": "image/tiff",
		"data":        "U1RBQ0tZQVJE",
	})
	return true
}

func parseGCPMapsSolarLatLng(w http.ResponseWriter, path string, r *http.Request) (lat, lng float64, ok bool) {
	latRaw := strings.TrimSpace(r.URL.Query().Get("location.latitude"))
	lngRaw := strings.TrimSpace(r.URL.Query().Get("location.longitude"))
	if latRaw == "" {
		respondGCPMapsSolarInvalidArgument(w, path, "location.latitude is required")
		return 0, 0, false
	}
	if lngRaw == "" {
		respondGCPMapsSolarInvalidArgument(w, path, "location.longitude is required")
		return 0, 0, false
	}

	latValue, err := strconv.ParseFloat(latRaw, 64)
	if err != nil {
		respondGCPMapsSolarInvalidArgument(w, path, "location.latitude must be a number")
		return 0, 0, false
	}
	lngValue, err := strconv.ParseFloat(lngRaw, 64)
	if err != nil {
		respondGCPMapsSolarInvalidArgument(w, path, "location.longitude must be a number")
		return 0, 0, false
	}
	return latValue, lngValue, true
}

func normalizeGCPMapsSolarPath(path string) string {
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func respondGCPMapsSolarInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
