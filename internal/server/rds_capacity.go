package server

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
	"time"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

type rdsCapacityAZXML struct {
	AvailabilityZone  string `xml:"AvailabilityZone"`
	AvailableCapacity int    `xml:"AvailableCapacity"`
}

type rdsCapacityReservationXML struct {
	Name          string             `xml:"DBCapacityReservationName"`
	InstanceClass string             `xml:"DBInstanceClass"`
	Engine        string             `xml:"Engine"`
	LicenseModel  string             `xml:"LicenseModel,omitempty"`
	CapacityPerAZ int                `xml:"ReservedCapacityPerAvailabilityZone"`
	Available     []rdsCapacityAZXML `xml:"AvailableCapacityPerAvailabilityZoneList>AvailableCapacityPerAvailabilityZone"`
	EndDate       string             `xml:"EndDate,omitempty"`
	CreateDate    string             `xml:"CreateDate"`
	StartDate     string             `xml:"StartDate"`
	State         string             `xml:"State"`
	Arn           string             `xml:"DBCapacityReservationArn"`
	Identifier    string             `xml:"DBCapacityReservationIdentifier"`
	Tags          []rdsTagXML        `xml:"TagList>Tag"`
}

func rdsCapacityReservationToXML(in rdssvc.CapacityReservation) rdsCapacityReservationXML {
	out := rdsCapacityReservationXML{
		Name: in.Name, InstanceClass: in.DBInstanceClass, Engine: in.Engine, LicenseModel: in.LicenseModel,
		CapacityPerAZ: in.CapacityPerAZ, EndDate: formatRDSTime(in.EndDate), CreateDate: formatRDSTime(in.CreateDate),
		StartDate: formatRDSTime(in.StartDate), State: in.State, Arn: in.Arn, Identifier: in.Identifier,
		Available: []rdsCapacityAZXML{}, Tags: []rdsTagXML{},
	}
	available := in.CapacityPerAZ
	if in.State != "active" {
		available = 0
	}
	for _, az := range in.AvailabilityZones {
		out.Available = append(out.Available, rdsCapacityAZXML{AvailabilityZone: az, AvailableCapacity: available})
	}
	for k, v := range in.Tags {
		out.Tags = append(out.Tags, rdsTagXML{Key: k, Value: v})
	}
	return out
}

func parseRDSOptionalTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func respondRDSCapacityError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, rdssvc.ErrAlreadyExists):
		respondRDSErrorXML(w, http.StatusBadRequest, "DBCapacityReservationAlreadyExistsFault", "capacity reservation already exists")
	case errors.Is(err, rdssvc.ErrNotFound):
		respondRDSErrorXML(w, http.StatusNotFound, "DBCapacityReservationNotFoundFault", "capacity reservation not found")
	case errors.Is(err, rdssvc.ErrInvalidState):
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidDBCapacityReservationStateFault", "capacity reservation is not in a valid state")
	default:
		respondRDSServiceError(w, action, err)
	}
}

func respondRDSCapacityReservation(w http.ResponseWriter, action string, item rdssvc.CapacityReservation) {
	respondRDSXML(w, action, struct {
		XMLName     xml.Name
		Reservation rdsCapacityReservationXML `xml:"DBCapacityReservation"`
	}{XMLName: xml.Name{Local: action + "Result"}, Reservation: rdsCapacityReservationToXML(item)})
}

func (s *Server) handleRDSCreateDBCapacityReservation(w http.ResponseWriter, r *http.Request) {
	capacity, err := parseOptionalRDSInt(r.Form.Get("CapacityPerAvailabilityZone"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "CapacityPerAvailabilityZone is invalid")
		return
	}
	start, err := parseRDSOptionalTime(r.Form.Get("StartDate"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "StartDate is invalid")
		return
	}
	end, err := parseRDSOptionalTime(r.Form.Get("EndDate"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "EndDate is invalid")
		return
	}
	tags := parseRDSTagMembers(r.Form, "Tags.Tag")
	if len(tags) == 0 {
		tags = parseRDSTagMembers(r.Form, "Tags.member")
	}
	item, err := s.rds.CreateCapacityReservation(rdssvc.CreateCapacityReservationInput{
		Name:              strings.TrimSpace(r.Form.Get("DBCapacityReservationName")),
		DBInstanceClass:   strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		Engine:            strings.TrimSpace(r.Form.Get("Engine")),
		LicenseModel:      strings.TrimSpace(r.Form.Get("LicenseModel")),
		CapacityPerAZ:     capacity,
		AvailabilityZones: parseRDSListMembers(r.Form, "AvailabilityZones.AvailabilityZone"),
		StartDate:         start,
		EndDate:           end,
		Tags:              tags,
	})
	if err != nil {
		respondRDSCapacityError(w, "CreateDBCapacityReservation", err)
		return
	}
	respondRDSCapacityReservation(w, "CreateDBCapacityReservation", item)
}

