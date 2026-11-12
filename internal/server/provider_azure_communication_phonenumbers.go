package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationPhoneNumbersAPIVersion = "2025-06-01"
const azureCommunicationPhoneNumbersPrefix = "/azure/communication-phonenumbers/2025-06-01"

type azureCommunicationPhoneNumbersReservation struct {
	ID                string
	SearchID          string
	DisplayName       string
	Description       string
	CountryCode       string
	PhoneNumberType   string
	AssignmentType    string
	Capabilities      map[string]any
	PhoneNumbers      []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ExpiresAt         time.Time
	DeletedAt         *time.Time
	PurchaseOperation string
}

type azureCommunicationPhoneNumbersSearchResult struct {
	ID              string
	OperationID     string
	CountryCode     string
	PhoneNumberType string
	AssignmentType  string
	Capabilities    map[string]any
	AreaCode        string
	Quantity        int
	PhoneNumbers    []*azureCommunicationPhoneNumbersRecord
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type azureCommunicationPhoneNumbersOperation struct {
	ID               string
	Kind             string
	Status           string
	ResourceLocation string
	SearchID         string
	ReservationID    string
	PhoneNumber      string
	PhoneNumbers     []string
	Capabilities     map[string]any
	Error            map[string]any
	PollCount        int
	CancelRequested  bool
	Completed        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureCommunicationPhoneNumbersRecord struct {
	PhoneNumber     string
	CountryCode     string
	CountryName     string
	PhoneNumberType string
	AssignmentType  string
	Cost            map[string]any
	Capabilities    map[string]any
	AreaCode        string
	Locality        string
	Administrative  string
	SearchID        string
	PurchasedAt     *time.Time
	ReservationID   string
}

const azureCommunicationPhoneNumbersMaxPrealloc = 100

func (s *Server) handleAzureCommunicationPhoneNumbersRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationPhoneNumbersPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationPhoneNumbersAPIVersion {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2025-06-01")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationPhoneNumbersPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "availablephonenumbers":
		s.handleAzureCommunicationPhoneNumbersAvailable(w, r, segments[1:])
		return true
	case "phonenumbers":
		s.handleAzureCommunicationPhoneNumbersNumbers(w, r, segments[1:])
		return true
	case "operatorinformation":
		if len(segments) == 2 && strings.EqualFold(segments[1], ":search") {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			s.handleAzureCommunicationPhoneNumbersOperatorInformationSearch(w, r)
			return true
		}
	}

	azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
	return true
}

func (s *Server) handleAzureCommunicationPhoneNumbersAvailable(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}

	if len(tail) == 1 && strings.EqualFold(tail[0], ":purchase") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationPhoneNumbersPurchase(w, r)
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "countries":
		s.handleAzureCommunicationPhoneNumbersCountries(w, r, tail[1:])
	case "searchresults":
		if len(tail) != 2 || r.Method != http.MethodGet {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		s.handleAzureCommunicationPhoneNumbersGetSearchResult(w, strings.TrimSpace(tail[1]))
	case "reservations":
		s.handleAzureCommunicationPhoneNumbersReservations(w, r, tail[1:])
	default:
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationPhoneNumbersCountries(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationPhoneNumbersListCountries(w, r)
		return
	}

	countryCode := strings.ToUpper(strings.TrimSpace(tail[0]))
	if _, ok := azureCommunicationPhoneNumbersCatalogCountry(countryCode); !ok {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "CountryNotFound", "country not found")
		return
	}

	if len(tail) == 2 {
		switch {
		case strings.EqualFold(tail[1], ":browse"):
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureCommunicationPhoneNumbersBrowseAvailableNumbers(w, r, countryCode)
			return
		case strings.EqualFold(tail[1], ":search"):
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureCommunicationPhoneNumbersSearchAvailableNumbers(w, r, countryCode)
			return
		case strings.EqualFold(tail[1], "localities"):
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureCommunicationPhoneNumbersListLocalities(w, r, countryCode)
			return
		case strings.EqualFold(tail[1], "offerings"):
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureCommunicationPhoneNumbersListOfferings(w, r, countryCode)
			return
		case strings.EqualFold(tail[1], "areacodes"):
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureCommunicationPhoneNumbersListAreaCodes(w, r, countryCode)
			return
		}
	}

	azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationPhoneNumbersReservations(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationPhoneNumbersListReservations(w, r)
		return
	}

	reservationID := strings.TrimSpace(tail[0])
	if reservationID == "" {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "reservationId is required")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodPatch:
			s.handleAzureCommunicationPhoneNumbersCreateOrUpdateReservation(w, r, reservationID)
		case http.MethodGet:
			s.handleAzureCommunicationPhoneNumbersGetReservation(w, reservationID)
		case http.MethodDelete:
			s.handleAzureCommunicationPhoneNumbersDeleteReservation(w, reservationID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 2 && strings.EqualFold(tail[1], ":purchase") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationPhoneNumbersPurchaseReservation(w, reservationID)
		return
	}

	azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationPhoneNumbersNumbers(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationPhoneNumbersListPhoneNumbers(w, r)
		return
	}

	if strings.EqualFold(tail[0], "operations") {
		if len(tail) != 2 {
			azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		operationID := strings.TrimSpace(tail[1])
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationPhoneNumbersGetOperation(w, operationID)
		case http.MethodDelete:
			s.handleAzureCommunicationPhoneNumbersCancelOperation(w, operationID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	number, ok := azureCommunicationPhoneNumbersDecodePathSegment(tail[0])
	if !ok {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "phone number path segment is invalid")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationPhoneNumbersGetByNumber(w, number)
		case http.MethodDelete:
			s.handleAzureCommunicationPhoneNumbersReleasePhoneNumber(w, number)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 2 && strings.EqualFold(tail[1], "capabilities") {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationPhoneNumbersUpdateCapabilities(w, r, number)
		return
	}

	azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationPhoneNumbersListCountries(w http.ResponseWriter, r *http.Request) {
	countries := azureCommunicationPhoneNumbersCatalogCountries()
	values := make([]any, 0, len(countries))
	for _, country := range countries {
		values = append(values, map[string]any{
			"countryCode":   country["countryCode"],
			"localizedName": country["localizedName"],
			"supportedPhoneNumberTypes": []any{
				"geographic",
				"tollFree",
			},
		})
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPaged(r, values))
}

func (s *Server) handleAzureCommunicationPhoneNumbersListAreaCodes(w http.ResponseWriter, r *http.Request, countryCode string) {
	phoneNumberType := strings.TrimSpace(r.URL.Query().Get("phoneNumberType"))
	if phoneNumberType == "" {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "phoneNumberType is required")
		return
	}
	country, _ := azureCommunicationPhoneNumbersCatalogCountry(countryCode)
	values := make([]any, 0, len(country.AreaCodes))
	for _, areaCode := range country.AreaCodes {
		values = append(values, map[string]any{
			"areaCode":               areaCode,
			"countryCode":            country.Code,
			"phoneNumberType":        phoneNumberType,
			"assignmentType":         azureCommunicationPhoneNumbersDefaultAssignmentType(strings.TrimSpace(r.URL.Query().Get("assignmentType"))),
			"localizedName":          country.Locality,
			"administrativeDivision": country.AdministrativeDivision,
		})
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPaged(r, values))
}

func (s *Server) handleAzureCommunicationPhoneNumbersListLocalities(w http.ResponseWriter, r *http.Request, countryCode string) {
	country, _ := azureCommunicationPhoneNumbersCatalogCountry(countryCode)
	values := []any{
		map[string]any{
			"localizedName":          country.Locality,
			"countryCode":            country.Code,
			"administrativeDivision": country.AdministrativeDivision,
		},
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPaged(r, values))
}

func (s *Server) handleAzureCommunicationPhoneNumbersListOfferings(w http.ResponseWriter, r *http.Request, countryCode string) {
	country, _ := azureCommunicationPhoneNumbersCatalogCountry(countryCode)
	values := make([]any, 0, len(country.Offerings))
	for _, offering := range country.Offerings {
		values = append(values, map[string]any{
			"phoneNumberType": offering.PhoneNumberType,
			"assignmentType":  offering.AssignmentType,
			"cost":            azureCommunicationPhoneNumbersCloneObject(offering.Cost),
			"capabilities":    azureCommunicationPhoneNumbersCloneObject(offering.Capabilities),
		})
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPaged(r, values))
}

func (s *Server) handleAzureCommunicationPhoneNumbersBrowseAvailableNumbers(w http.ResponseWriter, r *http.Request, countryCode string) {
	payload, _, ok := azureCommunicationPhoneNumbersReadObject(w, r, false)
	if !ok {
		return
	}
	params, ok := azureCommunicationPhoneNumbersBrowseParams(payload)
	if !ok {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "request body must include a valid phoneNumberType")
		return
	}
	records := s.azureCommunicationPhoneNumbersAvailableRecords(countryCode, params, false)
	values := make([]any, 0, len(records))
	for _, record := range records {
		values = append(values, azureCommunicationPhoneNumbersAvailableRecordFixture(record))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": values,
	})
}

func (s *Server) handleAzureCommunicationPhoneNumbersSearchAvailableNumbers(w http.ResponseWriter, r *http.Request, countryCode string) {
	payload, body, ok := azureCommunicationPhoneNumbersReadObject(w, r, false)
	if !ok {
		return
	}
	params, ok := azureCommunicationPhoneNumbersSearchParams(payload)
	if !ok {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "request body must include assignmentType, capabilities, and phoneNumberType")
		return
	}

	records := s.azureCommunicationPhoneNumbersAvailableRecords(countryCode, params, true)
	if len(records) == 0 {
		azureCommunicationPhoneNumbersError(w, http.StatusConflict, "NoAvailablePhoneNumbers", "no available phone numbers match the search criteria")
		return
	}

	now := time.Now().UTC()
	requestDigest := azureCommunicationPhoneNumbersDigest(countryCode + "|" + string(body))
	searchID := "search-" + requestDigest[:24]
	operationID := strings.TrimSpace(r.Header.Get("Operation-Id"))
	if operationID == "" {
		operationID = "operation-" + azureCommunicationPhoneNumbersDigest("operation|" + requestDigest)[:24]
	}

	s.providerStorageMu.Lock()
	result := s.azureCommunicationPhoneNumbersSearchResults[searchID]
	if result == nil {
		cloned := make([]*azureCommunicationPhoneNumbersRecord, 0, len(records))
		for _, record := range records {
			item := *record
			item.SearchID = searchID
			cloned = append(cloned, &item)
		}
		result = &azureCommunicationPhoneNumbersSearchResult{
			ID:              searchID,
			OperationID:     operationID,
			CountryCode:     countryCode,
			PhoneNumberType: params.PhoneNumberType,
			AssignmentType:  params.AssignmentType,
			Capabilities:    azureCommunicationPhoneNumbersCloneObject(params.Capabilities),
			AreaCode:        params.AreaCode,
			Quantity:        len(cloned),
			PhoneNumbers:    cloned,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		s.azureCommunicationPhoneNumbersSearchResults[searchID] = result
	}
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		op = &azureCommunicationPhoneNumbersOperation{
			ID:               operationID,
			Kind:             "search",
			Status:           "running",
			ResourceLocation: azureCommunicationPhoneNumbersSearchResultPath(searchID),
			SearchID:         searchID,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.azureCommunicationPhoneNumbersOperations[operationID] = op
	}
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureCommunicationPhoneNumbersOperationPath(operationID))
	w.Header().Set("Location", azureCommunicationPhoneNumbersSearchResultPath(searchID))
	w.Header().Set("operation-id", operationID)
	w.Header().Set("search-id", searchID)
	w.Header().Set("Retry-After", "1")
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"searchId":    searchID,
		"status":      "running",
	})
}

