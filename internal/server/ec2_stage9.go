package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage9Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeInstanceAttribute":
		attributeName := normalizeEC2InstanceAttributeName(r.Form.Get("Attribute"))
		description, err := s.ec2.DescribeInstanceAttribute(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			attributeName,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		resp := ec2DescribeInstanceAttributeResponse{
			XMLName:    xml.Name{Local: "DescribeInstanceAttributeResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			InstanceID: description.InstanceID,
		}
		switch attributeName {
		case "instanceType":
			resp.InstanceType = &ec2StringAttributeValue{Value: description.InstanceType}
		case "disableApiTermination":
			resp.DisableAPITermination = &ec2AttributeBooleanValue{Value: description.DisableAPITermination}
		case "sourceDestCheck":
			resp.SourceDestCheck = &ec2AttributeBooleanValue{Value: description.SourceDestCheck}
		case "instanceInitiatedShutdownBehavior":
			resp.InstanceInitiatedShutdownBehavior = &ec2StringAttributeValue{Value: description.InstanceInitiatedShutdownBehavior}
		case "userData":
			resp.UserData = &ec2StringAttributeValue{Value: description.UserData}
		case "groupSet":
			resp.GroupSet = &ec2GroupIdentifierSet{Items: ec2GroupIdentifierItems(description.GroupIDs)}
		}
		respondEC2XML(w, resp)
		return true
	case "ModifyInstanceAttribute":
		patch, attributeName := parseEC2ModifyInstanceAttributeInput(r)
		if err := s.ec2.ModifyInstanceAttribute(strings.TrimSpace(r.Form.Get("InstanceId")), attributeName, patch); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyInstanceAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "ResetInstanceAttribute":
		if err := s.ec2.ResetInstanceAttribute(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			normalizeEC2InstanceAttributeName(r.Form.Get("Attribute")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ResetInstanceAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "MonitorInstances":
		items, err := s.ec2.MonitorInstances(parseEC2Members(r.Form, "InstanceId."))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2MonitorInstancesResponse{
			XMLName:   xml.Name{Local: "MonitorInstancesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			InstancesSet: ec2InstanceMonitoringSet{
				Items: ec2InstanceMonitoringItems(items),
			},
		})
		return true
	case "UnmonitorInstances":
		items, err := s.ec2.UnmonitorInstances(parseEC2Members(r.Form, "InstanceId."))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2MonitorInstancesResponse{
			XMLName:   xml.Name{Local: "UnmonitorInstancesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			InstancesSet: ec2InstanceMonitoringSet{
				Items: ec2InstanceMonitoringItems(items),
			},
		})
		return true
	default:
		return false
	}
}

func parseEC2ModifyInstanceAttributeInput(r *http.Request) (ec2svc.InstanceAttributePatch, string) {
	attributeName := normalizeEC2InstanceAttributeName(r.Form.Get("Attribute"))
	patch := ec2svc.InstanceAttributePatch{}

	if value, ok := ec2OptionalStringFromForm(r.Form, "InstanceType.Value"); ok {
		patch.InstanceType = &value
	}
	if value, ok := parseEC2OptionalBool(r.Form.Get("DisableApiTermination.Value")); ok {
		patch.DisableAPITermination = &value
	}
	if value, ok := parseEC2OptionalBool(r.Form.Get("SourceDestCheck.Value")); ok {
		patch.SourceDestCheck = &value
	}
	if value, ok := ec2OptionalStringFromForm(r.Form, "InstanceInitiatedShutdownBehavior.Value"); ok {
		patch.InstanceInitiatedShutdownBehavior = &value
	}
	if value, ok := ec2OptionalStringFromForm(r.Form, "UserData.Value"); ok {
		patch.UserData = &value
	}
	if groupIDs := parseEC2Members(r.Form, "GroupId."); len(groupIDs) > 0 {
		patch.GroupIDs = groupIDs
	}

	if value, ok := ec2OptionalStringFromForm(r.Form, "Value"); ok {
		switch attributeName {
		case "instanceType":
			patch.InstanceType = &value
		case "instanceInitiatedShutdownBehavior":
			patch.InstanceInitiatedShutdownBehavior = &value
		case "userData":
			patch.UserData = &value
		case "disableApiTermination":
			boolValue := parseEC2Bool(value, false)
			patch.DisableAPITermination = &boolValue
		case "sourceDestCheck":
			boolValue := parseEC2Bool(value, false)
			patch.SourceDestCheck = &boolValue
		}
	}

	if attributeName == "" {
		switch {
		case patch.InstanceType != nil:
			attributeName = "instanceType"
		case patch.DisableAPITermination != nil:
			attributeName = "disableApiTermination"
		case patch.SourceDestCheck != nil:
			attributeName = "sourceDestCheck"
		case patch.InstanceInitiatedShutdownBehavior != nil:
			attributeName = "instanceInitiatedShutdownBehavior"
		case patch.UserData != nil:
			attributeName = "userData"
		case len(patch.GroupIDs) > 0:
			attributeName = "groupSet"
		}
	}

	return patch, attributeName
}

func normalizeEC2InstanceAttributeName(attribute string) string {
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

func ec2InstanceMonitoringItems(in []ec2svc.InstanceMonitoring) []ec2InstanceMonitoringItem {
	out := make([]ec2InstanceMonitoringItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2InstanceMonitoringItem{
			InstanceID: item.InstanceID,
			Monitoring: ec2MonitoringStateItem{State: item.State},
		})
	}
	return out
}

type ec2DescribeInstanceAttributeResponse struct {
	XMLName                           xml.Name
	Xmlns                             string                    `xml:"xmlns,attr"`
	RequestID                         string                    `xml:"requestId"`
	InstanceID                        string                    `xml:"instanceId"`
	InstanceType                      *ec2StringAttributeValue  `xml:"instanceType,omitempty"`
	DisableAPITermination             *ec2AttributeBooleanValue `xml:"disableApiTermination,omitempty"`
	SourceDestCheck                   *ec2AttributeBooleanValue `xml:"sourceDestCheck,omitempty"`
	InstanceInitiatedShutdownBehavior *ec2StringAttributeValue  `xml:"instanceInitiatedShutdownBehavior,omitempty"`
	UserData                          *ec2StringAttributeValue  `xml:"userData,omitempty"`
	GroupSet                          *ec2GroupIdentifierSet    `xml:"groupSet,omitempty"`
}

type ec2MonitorInstancesResponse struct {
	XMLName      xml.Name
	Xmlns        string                   `xml:"xmlns,attr"`
	RequestID    string                   `xml:"requestId"`
	InstancesSet ec2InstanceMonitoringSet `xml:"instancesSet"`
}

type ec2InstanceMonitoringSet struct {
	Items []ec2InstanceMonitoringItem `xml:"item"`
}

type ec2InstanceMonitoringItem struct {
	InstanceID string                 `xml:"instanceId"`
	Monitoring ec2MonitoringStateItem `xml:"monitoring"`
}

type ec2MonitoringStateItem struct {
	State string `xml:"state"`
}