func (s *Server) handleRDSModifyDBCapacityReservation(w http.ResponseWriter, r *http.Request) {
	capacity, err := parseOptionalRDSInt(r.Form.Get("CapacityPerAvailabilityZone"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "CapacityPerAvailabilityZone is invalid")
		return
	}
	end, err := parseRDSOptionalTime(r.Form.Get("EndDate"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "EndDate is invalid")
		return
	}
	item, err := s.rds.ModifyCapacityReservation(rdssvc.ModifyCapacityReservationInput{
		Identifier:    strings.TrimSpace(r.Form.Get("DBCapacityReservationIdentifier")),
		CapacityPerAZ: capacity,
		EndDateType:   strings.TrimSpace(r.Form.Get("EndDateType")),
		EndDate:       end,
	})
	if err != nil {
		respondRDSCapacityError(w, "ModifyDBCapacityReservation", err)
		return
	}
	respondRDSCapacityReservation(w, "ModifyDBCapacityReservation", item)
}

func (s *Server) handleRDSCancelDBCapacityReservation(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.CancelCapacityReservation(r.Form.Get("DBCapacityReservationIdentifier"))
	if err != nil {
		respondRDSCapacityError(w, "CancelDBCapacityReservation", err)
		return
	}
	respondRDSCapacityReservation(w, "CancelDBCapacityReservation", item)
}

func (s *Server) handleRDSDescribeDBCapacityReservations(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeCapacityReservations(r.Form.Get("DBCapacityReservationIdentifier"), strings.TrimSpace(r.Form.Get("Marker")), maxRecords)
	if err != nil {
		respondRDSCapacityError(w, "DescribeDBCapacityReservations", err)
		return
	}
	out := make([]rdsCapacityReservationXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsCapacityReservationToXML(item))
	}
	respondRDSXML(w, "DescribeDBCapacityReservations", struct {
		XMLName      xml.Name                    `xml:"DescribeDBCapacityReservationsResult"`
		Marker       string                      `xml:"Marker,omitempty"`
		Reservations []rdsCapacityReservationXML `xml:"DBCapacityReservations>DBCapacityReservation"`
	}{Marker: marker, Reservations: out})
}

// ponytail: static catalog of Aurora Serverless v2 platform versions; no per-account state.
func (s *Server) handleRDSDescribeServerlessV2PlatformVersions(w http.ResponseWriter, r *http.Request) {
	type capacity struct {
		Min float64 `xml:"MinCapacity"`
		Max float64 `xml:"MaxCapacity"`
	}
	type info struct {
		Version     string   `xml:"ServerlessV2PlatformVersion"`
		Description string   `xml:"ServerlessV2PlatformVersionDescription"`
		Engine      string   `xml:"Engine"`
		Support     capacity `xml:"ServerlessV2FeaturesSupport"`
		Status      string   `xml:"Status"`
		IsDefault   bool     `xml:"IsDefault"`
	}
	wantVersion := strings.TrimSpace(r.Form.Get("ServerlessV2PlatformVersion"))
	wantEngine := strings.TrimSpace(r.Form.Get("Engine"))
	defaultOnly := strings.EqualFold(strings.TrimSpace(r.Form.Get("DefaultOnly")), "true")
	out := []info{}
	for _, v := range []info{
		{Version: "1", Description: "Aurora Serverless v2 platform version 1", Engine: "aurora-mysql", Support: capacity{0, 256}, Status: "enabled", IsDefault: true},
		{Version: "1", Description: "Aurora Serverless v2 platform version 1", Engine: "aurora-postgresql", Support: capacity{0, 256}, Status: "enabled", IsDefault: true},
	} {
		if (wantVersion == "" || wantVersion == v.Version) && (wantEngine == "" || wantEngine == v.Engine) && (!defaultOnly || v.IsDefault) {
			out = append(out, v)
		}
	}
	respondRDSXML(w, "DescribeServerlessV2PlatformVersions", struct {
		XMLName  xml.Name `xml:"DescribeServerlessV2PlatformVersionsResult"`
		Versions []info   `xml:"ServerlessV2PlatformVersions>ServerlessV2PlatformVersionInfo"`
	}{Versions: out})
}
