package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage104Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCapacityReservationFleet":
		totalTargetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("TotalTargetCapacity"))
		if !ok || totalTargetCapacity == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		instanceTypeSpecifications, ok := parseEC2ReservationFleetInstanceSpecifications(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		var endDate *time.Time
		if hasEC2Field(r.Form, "EndDate") {
			raw := strings.TrimSpace(r.Form.Get("EndDate"))
			if raw != "" {
				parsed, err := parseEC2RFC3339Time(raw)
				if err != nil {
					respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
					return true
				}
				endDate = &parsed
			}
		}

		allocationStrategy := parseEC2OptionalString(r.Form.Get("AllocationStrategy"))
		instanceMatchCriteria := parseEC2OptionalString(r.Form.Get("InstanceMatchCriteria"))
		tenancy := parseEC2OptionalString(r.Form.Get("Tenancy"))

		tags := parseEC2TagSpecificationsForResource(r.Form, "capacity-reservation-fleet")
		if len(tags) == 0 {
			tags = parseEC2TagSpecificationsForResource(r.Form, "capacity-reservation")
		}

		fleet, err := s.ec2.CreateCapacityReservationFleet(
			instanceTypeSpecifications,
			*totalTargetCapacity,
			allocationStrategy,
			endDate,
			instanceMatchCriteria,
			tenancy,
			tags,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage104CreateCapacityReservationFleetResponse{
			XMLName:                    xml.Name{Local: "CreateCapacityReservationFleetResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			AllocationStrategy:         fleet.AllocationStrategy,
			CapacityReservationFleetID: fleet.ID,
			CreateTime:                 fleet.CreateTime.Format(time.RFC3339),
			EndDate:                    ec2OptionalRFC3339(fleet.EndDate),
			FleetCapacityReservations: ec2Stage104FleetCapacityReservationSet{
				Items: ec2Stage104FleetCapacityReservationItemsFrom(fleet.FleetCapacityReservations),
			},
			InstanceMatchCriteria:  fleet.InstanceMatchCriteria,
			State:                  fleet.State,
			TagSet:                 ec2TagSet{Items: ec2TagItemsFromMap(fleet.Tags)},
			Tenancy:                fleet.Tenancy,
			TotalFulfilledCapacity: fleet.TotalFulfilledCapacity,
			TotalTargetCapacity:    fleet.TotalTargetCapacity,
		})
		return true
	default:
		return false
	}
}

func parseEC2ReservationFleetInstanceSpecifications(values url.Values) ([]ec2svc.ReservationFleetInstanceSpecification, bool) {
	prefixes := []string{"InstanceTypeSpecification.", "InstanceTypeSpecifications."}
	indices := map[int]struct{}{}
	for key := range values {
		for _, prefix := range prefixes {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			rest := strings.TrimPrefix(key, prefix)
			if strings.HasPrefix(rest, "Item.") {
				rest = strings.TrimPrefix(rest, "Item.")
			}
			if strings.HasPrefix(rest, "Member.") {
				rest = strings.TrimPrefix(rest, "Member.")
			}
			part := rest
			if dot := strings.IndexByte(rest, '.'); dot >= 0 {
				part = rest[:dot]
			}
			idx, err := strconv.Atoi(part)
			if err != nil || idx <= 0 {
				continue
			}
			indices[idx] = struct{}{}
		}
	}

	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)

	out := make([]ec2svc.ReservationFleetInstanceSpecification, 0, len(ordered))
	for _, idx := range ordered {
		instanceType := ec2ReservationFleetIndexedField(values, prefixes, idx, "InstanceType")
		instancePlatform := ec2ReservationFleetIndexedField(values, prefixes, idx, "InstancePlatform")
		if instanceType == "" || instancePlatform == "" {
			return nil, false
		}

		priorityRaw := ec2ReservationFleetIndexedField(values, prefixes, idx, "Priority")
		priority, ok := parseEC2OptionalInt32(priorityRaw)
		if !ok {
			return nil, false
		}

		var weight *float64
		weightRaw := strings.TrimSpace(ec2ReservationFleetIndexedField(values, prefixes, idx, "Weight"))
		if weightRaw != "" {
			v, err := strconv.ParseFloat(weightRaw, 64)
			if err != nil {
				return nil, false
			}
			weight = &v
		}

		ebsOptimized, ok := parseEC2OptionalBoolValue(ec2ReservationFleetIndexedField(values, prefixes, idx, "EbsOptimized"))
		if !ok {
			return nil, false
		}

		out = append(out, ec2svc.ReservationFleetInstanceSpecification{
			AvailabilityZone:   ec2ReservationFleetIndexedField(values, prefixes, idx, "AvailabilityZone"),
			AvailabilityZoneID: ec2ReservationFleetIndexedField(values, prefixes, idx, "AvailabilityZoneId"),
			EbsOptimized:       ebsOptimized,
			InstancePlatform:   instancePlatform,
			InstanceType:       instanceType,
			Priority:           priority,
			Weight:             weight,
		})
	}
	return out, true
}

