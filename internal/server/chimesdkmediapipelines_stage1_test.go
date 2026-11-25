package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const cmplTestConfig = `{"MediaInsightsPipelineConfigurationName":"calls","ResourceAccessRoleArn":"arn:aws:iam::123456789012:role/insights",
	"Elements":[{"Type":"VoiceAnalyticsProcessor","VoiceAnalyticsProcessorConfiguration":{"SpeakerSearchStatus":"Enabled","VoiceToneAnalysisStatus":"Enabled"}},
	{"Type":"S3RecordingSink","S3RecordingSinkConfiguration":{"Destination":"arn:aws:s3:::recordings","RecordingFileFormat":"Wav"}}]}`

func TestChimeSDKMediaPipelinesStage1(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	st := srv.chimeSDKPipelines
	setClock := func(d time.Duration) {
		st.mu.Lock()
		st.now = func() time.Time { return time.Now().Add(d) }
		st.mu.Unlock()
	}
	meeting := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/meetings", cmtgMeetingBody("cap"), "")["Meeting"].(map[string]any)
	meetingArn := meeting["MeetingArn"].(string)

	capture := `{"SourceType":"ChimeSdkMeeting","SourceArn":"` + meetingArn + `","SinkType":"S3Bucket","SinkArn":"arn:aws:s3:::captures","ClientRequestToken":"cap-1","Tags":[{"Key":"k","Value":"v"}]}`
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/sdk-media-capture-pipelines", strings.Replace(capture, "arn:aws:s3:::captures", "arn:aws:sqs:us-east-1:1:q", 1), "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodPost, "/sdk-media-capture-pipelines", strings.Replace(strings.Replace(capture, meeting["MeetingId"].(string), fakeUUID(7), 1), "cap-1", "cap-2", 1), "")
	p := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/sdk-media-capture-pipelines", capture, "")["MediaCapturePipeline"].(map[string]any)
	id := p["MediaPipelineId"].(string)
	if p["Status"] != "Initializing" || p["SinkArn"] != "arn:aws:s3:::captures" {
		t.Fatalf("CreateMediaCapturePipeline: %v", p)
	}
	// Timestamps travel as epoch seconds.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/sdk-media-capture-pipelines/"+id, nil, nil, "chime")
	var wire struct {
		MediaCapturePipeline struct {
			CreatedTimestamp float64
			Status           string
		}
	}
	if err := json.Unmarshal(mustBody(t, resp), &wire); err != nil || wire.MediaCapturePipeline.CreatedTimestamp < 1e9 || resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("wire shape: %v %+v", err, wire)
	}
	if again := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/sdk-media-capture-pipelines", capture, ""); again["MediaCapturePipeline"].(map[string]any)["MediaPipelineId"] != id {
		t.Fatalf("idempotent create: %v", again)
	}
	setClock(3 * time.Second)
	got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/sdk-media-capture-pipelines/"+id, "", "")["MediaCapturePipeline"].(map[string]any)
	if got["Status"] != "InProgress" {
		t.Fatalf("settled capture: %v", got)
	}

	// Concatenation reads the capture pipeline; live connectors need RTMP.
	concat := `{"Sources":[{"Type":"MediaCapturePipeline","MediaCapturePipelineSourceConfiguration":{"MediaPipelineArn":"` + p["MediaPipelineArn"].(string) + `",
		"ChimeSdkMeetingConfiguration":{"ArtifactsConfiguration":{"Audio":{"State":"Enabled"},"Video":{"State":"Enabled"},"Content":{"State":"Disabled"},
		"DataChannel":{"State":"Enabled"},"TranscriptionMessages":{"State":"Enabled"},"MeetingEvents":{"State":"Enabled"},"CompositedVideo":{"State":"Disabled"}}}}}],
		"Sinks":[{"Type":"S3Bucket","S3BucketSinkConfiguration":{"Destination":"arn:aws:s3:::concat"}}]}`
	chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/sdk-media-concatenation-pipelines", concat, "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodPost, "/sdk-media-concatenation-pipelines", strings.Replace(concat, id, fakeUUID(9), 1), "")
	live := `{"Sources":[{"SourceType":"ChimeSdkMeeting","ChimeSdkMeetingLiveConnectorConfiguration":{"Arn":"` + meetingArn + `","MuxType":"AudioWithActiveSpeakerVideo"}}],
		"Sinks":[{"SinkType":"RTMP","RTMPConfiguration":{"Url":"rtmps://live.example.com/app/key"}}]}`
	liveID := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/sdk-media-live-connector-pipelines", live, "")["MediaLiveConnectorPipeline"].(map[string]any)["MediaPipelineId"].(string)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/sdk-media-live-connector-pipelines", strings.Replace(live, "rtmps://", "https://", 1), "")
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/sdk-media-live-connector-pipelines", strings.Replace(live, "AudioWithActiveSpeakerVideo", "AudioWithCompositedVideo", 1), "")

	// Stream pipelines reserve capacity in a Kinesis video stream pool.
	pool := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-pipeline-kinesis-video-stream-pools", `{"PoolName":"pool-1","StreamConfiguration":{"Region":"us-east-1","DataRetentionInHours":24}}`, "")
	poolCfg := pool["KinesisVideoStreamPoolConfiguration"].(map[string]any)
	if poolCfg["PoolStatus"] != "ACTIVE" && poolCfg["PoolStatus"] != "CREATING" {
		t.Fatalf("CreateMediaPipelineKinesisVideoStreamPool: %v", pool)
	}
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPost, "/media-pipeline-kinesis-video-stream-pools", `{"PoolName":"pool-1","StreamConfiguration":{"Region":"us-east-1"}}`, "")
	stream := `{"Sources":[{"SourceType":"ChimeSdkMeeting","SourceArn":"` + meetingArn + `"}],"Sinks":[{"SinkArn":"` + poolCfg["PoolArn"].(string) + `","SinkType":"KinesisVideoStreamPool","ReservedStreamCapacity":3,"MediaStreamType":"IndividualAudio"}]}`
	streamID := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/sdk-media-stream-pipelines", stream, "")["MediaStreamPipeline"].(map[string]any)["MediaPipelineId"].(string)
	if got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/media-pipeline-kinesis-video-stream-pools/pool-1", "", ""); got["KinesisVideoStreamPoolConfiguration"].(map[string]any)["PoolSize"] != float64(3) {
		t.Fatalf("pool size: %v", got)
	}
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodDelete, "/media-pipeline-kinesis-video-stream-pools/pool-1", "", "")
	setClock(10 * time.Second)
	upd := chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, "/media-pipeline-kinesis-video-stream-pools/"+url.PathEscape(poolCfg["PoolArn"].(string)), `{"StreamConfiguration":{"DataRetentionInHours":48}}`, "")
	if c := upd["KinesisVideoStreamPoolConfiguration"].(map[string]any); c["PoolStatus"] != "UPDATING" || c["StreamConfiguration"].(map[string]any)["DataRetentionInHours"] != float64(48) {
		t.Fatalf("UpdateMediaPipelineKinesisVideoStreamPool: %v", upd)
	}

	// Deleting the meeting stops the pipelines reading it.
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/meetings/"+meeting["MeetingId"].(string), "", "")
	for _, pid := range []string{id, liveID, streamID} {
		mp := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/sdk-media-pipelines/"+pid, "", "")["MediaPipeline"].(map[string]any)
		for _, v := range mp {
			if v.(map[string]any)["Status"] != "Stopped" {
				t.Fatalf("pipeline of a deleted meeting: %v", mp)
			}
		}
	}
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/sdk-media-pipelines/"+streamID, "", "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/media-pipeline-kinesis-video-stream-pools/pool-1", "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodDelete, "/sdk-media-capture-pipelines/"+liveID, "", "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/sdk-media-capture-pipelines/"+id, "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, "/sdk-media-capture-pipelines/"+id, "", "")
	page := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/sdk-media-pipelines?max-results=1", "", "")
	rest := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/sdk-media-pipelines?max-results=1&next-token="+page["NextToken"].(string), "", "")
	if len(page["MediaPipelines"].([]any)) != 1 || len(rest["MediaPipelines"].([]any)) != 1 || rest["NextToken"] != nil {
		t.Fatalf("ListMediaPipelines: %v / %v", page, rest)
	}
}

