package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage126Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "GetManagedPrefixListEntries":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		targetVersion, ok := parseEC2OptionalInt64(r.Form.Get("TargetVersion"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		entries, nextToken, err := s.ec2.GetManagedPrefixListEntries(
			strings.TrimSpace(r.Form.Get("PrefixListId")),
			targetVersion,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126GetManagedPrefixListEntriesResponse{
			XMLName:            xml.Name{Local: "GetManagedPrefixListEntriesResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			PrefixListEntrySet: ec2Stage126PrefixListEntrySet{Items: ec2Stage126PrefixListEntryItemsFrom(entries)},
			NextToken:          nextToken,
		})
		return true
	case "GetNetworkInsightsAccessScopeAnalysisFindings":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		networkInsightsAccessScopeAnalysisID := strings.TrimSpace(r.Form.Get("NetworkInsightsAccessScopeAnalysisId"))
		findings, analysisStatus, nextToken, err := s.ec2.GetNetworkInsightsAccessScopeAnalysisFindings(
			networkInsightsAccessScopeAnalysisID,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126GetNetworkInsightsAccessScopeAnalysisFindingsResponse{
			XMLName:                              xml.Name{Local: "GetNetworkInsightsAccessScopeAnalysisFindingsResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			AnalysisFindingSet:                   ec2Stage126AnalysisFindingSet{Items: ec2Stage126AnalysisFindingItemsFrom(findings)},
			AnalysisStatus:                       analysisStatus,
			NetworkInsightsAccessScopeAnalysisID: networkInsightsAccessScopeAnalysisID,
			NextToken:                            nextToken,
		})
		return true
	case "GetNetworkInsightsAccessScopeContent":
		content, err := s.ec2.GetNetworkInsightsAccessScopeContent(strings.TrimSpace(r.Form.Get("NetworkInsightsAccessScopeId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126GetNetworkInsightsAccessScopeContentResponse{
			XMLName:   xml.Name{Local: "GetNetworkInsightsAccessScopeContentResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			NetworkInsightsAccessScopeContent: ec2Stage126NetworkInsightsAccessScopeContentItem{
				NetworkInsightsAccessScopeID: content.NetworkInsightsAccessScopeID,
			},
		})
		return true
	case "GetReservedInstancesExchangeQuote":
		quote, err := s.ec2.GetReservedInstancesExchangeQuote(parseEC2MembersOrItemList(r.Form, "ReservedInstanceId"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		isValidExchange := quote.IsValidExchange
		respondEC2XML(w, ec2Stage126GetReservedInstancesExchangeQuoteResponse{
			XMLName:                             xml.Name{Local: "GetReservedInstancesExchangeQuoteResponse"},
			Xmlns:                               ec2Namespace,
			RequestID:                           "stackyard-request",
			CurrencyCode:                        quote.CurrencyCode,
			IsValidExchange:                     &isValidExchange,
			OutputReservedInstancesWillExpireAt: ec2TimeString(quote.OutputReservedInstancesWillExpireAt),
			PaymentDue:                          quote.PaymentDue,
			ReservedInstanceValueRollup:         ec2Stage126ReservationValueItemFrom(quote.ReservedInstanceValueRollup),
			ReservedInstanceValueSet:            ec2Stage126ReservedInstanceReservationValueSet{Items: ec2Stage126ReservedInstanceReservationValueItemsFrom(quote.ReservedInstanceValueSet)},
			TargetConfigurationValueRollup:      ec2Stage126ReservationValueItemFrom(quote.TargetConfigurationValueRollup),
			TargetConfigurationValueSet:         ec2Stage126TargetReservationValueSet{Items: ec2Stage126TargetReservationValueItemsFrom(quote.TargetConfigurationValueSet)},
			ValidationFailureReason:             quote.ValidationFailureReason,
		})
		return true
	case "GetSpotPlacementScores":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		targetCapacity, ok := parseEC2Stage126RequiredInt32(r.Form.Get("TargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		singleAvailabilityZone, ok := parseEC2OptionalBoolValue(r.Form.Get("SingleAvailabilityZone"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		scores, nextToken, err := s.ec2.GetSpotPlacementScores(
			targetCapacity,
			parseEC2MembersOrItemList(r.Form, "InstanceType"),
			parseEC2MembersOrItemList(r.Form, "RegionName"),
			singleAvailabilityZone,
			strings.TrimSpace(r.Form.Get("TargetCapacityUnitType")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126GetSpotPlacementScoresResponse{
			XMLName:               xml.Name{Local: "GetSpotPlacementScoresResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			SpotPlacementScoreSet: ec2Stage126SpotPlacementScoreSet{Items: ec2Stage126SpotPlacementScoreItemsFrom(scores)},
			NextToken:             nextToken,
		})
		return true
	case "ImportImage":
		encrypted, ok := parseEC2OptionalBoolValue(r.Form.Get("Encrypted"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		result, err := s.ec2.ImportImage(
			strings.TrimSpace(r.Form.Get("Architecture")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("Hypervisor")),
			strings.TrimSpace(r.Form.Get("KmsKeyId")),
			strings.TrimSpace(r.Form.Get("LicenseType")),
			strings.TrimSpace(r.Form.Get("Platform")),
			strings.TrimSpace(r.Form.Get("RoleName")),
			encrypted,
			ec2Stage126TagSpecificationsFromForm(r.Form, "import-image-task", "image", "snapshot"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126ImportImageResponse{
			XMLName:       xml.Name{Local: "ImportImageResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			Architecture:  result.Architecture,
			Description:   result.Description,
			Encrypted:     result.Encrypted,
			Hypervisor:    result.Hypervisor,
			ImageID:       result.ImageID,
			ImportTaskID:  result.ImportTaskID,
			KmsKeyID:      result.KmsKeyID,
			LicenseType:   result.LicenseType,
			Platform:      result.Platform,
			Progress:      result.Progress,
			Status:        result.Status,
			StatusMessage: result.StatusMessage,
		})
		return true
	case "ImportInstance":
		conversionTask, err := s.ec2.ImportInstance(
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("Platform")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126ImportInstanceResponse{
			XMLName:        xml.Name{Local: "ImportInstanceResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			ConversionTask: ec2Stage126ConversionTaskItemFrom(conversionTask),
		})
		return true
	case "ImportSnapshot":
		result, err := s.ec2.ImportSnapshot(
			strings.TrimSpace(r.Form.Get("Description")),
			ec2Stage126TagSpecificationsFromForm(r.Form, "import-snapshot-task", "snapshot"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126ImportSnapshotResponse{
			XMLName:            xml.Name{Local: "ImportSnapshotResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			Description:        result.Description,
			ImportTaskID:       result.ImportTaskID,
			SnapshotTaskDetail: ec2Stage126SnapshotTaskDetailItemFrom(result.SnapshotTaskDetail),
			TagSet:             ec2TagSet{Items: ec2Stage117TagItemsFromEC2Tags(result.Tags)},
		})
		return true
	case "ImportVolume":
		conversionTask, err := s.ec2.ImportVolume(
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("AvailabilityZone")),
			strings.TrimSpace(r.Form.Get("AvailabilityZoneId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126ImportVolumeResponse{
			XMLName:        xml.Name{Local: "ImportVolumeResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			ConversionTask: ec2Stage126ConversionTaskItemFrom(conversionTask),
		})
		return true
	case "ListImagesInRecycleBin":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		images, nextToken, err := s.ec2.ListImagesInRecycleBin(
			parseEC2MembersOrItemList(r.Form, "ImageId"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage126ListImagesInRecycleBinResponse{
			XMLName:   xml.Name{Local: "ListImagesInRecycleBinResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageSet:  ec2Stage126ImageRecycleBinInfoSet{Items: ec2Stage126ImageRecycleBinInfoItemsFrom(images)},
			NextToken: nextToken,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage126RequiredInt32(value string) (int32, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(n), true
}

func ec2Stage126PrefixListEntryItemsFrom(in []ec2svc.ManagedPrefixListEntry) []ec2Stage126PrefixListEntryItem {
	out := make([]ec2Stage126PrefixListEntryItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage126PrefixListEntryItem{
			CIDR:        item.CIDR,
			Description: item.Description,
		})
	}
	return out
}

func ec2Stage126AnalysisFindingItemsFrom(in []ec2svc.AccessScopeAnalysisFinding) []ec2Stage126AnalysisFindingItem {
	out := make([]ec2Stage126AnalysisFindingItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage126AnalysisFindingItem{
			FindingID:                            item.FindingID,
			NetworkInsightsAccessScopeAnalysisID: item.NetworkInsightsAccessScopeAnalysisID,
			NetworkInsightsAccessScopeID:         item.NetworkInsightsAccessScopeID,
		})
	}
	return out
}

func ec2Stage126ReservationValueItemFrom(in ec2svc.ReservationValue) ec2Stage126ReservationValueItem {
	return ec2Stage126ReservationValueItem{
		HourlyPrice:           in.HourlyPrice,
		RemainingTotalValue:   in.RemainingTotalValue,
		RemainingUpfrontValue: in.RemainingUpfrontValue,
	}
}

func ec2Stage126ReservedInstanceReservationValueItemsFrom(in []ec2svc.ReservedInstanceReservationValue) []ec2Stage126ReservedInstanceReservationValueItem {
	out := make([]ec2Stage126ReservedInstanceReservationValueItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage126ReservedInstanceReservationValueItem{
			ReservedInstanceID: item.ReservedInstanceID,
			ReservationValue:   ec2Stage126ReservationValueItemFrom(item.ReservationValue),
		})
	}
	return out
}

func ec2Stage126TargetReservationValueItemsFrom(in []ec2svc.TargetReservationValue) []ec2Stage126TargetReservationValueItem {
	out := make([]ec2Stage126TargetReservationValueItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage126TargetReservationValueItem{ReservationValue: ec2Stage126ReservationValueItemFrom(item.ReservationValue)})
	}
	return out
}

func ec2Stage126SpotPlacementScoreItemsFrom(in []ec2svc.SpotPlacementScore) []ec2Stage126SpotPlacementScoreItem {
	out := make([]ec2Stage126SpotPlacementScoreItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage126SpotPlacementScoreItem{
			AvailabilityZoneID: item.AvailabilityZoneID,
			Region:             item.Region,
			Score:              item.Score,
		})
	}
	return out
}

func ec2Stage126ConversionTaskItemFrom(in ec2svc.ConversionTask) ec2Stage115ConversionTaskItem {
	items := ec2Stage115ConversionTaskItemsFrom([]ec2svc.ConversionTask{in})
	if len(items) == 0 {
		return ec2Stage115ConversionTaskItem{}
	}
	return items[0]
}

func ec2Stage126SnapshotTaskDetailItemFrom(in ec2svc.ImportSnapshotTaskDetail) ec2Stage117SnapshotTaskDetailItem {
	return ec2Stage117SnapshotTaskDetailItem{
		Description:   in.Description,
		Progress:      in.Progress,
		SnapshotID:    in.SnapshotID,
		Status:        in.Status,
		StatusMessage: in.StatusMessage,
	}
}

func ec2Stage126ImageRecycleBinInfoItemsFrom(in []ec2svc.ImageRecycleBinInfo) []ec2Stage126ImageRecycleBinInfoItem {
	out := make([]ec2Stage126ImageRecycleBinInfoItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage126ImageRecycleBinInfoItem{
			Description:         item.Description,
			ImageID:             item.ImageID,
			Name:                item.Name,
			RecycleBinEnterTime: ec2TimeString(item.RecycleBinEnterTime),
			RecycleBinExitTime:  ec2TimeString(item.RecycleBinExitTime),
		})
	}
	return out
}

func ec2Stage126TagSpecificationsFromForm(values url.Values, resourceTypes ...string) []ec2svc.Tag {
	out := make([]ec2svc.Tag, 0)
	seen := map[string]struct{}{}
	appendTags := func(tags []ec2svc.Tag) {
		for _, tag := range tags {
			key := strings.TrimSpace(tag.Key)
			if key == "" {
				continue
			}
			value := strings.TrimSpace(tag.Value)
			sig := key + "\x00" + value
			if _, ok := seen[sig]; ok {
				continue
			}
			seen[sig] = struct{}{}
			out = append(out, ec2svc.Tag{Key: key, Value: value})
		}
	}
	for _, resourceType := range resourceTypes {
		appendTags(parseEC2TagSpecificationsForResource(values, resourceType))
	}
	if len(out) == 0 {
		appendTags(parseEC2Tags(values, "Tag."))
	}
	return out
}

type ec2Stage126GetManagedPrefixListEntriesResponse struct {
	XMLName            xml.Name                      `xml:"GetManagedPrefixListEntriesResponse"`
	Xmlns              string                        `xml:"xmlns,attr"`
	RequestID          string                        `xml:"requestId"`
	PrefixListEntrySet ec2Stage126PrefixListEntrySet `xml:"entrySet"`
	NextToken          *string                       `xml:"nextToken,omitempty"`
}

type ec2Stage126PrefixListEntrySet struct {
	Items []ec2Stage126PrefixListEntryItem `xml:"item"`
}

type ec2Stage126PrefixListEntryItem struct {
	CIDR        string `xml:"cidr,omitempty"`
	Description string `xml:"description,omitempty"`
}

type ec2Stage126GetNetworkInsightsAccessScopeAnalysisFindingsResponse struct {
	XMLName                              xml.Name                      `xml:"GetNetworkInsightsAccessScopeAnalysisFindingsResponse"`
	Xmlns                                string                        `xml:"xmlns,attr"`
	RequestID                            string                        `xml:"requestId"`
	AnalysisFindingSet                   ec2Stage126AnalysisFindingSet `xml:"analysisFindingSet"`
	AnalysisStatus                       string                        `xml:"analysisStatus,omitempty"`
	NetworkInsightsAccessScopeAnalysisID string                        `xml:"networkInsightsAccessScopeAnalysisId,omitempty"`
	NextToken                            *string                       `xml:"nextToken,omitempty"`
}

type ec2Stage126AnalysisFindingSet struct {
	Items []ec2Stage126AnalysisFindingItem `xml:"item"`
}

type ec2Stage126AnalysisFindingItem struct {
	FindingID                            string `xml:"findingId,omitempty"`
	NetworkInsightsAccessScopeAnalysisID string `xml:"networkInsightsAccessScopeAnalysisId,omitempty"`
	NetworkInsightsAccessScopeID         string `xml:"networkInsightsAccessScopeId,omitempty"`
}

type ec2Stage126GetNetworkInsightsAccessScopeContentResponse struct {
	XMLName                           xml.Name                                         `xml:"GetNetworkInsightsAccessScopeContentResponse"`
	Xmlns                             string                                           `xml:"xmlns,attr"`
	RequestID                         string                                           `xml:"requestId"`
	NetworkInsightsAccessScopeContent ec2Stage126NetworkInsightsAccessScopeContentItem `xml:"networkInsightsAccessScopeContent"`
}

type ec2Stage126NetworkInsightsAccessScopeContentItem struct {
	NetworkInsightsAccessScopeID string `xml:"networkInsightsAccessScopeId,omitempty"`
}

type ec2Stage126GetReservedInstancesExchangeQuoteResponse struct {
	XMLName                             xml.Name                                       `xml:"GetReservedInstancesExchangeQuoteResponse"`
	Xmlns                               string                                         `xml:"xmlns,attr"`
	RequestID                           string                                         `xml:"requestId"`
	CurrencyCode                        string                                         `xml:"currencyCode,omitempty"`
	IsValidExchange                     *bool                                          `xml:"isValidExchange,omitempty"`
	OutputReservedInstancesWillExpireAt string                                         `xml:"outputReservedInstancesWillExpireAt,omitempty"`
	PaymentDue                          string                                         `xml:"paymentDue,omitempty"`
	ReservedInstanceValueRollup         ec2Stage126ReservationValueItem                `xml:"reservedInstanceValueRollup"`
	ReservedInstanceValueSet            ec2Stage126ReservedInstanceReservationValueSet `xml:"reservedInstanceValueSet"`
	TargetConfigurationValueRollup      ec2Stage126ReservationValueItem                `xml:"targetConfigurationValueRollup"`
	TargetConfigurationValueSet         ec2Stage126TargetReservationValueSet           `xml:"targetConfigurationValueSet"`
	ValidationFailureReason             string                                         `xml:"validationFailureReason,omitempty"`
}

type ec2Stage126ReservationValueItem struct {
	HourlyPrice           string `xml:"hourlyPrice,omitempty"`
	RemainingTotalValue   string `xml:"remainingTotalValue,omitempty"`
	RemainingUpfrontValue string `xml:"remainingUpfrontValue,omitempty"`
}

type ec2Stage126ReservedInstanceReservationValueSet struct {
	Items []ec2Stage126ReservedInstanceReservationValueItem `xml:"item"`
}

type ec2Stage126ReservedInstanceReservationValueItem struct {
	ReservationValue   ec2Stage126ReservationValueItem `xml:"reservationValue"`
	ReservedInstanceID string                          `xml:"reservedInstanceId,omitempty"`
}

type ec2Stage126TargetReservationValueSet struct {
	Items []ec2Stage126TargetReservationValueItem `xml:"item"`
}

type ec2Stage126TargetReservationValueItem struct {
	ReservationValue ec2Stage126ReservationValueItem `xml:"reservationValue"`
}

type ec2Stage126GetSpotPlacementScoresResponse struct {
	XMLName               xml.Name                         `xml:"GetSpotPlacementScoresResponse"`
	Xmlns                 string                           `xml:"xmlns,attr"`
	RequestID             string                           `xml:"requestId"`
	SpotPlacementScoreSet ec2Stage126SpotPlacementScoreSet `xml:"spotPlacementScoreSet"`
	NextToken             *string                          `xml:"nextToken,omitempty"`
}

type ec2Stage126SpotPlacementScoreSet struct {
	Items []ec2Stage126SpotPlacementScoreItem `xml:"item"`
}

type ec2Stage126SpotPlacementScoreItem struct {
	AvailabilityZoneID string `xml:"availabilityZoneId,omitempty"`
	Region             string `xml:"region,omitempty"`
	Score              int32  `xml:"score,omitempty"`
}

type ec2Stage126ImportImageResponse struct {
	XMLName       xml.Name `xml:"ImportImageResponse"`
	Xmlns         string   `xml:"xmlns,attr"`
	RequestID     string   `xml:"requestId"`
	Architecture  string   `xml:"architecture,omitempty"`
	Description   string   `xml:"description,omitempty"`
	Encrypted     *bool    `xml:"encrypted,omitempty"`
	Hypervisor    string   `xml:"hypervisor,omitempty"`
	ImageID       string   `xml:"imageId,omitempty"`
	ImportTaskID  string   `xml:"importTaskId,omitempty"`
	KmsKeyID      string   `xml:"kmsKeyId,omitempty"`
	LicenseType   string   `xml:"licenseType,omitempty"`
	Platform      string   `xml:"platform,omitempty"`
	Progress      string   `xml:"progress,omitempty"`
	Status        string   `xml:"status,omitempty"`
	StatusMessage string   `xml:"statusMessage,omitempty"`
}

type ec2Stage126ImportInstanceResponse struct {
	XMLName        xml.Name                      `xml:"ImportInstanceResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	ConversionTask ec2Stage115ConversionTaskItem `xml:"conversionTask"`
}

type ec2Stage126ImportSnapshotResponse struct {
	XMLName            xml.Name                          `xml:"ImportSnapshotResponse"`
	Xmlns              string                            `xml:"xmlns,attr"`
	RequestID          string                            `xml:"requestId"`
	Description        string                            `xml:"description,omitempty"`
	ImportTaskID       string                            `xml:"importTaskId,omitempty"`
	SnapshotTaskDetail ec2Stage117SnapshotTaskDetailItem `xml:"snapshotTaskDetail"`
	TagSet             ec2TagSet                         `xml:"tagSet"`
}

type ec2Stage126ImportVolumeResponse struct {
	XMLName        xml.Name                      `xml:"ImportVolumeResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	ConversionTask ec2Stage115ConversionTaskItem `xml:"conversionTask"`
}

type ec2Stage126ListImagesInRecycleBinResponse struct {
	XMLName   xml.Name                          `xml:"ListImagesInRecycleBinResponse"`
	Xmlns     string                            `xml:"xmlns,attr"`
	RequestID string                            `xml:"requestId"`
	ImageSet  ec2Stage126ImageRecycleBinInfoSet `xml:"imageSet"`
	NextToken *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage126ImageRecycleBinInfoSet struct {
	Items []ec2Stage126ImageRecycleBinInfoItem `xml:"item"`
}

type ec2Stage126ImageRecycleBinInfoItem struct {
	Description         string `xml:"description,omitempty"`
	ImageID             string `xml:"imageId,omitempty"`
	Name                string `xml:"name,omitempty"`
	RecycleBinEnterTime string `xml:"recycleBinEnterTime,omitempty"`
	RecycleBinExitTime  string `xml:"recycleBinExitTime,omitempty"`
}
