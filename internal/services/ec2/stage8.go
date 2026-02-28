package ec2

import (
	"sort"
	"strings"
	"time"
)

type Image struct {
	ID                       string
	Name                     string
	Description              string
	DeprecationTime          *time.Time
	DeregistrationProtection string
	State                    string
	OwnerID                  string
	ImageLocation            string
	Architecture             string
	ImageType                string
	RootDeviceType           string
	RootDeviceName           string
	VirtualizationType       string
	CreationDate             time.Time
	Tags                     map[string]string
	LaunchPermissions        []LaunchPermission
}

type LaunchPermission struct {
	UserID string
	Group  string
}

type ImageAttribute struct {
	ImageID           string
	Description       string
	LaunchPermissions []LaunchPermission
}

type LaunchPermissionModifications struct {
	Add    []LaunchPermission
	Remove []LaunchPermission
}

func (s *Service) CreateImage(instanceID, name, description string, noReboot bool, tags []Tag) (Image, error) {
	instanceID = strings.TrimSpace(instanceID)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	_ = noReboot
	if instanceID == "" || name == "" {
		return Image{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return Image{}, ErrNotFound
	}

	image := &Image{
		ID:                       s.nextIDLocked("ami"),
		Name:                     name,
		Description:              description,
		DeprecationTime:          nil,
		DeregistrationProtection: "disabled",
		State:                    "available",
		OwnerID:                  DefaultAccountID,
		ImageLocation:            DefaultAccountID + "/" + name,
		Architecture:             "x86_64",
		ImageType:                "machine",
		RootDeviceType:           "ebs",
		RootDeviceName:           "/dev/sda1",
		VirtualizationType:       "hvm",
		CreationDate:             time.Now().UTC(),
		Tags:                     tagsToMap(tags),
		LaunchPermissions:        nil,
	}
	s.images[image.ID] = image
	return cloneImage(image), nil
}

func (s *Service) DescribeImages(imageIDs, owners []string) []Image {
	s.mu.Lock()
	defer s.mu.Unlock()

	imageIDSet := toStringSet(imageIDs)
	ownerSet := map[string]struct{}{}
	for _, owner := range owners {
		owner = strings.TrimSpace(owner)
		if owner == "" {
			continue
		}
		if strings.EqualFold(owner, "self") {
			owner = DefaultAccountID
		}
		ownerSet[owner] = struct{}{}
	}

	out := make([]Image, 0, len(s.images))
	for _, image := range s.images {
		if len(imageIDSet) > 0 {
			if _, ok := imageIDSet[image.ID]; !ok {
				continue
			}
		}
		if len(ownerSet) > 0 {
			if _, ok := ownerSet[image.OwnerID]; !ok {
				continue
			}
		}
		out = append(out, cloneImage(image))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeregisterImage(imageID string) error {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.images[imageID] == nil {
		return ErrNotFound
	}
	delete(s.images, imageID)
	return nil
}

func (s *Service) DescribeImageAttribute(imageID, attribute string) (ImageAttribute, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ImageAttribute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return ImageAttribute{}, ErrNotFound
	}

	switch normalizeImageAttributeName(attribute) {
	case "", "description":
		return ImageAttribute{
			ImageID:     image.ID,
			Description: image.Description,
		}, nil
	case "launchPermission":
		return ImageAttribute{
			ImageID:           image.ID,
			LaunchPermissions: cloneLaunchPermissions(image.LaunchPermissions),
		}, nil
	default:
		return ImageAttribute{}, ErrInvalidParameter
	}
}

func (s *Service) ModifyImageAttribute(imageID, attribute string, description *string, launchPermission *LaunchPermissionModifications) error {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return ErrNotFound
	}

	attrName := normalizeImageAttributeName(attribute)
	if attrName == "" {
		switch {
		case description != nil:
			attrName = "description"
		case launchPermission != nil:
			attrName = "launchPermission"
		default:
			return ErrInvalidParameter
		}
	}

	switch attrName {
	case "description":
		if description == nil {
			return ErrInvalidParameter
		}
		image.Description = strings.TrimSpace(*description)
	case "launchPermission":
		if launchPermission == nil {
			return ErrInvalidParameter
		}
		image.LaunchPermissions = applyLaunchPermissionChanges(image.LaunchPermissions, launchPermission.Add, launchPermission.Remove)
	default:
		return ErrInvalidParameter
	}
	return nil
}

func (s *Service) ResetImageAttribute(imageID, attribute string) error {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return ErrNotFound
	}

	attrName := normalizeImageAttributeName(attribute)
	if attrName == "" {
		attrName = "launchPermission"
	}
	if attrName != "launchPermission" {
		return ErrInvalidParameter
	}
	image.LaunchPermissions = nil
	return nil
}

func normalizeImageAttributeName(attribute string) string {
	attribute = strings.TrimSpace(attribute)
	switch strings.ToLower(attribute) {
	case "":
		return ""
	case "description":
		return "description"
	case "launchpermission":
		return "launchPermission"
	default:
		return ""
	}
}

func applyLaunchPermissionChanges(current, add, remove []LaunchPermission) []LaunchPermission {
	index := map[string]LaunchPermission{}
	for _, permission := range current {
		normalized, ok := normalizeLaunchPermission(permission)
		if !ok {
			continue
		}
		index[launchPermissionKey(normalized)] = normalized
	}
	for _, permission := range add {
		normalized, ok := normalizeLaunchPermission(permission)
		if !ok {
			continue
		}
		index[launchPermissionKey(normalized)] = normalized
	}
	for _, permission := range remove {
		normalized, ok := normalizeLaunchPermission(permission)
		if !ok {
			continue
		}
		delete(index, launchPermissionKey(normalized))
	}

	out := make([]LaunchPermission, 0, len(index))
	for _, permission := range index {
		out = append(out, permission)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].UserID < out[j].UserID
	})
	return out
}

func normalizeLaunchPermission(permission LaunchPermission) (LaunchPermission, bool) {
	permission.UserID = strings.TrimSpace(permission.UserID)
	permission.Group = strings.TrimSpace(permission.Group)
	if permission.UserID == "" && permission.Group == "" {
		return LaunchPermission{}, false
	}
	return permission, true
}

func launchPermissionKey(permission LaunchPermission) string {
	return permission.Group + "|" + permission.UserID
}

func cloneImage(in *Image) Image {
	if in == nil {
		return Image{}
	}
	return Image{
		ID:                       in.ID,
		Name:                     in.Name,
		Description:              in.Description,
		DeprecationTime:          cloneTimePointer(in.DeprecationTime),
		DeregistrationProtection: in.DeregistrationProtection,
		State:                    in.State,
		OwnerID:                  in.OwnerID,
		ImageLocation:            in.ImageLocation,
		Architecture:             in.Architecture,
		ImageType:                in.ImageType,
		RootDeviceType:           in.RootDeviceType,
		RootDeviceName:           in.RootDeviceName,
		VirtualizationType:       in.VirtualizationType,
		CreationDate:             in.CreationDate,
		Tags:                     cloneStringMap(in.Tags),
		LaunchPermissions:        cloneLaunchPermissions(in.LaunchPermissions),
	}
}

func cloneLaunchPermissions(in []LaunchPermission) []LaunchPermission {
	out := make([]LaunchPermission, 0, len(in))
	for _, permission := range in {
		out = append(out, permission)
	}
	return out
}
