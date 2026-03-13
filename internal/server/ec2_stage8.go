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

func (s *Server) handleEC2Stage8Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateImage":
		image, err := s.ec2.CreateImage(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("Name")),
			strings.TrimSpace(r.Form.Get("Description")),
			parseEC2Bool(r.Form.Get("NoReboot"), false),
			parseEC2TagSpecificationsForResource(r.Form, "image"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateImageResponse{
			XMLName:   xml.Name{Local: "CreateImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageID:   image.ID,
		})
		return true
	case "DescribeImages":
		images := s.ec2.DescribeImages(
			parseEC2Members(r.Form, "ImageId."),
			parseEC2Members(r.Form, "Owner."),
		)
		respondEC2XML(w, ec2DescribeImagesResponse{
			XMLName:   xml.Name{Local: "DescribeImagesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImagesSet: ec2ImageSet{Items: ec2ImageItems(images)},
		})
		return true
	case "DeregisterImage":
		if err := s.ec2.DeregisterImage(strings.TrimSpace(r.Form.Get("ImageId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeregisterImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DescribeImageAttribute":
		attribute, err := s.ec2.DescribeImageAttribute(
			strings.TrimSpace(r.Form.Get("ImageId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		resp := ec2DescribeImageAttributeResponse{
			XMLName:   xml.Name{Local: "DescribeImageAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageID:   attribute.ImageID,
		}
		if attribute.Description != "" {
			resp.Description = &ec2StringAttributeValue{Value: attribute.Description}
		}
		if len(attribute.LaunchPermissions) > 0 {
			resp.LaunchPermission = &ec2ImageLaunchPermissionSet{
				Items: ec2ImageLaunchPermissionItems(attribute.LaunchPermissions),
			}
		}
		respondEC2XML(w, resp)
		return true
	case "ModifyImageAttribute":
		var description *string
		if value, ok := ec2OptionalStringFromForm(r.Form, "Description.Value"); ok {
			description = &value
		} else if value, ok := ec2OptionalStringFromForm(r.Form, "Description"); ok {
			description = &value
		}
		launchPermission := parseEC2LaunchPermissionModifications(r.Form)
		if err := s.ec2.ModifyImageAttribute(
			strings.TrimSpace(r.Form.Get("ImageId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
			description,
			launchPermission,
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyImageAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "ResetImageAttribute":
		if err := s.ec2.ResetImageAttribute(
			strings.TrimSpace(r.Form.Get("ImageId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ResetImageAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func parseEC2LaunchPermissionModifications(values url.Values) *ec2svc.LaunchPermissionModifications {
	add := parseEC2LaunchPermissions(values, "LaunchPermission.Add.")
	remove := parseEC2LaunchPermissions(values, "LaunchPermission.Remove.")

	operationType := strings.ToLower(strings.TrimSpace(values.Get("OperationType")))
	if operationType == "add" || operationType == "remove" {
		legacyPermissions := make([]ec2svc.LaunchPermission, 0)
		for _, userID := range parseEC2Members(values, "UserId.") {
			legacyPermissions = append(legacyPermissions, ec2svc.LaunchPermission{UserID: userID})
		}
		for _, group := range parseEC2Members(values, "UserGroup.") {
			legacyPermissions = append(legacyPermissions, ec2svc.LaunchPermission{Group: group})
		}
		if operationType == "add" {
			add = append(add, legacyPermissions...)
		} else {
			remove = append(remove, legacyPermissions...)
		}
	}

	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	return &ec2svc.LaunchPermissionModifications{Add: add, Remove: remove}
}

func parseEC2LaunchPermissions(values url.Values, prefix string) []ec2svc.LaunchPermission {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}
	if len(indices) == 0 {
		return nil
	}
	sorted := make([]int, 0, len(indices))
	for idx := range indices {
		sorted = append(sorted, idx)
	}
	sort.Ints(sorted)

	out := make([]ec2svc.LaunchPermission, 0, len(sorted))
	for _, idx := range sorted {
		group := strings.TrimSpace(values.Get(prefix + strconv.Itoa(idx) + ".Group"))
		userID := strings.TrimSpace(values.Get(prefix + strconv.Itoa(idx) + ".UserId"))
		if group == "" && userID == "" {
			continue
		}
		out = append(out, ec2svc.LaunchPermission{
			Group:  group,
			UserID: userID,
		})
	}
	return out
}

func ec2OptionalStringFromForm(values url.Values, key string) (string, bool) {
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return "", false
	}
	return value, true
}

func ec2ImageItems(in []ec2svc.Image) []ec2ImageItem {
	out := make([]ec2ImageItem, 0, len(in))
	for _, image := range in {
		tags := make([]ec2TagItem, 0, len(image.Tags))
		for key, value := range image.Tags {
			tags = append(tags, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
		out = append(out, ec2ImageItem{
			ImageID:                  image.ID,
			ImageLocation:            image.ImageLocation,
			ImageState:               image.State,
			ImageOwnerID:             image.OwnerID,
			IsPublic:                 false,
			Architecture:             image.Architecture,
			ImageType:                image.ImageType,
			Name:                     image.Name,
			Description:              image.Description,
			DeprecationTime:          ec2TimeString(image.DeprecationTime),
			RootDeviceType:           image.RootDeviceType,
			RootDeviceName:           image.RootDeviceName,
			VirtualizationType:       image.VirtualizationType,
			CreationDate:             image.CreationDate.Format(timeRFC3339UTC),
			TagSet:                   ec2TagSet{Items: tags},
		})
	}
	return out
}

func ec2ImageLaunchPermissionItems(in []ec2svc.LaunchPermission) []ec2ImageLaunchPermissionItem {
	out := make([]ec2ImageLaunchPermissionItem, 0, len(in))
	for _, permission := range in {
		out = append(out, ec2ImageLaunchPermissionItem{
			UserID: permission.UserID,
			Group:  permission.Group,
		})
	}
	return out
}

func ec2TimeString(in *time.Time) string {
	if in == nil {
		return ""
	}
	return in.UTC().Format(timeRFC3339UTC)
}

const timeRFC3339UTC = "2006-01-02T15:04:05Z"

type ec2CreateImageResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	ImageID   string `xml:"imageId"`
}

type ec2DescribeImagesResponse struct {
	XMLName   xml.Name    `xml:"DescribeImagesResponse"`
	Xmlns     string      `xml:"xmlns,attr"`
	RequestID string      `xml:"requestId"`
	ImagesSet ec2ImageSet `xml:"imagesSet"`
}

type ec2ImageSet struct {
	Items []ec2ImageItem `xml:"item"`
}

type ec2ImageItem struct {
	ImageID                  string    `xml:"imageId"`
	ImageLocation            string    `xml:"imageLocation,omitempty"`
	ImageState               string    `xml:"imageState,omitempty"`
	ImageOwnerID             string    `xml:"imageOwnerId,omitempty"`
	IsPublic                 bool      `xml:"isPublic"`
	Architecture             string    `xml:"architecture,omitempty"`
	ImageType                string    `xml:"imageType,omitempty"`
	Name                     string    `xml:"name,omitempty"`
	Description              string    `xml:"description,omitempty"`
	DeprecationTime          string    `xml:"deprecationTime,omitempty"`
	RootDeviceType           string    `xml:"rootDeviceType,omitempty"`
	RootDeviceName           string    `xml:"rootDeviceName,omitempty"`
	VirtualizationType       string    `xml:"virtualizationType,omitempty"`
	CreationDate             string    `xml:"creationDate,omitempty"`
	TagSet                   ec2TagSet `xml:"tagSet"`
}

type ec2DescribeImageAttributeResponse struct {
	XMLName          xml.Name                     `xml:"DescribeImageAttributeResponse"`
	Xmlns            string                       `xml:"xmlns,attr"`
	RequestID        string                       `xml:"requestId"`
	ImageID          string                       `xml:"imageId"`
	Description      *ec2StringAttributeValue     `xml:"description,omitempty"`
	LaunchPermission *ec2ImageLaunchPermissionSet `xml:"launchPermission,omitempty"`
}

type ec2ImageLaunchPermissionSet struct {
	Items []ec2ImageLaunchPermissionItem `xml:"item"`
}

type ec2ImageLaunchPermissionItem struct {
	UserID string `xml:"userId,omitempty"`
	Group  string `xml:"group,omitempty"`
}
