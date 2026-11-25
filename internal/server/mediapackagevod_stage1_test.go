package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const mpvodTestRole = "arn:aws:iam::123456789012:role/mediapackage-vod"

func TestMediaPackageVODStage1Lifecycle(t *testing.T) {
	ts := newMediaPackageVODTestServer(t)

	// Packaging groups.
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_groups", `{"id":"bad id"}`)
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_groups", `{}`)
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_groups",
		`{"id":"g1","authorization":{"cdnIdentifierSecret":"nope","secretsRoleArn":"`+mpvodTestRole+`"}}`)
	g := mpvodCall(t, ts, http.StatusOK, http.MethodPost, "/packaging_groups", `{"id":"g1","egressAccessLogs":{},"tags":{"env":"dev"}}`)
	if g["arn"] != "arn:aws:mediapackage-vod:us-east-1:123456789012:packaging-groups/g1" || !strings.HasPrefix(g["domainName"].(string), "https://") ||
		g["egressAccessLogs"].(map[string]any)["logGroupName"] != mpvodDefaultLogs {
		t.Fatalf("group: %v", g)
	}
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_groups", `{"id":"g1"}`)
	mpvodCall(t, ts, http.StatusNotFound, http.MethodGet, "/packaging_groups/nope", "")
	secret := "arn:aws:secretsmanager:us-east-1:123456789012:secret:cdn-abc"
	up := mpvodCall(t, ts, http.StatusOK, http.MethodPut, "/packaging_groups/g1",
		`{"authorization":{"cdnIdentifierSecret":"`+secret+`","secretsRoleArn":"`+mpvodTestRole+`"}}`)
	if up["authorization"].(map[string]any)["cdnIdentifierSecret"] != secret {
		t.Fatalf("update: %v", up)
	}
	logs := mpvodCall(t, ts, http.StatusOK, http.MethodPut, "/packaging_groups/g1/configure_logs", `{"egressAccessLogs":{"logGroupName":"/aws/MediaPackage/custom"}}`)
	if logs["egressAccessLogs"].(map[string]any)["logGroupName"] != "/aws/MediaPackage/custom" || logs["authorization"] == nil {
		t.Fatalf("configure logs: %v", logs)
	}
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPut, "/packaging_groups/g1/configure_logs", `{"egressAccessLogs":{"logGroupName":"elsewhere"}}`)

	// Packaging configurations.
	hls := `"hlsPackage":{"hlsManifests":[{"manifestName":"main","adMarkers":"NONE"}],"segmentDurationSeconds":6}`
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_configurations", `{"id":"c1","packagingGroupId":"nope",`+hls+`}`)
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_configurations", `{"id":"c1","packagingGroupId":"g1"}`)
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_configurations",
		`{"id":"c1","packagingGroupId":"g1","hlsPackage":{"hlsManifests":[{"adMarkers":"SOMETIMES"}]}}`)
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/packaging_configurations",
		`{"id":"c1","packagingGroupId":"g1","hlsPackage":{"hlsManifests":[{"manifestName":"a"},{"manifestName":"a"}]}}`)
	c := mpvodCall(t, ts, http.StatusOK, http.MethodPost, "/packaging_configurations", `{"id":"c1","packagingGroupId":"g1",`+hls+`}`)
	if c["hlsPackage"].(map[string]any)["segmentDurationSeconds"] != float64(6) {
		t.Fatalf("config: %v", c)
	}
	mpvodCall(t, ts, http.StatusOK, http.MethodPost, "/packaging_configurations",
		`{"id":"c2","packagingGroupId":"g1","dashPackage":{"dashManifests":[{"profile":"NONE"}]}}`)
	if got := mpvodCall(t, ts, http.StatusOK, http.MethodGet, "/packaging_configurations?packagingGroupId=g1&maxResults=1", ""); len(got["packagingConfigurations"].([]any)) != 1 || got["nextToken"] == nil {
		t.Fatalf("list configs: %v", got)
	}

	// Assets need an existing S3 source.
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/assets",
		`{"id":"a1","packagingGroupId":"g1","sourceArn":"arn:aws:s3:::vod-src/movie.smil","sourceRoleArn":"`+mpvodTestRole+`"}`)
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/vod-src", nil, nil, "s3")
	resp.Body.Close()
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/vod-src/movie.smil", []byte("<smil/>"), nil, "s3")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("s3 put: %d", resp.StatusCode)
	}
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodPost, "/assets",
		`{"id":"a1","packagingGroupId":"g1","sourceArn":"arn:aws:s3:::vod-src/movie.smil","sourceRoleArn":"not-a-role"}`)
	a := mpvodCall(t, ts, http.StatusOK, http.MethodPost, "/assets",
		`{"id":"a1","packagingGroupId":"g1","sourceArn":"arn:aws:s3:::vod-src/movie.smil","sourceRoleArn":"`+mpvodTestRole+`"}`)
	eps := a["egressEndpoints"].([]any)
	if len(eps) != 2 || a["resourceId"] != "a1" || eps[0].(map[string]any)["status"] != "QUEUED" ||
		!strings.HasSuffix(eps[0].(map[string]any)["url"].(string), "/main.m3u8") || !strings.HasSuffix(eps[1].(map[string]any)["url"].(string), "/index.mpd") {
		t.Fatalf("asset: %v", a)
	}
	d := mpvodCall(t, ts, http.StatusOK, http.MethodGet, "/assets/a1", "")
	if d["egressEndpoints"].([]any)[0].(map[string]any)["status"] != "PLAYABLE" {
		t.Fatalf("describe asset: %v", d)
	}
	if got := mpvodCall(t, ts, http.StatusOK, http.MethodGet, "/packaging_groups/g1", ""); got["approximateAssetCount"] != float64(1) {
		t.Fatalf("asset count: %v", got)
	}
	if got := mpvodCall(t, ts, http.StatusOK, http.MethodGet, "/assets?packagingGroupId=g1", ""); len(got["assets"].([]any)) != 1 {
		t.Fatalf("list assets: %v", got)
	}

	// Tags on every resource ARN.
	arn := url.PathEscape(a["arn"].(string))
	mpvodCall(t, ts, http.StatusNoContent, http.MethodPost, "/tags/"+arn, `{"tags":{"team":"video","tmp":"x"}}`)
	mpvodCall(t, ts, http.StatusNoContent, http.MethodDelete, "/tags/"+arn+"?tagKeys=tmp", "")
	if tags := mpvodCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any); len(tags) != 1 || tags["team"] != "video" {
		t.Fatalf("tags: %v", tags)
	}
	mpvodCall(t, ts, http.StatusNotFound, http.MethodGet, "/tags/"+url.PathEscape("arn:aws:mediapackage-vod:us-east-1:123456789012:assets/nope"), "")

	// Groups in use can't be deleted.
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodDelete, "/packaging_groups/g1", "")
	mpvodCall(t, ts, http.StatusAccepted, http.MethodDelete, "/assets/a1", "")
	mpvodCall(t, ts, http.StatusUnprocessableEntity, http.MethodDelete, "/packaging_groups/g1", "")
	mpvodCall(t, ts, http.StatusAccepted, http.MethodDelete, "/packaging_configurations/c1", "")
	mpvodCall(t, ts, http.StatusAccepted, http.MethodDelete, "/packaging_configurations/c2", "")
	mpvodCall(t, ts, http.StatusAccepted, http.MethodDelete, "/packaging_groups/g1", "")
	mpvodCall(t, ts, http.StatusNotFound, http.MethodDelete, "/packaging_groups/g1", "")
	if got := mpvodCall(t, ts, http.StatusOK, http.MethodGet, "/packaging_groups", ""); len(got["packagingGroups"].([]any)) != 0 {
		t.Fatalf("groups left: %v", got)
	}
}
