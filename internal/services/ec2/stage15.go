package ec2

import (
	"sort"
	"strings"
	"time"
)

type AddressTransfer struct {
	AddressTransferStatus          string
	AllocationID                   string
	PublicIP                       string
	TransferAccountID              string
	TransferOfferAcceptedTimestamp *time.Time
	TransferOfferExpirationTime    *time.Time
}

type MovingAddressStatus struct {
	MoveStatus string
	PublicIP   string
}

func (s *Service) EnableAddressTransfer(allocationID, transferAccountID string) (AddressTransfer, error) {
	allocationID = strings.TrimSpace(allocationID)
	transferAccountID = strings.TrimSpace(transferAccountID)
	if allocationID == "" || transferAccountID == "" {
		return AddressTransfer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.addresses[allocationID]
	if address == nil {
		return AddressTransfer{}, ErrNotFound
	}

	now := time.Now().UTC()
	expiration := now.Add(7 * 24 * time.Hour)
	transfer := &AddressTransfer{
		AddressTransferStatus:       "pending",
		AllocationID:                address.AllocationID,
		PublicIP:                    address.PublicIP,
		TransferAccountID:           transferAccountID,
		TransferOfferExpirationTime: &expiration,
	}
	s.addressTransfers[allocationID] = transfer
	return cloneAddressTransfer(transfer), nil
}

func (s *Service) DisableAddressTransfer(allocationID string) (AddressTransfer, error) {
	allocationID = strings.TrimSpace(allocationID)
	if allocationID == "" {
		return AddressTransfer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	transfer := s.addressTransfers[allocationID]
	if transfer == nil {
		return AddressTransfer{}, ErrNotFound
	}
	transfer.AddressTransferStatus = "disabled"
	return cloneAddressTransfer(transfer), nil
}

func (s *Service) AcceptAddressTransfer(publicIP string) (AddressTransfer, error) {
	publicIP = strings.TrimSpace(publicIP)
	if publicIP == "" {
		return AddressTransfer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	address := s.resolveElasticAddressLocked("", publicIP)
	if address == nil {
		return AddressTransfer{}, ErrNotFound
	}
	transfer := s.addressTransfers[address.AllocationID]
	if transfer == nil {
		return AddressTransfer{}, ErrNotFound
	}
	if transfer.AddressTransferStatus != "pending" {
		return AddressTransfer{}, ErrConflict
	}

	now := time.Now().UTC()
	transfer.AddressTransferStatus = "accepted"
	transfer.TransferOfferAcceptedTimestamp = &now
	return cloneAddressTransfer(transfer), nil
}

func (s *Service) DescribeAddressTransfers(allocationIDs []string) []AddressTransfer {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(allocationIDs)
	out := make([]AddressTransfer, 0, len(s.addressTransfers))
	for allocationID, transfer := range s.addressTransfers {
		if len(idSet) > 0 {
			if _, ok := idSet[allocationID]; !ok {
				continue
			}
		}
		out = append(out, cloneAddressTransfer(transfer))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AllocationID < out[j].AllocationID })
	return out
}

func (s *Service) DescribeMovingAddresses(publicIPs []string) []MovingAddressStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	publicIPSet := toStringSet(publicIPs)
	out := make([]MovingAddressStatus, 0)
	for _, transfer := range s.addressTransfers {
		if len(publicIPSet) > 0 {
			if _, ok := publicIPSet[transfer.PublicIP]; !ok {
				continue
			}
		}
		switch transfer.AddressTransferStatus {
		case "pending":
			out = append(out, MovingAddressStatus{MoveStatus: "movingToVpc", PublicIP: transfer.PublicIP})
		case "disabled":
			out = append(out, MovingAddressStatus{MoveStatus: "restoringToClassic", PublicIP: transfer.PublicIP})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PublicIP < out[j].PublicIP })
	return out
}

func cloneAddressTransfer(in *AddressTransfer) AddressTransfer {
	out := *in
	if in.TransferOfferAcceptedTimestamp != nil {
		ts := *in.TransferOfferAcceptedTimestamp
		out.TransferOfferAcceptedTimestamp = &ts
	}
	if in.TransferOfferExpirationTime != nil {
		ts := *in.TransferOfferExpirationTime
		out.TransferOfferExpirationTime = &ts
	}
	return out
}