func (s *Server) handleAzureCommunicationPhoneNumbersGetSearchResult(w http.ResponseWriter, searchID string) {
	if searchID == "" {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "searchId is required")
		return
	}
	s.providerStorageMu.Lock()
	result := s.azureCommunicationPhoneNumbersSearchResults[searchID]
	s.providerStorageMu.Unlock()
	if result == nil {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "SearchResultNotFound", "search result not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersSearchResultFixture(result))
}

func (s *Server) handleAzureCommunicationPhoneNumbersCreateOrUpdateReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	payload, _, ok := azureCommunicationPhoneNumbersReadObject(w, r, false)
	if !ok {
		return
	}

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureCommunicationPhoneNumbersCleanupReservationsLocked(now)

	reservation := s.azureCommunicationPhoneNumbersReservations[reservationID]
	created := false
	if reservation == nil {
		reservation = &azureCommunicationPhoneNumbersReservation{
			ID:           reservationID,
			CreatedAt:    now,
			UpdatedAt:    now,
			ExpiresAt:    now.Add(15 * time.Minute),
			Capabilities: map[string]any{},
		}
		s.azureCommunicationPhoneNumbersReservations[reservationID] = reservation
		created = true
	}

	if searchID := strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["searchId"])); searchID != "" {
		result := s.azureCommunicationPhoneNumbersSearchResults[searchID]
		if result == nil {
			azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "SearchResultNotFound", "search result not found")
			return
		}
		reservation.SearchID = searchID
		reservation.CountryCode = result.CountryCode
		reservation.PhoneNumberType = result.PhoneNumberType
		reservation.AssignmentType = result.AssignmentType
		reservation.Capabilities = azureCommunicationPhoneNumbersCloneObject(result.Capabilities)
		if payload["phoneNumbers"] == nil {
			numbers := make([]string, 0, len(result.PhoneNumbers))
			for _, record := range result.PhoneNumbers {
				numbers = append(numbers, record.PhoneNumber)
			}
			reservation.PhoneNumbers = numbers
		}
	}

	if displayName, ok := payload["displayName"]; ok {
		if displayName == nil {
			reservation.DisplayName = ""
		} else {
			reservation.DisplayName = strings.TrimSpace(azureCommunicationPhoneNumbersString(displayName))
		}
	}
	if description, ok := payload["description"]; ok {
		if description == nil {
			reservation.Description = ""
		} else {
			reservation.Description = strings.TrimSpace(azureCommunicationPhoneNumbersString(description))
		}
	}
	if countryCode, ok := payload["countryCode"]; ok && countryCode != nil {
		reservation.CountryCode = strings.ToUpper(strings.TrimSpace(azureCommunicationPhoneNumbersString(countryCode)))
	}
	if phoneNumberType, ok := payload["phoneNumberType"]; ok && phoneNumberType != nil {
		reservation.PhoneNumberType = strings.TrimSpace(azureCommunicationPhoneNumbersString(phoneNumberType))
	}
	if assignmentType, ok := payload["assignmentType"]; ok && assignmentType != nil {
		reservation.AssignmentType = strings.TrimSpace(azureCommunicationPhoneNumbersString(assignmentType))
	}
	if capabilities, ok := payload["capabilities"]; ok && capabilities != nil {
		caps, valid := azureCommunicationPhoneNumbersCapabilities(capabilities, true)
		if !valid {
			azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "capabilities must be an object with valid calling and sms values")
			return
		}
		reservation.Capabilities = caps
	}
	if phoneNumbers, ok := payload["phoneNumbers"]; ok {
		if phoneNumbers == nil {
			reservation.PhoneNumbers = nil
		} else {
			numbers, valid := azureCommunicationPhoneNumbersStringList(phoneNumbers)
			if !valid {
				azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "phoneNumbers must be an array of phone number strings")
				return
			}
			for _, number := range numbers {
				if !azureCommunicationPhoneNumbersValidE164(number) {
					azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "phoneNumbers must contain valid E.164 numbers")
					return
				}
			}
			reservation.PhoneNumbers = numbers
		}
	}

	if reservation.CountryCode == "" {
		if len(reservation.PhoneNumbers) > 0 {
			reservation.CountryCode = azureCommunicationPhoneNumbersCountryCodeFromNumber(reservation.PhoneNumbers[0])
		}
		if reservation.CountryCode == "" {
			azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "countryCode or searchId is required")
			return
		}
	}
	if _, ok := azureCommunicationPhoneNumbersCatalogCountry(strings.ToUpper(reservation.CountryCode)); !ok {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "CountryNotFound", "country not found")
		return
	}
	if reservation.PhoneNumberType == "" {
		reservation.PhoneNumberType = "geographic"
	}
	if reservation.AssignmentType == "" {
		reservation.AssignmentType = "person"
	}
	if len(reservation.Capabilities) == 0 {
		reservation.Capabilities = azureCommunicationPhoneNumbersDefaultCapabilities()
	}

	reservation.UpdatedAt = now
	maxExpiry := reservation.CreatedAt.Add(2 * time.Hour)
	extended := now.Add(15 * time.Minute)
	if extended.After(maxExpiry) {
		extended = maxExpiry
	}
	reservation.ExpiresAt = extended

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureCommunicationPhoneNumbersReservationFixture(reservation))
}

