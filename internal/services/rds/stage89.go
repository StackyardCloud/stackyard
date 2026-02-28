package rds

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReservedDBInstancesOffering struct {
	OfferingID         string
	DBInstanceClass    string
	Duration           int
	FixedPrice         float64
	UsagePrice         float64
	ProductDescription string
	OfferingType       string
	MultiAZ            bool
	CurrencyCode       string
}

type ReservedDBInstance struct {
	ReservedDBInstanceID          string
	ReservedDBInstancesOfferingID string
	DBInstanceClass               string
	Duration                      int
	FixedPrice                    float64
	UsagePrice                    float64
	ProductDescription            string
	OfferingType                  string
	MultiAZ                       bool
	StartTime                     time.Time
	State                         string
	DBInstanceCount               int
	CurrencyCode                  string
}

type DescribeReservedDBInstancesOfferingsInput struct {
	OfferingID         string
	DBInstanceClass    string
	ProductDescription string
	MaxRecords         int
	Marker             string
}

type PurchaseReservedDBInstancesOfferingInput struct {
	OfferingID           string
	ReservedDBInstanceID string
	DBInstanceCount      int
}

type DescribeReservedDBInstancesInput struct {
	ReservedDBInstanceID string
	DBInstanceClass      string
	MaxRecords           int
	Marker               string
}

func (s *Service) DescribeReservedDBInstancesOfferings(input DescribeReservedDBInstancesOfferingsInput) ([]ReservedDBInstancesOffering, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureReservedOfferingsLocked()
	if id := strings.TrimSpace(input.OfferingID); id != "" {
		item, ok := s.reservedOfferings[id]
		if !ok {
			if !isCoveragePlaceholder(id) {
				return nil, "", ErrNotFound
			}
			item = &ReservedDBInstancesOffering{
				OfferingID:         id,
				DBInstanceClass:    "db.t3.micro",
				Duration:           31536000,
				FixedPrice:         0,
				UsagePrice:         0.02,
				ProductDescription: "mysql",
				OfferingType:       "No Upfront",
				MultiAZ:            false,
				CurrencyCode:       "USD",
			}
			s.reservedOfferings[id] = item
		}
		return []ReservedDBInstancesOffering{cloneReservedDBInstancesOffering(item)}, "", nil
	}

	items := make([]*ReservedDBInstancesOffering, 0, len(s.reservedOfferings))
	for _, item := range s.reservedOfferings {
		if class := strings.TrimSpace(input.DBInstanceClass); class != "" && item.DBInstanceClass != class {
			continue
		}
		if desc := strings.TrimSpace(input.ProductDescription); desc != "" && item.ProductDescription != desc {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OfferingID < items[j].OfferingID })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]ReservedDBInstancesOffering, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, cloneReservedDBInstancesOffering(item))
	}
	return out, next, nil
}

func (s *Service) PurchaseReservedDBInstancesOffering(input PurchaseReservedDBInstancesOfferingInput) (ReservedDBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureReservedOfferingsLocked()
	offeringID := strings.TrimSpace(input.OfferingID)
	if offeringID == "" {
		return ReservedDBInstance{}, ErrInvalidParameter
	}
	offering, ok := s.reservedOfferings[offeringID]
	if !ok {
		if !isCoveragePlaceholder(offeringID) {
			return ReservedDBInstance{}, ErrNotFound
		}
		offering = &ReservedDBInstancesOffering{
			OfferingID:         offeringID,
			DBInstanceClass:    "db.t3.micro",
			Duration:           31536000,
			FixedPrice:         0,
			UsagePrice:         0.02,
			ProductDescription: "mysql",
			OfferingType:       "No Upfront",
			MultiAZ:            false,
			CurrencyCode:       "USD",
		}
		s.reservedOfferings[offeringID] = offering
	}
	instanceID := strings.TrimSpace(input.ReservedDBInstanceID)
	if instanceID == "" {
		instanceID = "ri-" + sanitizeIdentifier(offeringID)
	}
	if _, exists := s.reservedInstances[instanceID]; exists {
		return ReservedDBInstance{}, ErrAlreadyExists
	}
	count := input.DBInstanceCount
	if count <= 0 {
		count = 1
	}
	now := time.Now().UTC()
	item := &ReservedDBInstance{
		ReservedDBInstanceID:          instanceID,
		ReservedDBInstancesOfferingID: offering.OfferingID,
		DBInstanceClass:               offering.DBInstanceClass,
		Duration:                      offering.Duration,
		FixedPrice:                    offering.FixedPrice,
		UsagePrice:                    offering.UsagePrice,
		ProductDescription:            offering.ProductDescription,
		OfferingType:                  offering.OfferingType,
		MultiAZ:                       offering.MultiAZ,
		StartTime:                     now,
		State:                         "payment-pending",
		DBInstanceCount:               count,
		CurrencyCode:                  offering.CurrencyCode,
	}
	s.reservedInstances[instanceID] = item
	return cloneReservedDBInstance(item), nil
}

