package ec2

import "strings"

func (s *Service) CopyImage(
	name string,
	sourceImageID string,
	sourceRegion string,
	description *string,
	clientToken *string,
	copyImageTags *bool,
	encrypted *bool,
	kmsKeyID *string,
) (string, error) {
	name = strings.TrimSpace(name)
	sourceImageID = strings.TrimSpace(sourceImageID)
	sourceRegion = strings.TrimSpace(sourceRegion)
	if name == "" || sourceImageID == "" || sourceRegion == "" {
		return "", ErrInvalidParameter
	}
	_ = clientToken
	_ = encrypted
	_ = kmsKeyID

	s.mu.Lock()
	defer s.mu.Unlock()

	source := s.images[sourceImageID]
	if source == nil {
		return "", ErrNotFound
	}

	newID := s.nextIDLocked("ami")
	newDescription := source.Description
	if description != nil {
		newDescription = strings.TrimSpace(*description)
	}
	newTags := map[string]string{}
	if copyImageTags != nil && *copyImageTags {
		newTags = cloneStringMap(source.Tags)
	}

	s.images[newID] = &Image{
		ID:                       newID,
		Name:                     name,
		Description:              newDescription,
		DeprecationTime:          nil,
		DeregistrationProtection: source.DeregistrationProtection,
		State:                    "available",
		OwnerID:                  DefaultAccountID,
		ImageLocation:            DefaultAccountID + "/" + name,
		Architecture:             source.Architecture,
		ImageType:                source.ImageType,
		RootDeviceType:           source.RootDeviceType,
		RootDeviceName:           source.RootDeviceName,
		VirtualizationType:       source.VirtualizationType,
		CreationDate:             source.CreationDate,
		Tags:                     newTags,
		LaunchPermissions:        cloneLaunchPermissions(source.LaunchPermissions),
	}
	return newID, nil
}
