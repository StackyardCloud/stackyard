package ec2

import "strings"

type SerialConsoleAccessStatus struct {
	ManagedBy                  string
	SerialConsoleAccessEnabled bool
}

func (s *Service) EnableSerialConsoleAccess() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.serialConsoleAccessEnabled = true
	s.serialConsoleManagedBy = "account"
	return true
}

func (s *Service) DisableSerialConsoleAccess() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.serialConsoleAccessEnabled = false
	s.serialConsoleManagedBy = "account"
	return false
}

func (s *Service) GetSerialConsoleAccessStatus() SerialConsoleAccessStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return SerialConsoleAccessStatus{
		ManagedBy:                  s.serialConsoleManagedBy,
		SerialConsoleAccessEnabled: s.serialConsoleAccessEnabled,
	}
}

func (s *Service) EnableIpamOrganizationAdminAccount(delegatedAdminAccountID string) (bool, error) {
	delegatedAdminAccountID = strings.TrimSpace(delegatedAdminAccountID)
	if delegatedAdminAccountID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ipamOrganizationAdminAccountID = delegatedAdminAccountID
	return true, nil
}

func (s *Service) DisableIpamOrganizationAdminAccount(delegatedAdminAccountID string) (bool, error) {
	delegatedAdminAccountID = strings.TrimSpace(delegatedAdminAccountID)
	if delegatedAdminAccountID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ipamOrganizationAdminAccountID = ""
	return true, nil
}