func ec2ReservationFleetIndexedField(values url.Values, prefixes []string, idx int, field string) string {
	i := strconv.Itoa(idx)
	candidates := make([]string, 0, len(prefixes)*3)
	for _, prefix := range prefixes {
		candidates = append(candidates,
			prefix+i+"."+field,
			prefix+"Item."+i+"."+field,
			prefix+"Member."+i+"."+field,
		)
	}
	for _, key := range candidates {
		if !hasEC2Field(values, key) {
			continue
		}
		return strings.TrimSpace(values.Get(key))
	}
	return ""
}

func parseEC2OptionalBoolValue(value string) (*bool, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	parsed := parseEC2Bool(value, false)
	if strings.EqualFold(value, "true") || value == "1" || strings.EqualFold(value, "yes") || strings.EqualFold(value, "on") ||
		strings.EqualFold(value, "false") || value == "0" || strings.EqualFold(value, "no") || strings.EqualFold(value, "off") {
		return &parsed, true
	}
	return nil, false
}

func ec2OptionalRFC3339(in *time.Time) string {
	if in == nil || in.IsZero() {
		return ""
	}
	return in.UTC().Format(time.RFC3339)
}

func ec2Stage104FleetCapacityReservationItemsFrom(in []ec2svc.FleetCapacityReservation) []ec2Stage104FleetCapacityReservationItem {
	out := make([]ec2Stage104FleetCapacityReservationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage104FleetCapacityReservationItem{
			AvailabilityZone:      item.AvailabilityZone,
			AvailabilityZoneID:    item.AvailabilityZoneID,
			CapacityReservationID: item.CapacityReservationID,
			CreateDate:            item.CreateDate.UTC().Format(time.RFC3339),
			EbsOptimized:          item.EbsOptimized,
			FulfilledCapacity:     item.FulfilledCapacity,
			InstancePlatform:      item.InstancePlatform,
			InstanceType:          item.InstanceType,
			Priority:              item.Priority,
			TotalInstanceCount:    item.TotalInstanceCount,
			Weight:                item.Weight,
		})
	}
	return out
}

type ec2Stage104CreateCapacityReservationFleetResponse struct {
	XMLName                    xml.Name                               `xml:"CreateCapacityReservationFleetResponse"`
	Xmlns                      string                                 `xml:"xmlns,attr"`
	RequestID                  string                                 `xml:"requestId"`
	AllocationStrategy         string                                 `xml:"allocationStrategy,omitempty"`
	CapacityReservationFleetID string                                 `xml:"capacityReservationFleetId,omitempty"`
	CreateTime                 string                                 `xml:"createTime,omitempty"`
	EndDate                    string                                 `xml:"endDate,omitempty"`
	FleetCapacityReservations  ec2Stage104FleetCapacityReservationSet `xml:"fleetCapacityReservationSet"`
	InstanceMatchCriteria      string                                 `xml:"instanceMatchCriteria,omitempty"`
	State                      string                                 `xml:"state,omitempty"`
	TagSet                     ec2TagSet                              `xml:"tagSet"`
	Tenancy                    string                                 `xml:"tenancy,omitempty"`
	TotalFulfilledCapacity     float64                                `xml:"totalFulfilledCapacity,omitempty"`
	TotalTargetCapacity        int32                                  `xml:"totalTargetCapacity,omitempty"`
}

type ec2Stage104FleetCapacityReservationSet struct {
	Items []ec2Stage104FleetCapacityReservationItem `xml:"item"`
}

type ec2Stage104FleetCapacityReservationItem struct {
	AvailabilityZone      string   `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID    string   `xml:"availabilityZoneId,omitempty"`
	CapacityReservationID string   `xml:"capacityReservationId,omitempty"`
	CreateDate            string   `xml:"createDate,omitempty"`
	EbsOptimized          *bool    `xml:"ebsOptimized,omitempty"`
	FulfilledCapacity     float64  `xml:"fulfilledCapacity,omitempty"`
	InstancePlatform      string   `xml:"instancePlatform,omitempty"`
	InstanceType          string   `xml:"instanceType,omitempty"`
	Priority              *int32   `xml:"priority,omitempty"`
	TotalInstanceCount    int32    `xml:"totalInstanceCount,omitempty"`
	Weight                *float64 `xml:"weight,omitempty"`
}