func (s *Server) handleAzureCommunicationPhoneNumbersGetReservation(w http.ResponseWriter, reservationID string) {
	s.providerStorageMu.Lock()
	s.azureCommunicationPhoneNumbersCleanupReservationsLocked(time.Now().UTC())
	reservation := s.azureCommunicationPhoneNumbersReservations[reservationID]
	s.providerStorageMu.Unlock()
	if reservation == nil {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "ReservationNotFound", "reservation not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersReservationFixture(reservation))
}

func (s *Server) handleAzureCommunicationPhoneNumbersListReservations(w http.ResponseWriter, r *http.Request) {
	s.providerStorageMu.Lock()
	s.azureCommunicationPhoneNumbersCleanupReservationsLocked(time.Now().UTC())
	values := make([]any, 0, len(s.azureCommunicationPhoneNumbersReservations))
	for _, reservation := range s.azureCommunicationPhoneNumbersReservations {
		values = append(values, azureCommunicationPhoneNumbersReservationFixture(reservation))
	}
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPaged(r, values))
}

func (s *Server) handleAzureCommunicationPhoneNumbersDeleteReservation(w http.ResponseWriter, reservationID string) {
	s.providerStorageMu.Lock()
	reservation := s.azureCommunicationPhoneNumbersReservations[reservationID]
	if reservation == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "ReservationNotFound", "reservation not found")
		return
	}
	now := time.Now().UTC()
	reservation.DeletedAt = &now
	delete(s.azureCommunicationPhoneNumbersReservations, reservationID)
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationPhoneNumbersPurchaseReservation(w http.ResponseWriter, reservationID string) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	reservation := s.azureCommunicationPhoneNumbersReservations[reservationID]
	if reservation == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "ReservationNotFound", "reservation not found")
		return
	}
	if len(reservation.PhoneNumbers) == 0 {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "reservation has no phone numbers to purchase")
		return
	}
	operationID := "operation-" + azureCommunicationPhoneNumbersDigest("purchase-reservation|" + reservationID)[:24]
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		op = &azureCommunicationPhoneNumbersOperation{
			ID:               operationID,
			Kind:             "purchaseReservation",
			Status:           "running",
			ResourceLocation: azureCommunicationPhoneNumbersReservationPath(reservationID),
			ReservationID:    reservationID,
			PhoneNumbers:     slices.Clone(reservation.PhoneNumbers),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.azureCommunicationPhoneNumbersOperations[operationID] = op
	}
	reservation.PurchaseOperation = operationID
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureCommunicationPhoneNumbersOperationPath(operationID))
	w.Header().Set("Location", azureCommunicationPhoneNumbersReservationPath(reservationID))
	w.Header().Set("operation-id", operationID)
	w.Header().Set("Retry-After", "1")
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"status":      "running",
	})
}