func TestChimeSDKMediaPipelinesStage1Insights(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	st := srv.chimeSDKPipelines
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/media-insights-pipeline-configurations",
		strings.Replace(cmplTestConfig, `"VoiceAnalyticsProcessorConfiguration"`, `"SqsQueueSinkConfiguration"`, 1), "")
	cfg := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-insights-pipeline-configurations", cmplTestConfig, "")["MediaInsightsPipelineConfiguration"].(map[string]any)
	cfgArn := cfg["MediaInsightsPipelineConfigurationArn"].(string)
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPost, "/media-insights-pipeline-configurations", cmplTestConfig, "")
	byID := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/media-insights-pipeline-configurations/"+cfg["MediaInsightsPipelineConfigurationId"].(string), "", "")
	if byID["MediaInsightsPipelineConfiguration"].(map[string]any)["MediaInsightsPipelineConfigurationName"] != "calls" {
		t.Fatalf("GetMediaInsightsPipelineConfiguration by ID: %v", byID)
	}

	insights := `{"MediaInsightsPipelineConfigurationArn":"` + cfgArn + `","KinesisVideoStreamSourceRuntimeConfiguration":{"MediaEncoding":"pcm","MediaSampleRate":8000,
		"Streams":[{"StreamArn":"arn:aws:kinesisvideo:us-east-1:123456789012:stream/call/1","StreamChannelDefinition":{"NumberOfChannels":1}}]}}`
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/media-insights-pipelines", `{"MediaInsightsPipelineConfigurationArn":"`+cfgArn+`"}`, "")
	mp := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-insights-pipelines", insights, "")["MediaInsightsPipeline"].(map[string]any)
	pid := mp["MediaPipelineId"].(string)
	if len(mp["ElementStatuses"].([]any)) != 2 {
		t.Fatalf("CreateMediaInsightsPipeline: %v", mp)
	}
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodDelete, "/media-insights-pipeline-configurations/calls", "", "")

	// Pause and resume.
	status := "/media-insights-pipeline-status/" + pid
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPut, status, `{"UpdateStatus":"Resume"}`, "")
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, status, `{"UpdateStatus":"Pause"}`, "")
	if got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/sdk-media-pipelines/"+pid, "", ""); got["MediaPipeline"].(map[string]any)["MediaInsightsPipeline"].(map[string]any)["Status"] != "Paused" {
		t.Fatalf("paused: %v", got)
	}
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, status, `{"UpdateStatus":"Resume"}`, "")

	// Voice analytics tasks settle asynchronously.
	base := "/media-insights-pipelines/" + url.PathEscape(mp["MediaPipelineArn"].(string))
	task := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, base+"/speaker-search-tasks?operation=start",
		`{"VoiceProfileDomainArn":"arn:aws:chime:us-east-1:123456789012:voice-profile-domain/vpd","ClientRequestToken":"task-1"}`, "")["SpeakerSearchTask"].(map[string]any)
	tid := task["SpeakerSearchTaskId"].(string)
	if task["SpeakerSearchTaskStatus"] != "Initializing" {
		t.Fatalf("StartSpeakerSearchTask: %v", task)
	}
	st.mu.Lock()
	st.now = func() time.Time { return time.Now().Add(3 * time.Second) }
	st.mu.Unlock()
	got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, base+"/speaker-search-tasks/"+tid, "", "")["SpeakerSearchTask"].(map[string]any)
	if got["SpeakerSearchTaskStatus"] != "InProgress" {
		t.Fatalf("settled task: %v", got)
	}
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodPost, base+"/speaker-search-tasks/"+tid+"?operation=stop", "", "")
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPost, base+"/speaker-search-tasks/"+tid+"?operation=stop", "", "")
	st.mu.Lock()
	st.now = func() time.Time { return time.Now().Add(10 * time.Second) }
	st.mu.Unlock()
	if got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, base+"/speaker-search-tasks/"+tid, "", ""); got["SpeakerSearchTask"].(map[string]any)["SpeakerSearchTaskStatus"] != "Stopped" {
		t.Fatalf("stopped task: %v", got)
	}
	vt := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-insights-pipelines/"+pid+"/voice-tone-analysis-tasks?operation=start", `{"LanguageCode":"en-US"}`, "")
	vid := vt["VoiceToneAnalysisTask"].(map[string]any)["VoiceToneAnalysisTaskId"].(string)
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, "/media-insights-pipelines/"+pid+"/speaker-search-tasks/"+vid, "", "")
	chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/media-insights-pipelines/"+pid+"/voice-tone-analysis-tasks/"+vid, "", "")

	// Without a voice analytics processor no tasks start.
	plain := strings.Replace(strings.Replace(cmplTestConfig, `"calls"`, `"plain"`, 1),
		`{"Type":"VoiceAnalyticsProcessor","VoiceAnalyticsProcessorConfiguration":{"SpeakerSearchStatus":"Enabled","VoiceToneAnalysisStatus":"Enabled"}},`, "", 1)
	plainArn := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-insights-pipeline-configurations", plain, "")["MediaInsightsPipelineConfiguration"].(map[string]any)["MediaInsightsPipelineConfigurationArn"].(string)
	plainID := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-insights-pipelines", strings.Replace(insights, cfgArn, plainArn, 1), "")["MediaInsightsPipeline"].(map[string]any)["MediaPipelineId"].(string)
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/media-insights-pipelines/"+plainID+"/voice-tone-analysis-tasks?operation=start", `{"LanguageCode":"en-US"}`, "")
	// A ClientRequestToken reused for another configuration is a conflict,
	// not a pipeline silently moved between configurations.
	tok := strings.Replace(insights, `"MediaInsightsPipelineConfigurationArn"`, `"ClientRequestToken":"ins-1","MediaInsightsPipelineConfigurationArn"`, 1)
	tokID := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-insights-pipelines", tok, "")["MediaInsightsPipeline"].(map[string]any)["MediaPipelineId"].(string)
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPost, "/media-insights-pipelines", strings.Replace(tok, cfgArn, plainArn, 1), "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/sdk-media-pipelines/"+tokID, "", "")

	upd := strings.Replace(cmplTestConfig, `"MediaInsightsPipelineConfigurationName":"calls",`, "", 1)
	if out := chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, "/media-insights-pipeline-configurations/"+url.PathEscape(cfgArn), upd, ""); out["MediaInsightsPipelineConfiguration"].(map[string]any)["MediaInsightsPipelineConfigurationName"] != "calls" {
		t.Fatalf("UpdateMediaInsightsPipelineConfiguration: %v", out)
	}
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/sdk-media-pipelines/"+pid, "", "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/media-insights-pipeline-configurations/calls", "", "")
	if l := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/media-insights-pipeline-configurations", "", ""); len(l["MediaInsightsPipelineConfigurations"].([]any)) != 1 {
		t.Fatalf("ListMediaInsightsPipelineConfigurations: %v", l)
	}
	st.mu.Lock()
	tokens := len(st.tokens)
	st.mu.Unlock()
	if tokens != 0 {
		t.Fatalf("deleted resources left %d idempotency tokens", tokens)
	}
}