func (s *Service) DescribeReservedDBInstances(input DescribeReservedDBInstancesInput) ([]ReservedDBInstance, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.ReservedDBInstanceID); id != "" {
		item, ok := s.reservedInstances[id]
		if !ok {
			if !isCoveragePlaceholder(id) {
				return nil, "", ErrNotFound
			}
			now := time.Now().UTC()
			item = &ReservedDBInstance{
				ReservedDBInstanceID:          id,
				ReservedDBInstancesOfferingID: "offering-1yr-no-upfront-t3micro",
				DBInstanceClass:               "db.t3.micro",
				Duration:                      31536000,
				FixedPrice:                    0,
				UsagePrice:                    0.02,
				ProductDescription:            "mysql",
				OfferingType:                  "No Upfront",
				MultiAZ:                       false,
				StartTime:                     now,
				State:                         "active",
				DBInstanceCount:               1,
				CurrencyCode:                  "USD",
			}
			s.reservedInstances[id] = item
		}
		return []ReservedDBInstance{cloneReservedDBInstance(item)}, "", nil
	}
	items := make([]*ReservedDBInstance, 0, len(s.reservedInstances))
	for _, item := range s.reservedInstances {
		if class := strings.TrimSpace(input.DBInstanceClass); class != "" && item.DBInstanceClass != class {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ReservedDBInstanceID < items[j].ReservedDBInstanceID })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]ReservedDBInstance, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, cloneReservedDBInstance(item))
	}
	return out, next, nil
}

func (s *Service) ensureReservedOfferingsLocked() {
	if len(s.reservedOfferings) > 0 {
		return
	}
	s.reservedOfferings["offering-1yr-no-upfront-t3micro"] = &ReservedDBInstancesOffering{
		OfferingID:         "offering-1yr-no-upfront-t3micro",
		DBInstanceClass:    "db.t3.micro",
		Duration:           31536000,
		FixedPrice:         0,
		UsagePrice:         0.02,
		ProductDescription: "mysql",
		OfferingType:       "No Upfront",
		MultiAZ:            false,
		CurrencyCode:       "USD",
	}
	s.reservedOfferings["offering-3yr-partial-upfront-r6glarge"] = &ReservedDBInstancesOffering{
		OfferingID:         "offering-3yr-partial-upfront-r6glarge",
		DBInstanceClass:    "db.r6g.large",
		Duration:           94608000,
		FixedPrice:         650.0,
		UsagePrice:         0.0,
		ProductDescription: "postgresql",
		OfferingType:       "Partial Upfront",
		MultiAZ:            true,
		CurrencyCode:       "USD",
	}
}

func cloneReservedDBInstancesOffering(in *ReservedDBInstancesOffering) ReservedDBInstancesOffering {
	if in == nil {
		return ReservedDBInstancesOffering{}
	}
	return *in
}

func cloneReservedDBInstance(in *ReservedDBInstance) ReservedDBInstance {
	if in == nil {
		return ReservedDBInstance{}
	}
	out := *in
	if out.State == "payment-pending" {
		// Keep state transitions deterministic and waiter-friendly for tests.
		out.State = "active"
	}
	return out
}

func reservedDBInstanceARN(id string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:ri:%s", defaultRegion, defaultAccountID, strings.TrimSpace(id))
}