func (s *Server) handleAzureCommunicationPhoneNumbersPurchase(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationPhoneNumbersReadObject(w, r, false)
	if !ok {
		return
	}
	numbers, searchID, errPayload := s.azureCommunicationPhoneNumbersPurchaseNumbersFromPayload(payload)
	if errPayload != nil {
		respondJSON(w, int(errPayload["status"].(int)), errPayload["body"])
		return
	}

	now := time.Now().UTC()
	operationID := strings.TrimSpace(r.Header.Get("Operation-Id"))
	if operationID == "" {
		operationID = "operation-" + azureCommunicationPhoneNumbersDigest("purchase|" + string(body))[:24]
	}

	s.providerStorageMu.Lock()
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		op = &azureCommunicationPhoneNumbersOperation{
			ID:               operationID,
			Kind:             "purchase",
			Status:           "running",
			ResourceLocation: azureCommunicationPhoneNumbersNumbersPath(),
			PhoneNumbers:     slices.Clone(numbers),
			SearchID:         searchID,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.azureCommunicationPhoneNumbersOperations[operationID] = op
	}
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureCommunicationPhoneNumbersOperationPath(operationID))
	w.Header().Set("Location", azureCommunicationPhoneNumbersNumbersPath())
	w.Header().Set("operation-id", operationID)
	w.Header().Set("Retry-After", "1")
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"status":      "running",
	})
}

func (s *Server) handleAzureCommunicationPhoneNumbersListPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	s.providerStorageMu.Lock()
	values := make([]any, 0, len(s.azureCommunicationPhoneNumbersNumbers))
	for _, number := range s.azureCommunicationPhoneNumbersNumbers {
		values = append(values, azureCommunicationPhoneNumbersPurchasedRecordFixture(number))
	}
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPaged(r, values))
}

func (s *Server) handleAzureCommunicationPhoneNumbersGetByNumber(w http.ResponseWriter, phoneNumber string) {
	normalized := azureCommunicationPhoneNumbersNormalizePhoneNumber(phoneNumber)
	s.providerStorageMu.Lock()
	record := s.azureCommunicationPhoneNumbersNumbers[normalized]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PhoneNumberNotFound", "phone number not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationPhoneNumbersPurchasedRecordFixture(record))
}

func (s *Server) handleAzureCommunicationPhoneNumbersReleasePhoneNumber(w http.ResponseWriter, phoneNumber string) {
	normalized := azureCommunicationPhoneNumbersNormalizePhoneNumber(phoneNumber)
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	record := s.azureCommunicationPhoneNumbersNumbers[normalized]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PhoneNumberNotFound", "phone number not found")
		return
	}
	operationID := "operation-" + azureCommunicationPhoneNumbersDigest("release|" + normalized)[:24]
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		op = &azureCommunicationPhoneNumbersOperation{
			ID:               operationID,
			Kind:             "release",
			Status:           "running",
			ResourceLocation: azureCommunicationPhoneNumbersNumberPath(normalized),
			PhoneNumber:      normalized,
			PhoneNumbers:     []string{normalized},
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.azureCommunicationPhoneNumbersOperations[operationID] = op
	}
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureCommunicationPhoneNumbersOperationPath(operationID))
	w.Header().Set("Location", azureCommunicationPhoneNumbersNumberPath(normalized))
	w.Header().Set("operation-id", operationID)
	w.Header().Set("Retry-After", "1")
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"status":      "running",
	})
}

func (s *Server) handleAzureCommunicationPhoneNumbersUpdateCapabilities(w http.ResponseWriter, r *http.Request, phoneNumber string) {
	payload, _, ok := azureCommunicationPhoneNumbersReadObject(w, r, false)
	if !ok {
		return
	}
	capabilitiesValue := payload["capabilities"]
	if capabilitiesValue == nil {
		capabilitiesValue = payload
	}
	caps, valid := azureCommunicationPhoneNumbersCapabilities(capabilitiesValue, true)
	if !valid || len(caps) == 0 {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "request body must include valid capabilities")
		return
	}
	normalized := azureCommunicationPhoneNumbersNormalizePhoneNumber(phoneNumber)

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	record := s.azureCommunicationPhoneNumbersNumbers[normalized]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "PhoneNumberNotFound", "phone number not found")
		return
	}
	operationID := "operation-" + azureCommunicationPhoneNumbersDigest("capabilities|" + normalized)[:24]
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		op = &azureCommunicationPhoneNumbersOperation{
			ID:               operationID,
			Kind:             "updateCapabilities",
			Status:           "running",
			ResourceLocation: azureCommunicationPhoneNumbersNumberPath(normalized),
			PhoneNumber:      normalized,
			Capabilities:     caps,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.azureCommunicationPhoneNumbersOperations[operationID] = op
	}
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureCommunicationPhoneNumbersOperationPath(operationID))
	w.Header().Set("Location", azureCommunicationPhoneNumbersNumberPath(normalized))
	w.Header().Set("operation-id", operationID)
	w.Header().Set("Retry-After", "1")
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"status":      "running",
	})
}

