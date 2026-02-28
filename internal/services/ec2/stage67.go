package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Service) CreateVpcBlockPublicAccessExclusion(
	internetGatewayExclusionMode, vpcID, subnetID string,
	tags []Tag,
) (VpcBlockPublicAccessExclusion, error) {
	internetGatewayExclusionMode = strings.ToLower(strings.TrimSpace(internetGatewayExclusionMode))
	vpcID = strings.TrimSpace(vpcID)
	subnetID = strings.TrimSpace(subnetID)
	if internetGatewayExclusionMode == "" || (vpcID == "" && subnetID == "") || (vpcID != "" && subnetID != "") {
		return VpcBlockPublicAccessExclusion{}, ErrInvalidParameter
	}

	switch internetGatewayExclusionMode {
	case "allow-bidirectional", "allow-egress":
	default:
		return VpcBlockPublicAccessExclusion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN := ""
	switch {
	case vpcID != "":
		if s.vpcs[vpcID] == nil {
			return VpcBlockPublicAccessExclusion{}, ErrNotFound
		}
		resourceARN = fmt.Sprintf("arn:aws:ec2:%s:%s:vpc/%s", DefaultRegion, DefaultAccountID, vpcID)
	case subnetID != "":
		if s.subnets[subnetID] == nil {
			return VpcBlockPublicAccessExclusion{}, ErrNotFound
		}
		resourceARN = fmt.Sprintf("arn:aws:ec2:%s:%s:subnet/%s", DefaultRegion, DefaultAccountID, subnetID)
	}

	for _, exclusion := range s.vpcBlockPublicAccessExclusions {
		if exclusion == nil {
			continue
		}
		if exclusion.ResourceARN == resourceARN {
			return VpcBlockPublicAccessExclusion{}, ErrConflict
		}
	}

	now := time.Now().UTC()
	exclusion := &VpcBlockPublicAccessExclusion{
		ExclusionID:                  s.nextIDLocked("vpcbpa-ex"),
		InternetGatewayExclusionMode: internetGatewayExclusionMode,
		ResourceARN:                  resourceARN,
		State:                        "create-complete",
		Reason:                       "",
		CreationTimestamp:            now,
		LastUpdateTimestamp:          now,
		Tags:                         tagsToMap(tags),
	}
	s.vpcBlockPublicAccessExclusions[exclusion.ExclusionID] = exclusion
	return cloneVpcBlockPublicAccessExclusion(exclusion), nil
}

func (s *Service) DeleteVpcBlockPublicAccessExclusion(exclusionID string) (VpcBlockPublicAccessExclusion, error) {
	exclusionID = strings.TrimSpace(exclusionID)
	if exclusionID == "" {
		return VpcBlockPublicAccessExclusion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	exclusion := s.vpcBlockPublicAccessExclusions[exclusionID]
	if exclusion == nil {
		return VpcBlockPublicAccessExclusion{}, ErrNotFound
	}

	now := time.Now().UTC()
	exclusion.State = "delete-complete"
	exclusion.Reason = ""
	exclusion.LastUpdateTimestamp = now
	exclusion.DeletionTimestamp = &now

	out := cloneVpcBlockPublicAccessExclusion(exclusion)
	delete(s.vpcBlockPublicAccessExclusions, exclusionID)
	return out, nil
}

func (s *Service) DescribeVpcBlockPublicAccessExclusions(
	exclusionIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcBlockPublicAccessExclusion, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	exclusionIDSet := toStringSet(dedupeTrimmedStrings(exclusionIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterExclusionIDSet := toStringSet(standardFilters["exclusion-id"])
	filterResourceARNSet := toStringSet(standardFilters["resource-arn"])
	filterModeSet := toLowerStringSet(standardFilters["internet-gateway-exclusion-mode"])
	filterStateSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VpcBlockPublicAccessExclusion, 0, len(s.vpcBlockPublicAccessExclusions))
	for _, exclusion := range s.vpcBlockPublicAccessExclusions {
		if exclusion == nil {
			continue
		}
		if len(exclusionIDSet) > 0 {
			if _, ok := exclusionIDSet[exclusion.ExclusionID]; !ok {
				continue
			}
		}
		if len(filterExclusionIDSet) > 0 {
			if _, ok := filterExclusionIDSet[exclusion.ExclusionID]; !ok {
				continue
			}
		}
		if len(filterResourceARNSet) > 0 {
			if _, ok := filterResourceARNSet[exclusion.ResourceARN]; !ok {
				continue
			}
		}
		if len(filterModeSet) > 0 {
			if _, ok := filterModeSet[strings.ToLower(exclusion.InternetGatewayExclusionMode)]; !ok {
				continue
			}
		}
		if len(filterStateSet) > 0 {
			if _, ok := filterStateSet[strings.ToLower(exclusion.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(exclusion.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVpcBlockPublicAccessExclusion(exclusion))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ExclusionID < items[j].ExclusionID
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VpcBlockPublicAccessExclusion(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeVpcBlockPublicAccessOptions() VpcBlockPublicAccessOptions {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vpcBlockPublicAccessOptions
}
