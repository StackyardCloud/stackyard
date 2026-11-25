// Command chime-sdk-media-pipelines runs Amazon Chime SDK Media Pipelines
// against Stackyard: record a Chime SDK meeting with a capture pipeline,
// concatenate its artifacts, stream its audio into a Kinesis video stream
// pool, analyse a call with a media insights pipeline and a voice tone
// analysis task, and watch the capture pipeline stop when the meeting ends.
//
// The SDK has no Chime SDK modules in this repository's go.mod, so requests
// are signed by hand.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// The Chime SDK APIs all sign as chime.
const signingName = "chime"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Chime SDK Media Pipelines client using %s\n", c.endpoint)
	run := fmt.Sprint(time.Now().UnixNano())
	meeting := c.rest("CreateMeeting", http.MethodPost, "/meetings", "", map[string]any{
		"ClientRequestToken": "m-" + run, "ExternalMeetingId": "webinar-" + run, "MediaRegion": "us-east-1",
	})["Meeting"].(map[string]any)
	meetingArn := meeting["MeetingArn"].(string)

	capture := c.rest("CreateMediaCapturePipeline", http.MethodPost, "/sdk-media-capture-pipelines", "", map[string]any{
		"SourceType": "ChimeSdkMeeting", "SourceArn": meetingArn, "SinkType": "S3Bucket", "SinkArn": "arn:aws:s3:::webinar-recordings",
		"ClientRequestToken": "cap-" + run,
	})["MediaCapturePipeline"].(map[string]any)
	captureID := capture["MediaPipelineId"].(string)
	fmt.Printf("capture pipeline %s is %v\n", captureID, capture["Status"])
	status := capture["Status"]
	for i := 0; i < 20 && status != "InProgress"; i++ {
		time.Sleep(250 * time.Millisecond)
		status = c.rest("GetMediaCapturePipeline", http.MethodGet, "/sdk-media-capture-pipelines/"+captureID, "", nil)["MediaCapturePipeline"].(map[string]any)["Status"]
	}
	fmt.Printf("capture pipeline is %v\n", status)

	on := map[string]any{"State": "Enabled"}
	concat := c.rest("CreateMediaConcatenationPipeline", http.MethodPost, "/sdk-media-concatenation-pipelines", "", map[string]any{
		"Sources": []any{map[string]any{"Type": "MediaCapturePipeline", "MediaCapturePipelineSourceConfiguration": map[string]any{
			"MediaPipelineArn": capture["MediaPipelineArn"], "ChimeSdkMeetingConfiguration": map[string]any{"ArtifactsConfiguration": map[string]any{
				"Audio": on, "Video": on, "Content": on, "DataChannel": on, "TranscriptionMessages": on, "MeetingEvents": on,
				"CompositedVideo": map[string]any{"State": "Disabled"}}}}}},
		"Sinks": []any{map[string]any{"Type": "S3Bucket", "S3BucketSinkConfiguration": map[string]any{"Destination": "arn:aws:s3:::webinar-final"}}},
	})["MediaConcatenationPipeline"].(map[string]any)
	fmt.Printf("concatenation pipeline %v\n", concat["MediaPipelineId"])

	pool := c.rest("CreateMediaPipelineKinesisVideoStreamPool", http.MethodPost, "/media-pipeline-kinesis-video-stream-pools", "", map[string]any{
		"PoolName": "webinar-audio-" + run, "StreamConfiguration": map[string]any{"Region": "us-east-1", "DataRetentionInHours": 24},
	})["KinesisVideoStreamPoolConfiguration"].(map[string]any)
	stream := c.rest("CreateMediaStreamPipeline", http.MethodPost, "/sdk-media-stream-pipelines", "", map[string]any{
		"Sources": []any{map[string]any{"SourceType": "ChimeSdkMeeting", "SourceArn": meetingArn}},
		"Sinks":   []any{map[string]any{"SinkArn": pool["PoolArn"], "SinkType": "KinesisVideoStreamPool", "ReservedStreamCapacity": 2, "MediaStreamType": "IndividualAudio"}},
	})["MediaStreamPipeline"].(map[string]any)
	poolNow := c.rest("GetMediaPipelineKinesisVideoStreamPool", http.MethodGet, "/media-pipeline-kinesis-video-stream-pools/"+url.PathEscape(pool["PoolName"].(string)), "", nil)
	fmt.Printf("stream pipeline %v reserves %v streams in pool %v\n", stream["MediaPipelineId"],
		poolNow["KinesisVideoStreamPoolConfiguration"].(map[string]any)["PoolSize"], pool["PoolName"])

	cfg := c.rest("CreateMediaInsightsPipelineConfiguration", http.MethodPost, "/media-insights-pipeline-configurations", "", map[string]any{
		"MediaInsightsPipelineConfigurationName": "call-insights-" + run, "ResourceAccessRoleArn": "arn:aws:iam::123456789012:role/insights",
		"Elements": []any{
			map[string]any{"Type": "VoiceAnalyticsProcessor", "VoiceAnalyticsProcessorConfiguration": map[string]any{"VoiceToneAnalysisStatus": "Enabled"}},
			map[string]any{"Type": "SnsTopicSink", "SnsTopicSinkConfiguration": map[string]any{"InsightsTarget": "arn:aws:sns:us-east-1:123456789012:insights"}},
		},
	})["MediaInsightsPipelineConfiguration"].(map[string]any)
	insights := c.rest("CreateMediaInsightsPipeline", http.MethodPost, "/media-insights-pipelines", "", map[string]any{
		"MediaInsightsPipelineConfigurationArn": cfg["MediaInsightsPipelineConfigurationArn"],
		"KinesisVideoStreamSourceRuntimeConfiguration": map[string]any{"MediaEncoding": "pcm", "MediaSampleRate": 8000,
			"Streams": []any{map[string]any{"StreamArn": "arn:aws:kinesisvideo:us-east-1:123456789012:stream/call/1", "StreamChannelDefinition": map[string]any{"NumberOfChannels": 2}}}},
	})["MediaInsightsPipeline"].(map[string]any)
	insightsID := insights["MediaPipelineId"].(string)
	task := c.rest("StartVoiceToneAnalysisTask", http.MethodPost, "/media-insights-pipelines/"+insightsID+"/voice-tone-analysis-tasks?operation=start", "", map[string]any{
		"LanguageCode": "en-US",
	})["VoiceToneAnalysisTask"].(map[string]any)
	taskPath := "/media-insights-pipelines/" + insightsID + "/voice-tone-analysis-tasks/" + task["VoiceToneAnalysisTaskId"].(string)
	c.rest("StopVoiceToneAnalysisTask", http.MethodPost, taskPath+"?operation=stop", "", nil)
	fmt.Printf("voice tone task %v\n", c.rest("GetVoiceToneAnalysisTask", http.MethodGet, taskPath, "", nil)["VoiceToneAnalysisTask"].(map[string]any)["VoiceToneAnalysisTaskStatus"])
	c.rest("UpdateMediaInsightsPipelineStatus", http.MethodPut, "/media-insights-pipeline-status/"+insightsID, "", map[string]any{"UpdateStatus": "Pause"})

	c.rest("DeleteMeeting", http.MethodDelete, "/meetings/"+meeting["MeetingId"].(string), "", nil)
	after := c.rest("GetMediaPipeline", http.MethodGet, "/sdk-media-pipelines/"+captureID, "", nil)["MediaPipeline"].(map[string]any)
	fmt.Printf("after the meeting ended the capture pipeline is %v\n", after["MediaCapturePipeline"].(map[string]any)["Status"])

	for _, id := range []any{stream["MediaPipelineId"], concat["MediaPipelineId"], insightsID} {
		c.rest("DeleteMediaPipeline", http.MethodDelete, fmt.Sprint("/sdk-media-pipelines/", id), "", nil)
	}
	c.rest("DeleteMediaCapturePipeline", http.MethodDelete, "/sdk-media-capture-pipelines/"+captureID, "", nil)
	c.rest("DeleteMediaPipelineKinesisVideoStreamPool", http.MethodDelete, "/media-pipeline-kinesis-video-stream-pools/"+url.PathEscape(pool["PoolName"].(string)), "", nil)
	c.rest("DeleteMediaInsightsPipelineConfiguration", http.MethodDelete, "/media-insights-pipeline-configurations/"+url.PathEscape(cfg["MediaInsightsPipelineConfigurationName"].(string)), "", nil)
	fmt.Println("cleaned up")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// rest sends one REST JSON operation signed for chime, acting as bearer
// when it is set, and exits unless it returns 2xx.
func (c client) rest(op, method, path, bearer string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if bearer != "" {
		headers["X-Amz-Chime-Bearer"] = bearer
	}
	raw := c.send(op, method, c.endpoint+path, body, headers)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request and exits unless it returns 2xx.
func (c client) send(op, method, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, signingName, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