func (s *Server) handleAzureCommunicationPhoneNumbersGetOperation(w http.ResponseWriter, operationID string) {
	if operationID == "" {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "operationId is required")
		return
	}

	s.providerStorageMu.Lock()
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "OperationNotFound", "operation not found")
		return
	}
	if strings.EqualFold(op.Status, "running") {
		op.PollCount++
		if op.CancelRequested {
			op.Status = "canceled"
			op.Completed = true
		} else if op.PollCount >= 2 {
			s.azureCommunicationPhoneNumbersCompleteOperationLocked(op)
		}
		op.UpdatedAt = time.Now().UTC()
	}
	fixture := azureCommunicationPhoneNumbersOperationFixture(op)
	running := strings.EqualFold(op.Status, "running")
	s.providerStorageMu.Unlock()

	if running {
		w.Header().Set("Retry-After", "1")
	}
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationPhoneNumbersCancelOperation(w http.ResponseWriter, operationID string) {
	if operationID == "" {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "operationId is required")
		return
	}
	s.providerStorageMu.Lock()
	op := s.azureCommunicationPhoneNumbersOperations[operationID]
	if op == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationPhoneNumbersError(w, http.StatusNotFound, "OperationNotFound", "operation not found")
		return
	}
	if strings.EqualFold(op.Status, "running") {
		op.CancelRequested = true
		op.Status = "canceled"
		op.Completed = true
		op.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationPhoneNumbersOperatorInformationSearch(w http.ResponseWriter, r *http.Request) {
	payload, _, ok := azureCommunicationPhoneNumbersReadObject(w, r, false)
	if !ok {
		return
	}
	numbers, ok := azureCommunicationPhoneNumbersStringList(payload["phoneNumbers"])
	if !ok || len(numbers) == 0 {
		if single := strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["phoneNumber"])); single != "" {
			numbers = []string{single}
		}
	}
	if len(numbers) == 0 {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "phoneNumbers must contain at least one phone number")
		return
	}

	values := make([]any, 0, len(numbers))
	for _, number := range numbers {
		normalized := azureCommunicationPhoneNumbersNormalizePhoneNumber(number)
		if !azureCommunicationPhoneNumbersValidE164(normalized) {
			azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "phoneNumbers must contain valid E.164 numbers")
			return
		}
		countryCode := azureCommunicationPhoneNumbersCountryCodeFromNumber(normalized)
		country, ok := azureCommunicationPhoneNumbersCatalogCountry(countryCode)
		if !ok {
			country = azureCommunicationPhoneNumbersCountryCatalog{
				Code:                   "ZZ",
				Name:                   "Unknown",
				Locality:               "Unknown",
				AdministrativeDivision: "Unknown",
			}
		}
		values = append(values, map[string]any{
			"phoneNumber": normalized,
			"countryCode": country.Code,
			"operatorDetails": map[string]any{
				"name":              country.Name + " Carrier",
				"localizedName":     country.Name + " Carrier",
				"mobileCountryCode": "001",
				"mobileNetworkCode": "01",
				"operatorType":      "mobile",
			},
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": values,
	})
}

func (s *Server) azureCommunicationPhoneNumbersPurchaseNumbersFromPayload(payload map[string]any) ([]string, string, map[string]any) {
	if searchID := strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["searchId"])); searchID != "" {
		s.providerStorageMu.Lock()
		result := s.azureCommunicationPhoneNumbersSearchResults[searchID]
		s.providerStorageMu.Unlock()
		if result == nil {
			return nil, "", map[string]any{
				"status": http.StatusNotFound,
				"body": map[string]any{"error": map[string]any{
					"code":    "SearchResultNotFound",
					"message": "search result not found",
				}},
			}
		}
		numbers := make([]string, 0, len(result.PhoneNumbers))
		for _, record := range result.PhoneNumbers {
			numbers = append(numbers, record.PhoneNumber)
		}
		return numbers, searchID, nil
	}

	numbers, ok := azureCommunicationPhoneNumbersStringList(payload["phoneNumbers"])
	if !ok || len(numbers) == 0 {
		return nil, "", map[string]any{
			"status": http.StatusBadRequest,
			"body": map[string]any{"error": map[string]any{
				"code":    "InvalidRequest",
				"message": "searchId or phoneNumbers is required",
			}},
		}
	}
	for _, number := range numbers {
		if !azureCommunicationPhoneNumbersValidE164(number) {
			return nil, "", map[string]any{
				"status": http.StatusBadRequest,
				"body": map[string]any{"error": map[string]any{
					"code":    "InvalidRequest",
					"message": "phoneNumbers must contain valid E.164 numbers",
				}},
			}
		}
	}
	return numbers, "", nil
}

func (s *Server) azureCommunicationPhoneNumbersCompleteOperationLocked(op *azureCommunicationPhoneNumbersOperation) {
	if op.Completed {
		return
	}

	switch op.Kind {
	case "search":
		op.Status = "succeeded"
	case "purchaseReservation":
		reservation := s.azureCommunicationPhoneNumbersReservations[op.ReservationID]
		if reservation == nil {
			op.Status = "failed"
			op.Error = map[string]any{"code": "ReservationNotFound", "message": "reservation not found"}
			break
		}
		for _, number := range reservation.PhoneNumbers {
			s.azureCommunicationPhoneNumbersAddPurchasedNumberLocked(number, reservation.CountryCode, reservation.PhoneNumberType, reservation.AssignmentType, reservation.Capabilities, op.SearchID, reservation.ID)
		}
		delete(s.azureCommunicationPhoneNumbersReservations, reservation.ID)
		op.Status = "succeeded"
	case "purchase":
		for _, number := range op.PhoneNumbers {
			countryCode := azureCommunicationPhoneNumbersCountryCodeFromNumber(number)
			s.azureCommunicationPhoneNumbersAddPurchasedNumberLocked(number, countryCode, "geographic", "person", azureCommunicationPhoneNumbersDefaultCapabilities(), op.SearchID, "")
		}
		op.Status = "succeeded"
	case "release":
		delete(s.azureCommunicationPhoneNumbersNumbers, azureCommunicationPhoneNumbersNormalizePhoneNumber(op.PhoneNumber))
		op.Status = "succeeded"
	case "updateCapabilities":
		record := s.azureCommunicationPhoneNumbersNumbers[azureCommunicationPhoneNumbersNormalizePhoneNumber(op.PhoneNumber)]
		if record == nil {
			op.Status = "failed"
			op.Error = map[string]any{"code": "PhoneNumberNotFound", "message": "phone number not found"}
			break
		}
		record.Capabilities = azureCommunicationPhoneNumbersCloneObject(op.Capabilities)
		op.Status = "succeeded"
	default:
		op.Status = "succeeded"
	}

	op.Completed = true
}

