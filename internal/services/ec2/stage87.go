package ec2

import (
	"strings"
	"time"
)

func (s *Service) BundleInstance(
	instanceID string,
	storage BundleStorage,
) (BundleTask, error) {
	instanceID = strings.TrimSpace(instanceID)
	storage.AWSAccessKeyID = strings.TrimSpace(storage.AWSAccessKeyID)
	storage.Bucket = strings.TrimSpace(storage.Bucket)
	storage.Prefix = strings.TrimSpace(storage.Prefix)
	storage.UploadPolicy = strings.TrimSpace(storage.UploadPolicy)
	storage.UploadPolicySignature = strings.TrimSpace(storage.UploadPolicySignature)
	if instanceID == "" || storage.Bucket == "" {
		return BundleTask{}, ErrInvalidParameter
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	task := &BundleTask{
		BundleID:   s.nextIDLocked("bun"),
		InstanceID: instanceID,
		Progress:   "0%",
		StartTime:  now,
		State:      "pending",
		Storage:    storage,
		UpdateTime: now,
	}
	s.bundleTasks[task.BundleID] = task
	return cloneBundleTask(task), nil
}

func cloneBundleTask(in *BundleTask) BundleTask {
	if in == nil {
		return BundleTask{}
	}
	out := *in
	if in.Error != nil {
		errCopy := *in.Error
		out.Error = &errCopy
	}
	return out
}
