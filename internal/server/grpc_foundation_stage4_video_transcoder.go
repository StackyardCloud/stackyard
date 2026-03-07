package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	transcoderpb "cloud.google.com/go/video/transcoder/apiv1/transcoderpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpVideoTranscoderCreateJobMethod         = "/google.cloud.video.transcoder.v1.TranscoderService/CreateJob"
	gcpVideoTranscoderListJobsMethod          = "/google.cloud.video.transcoder.v1.TranscoderService/ListJobs"
	gcpVideoTranscoderGetJobMethod            = "/google.cloud.video.transcoder.v1.TranscoderService/GetJob"
	gcpVideoTranscoderDeleteJobMethod         = "/google.cloud.video.transcoder.v1.TranscoderService/DeleteJob"
	gcpVideoTranscoderCreateJobTemplateMethod = "/google.cloud.video.transcoder.v1.TranscoderService/CreateJobTemplate"
	gcpVideoTranscoderListJobTemplatesMethod  = "/google.cloud.video.transcoder.v1.TranscoderService/ListJobTemplates"
	gcpVideoTranscoderGetJobTemplateMethod    = "/google.cloud.video.transcoder.v1.TranscoderService/GetJobTemplate"
	gcpVideoTranscoderDeleteJobTemplateMethod = "/google.cloud.video.transcoder.v1.TranscoderService/DeleteJobTemplate"
)