func (s *Server) azureCommunicationPhoneNumbersAddPurchasedNumberLocked(number, countryCode, phoneNumberType, assignmentType string, capabilities map[string]any, searchID, reservationID string) {
	normalized := azureCommunicationPhoneNumbersNormalizePhoneNumber(number)
	country, ok := azureCommunicationPhoneNumbersCatalogCountry(countryCode)
	if !ok {
		country = azureCommunicationPhoneNumbersCountryCatalog{
			Code:                   countryCode,
			Name:                   "Unknown",
			Locality:               "Unknown",
			AdministrativeDivision: "Unknown",
			AreaCodes:              []string{""},
		}
	}
	now := time.Now().UTC()
	s.azureCommunicationPhoneNumbersNumbers[normalized] = &azureCommunicationPhoneNumbersRecord{
		PhoneNumber:     normalized,
		CountryCode:     country.Code,
		CountryName:     country.Name,
		PhoneNumberType: phoneNumberType,
		AssignmentType:  assignmentType,
		Cost:            map[string]any{"amount": 1.25, "currencyCode": "USD", "billingFrequency": "monthly"},
		Capabilities:    azureCommunicationPhoneNumbersCloneObject(capabilities),
		AreaCode:        azureCommunicationPhoneNumbersAreaCodeFromNumber(normalized, country.Code),
		Locality:        country.Locality,
		Administrative:  country.AdministrativeDivision,
		SearchID:        searchID,
		PurchasedAt:     &now,
		ReservationID:   reservationID,
	}
}

func (s *Server) azureCommunicationPhoneNumbersCleanupReservationsLocked(now time.Time) {
	for id, reservation := range s.azureCommunicationPhoneNumbersReservations {
		if reservation == nil {
			delete(s.azureCommunicationPhoneNumbersReservations, id)
			continue
		}
		if reservation.DeletedAt != nil || now.After(reservation.ExpiresAt) {
			delete(s.azureCommunicationPhoneNumbersReservations, id)
		}
	}
}

func (s *Server) azureCommunicationPhoneNumbersAvailableRecords(countryCode string, params azureCommunicationPhoneNumbersSearchOptions, respectQuantity bool) []*azureCommunicationPhoneNumbersRecord {
	country, _ := azureCommunicationPhoneNumbersCatalogCountry(countryCode)
	quantity := params.Quantity
	if quantity <= 0 {
		quantity = 5
	}
	if !respectQuantity {
		quantity = max(quantity, 5)
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureCommunicationPhoneNumbersCleanupReservationsLocked(time.Now().UTC())

	reserved := map[string]struct{}{}
	for _, reservation := range s.azureCommunicationPhoneNumbersReservations {
		for _, number := range reservation.PhoneNumbers {
			reserved[azureCommunicationPhoneNumbersNormalizePhoneNumber(number)] = struct{}{}
		}
	}

	// quantity comes from the request; cap the preallocation, append grows past it if needed.
	capacity := quantity
	if capacity > azureCommunicationPhoneNumbersMaxPrealloc {
		capacity = azureCommunicationPhoneNumbersMaxPrealloc
	}
	results := make([]*azureCommunicationPhoneNumbersRecord, 0, capacity)
	prefixes := params.PhoneNumberPrefixes
	for _, areaCode := range country.AreaCodes {
		if params.AreaCode != "" && params.AreaCode != areaCode {
			continue
		}
		for idx := 0; idx < 25; idx++ {
			number := azureCommunicationPhoneNumbersGenerateNumber(country, areaCode, idx)
			if len(prefixes) > 0 && !azureCommunicationPhoneNumbersMatchesPrefix(number, prefixes) {
				continue
			}
			if _, exists := s.azureCommunicationPhoneNumbersNumbers[number]; exists {
				continue
			}
			if _, exists := reserved[number]; exists {
				continue
			}
			results = append(results, &azureCommunicationPhoneNumbersRecord{
				PhoneNumber:     number,
				CountryCode:     country.Code,
				CountryName:     country.Name,
				PhoneNumberType: params.PhoneNumberType,
				AssignmentType:  params.AssignmentType,
				Cost:            map[string]any{"amount": 1.25, "currencyCode": "USD", "billingFrequency": "monthly"},
				Capabilities:    azureCommunicationPhoneNumbersCloneObject(params.Capabilities),
				AreaCode:        areaCode,
				Locality:        country.Locality,
				Administrative:  country.AdministrativeDivision,
			})
			if len(results) >= quantity {
				return results
			}
		}
	}
	return results
}

func azureCommunicationPhoneNumbersBrowseParams(payload map[string]any) (azureCommunicationPhoneNumbersSearchOptions, bool) {
	options := azureCommunicationPhoneNumbersSearchOptions{
		PhoneNumberType: strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["phoneNumberType"])),
		AssignmentType:  azureCommunicationPhoneNumbersDefaultAssignmentType(strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["assignmentType"]))),
		Quantity:        5,
	}
	if options.PhoneNumberType == "" {
		return options, false
	}
	if caps, ok := azureCommunicationPhoneNumbersCapabilities(payload["capabilities"], false); ok && len(caps) > 0 {
		options.Capabilities = caps
	} else {
		options.Capabilities = azureCommunicationPhoneNumbersDefaultCapabilities()
	}
	if prefixes, ok := azureCommunicationPhoneNumbersStringList(payload["phoneNumberPrefixes"]); ok {
		options.PhoneNumberPrefixes = prefixes
	}
	return options, true
}

func azureCommunicationPhoneNumbersSearchParams(payload map[string]any) (azureCommunicationPhoneNumbersSearchOptions, bool) {
	options := azureCommunicationPhoneNumbersSearchOptions{
		PhoneNumberType: strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["phoneNumberType"])),
		AssignmentType:  strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["assignmentType"])),
		AreaCode:        strings.TrimSpace(azureCommunicationPhoneNumbersString(payload["areaCode"])),
		Quantity:        1,
	}
	if quantity, ok := azureCommunicationPhoneNumbersInt(payload["quantity"]); ok && quantity > 0 {
		options.Quantity = quantity
	}
	caps, ok := azureCommunicationPhoneNumbersCapabilities(payload["capabilities"], true)
	if options.PhoneNumberType == "" || options.AssignmentType == "" || !ok || len(caps) == 0 {
		return options, false
	}
	options.Capabilities = caps
	return options, true
}

type azureCommunicationPhoneNumbersSearchOptions struct {
	PhoneNumberType     string
	AssignmentType      string
	AreaCode            string
	Capabilities        map[string]any
	PhoneNumberPrefixes []string
	Quantity            int
}

type azureCommunicationPhoneNumbersCountryCatalog struct {
	Code                   string
	Name                   string
	Locality               string
	AdministrativeDivision string
	AreaCodes              []string
	Offerings              []azureCommunicationPhoneNumbersOffering
}

