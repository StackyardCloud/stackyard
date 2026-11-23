package rds

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	maxCapacityReservations     = 1000
	maxCapacityReservationAZs   = 16
	maxCapacityReservationName  = 255
	maxCapacityReservationPerAZ = 10000
)

// ErrCapacityReservationState is returned when a reservation cannot change in its current state.
var ErrCapacityReservationState = fmt.Errorf("%w: capacity reservation", ErrInvalidState)

type CapacityReservation struct {
	Identifier        string
	Arn               string
	Name              string
	DBInstanceClass   string
	Engine            string
	LicenseModel      string
	CapacityPerAZ     int
	AvailabilityZones []string
	StartDate         time.Time
	EndDate           time.Time
	CreateDate        time.Time
	State             string
	Tags              map[string]string
}

type CreateCapacityReservationInput struct {
	Name              string
	DBInstanceClass   string
	Engine            string
	LicenseModel      string
	CapacityPerAZ     int
	AvailabilityZones []string
	StartDate         time.Time
	EndDate           time.Time
	Tags              map[string]string
}

type ModifyCapacityReservationInput struct {
	Identifier    string
	CapacityPerAZ int // 0 = unchanged
	EndDateType   string
	EndDate       time.Time
}

func (s *Service) CreateCapacityReservation(in CreateCapacityReservationInput) (CapacityReservation, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > maxCapacityReservationName || strings.TrimSpace(in.DBInstanceClass) == "" ||
		strings.TrimSpace(in.Engine) == "" || in.CapacityPerAZ < 1 || in.CapacityPerAZ > maxCapacityReservationPerAZ ||
		len(in.AvailabilityZones) == 0 || len(in.AvailabilityZones) > maxCapacityReservationAZs {
		return CapacityReservation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capacityReservations == nil {
		s.capacityReservations = map[string]*CapacityReservation{}
	}
	for _, existing := range s.capacityReservations {
		if existing.Name == name {
			return CapacityReservation{}, ErrAlreadyExists
		}
	}
	if len(s.capacityReservations) >= maxCapacityReservations {
		return CapacityReservation{}, ErrInvalidParameter
	}
	now := time.Now().UTC()
	start := in.StartDate
	if start.IsZero() {
		start = now
	}
	id := newCapacityReservationID()
	item := &CapacityReservation{
		Identifier:        id,
		Arn:               fmt.Sprintf("arn:aws:rds:%s:%s:db-capacity-reservation:%s", defaultRegion, defaultAccountID, id),
		Name:              name,
		DBInstanceClass:   strings.TrimSpace(in.DBInstanceClass),
		Engine:            strings.TrimSpace(in.Engine),
		LicenseModel:      strings.TrimSpace(in.LicenseModel),
		CapacityPerAZ:     in.CapacityPerAZ,
		AvailabilityZones: append([]string(nil), in.AvailabilityZones...),
		StartDate:         start,
		EndDate:           in.EndDate,
		CreateDate:        now,
		State:             "active",
		Tags:              cloneTagMap(in.Tags),
	}
	s.capacityReservations[id] = item
	return cloneCapacityReservation(item), nil
}

// findCapacityReservationLocked resolves an identifier or name; coverage
// placeholders resolve to a seeded reservation like other RDS resources.
func (s *Service) findCapacityReservationLocked(ref string) (*CapacityReservation, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrInvalidParameter
	}
	if item, ok := s.capacityReservations[ref]; ok {
		return item, nil
	}
	for _, item := range s.capacityReservations {
		if item.Name == ref {
			return item, nil
		}
	}
	if !isCoveragePlaceholder(ref) {
		return nil, ErrNotFound
	}
	if s.capacityReservations == nil {
		s.capacityReservations = map[string]*CapacityReservation{}
	}
	if len(s.capacityReservations) >= maxCapacityReservations {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	item := &CapacityReservation{
		Identifier:        ref,
		Arn:               fmt.Sprintf("arn:aws:rds:%s:%s:db-capacity-reservation:%s", defaultRegion, defaultAccountID, ref),
		Name:              ref,
		DBInstanceClass:   "db.r6g.large",
		Engine:            "postgres",
		CapacityPerAZ:     1,
		AvailabilityZones: []string{defaultRegion + "a"},
		StartDate:         now,
		CreateDate:        now,
		State:             "active",
	}
	s.capacityReservations[ref] = item
	return item, nil
}

func (s *Service) DescribeCapacityReservations(ref, marker string, maxRecords int) ([]CapacityReservation, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(ref) != "" {
		item, err := s.findCapacityReservationLocked(ref)
		if err != nil {
			return nil, "", err
		}
		return []CapacityReservation{cloneCapacityReservation(item)}, "", nil
	}
	items := make([]*CapacityReservation, 0, len(s.capacityReservations))
	for _, item := range s.capacityReservations {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), marker, maxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]CapacityReservation, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, cloneCapacityReservation(item))
	}
	return out, next, nil
}

func (s *Service) ModifyCapacityReservation(in ModifyCapacityReservationInput) (CapacityReservation, error) {
	if in.CapacityPerAZ < 0 || in.CapacityPerAZ > maxCapacityReservationPerAZ {
		return CapacityReservation{}, ErrInvalidParameter
	}
	endType := strings.TrimSpace(in.EndDateType)
	if endType != "" && endType != "unlimited" && endType != "limited" {
		return CapacityReservation{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.findCapacityReservationLocked(in.Identifier)
	if err != nil {
		return CapacityReservation{}, err
	}
	if item.State == "cancelled" || item.State == "expired" {
		return CapacityReservation{}, ErrCapacityReservationState
	}
	if in.CapacityPerAZ > 0 {
		item.CapacityPerAZ = in.CapacityPerAZ
	}
	switch {
	case endType == "unlimited":
		item.EndDate = time.Time{}
	case !in.EndDate.IsZero():
		item.EndDate = in.EndDate
	}
	return cloneCapacityReservation(item), nil
}

func (s *Service) CancelCapacityReservation(ref string) (CapacityReservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.findCapacityReservationLocked(ref)
	if err != nil {
		return CapacityReservation{}, err
	}
	if item.State == "cancelled" || item.State == "expired" {
		return CapacityReservation{}, ErrCapacityReservationState
	}
	item.State = "cancelled"
	return cloneCapacityReservation(item), nil
}

func cloneCapacityReservation(in *CapacityReservation) CapacityReservation {
	out := *in
	out.AvailabilityZones = append([]string(nil), in.AvailabilityZones...)
	out.Tags = cloneTagMap(in.Tags)
	return out
}

func newCapacityReservationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "dbcr-" + hex.EncodeToString(b)
}
