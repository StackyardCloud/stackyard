package ec2

import (
	"sort"
	"strings"
)

type InstanceAttributeDescription struct {
	InstanceID                        string
	InstanceType                      string
	DisableAPITermination             bool
	SourceDestCheck                   bool
	InstanceInitiatedShutdownBehavior string
	UserData                          string
	GroupIDs                          []string
}

type InstanceAttributePatch struct {
	InstanceType                      *string
	DisableAPITermination             *bool
	SourceDestCheck                   *bool
	InstanceInitiatedShutdownBehavior *string
	UserData                          *string
	GroupIDs                          []string
}

type InstanceMonitoring struct {
	InstanceID string
	State      string
}

func (s *Service) DescribeInstanceAttribute(instanceID, attribute string) (InstanceAttributeDescription, error) {
	instanceID = strings.TrimSpace(instanceID)
	attribute = normalizeInstanceAttributeName(attribute)
	if instanceID == "" || attribute == "" {
		return InstanceAttributeDescription{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return InstanceAttributeDescription{}, ErrNotFound
	}
	return InstanceAttributeDescription{
		InstanceID:                        instance.ID,
		InstanceType:                      instance.InstanceType,
		DisableAPITermination:             instance.DisableAPITermination,
		SourceDestCheck:                   instance.SourceDestCheck,
		InstanceInitiatedShutdownBehavior: instance.InstanceInitiatedShutdownBehavior,
		UserData:                          instance.UserData,
		GroupIDs:                          append([]string(nil), instance.SecurityGroupIDs...),
	}, nil
}

func (s *Service) ModifyInstanceAttribute(instanceID, attribute string, patch InstanceAttributePatch) error {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return ErrNotFound
	}

	attrName := normalizeInstanceAttributeName(attribute)
	if attrName == "" {
		switch {
		case patch.InstanceType != nil:
			attrName = "instanceType"
		case patch.DisableAPITermination != nil:
			attrName = "disableApiTermination"
		case patch.SourceDestCheck != nil:
			attrName = "sourceDestCheck"
		case patch.InstanceInitiatedShutdownBehavior != nil:
			attrName = "instanceInitiatedShutdownBehavior"
		case patch.UserData != nil:
			attrName = "userData"
		case len(patch.GroupIDs) > 0:
			attrName = "groupSet"
		default:
			return ErrInvalidParameter
		}
	}

	switch attrName {
	case "instanceType":
		if patch.InstanceType == nil || strings.TrimSpace(*patch.InstanceType) == "" {
			return ErrInvalidParameter
		}
		instance.InstanceType = strings.TrimSpace(*patch.InstanceType)
	case "disableApiTermination":
		if patch.DisableAPITermination == nil {
			return ErrInvalidParameter
		}
		instance.DisableAPITermination = *patch.DisableAPITermination
	case "sourceDestCheck":
		if patch.SourceDestCheck == nil {
			return ErrInvalidParameter
		}
		instance.SourceDestCheck = *patch.SourceDestCheck
	case "instanceInitiatedShutdownBehavior":
		if patch.InstanceInitiatedShutdownBehavior == nil {
			return ErrInvalidParameter
		}
		behavior := strings.ToLower(strings.TrimSpace(*patch.InstanceInitiatedShutdownBehavior))
		if behavior != "stop" && behavior != "terminate" {
			return ErrInvalidParameter
		}
		instance.InstanceInitiatedShutdownBehavior = behavior
	case "userData":
		if patch.UserData == nil {
			return ErrInvalidParameter
		}
		instance.UserData = strings.TrimSpace(*patch.UserData)
	case "groupSet":
		if len(patch.GroupIDs) == 0 {
			return ErrInvalidParameter
		}
		groupIDs := make([]string, 0, len(patch.GroupIDs))
		for _, groupID := range patch.GroupIDs {
			groupID = strings.TrimSpace(groupID)
			if groupID == "" {
				continue
			}
			group := s.securityGroups[groupID]
			if group == nil {
				return ErrNotFound
			}
			if group.VpcID != instance.VpcID {
				return ErrInvalidParameter
			}
			groupIDs = append(groupIDs, groupID)
		}
		if len(groupIDs) == 0 {
			return ErrInvalidParameter
		}
		instance.SecurityGroupIDs = groupIDs
	default:
		return ErrInvalidParameter
	}
	return nil
}

func (s *Service) ResetInstanceAttribute(instanceID, attribute string) error {
	instanceID = strings.TrimSpace(instanceID)
	attribute = normalizeInstanceAttributeName(attribute)
	if instanceID == "" || attribute == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return ErrNotFound
	}

	switch attribute {
	case "sourceDestCheck":
		instance.SourceDestCheck = true
	case "kernel", "ramdisk":
		// Stackyard does not model kernel or ramdisk IDs; reset is a no-op.
	default:
		return ErrInvalidParameter
	}
	return nil
}

func (s *Service) MonitorInstances(instanceIDs []string) ([]InstanceMonitoring, error) {
	return s.setMonitoringState(instanceIDs, "enabled")
}

func (s *Service) UnmonitorInstances(instanceIDs []string) ([]InstanceMonitoring, error) {
	return s.setMonitoringState(instanceIDs, "disabled")
}

func (s *Service) setMonitoringState(instanceIDs []string, state string) ([]InstanceMonitoring, error) {
	if len(instanceIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]InstanceMonitoring, 0, len(instanceIDs))
	for _, instanceID := range instanceIDs {
		instanceID = strings.TrimSpace(instanceID)
		if instanceID == "" {
			continue
		}
		instance := s.instances[instanceID]
		if instance == nil {
			return nil, ErrNotFound
		}
		instance.MonitoringState = state
		items = append(items, InstanceMonitoring{
			InstanceID: instance.ID,
			State:      instance.MonitoringState,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].InstanceID < items[j].InstanceID })
	return items, nil
}

func normalizeInstanceAttributeName(attribute string) string {
	switch strings.ToLower(strings.TrimSpace(attribute)) {
	case "instancetype":
		return "instanceType"
	case "disableapitermination":
		return "disableApiTermination"
	case "sourcedestcheck":
		return "sourceDestCheck"
	case "instanceinitiatedshutdownbehavior":
		return "instanceInitiatedShutdownBehavior"
	case "userdata":
		return "userData"
	case "groupset":
		return "groupSet"
	case "kernel":
		return "kernel"
	case "ramdisk":
		return "ramdisk"
	default:
		return ""
	}
}