type azureCommunicationPhoneNumbersOffering struct {
	PhoneNumberType string
	AssignmentType  string
	Cost            map[string]any
	Capabilities    map[string]any
}

func azureCommunicationPhoneNumbersCatalogCountries() []map[string]any {
	return []map[string]any{
		{"countryCode": "US", "localizedName": "United States"},
		{"countryCode": "GB", "localizedName": "United Kingdom"},
	}
}

func azureCommunicationPhoneNumbersCatalogCountry(countryCode string) (azureCommunicationPhoneNumbersCountryCatalog, bool) {
	switch strings.ToUpper(strings.TrimSpace(countryCode)) {
	case "US":
		return azureCommunicationPhoneNumbersCountryCatalog{
			Code:                   "US",
			Name:                   "United States",
			Locality:               "Seattle",
			AdministrativeDivision: "WA",
			AreaCodes:              []string{"425", "206"},
			Offerings: []azureCommunicationPhoneNumbersOffering{
				{
					PhoneNumberType: "geographic",
					AssignmentType:  "person",
					Cost:            map[string]any{"amount": 1.25, "currencyCode": "USD", "billingFrequency": "monthly"},
					Capabilities:    azureCommunicationPhoneNumbersDefaultCapabilities(),
				},
				{
					PhoneNumberType: "tollFree",
					AssignmentType:  "application",
					Cost:            map[string]any{"amount": 2.50, "currencyCode": "USD", "billingFrequency": "monthly"},
					Capabilities:    map[string]any{"calling": "inbound", "sms": "outbound"},
				},
			},
		}, true
	case "GB":
		return azureCommunicationPhoneNumbersCountryCatalog{
			Code:                   "GB",
			Name:                   "United Kingdom",
			Locality:               "London",
			AdministrativeDivision: "England",
			AreaCodes:              []string{"20", "161"},
			Offerings: []azureCommunicationPhoneNumbersOffering{
				{
					PhoneNumberType: "geographic",
					AssignmentType:  "person",
					Cost:            map[string]any{"amount": 1.10, "currencyCode": "GBP", "billingFrequency": "monthly"},
					Capabilities:    azureCommunicationPhoneNumbersDefaultCapabilities(),
				},
			},
		}, true
	default:
		return azureCommunicationPhoneNumbersCountryCatalog{}, false
	}
}

func azureCommunicationPhoneNumbersGenerateNumber(country azureCommunicationPhoneNumbersCountryCatalog, areaCode string, index int) string {
	switch country.Code {
	case "US":
		return fmt.Sprintf("+1%s555%04d", areaCode, 1000+index)
	case "GB":
		return fmt.Sprintf("+44%s123%04d", areaCode, 1000+index)
	default:
		return fmt.Sprintf("+999%s%04d", areaCode, 1000+index)
	}
}

func azureCommunicationPhoneNumbersMatchesPrefix(number string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(number, strings.TrimSpace(prefix)) {
			return true
		}
	}
	return false
}

func azureCommunicationPhoneNumbersAvailableRecordFixture(record *azureCommunicationPhoneNumbersRecord) map[string]any {
	return map[string]any{
		"phoneNumber":     record.PhoneNumber,
		"countryCode":     record.CountryCode,
		"countryName":     record.CountryName,
		"phoneNumberType": record.PhoneNumberType,
		"assignmentType":  record.AssignmentType,
		"capabilities":    azureCommunicationPhoneNumbersCloneObject(record.Capabilities),
		"cost":            azureCommunicationPhoneNumbersCloneObject(record.Cost),
	}
}

func azureCommunicationPhoneNumbersSearchResultFixture(result *azureCommunicationPhoneNumbersSearchResult) map[string]any {
	values := make([]any, 0, len(result.PhoneNumbers))
	for _, record := range result.PhoneNumbers {
		values = append(values, azureCommunicationPhoneNumbersAvailableRecordFixture(record))
	}
	return map[string]any{
		"searchId":        result.ID,
		"operationId":     result.OperationID,
		"status":          "succeeded",
		"phoneNumbers":    values,
		"createdDateTime": result.CreatedAt.Format(time.RFC3339Nano),
	}
}

func azureCommunicationPhoneNumbersReservationFixture(reservation *azureCommunicationPhoneNumbersReservation) map[string]any {
	phoneNumbers := make([]any, 0, len(reservation.PhoneNumbers))
	for _, number := range reservation.PhoneNumbers {
		phoneNumbers = append(phoneNumbers, number)
	}
	return map[string]any{
		"reservationId":       reservation.ID,
		"searchId":            reservation.SearchID,
		"displayName":         reservation.DisplayName,
		"description":         reservation.Description,
		"countryCode":         reservation.CountryCode,
		"phoneNumberType":     reservation.PhoneNumberType,
		"assignmentType":      reservation.AssignmentType,
		"capabilities":        azureCommunicationPhoneNumbersCloneObject(reservation.Capabilities),
		"phoneNumbers":        phoneNumbers,
		"purchaseOperationId": reservation.PurchaseOperation,
		"createdDateTime":     reservation.CreatedAt.Format(time.RFC3339Nano),
		"expiresAt":           reservation.ExpiresAt.Format(time.RFC3339Nano),
	}
}

func azureCommunicationPhoneNumbersPurchasedRecordFixture(record *azureCommunicationPhoneNumbersRecord) map[string]any {
	payload := map[string]any{
		"phoneNumber":            record.PhoneNumber,
		"countryCode":            record.CountryCode,
		"countryName":            record.CountryName,
		"phoneNumberType":        record.PhoneNumberType,
		"assignmentType":         record.AssignmentType,
		"capabilities":           azureCommunicationPhoneNumbersCloneObject(record.Capabilities),
		"cost":                   azureCommunicationPhoneNumbersCloneObject(record.Cost),
		"areaCode":               record.AreaCode,
		"locality":               record.Locality,
		"administrativeDivision": record.Administrative,
		"searchId":               record.SearchID,
		"reservationId":          record.ReservationID,
	}
	if record.PurchasedAt != nil {
		payload["purchasedAt"] = record.PurchasedAt.Format(time.RFC3339Nano)
	}
	return payload
}

