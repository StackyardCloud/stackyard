package rds

import (
	"errors"
	"testing"
)

func TestCapacityReservationLifecycle(t *testing.T) {
	s := NewService()
	in := CreateCapacityReservationInput{Name: "cr1", DBInstanceClass: "db.r6g.large", Engine: "postgres", CapacityPerAZ: 2, AvailabilityZones: []string{"us-east-1a"}}
	created, err := s.CreateCapacityReservation(in)
	if err != nil || created.State != "active" || created.Identifier == "" {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := s.CreateCapacityReservation(in); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := s.CreateCapacityReservation(CreateCapacityReservationInput{Name: "x"}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("missing fields: %v", err)
	}
	mod, err := s.ModifyCapacityReservation(ModifyCapacityReservationInput{Identifier: "cr1", CapacityPerAZ: 5})
	if err != nil || mod.CapacityPerAZ != 5 {
		t.Fatalf("modify by name: %+v %v", mod, err)
	}
	if _, err := s.CancelCapacityReservation(created.Identifier); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := s.CancelCapacityReservation(created.Identifier); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("double cancel: %v", err)
	}
	if _, _, err := s.DescribeCapacityReservations("nope", "", 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("describe missing: %v", err)
	}
	items, _, err := s.DescribeCapacityReservations("", "", 0)
	if err != nil || len(items) != 1 || items[0].State != "cancelled" {
		t.Fatalf("describe all: %+v %v", items, err)
	}
}