func gcpStage4GRPCVideoTranscoder(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVideoTranscoderCreateJobMethod:
		return gcpStage4GRPCVideoTranscoderCreateJob(grpcReqBody)
	case gcpVideoTranscoderListJobsMethod:
		return gcpStage4GRPCVideoTranscoderListJobs(grpcReqBody)
	case gcpVideoTranscoderGetJobMethod:
		return gcpStage4GRPCVideoTranscoderGetJob(grpcReqBody)
	case gcpVideoTranscoderDeleteJobMethod:
		return gcpStage4GRPCVideoTranscoderDeleteJob(grpcReqBody)
	case gcpVideoTranscoderCreateJobTemplateMethod:
		return gcpStage4GRPCVideoTranscoderCreateJobTemplate(grpcReqBody)
	case gcpVideoTranscoderListJobTemplatesMethod:
		return gcpStage4GRPCVideoTranscoderListJobTemplates(grpcReqBody)
	case gcpVideoTranscoderGetJobTemplateMethod:
		return gcpStage4GRPCVideoTranscoderGetJobTemplate(grpcReqBody)
	case gcpVideoTranscoderDeleteJobTemplateMethod:
		return gcpStage4GRPCVideoTranscoderDeleteJobTemplate(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCVideoTranscoderCreateJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.CreateJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetJob() == nil {
		return grpcInvalidArgument("job-required")
	}
	if strings.TrimSpace(req.GetJob().GetInputUri()) == "" {
		return grpcInvalidArgument("job-input_uri-required")
	}
	if strings.TrimSpace(req.GetJob().GetOutputUri()) == "" {
		return grpcInvalidArgument("job-output_uri-required")
	}
	if strings.TrimSpace(req.GetJob().GetTemplateId()) == "" && req.GetJob().GetConfig() == nil {
		return grpcInvalidArgument("job-template_or_config-required")
	}

	jobID := "job-1"
	if name := strings.TrimSpace(req.GetJob().GetName()); name != "" {
		parent, parsedID, ok := gcpVideoTranscoderParseJobName(name)
		if !ok {
			return grpcInvalidArgument("job-name-invalid")
		}
		if parent != strings.TrimSpace(req.GetParent()) {
			return grpcInvalidArgument("job-name-mismatch")
		}
		jobID = parsedID
	}

	resp := gcpStage4VideoTranscoderJob(project, location, jobID)
	resp.InputUri = strings.TrimSpace(req.GetJob().GetInputUri())
	resp.OutputUri = strings.TrimSpace(req.GetJob().GetOutputUri())
	if req.GetJob().GetConfig() != nil {
		resp.JobConfig = &transcoderpb.Job_Config{Config: req.GetJob().GetConfig()}
	} else if templateID := strings.TrimSpace(req.GetJob().GetTemplateId()); templateID != "" {
		resp.JobConfig = &transcoderpb.Job_TemplateId{TemplateId: templateID}
	}
	if len(req.GetJob().GetLabels()) > 0 {
		resp.Labels = req.GetJob().GetLabels()
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCVideoTranscoderListJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.ListJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*transcoderpb.Job{
		gcpStage4VideoTranscoderJob(project, location, "job-1"),
		gcpStage4VideoTranscoderJob(project, location, "job-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoTranscoderPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&transcoderpb.ListJobsResponse{
		Jobs:          items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCVideoTranscoderGetJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.GetJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, jobID, ok := gcpVideoTranscoderParseJobName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoTranscoderMissingID(jobID) {
		return grpcNotFound("job-not-found")
	}
	project, location, _ := gcpVideoTranscoderProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoTranscoderJob(project, location, jobID))
}

func gcpStage4GRPCVideoTranscoderDeleteJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.DeleteJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, jobID, ok := gcpVideoTranscoderParseJobName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoTranscoderMissingID(jobID) && !req.GetAllowMissing() {
		return grpcNotFound("job-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVideoTranscoderCreateJobTemplate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.CreateJobTemplateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetJobTemplate() == nil {
		return grpcInvalidArgument("job_template-required")
	}
	jobTemplateID := strings.TrimSpace(req.GetJobTemplateId())
	if jobTemplateID == "" {
		if name := strings.TrimSpace(req.GetJobTemplate().GetName()); name != "" {
			_, parsedID, ok := gcpVideoTranscoderParseJobTemplateName(name)
			if !ok {
				return grpcInvalidArgument("job_template-name-invalid")
			}
			jobTemplateID = parsedID
		}
	}
	if jobTemplateID == "" {
		return grpcInvalidArgument("job_template_id-required")
	}
	if !gcpVideoTranscoderJobTemplatePattern.MatchString(jobTemplateID) {
		return grpcInvalidArgument("job_template_id-invalid")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/jobTemplates/%s", project, location, jobTemplateID)
	if name := strings.TrimSpace(req.GetJobTemplate().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("job_template-name-mismatch")
	}
	resp := gcpStage4VideoTranscoderJobTemplate(project, location, jobTemplateID)
	if req.GetJobTemplate().GetConfig() != nil {
		resp.Config = req.GetJobTemplate().GetConfig()
	}
	if len(req.GetJobTemplate().GetLabels()) > 0 {
		resp.Labels = req.GetJobTemplate().GetLabels()
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCVideoTranscoderListJobTemplates(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.ListJobTemplatesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*transcoderpb.JobTemplate{
		gcpStage4VideoTranscoderJobTemplate(project, location, "template-1"),
		gcpStage4VideoTranscoderJobTemplate(project, location, "template-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoTranscoderPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&transcoderpb.ListJobTemplatesResponse{
		JobTemplates:  items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCVideoTranscoderGetJobTemplate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.GetJobTemplateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, jobTemplateID, ok := gcpVideoTranscoderParseJobTemplateName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoTranscoderMissingID(jobTemplateID) {
		return grpcNotFound("job_template-not-found")
	}
	project, location, _ := gcpVideoTranscoderProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoTranscoderJobTemplate(project, location, jobTemplateID))
}

func gcpStage4GRPCVideoTranscoderDeleteJobTemplate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &transcoderpb.DeleteJobTemplateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, jobTemplateID, ok := gcpVideoTranscoderParseJobTemplateName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoTranscoderMissingID(jobTemplateID) && !req.GetAllowMissing() {
		return grpcNotFound("job_template-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4VideoTranscoderJob(project, location, jobID string) *transcoderpb.Job {
	return &transcoderpb.Job{
		Name:       fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID),
		InputUri:   fmt.Sprintf("gs://stackyard-inputs/%s.mp4", jobID),
		OutputUri:  fmt.Sprintf("gs://stackyard-outputs/%s/", jobID),
		JobConfig:  &transcoderpb.Job_TemplateId{TemplateId: "preset/web-hd"},
		State:      transcoderpb.Job_SUCCEEDED,
		CreateTime: timestamppb.New(gcpVideoTranscoderReferenceTime),
		StartTime:  timestamppb.New(gcpVideoTranscoderReferenceTime.Add(5 * time.Second)),
		EndTime:    timestamppb.New(gcpVideoTranscoderReferenceTime.Add(15 * time.Second)),
		Labels: map[string]string{
			"env": "staged",
			"id":  jobID,
		},
	}
}

func gcpStage4VideoTranscoderJobTemplate(project, location, jobTemplateID string) *transcoderpb.JobTemplate {
	return &transcoderpb.JobTemplate{
		Name: fmt.Sprintf("projects/%s/locations/%s/jobTemplates/%s", project, location, jobTemplateID),
		Config: &transcoderpb.JobConfig{
			Output: &transcoderpb.Output{
				Uri: fmt.Sprintf("gs://stackyard-outputs/templates/%s/", jobTemplateID),
			},
		},
		Labels: map[string]string{
			"env": "staged",
			"id":  jobTemplateID,
		},
	}
}

func gcpStage4VideoTranscoderPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
	if pageSize < 0 {
		return 0, 0, "", "page_size-negative", false
	}
	if pageSize > int32(max) {
		return 0, 0, "", "page_size-too-large", false
	}
	start = 0
	if strings.TrimSpace(pageToken) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(pageToken))
		if err != nil || parsed < 0 {
			return 0, 0, "", "page_token-invalid", false
		}
		start = parsed
	}
	if start > total {
		return 0, 0, "", "page_token-out-of-range", false
	}
	end = total
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	nextPageToken = ""
	if end < total {
		nextPageToken = strconv.Itoa(end)
	}
	return start, end, nextPageToken, "", true
}