func azureCommunicationPhoneNumbersOperationFixture(op *azureCommunicationPhoneNumbersOperation) map[string]any {
	payload := map[string]any{
		"id":                  op.ID,
		"status":              op.Status,
		"kind":                op.Kind,
		"resourceLocation":    op.ResourceLocation,
		"createdDateTime":     op.CreatedAt.Format(time.RFC3339Nano),
		"lastUpdatedDateTime": op.UpdatedAt.Format(time.RFC3339Nano),
	}
	if op.SearchID != "" {
		payload["searchId"] = op.SearchID
	}
	if op.ReservationID != "" {
		payload["reservationId"] = op.ReservationID
	}
	if op.PhoneNumber != "" {
		payload["phoneNumber"] = op.PhoneNumber
	}
	if len(op.PhoneNumbers) > 0 {
		items := make([]any, 0, len(op.PhoneNumbers))
		for _, item := range op.PhoneNumbers {
			items = append(items, item)
		}
		payload["phoneNumbers"] = items
	}
	if len(op.Error) > 0 {
		payload["error"] = azureCommunicationPhoneNumbersCloneObject(op.Error)
	}
	return payload
}

func azureCommunicationPhoneNumbersPaged(r *http.Request, values []any) map[string]any {
	skip, _ := azureCommunicationPhoneNumbersQueryInt(r, "skip")
	maxPageSize, ok := azureCommunicationPhoneNumbersQueryInt(r, "maxPageSize")
	if !ok || maxPageSize <= 0 {
		maxPageSize, ok = azureCommunicationPhoneNumbersQueryInt(r, "top")
	}
	if !ok || maxPageSize <= 0 {
		maxPageSize = len(values)
	}
	if skip < 0 {
		skip = 0
	}
	if skip > len(values) {
		skip = len(values)
	}
	end := skip + maxPageSize
	if end > len(values) {
		end = len(values)
	}
	page := values[skip:end]
	payload := map[string]any{"value": page}
	if end < len(values) {
		payload["nextLink"] = fmt.Sprintf("%s?api-version=%s&skip=%d&maxPageSize=%d", rawRequestPath(r), azureCommunicationPhoneNumbersAPIVersion, end, maxPageSize)
	}
	return payload
}

func azureCommunicationPhoneNumbersDefaultCapabilities() map[string]any {
	return map[string]any{
		"calling": "inbound+outbound",
		"sms":     "inbound+outbound",
	}
}

func azureCommunicationPhoneNumbersCapabilities(value any, requireObject bool) (map[string]any, bool) {
	if value == nil {
		if requireObject {
			return nil, false
		}
		return map[string]any{}, true
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	out := map[string]any{}
	for _, key := range []string{"calling", "sms"} {
		if raw, exists := object[key]; exists {
			text := strings.ToLower(strings.TrimSpace(azureCommunicationPhoneNumbersString(raw)))
			switch text {
			case "none", "inbound", "outbound", "inbound+outbound":
				out[key] = text
			default:
				return nil, false
			}
		}
	}
	if requireObject && len(out) == 0 {
		return nil, false
	}
	return out, true
}

func azureCommunicationPhoneNumbersReadObject(w http.ResponseWriter, r *http.Request, allowEmpty bool) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 && allowEmpty {
		return map[string]any{}, body, true
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationPhoneNumbersError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationPhoneNumbersQueryInt(r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func azureCommunicationPhoneNumbersDecodePathSegment(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", false
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", false
	}
	return decoded, true
}

func azureCommunicationPhoneNumbersNormalizePhoneNumber(value string) string {
	return strings.TrimSpace(value)
}

func azureCommunicationPhoneNumbersValidE164(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 20 || !strings.HasPrefix(value, "+") {
		return false
	}
	for _, ch := range value[1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func azureCommunicationPhoneNumbersCountryCodeFromNumber(number string) string {
	number = strings.TrimSpace(number)
	switch {
	case strings.HasPrefix(number, "+1"):
		return "US"
	case strings.HasPrefix(number, "+44"):
		return "GB"
	default:
		return ""
	}
}

func azureCommunicationPhoneNumbersAreaCodeFromNumber(number, countryCode string) string {
	switch countryCode {
	case "US":
		if len(number) >= 5 {
			return number[2:5]
		}
	case "GB":
		trimmed := strings.TrimPrefix(number, "+44")
		if len(trimmed) >= 2 {
			if strings.HasPrefix(trimmed, "20") {
				return "20"
			}
			return trimmed[:3]
		}
	}
	return ""
}

func azureCommunicationPhoneNumbersString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func azureCommunicationPhoneNumbersStringList(value any) ([]string, bool) {
	if value == nil {
		return nil, false
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(azureCommunicationPhoneNumbersString(item))
		if text == "" {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}

func azureCommunicationPhoneNumbersInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
}

func azureCommunicationPhoneNumbersDefaultAssignmentType(value string) string {
	if strings.TrimSpace(value) == "" {
		return "person"
	}
	return value
}

func azureCommunicationPhoneNumbersOperationPath(operationID string) string {
	return fmt.Sprintf("%s/phoneNumbers/operations/%s?api-version=%s", azureCommunicationPhoneNumbersPrefix, operationID, azureCommunicationPhoneNumbersAPIVersion)
}

func azureCommunicationPhoneNumbersSearchResultPath(searchID string) string {
	return fmt.Sprintf("%s/availablePhoneNumbers/searchResults/%s?api-version=%s", azureCommunicationPhoneNumbersPrefix, searchID, azureCommunicationPhoneNumbersAPIVersion)
}

func azureCommunicationPhoneNumbersReservationPath(reservationID string) string {
	return fmt.Sprintf("%s/availablePhoneNumbers/reservations/%s?api-version=%s", azureCommunicationPhoneNumbersPrefix, reservationID, azureCommunicationPhoneNumbersAPIVersion)
}

func azureCommunicationPhoneNumbersNumberPath(number string) string {
	return fmt.Sprintf("%s/phoneNumbers/%s?api-version=%s", azureCommunicationPhoneNumbersPrefix, url.PathEscape(number), azureCommunicationPhoneNumbersAPIVersion)
}

func azureCommunicationPhoneNumbersNumbersPath() string {
	return fmt.Sprintf("%s/phoneNumbers?api-version=%s", azureCommunicationPhoneNumbersPrefix, azureCommunicationPhoneNumbersAPIVersion)
}

func azureCommunicationPhoneNumbersCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCommunicationPhoneNumbersCloneObject(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureCommunicationPhoneNumbersCloneObject(object)
				} else {
					cloned[i] = item
				}
			}
			out[key] = cloned
		default:
			out[key] = value
		}
	}
	return out
}

func azureCommunicationPhoneNumbersDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationPhoneNumbersError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
