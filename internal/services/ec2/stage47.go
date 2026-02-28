package ec2

import (
	"strings"
	"time"
)

type VpcBlockPublicAccessExclusion struct {
	ExclusionID                  string
	InternetGatewayExclusionMode string
	ResourceARN                  string
	State                        string
	Reason                       string
	CreationTimestamp            time.Time
	LastUpdateTimestamp          time.Time
	DeletionTimestamp            *time.Time
	Tags                         map[string]string
}

func (s *Service) ModifyVpcBlockPublicAccessExclusion(exclusionID, internetGatewayExclusionMode string) (VpcBlockPublicAccessExclusion, error) {
	exclusionID = strings.TrimSpace(exclusionID)
	internetGatewayExclusionMode = strings.ToLower(strings.TrimSpace(internetGatewayExclusionMode))
	if exclusionID == "" || internetGatewayExclusionMode == "" {
		return VpcBlockPublicAccessExclusion{}, ErrInvalidParameter
	}

	switch internetGatewayExclusionMode {
	case "allow-bidirectional", "allow-egress":
	default:
		return VpcBlockPublicAccessExclusion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	exclusion := s.vpcBlockPublicAccessExclusions[exclusionID]
	if exclusion == nil {
		return VpcBlockPublicAccessExclusion{}, ErrNotFound
	}

	exclusion.InternetGatewayExclusionMode = internetGatewayExclusionMode
	exclusion.State = "update-complete"
	exclusion.Reason = ""
	exclusion.LastUpdateTimestamp = time.Now().UTC()

	return cloneVpcBlockPublicAccessExclusion(exclusion), nil
}

func cloneVpcBlockPublicAccessExclusion(in *VpcBlockPublicAccessExclusion) VpcBlockPublicAccessExclusion {
	if in == nil {
		return VpcBlockPublicAccessExclusion{}
	}
	out := *in
	if in.DeletionTimestamp != nil {
		deletedAt := *in.DeletionTimestamp
		out.DeletionTimestamp = &deletedAt
	}
	out.Tags = cloneStringMap(in.Tags)
	return out
}
