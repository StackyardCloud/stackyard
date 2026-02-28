package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage87Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "BundleInstance":
		task, err := s.ec2.BundleInstance(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			ec2svc.BundleStorage{
				AWSAccessKeyID:        strings.TrimSpace(r.Form.Get("Storage.S3.AWSAccessKeyId")),
				Bucket:                strings.TrimSpace(r.Form.Get("Storage.S3.Bucket")),
				Prefix:                strings.TrimSpace(r.Form.Get("Storage.S3.Prefix")),
				UploadPolicy:          strings.TrimSpace(r.Form.Get("Storage.S3.UploadPolicy")),
				UploadPolicySignature: strings.TrimSpace(r.Form.Get("Storage.S3.UploadPolicySignature")),
			},
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage87BundleInstanceResponse{
			XMLName:            xml.Name{Local: "BundleInstanceResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			BundleInstanceTask: ec2Stage87BundleTaskItemFrom(task),
		})
		return true
	default:
		return false
	}
}

type ec2Stage87BundleInstanceResponse struct {
	XMLName            xml.Name                 `xml:"BundleInstanceResponse"`
	Xmlns              string                   `xml:"xmlns,attr"`
	RequestID          string                   `xml:"requestId"`
	BundleInstanceTask ec2Stage87BundleTaskItem `xml:"bundleInstanceTask"`
}

type ec2Stage87BundleTaskItem struct {
	BundleID   string                         `xml:"bundleId,omitempty"`
	Error      *ec2Stage87BundleTaskErrorItem `xml:"error,omitempty"`
	InstanceID string                         `xml:"instanceId,omitempty"`
	Progress   string                         `xml:"progress,omitempty"`
	StartTime  string                         `xml:"startTime,omitempty"`
	State      string                         `xml:"state,omitempty"`
	Storage    ec2Stage87StorageItem          `xml:"storage"`
	UpdateTime string                         `xml:"updateTime,omitempty"`
}

type ec2Stage87BundleTaskErrorItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage87StorageItem struct {
	S3 ec2Stage87S3StorageItem `xml:"S3"`
}

type ec2Stage87S3StorageItem struct {
	AWSAccessKeyID        string `xml:"AWSAccessKeyId,omitempty"`
	Bucket                string `xml:"bucket,omitempty"`
	Prefix                string `xml:"prefix,omitempty"`
	UploadPolicy          string `xml:"uploadPolicy,omitempty"`
	UploadPolicySignature string `xml:"uploadPolicySignature,omitempty"`
}

func ec2Stage87BundleTaskItemFrom(task ec2svc.BundleTask) ec2Stage87BundleTaskItem {
	out := ec2Stage87BundleTaskItem{
		BundleID:   task.BundleID,
		InstanceID: task.InstanceID,
		Progress:   task.Progress,
		StartTime:  task.StartTime.Format(time.RFC3339),
		State:      task.State,
		Storage: ec2Stage87StorageItem{
			S3: ec2Stage87S3StorageItem{
				AWSAccessKeyID:        task.Storage.AWSAccessKeyID,
				Bucket:                task.Storage.Bucket,
				Prefix:                task.Storage.Prefix,
				UploadPolicy:          task.Storage.UploadPolicy,
				UploadPolicySignature: task.Storage.UploadPolicySignature,
			},
		},
		UpdateTime: task.UpdateTime.Format(time.RFC3339),
	}
	if task.Error != nil {
		out.Error = &ec2Stage87BundleTaskErrorItem{
			Code:    task.Error.Code,
			Message: task.Error.Message,
		}
	}
	return out
}
