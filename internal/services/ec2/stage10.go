package ec2

import (
	"encoding/base64"
	"strings"
	"time"
)

type ConsoleOutput struct {
	InstanceID string
	Output     string
	Timestamp  time.Time
}

type ConsoleScreenshot struct {
	InstanceID string
	ImageData  string
}

type PasswordData struct {
	InstanceID   string
	PasswordData string
	Timestamp    time.Time
}

func (s *Service) GetConsoleOutput(instanceID string, latest bool) (ConsoleOutput, error) {
	instanceID = strings.TrimSpace(instanceID)
	_ = latest
	if instanceID == "" {
		return ConsoleOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return ConsoleOutput{}, ErrNotFound
	}
	raw := "Stackyard console output for " + instance.ID + "\nstate=" + instance.StateName + "\ninstanceType=" + instance.InstanceType
	return ConsoleOutput{
		InstanceID: instance.ID,
		Output:     base64.StdEncoding.EncodeToString([]byte(raw)),
		Timestamp:  time.Now().UTC(),
	}, nil
}

func (s *Service) GetConsoleScreenshot(instanceID string, wakeUp bool) (ConsoleScreenshot, error) {
	instanceID = strings.TrimSpace(instanceID)
	_ = wakeUp
	if instanceID == "" {
		return ConsoleScreenshot{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return ConsoleScreenshot{}, ErrNotFound
	}

	// 1x1 transparent PNG.
	const pixelPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8Xw8AAusB9Y1nX8sAAAAASUVORK5CYII="
	return ConsoleScreenshot{
		InstanceID: instance.ID,
		ImageData:  pixelPNGBase64,
	}, nil
}

func (s *Service) GetPasswordData(instanceID string) (PasswordData, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return PasswordData{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return PasswordData{}, ErrNotFound
	}

	secret := "stackyard-password-" + instance.ID
	return PasswordData{
		InstanceID:   instance.ID,
		PasswordData: base64.StdEncoding.EncodeToString([]byte(secret)),
		Timestamp:    time.Now().UTC(),
	}, nil
}
