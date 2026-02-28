package ec2

import (
	"strings"
	"time"
)

type VpcBlockPublicAccessOptions struct {
	AwsAccountID             string
	AwsRegion                string
	ExclusionsAllowed        string
	InternetGatewayBlockMode string
	LastUpdateTimestamp      time.Time
	ManagedBy                string
	Reason                   string
	State                    string
}

func (s *Service) ModifyVpcBlockPublicAccessOptions(internetGatewayBlockMode string) (VpcBlockPublicAccessOptions, error) {
	internetGatewayBlockMode = strings.ToLower(strings.TrimSpace(internetGatewayBlockMode))
	if internetGatewayBlockMode == "" {
		return VpcBlockPublicAccessOptions{}, ErrInvalidParameter
	}

	switch internetGatewayBlockMode {
	case "off", "block-bidirectional", "block-ingress":
	default:
		return VpcBlockPublicAccessOptions{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.vpcBlockPublicAccessOptions.InternetGatewayBlockMode = internetGatewayBlockMode
	s.vpcBlockPublicAccessOptions.LastUpdateTimestamp = time.Now().UTC()
	s.vpcBlockPublicAccessOptions.State = "update-complete"
	s.vpcBlockPublicAccessOptions.Reason = ""

	return s.vpcBlockPublicAccessOptions, nil
}
