package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Account: REST JSON, signed account, every operation POST /<op> with a
// body-only input. Requests run through the shared budRun plumbing (model
// validation and conformance on the way in and out).
//
// The caller is the Organizations management account (acctCaller). An
// AccountId parameter is resolved against the Organizations store; the
// caller's own ID is refused, as AWS refuses it, and accounts outside the
// organization are AccessDenied.

const acctCaller = "123456789012"

var acctAPI = &budJSONAPI{
	signing: "account", contentType: "application/json",
	invalidCode: "ValidationException", model: awsmodels.AccountModel, respondError: acctRespondError,
}

// acctRespondError writes a REST JSON error in the Account model's shape:
// lower-case message, and a reason on ValidationException.
func acctRespondError(w http.ResponseWriter, e *budErr) {
	body := map[string]any{"message": e.msg}
	if e.code == "ValidationException" {
		body["reason"] = "fieldValidationFailed"
	}
	for k, v := range e.extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

// acctRegionInvalid is a ValidationException about the Region named.
func acctRegionInvalid(format string, args ...any) *budErr {
	e := acctValidation(format, args...)
	e.extra = map[string]any{"reason": "invalidRegionOptTarget"}
	return e
}

var acctPaths = func() map[string]string {
	out := make(map[string]string, len(acctOperations))
	for _, op := range acctOperations {
		out["/"+string(unicode.ToLower(rune(op.Name[0])))+op.Name[1:]] = op.Name
	}
	return out
}()

func init() {
	acctAPI.handlers = map[string]budHandler{
		"GetAccountInformation":         (*Server).acctGetAccountInformation,
		"PutAccountName":                (*Server).acctPutAccountName,
		"GetGovCloudAccountInformation": (*Server).acctGetGovCloud,
		"GetAlternateContact":           (*Server).acctGetAlternateContact,
		"PutAlternateContact":           (*Server).acctPutAlternateContact,
		"DeleteAlternateContact":        (*Server).acctDeleteAlternateContact,
		"GetContactInformation":         (*Server).acctGetContactInformation,
		"PutContactInformation":         (*Server).acctPutContactInformation,
		"SendPhoneNumberVerification":   (*Server).acctSendPhoneVerification,
		"VerifyPhoneNumber":             (*Server).acctVerifyPhoneNumber,
		"GetPrimaryEmail":               (*Server).acctGetPrimaryEmail,
		"StartPrimaryEmailUpdate":       (*Server).acctStartPrimaryEmailUpdate,
		"AcceptPrimaryEmailUpdate":      (*Server).acctAcceptPrimaryEmailUpdate,
		"GetPrimaryEmailUpdateStatus":   (*Server).acctGetPrimaryEmailUpdateStatus,
		"GetRegionOptStatus":            (*Server).acctGetRegionOptStatus,
		"ListRegions":                   (*Server).acctListRegions,
		"EnableRegion":                  (*Server).acctEnableRegion,
		"DisableRegion":                 (*Server).acctDisableRegion,
	}
}

func (s *Server) handleAccountRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.acctServeREST(w, r, acctAPI, acctPaths, "account.")
}

// acctServeREST claims POSTs to one of paths that are signed for api.signing
// (or unsigned and sent to a host starting with hostPrefix) and carry no
// X-Amz-Target. Tax Settings shares it.
func (s *Server) acctServeREST(w http.ResponseWriter, r *http.Request, api *budJSONAPI, paths map[string]string, hostPrefix string) bool {
	if r.Method != http.MethodPost || r.Header.Get("X-Amz-Target") != "" {
		return false
	}
	op, ok := paths[r.URL.Path]
	if !ok {
		return false
	}
	switch hint := strings.TrimSpace(sigV4ServiceHint(r)); {
	case hint == api.signing:
	case hint == "" && r.URL.RawQuery == "" && strings.HasPrefix(strings.ToLower(r.Host), hostPrefix):
	default:
		return false
	}
	s.budRun(w, r, api, op, true)
	return true
}

func acctValidation(format string, args ...any) *budErr {
	return budErrorf("ValidationException", format, args...)
}

// acctOTP is the one-time code Stackyard "sends" for value (a new primary
// email or a phone number): Stackyard sends no mail or SMS, so clients
// derive it the same way.
func acctOTP(account, value string) string {
	sum := sha256.Sum256([]byte(account + ":" + value))
	return strings.ToUpper(hex.EncodeToString(sum[:3]))
}

// acctTarget resolves the account named by in[key] (empty: the caller)
// through the Organizations store, which is locked and released here,
// before any Account lock is taken.
func (s *Server) acctTarget(in map[string]any, key string) (organizationsAccount, *budErr) {
	id := budStr(in, key)
	if id == "" {
		id = acctCaller
	} else if id == acctCaller {
		return organizationsAccount{}, acctValidation("The management account can't specify its own %s; call the operation without it.", key)
	}
	a, ok := s.organizations.account(id)
	if !ok {
		return organizationsAccount{}, budErrorf("AccessDeniedException", "Account %s is not a member of your organization.", id)
	}
	return a, nil
}

// acctDo runs fn on the target account's record under the Account lock.
func (s *Server) acctDo(in map[string]any, fn func(*acctAccount) (map[string]any, *budErr)) (map[string]any, *budErr) {
	a, err := s.acctTarget(in, "AccountId")
	if err != nil {
		return nil, err
	}
	st := s.account
	st.mu.Lock()
	defer st.mu.Unlock()
	return fn(st.get(a))
}

// ---- account information ----

func (s *Server) acctGetAccountInformation(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		return map[string]any{"AccountId": a.id, "AccountName": a.name, "AccountCreatedDate": budEpoch(a.created), "AccountState": "ACTIVE"}, nil
	})
}

func (s *Server) acctPutAccountName(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		a.name = budStr(in, "AccountName")
		return nil, nil
	})
}

func (s *Server) acctGetGovCloud(in map[string]any) (map[string]any, *budErr) {
	a, err := s.acctTarget(in, "StandardAccountId")
	if err != nil {
		return nil, err
	}
	if a.GovCloudID == "" {
		return nil, budErrorf("ResourceNotFoundException", "Account %s has no linked AWS GovCloud (US) account.", a.ID)
	}
	return map[string]any{"GovCloudAccountId": a.GovCloudID, "AccountState": "ACTIVE"}, nil
}

// ---- alternate contacts ----

func (s *Server) acctGetAlternateContact(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		kind := budStr(in, "AlternateContactType")
		c := a.contacts[kind]
		if c == nil {
			return nil, budErrorf("ResourceNotFoundException", "No contact of the inputted alternate contact type found.")
		}
		return map[string]any{"AlternateContact": budCloneMap(c)}, nil
	})
}

func (s *Server) acctPutAlternateContact(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		c := map[string]any{}
		for _, k := range []string{"Name", "Title", "EmailAddress", "PhoneNumber", "AlternateContactType"} {
			c[k] = budStr(in, k)
		}
		a.contacts[budStr(in, "AlternateContactType")] = c // three types at most
		return nil, nil
	})
}

func (s *Server) acctDeleteAlternateContact(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		kind := budStr(in, "AlternateContactType")
		if a.contacts[kind] == nil {
			return nil, budErrorf("ResourceNotFoundException", "No contact of the inputted alternate contact type found.")
		}
		delete(a.contacts, kind)
		return nil, nil
	})
}

// ---- contact information and phone verification ----

func (s *Server) acctGetContactInformation(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		if a.info == nil {
			return nil, budErrorf("ResourceNotFoundException", "No contact information found for account %s.", a.id)
		}
		return map[string]any{"ContactInformation": budCloneMap(a.info), "VerificationStatus": a.phoneStatus}, nil
	})
}

func (s *Server) acctPutContactInformation(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		info := budCloneMap(budMap(in, "ContactInformation")) // conformed and size-checked by the model
		if a.info == nil || budStr(a.info, "PhoneNumber") != budStr(info, "PhoneNumber") {
			a.phoneStatus = "UNVERIFIED" // a new number needs verifying again
		}
		a.info = info
		return nil, nil
	})
}

func (s *Server) acctSendPhoneVerification(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		if a.info == nil {
			return nil, budErrorf("ResourceNotFoundException", "No contact information found for account %s.", a.id)
		}
		if a.phoneStatus == "VERIFIED" {
			return nil, budErrorf("ConflictException", "The phone number %s is already verified.", budStr(a.info, "PhoneNumber"))
		}
		a.phoneStatus = "PENDING"
		return map[string]any{"Status": a.phoneStatus}, nil
	})
}

func (s *Server) acctVerifyPhoneNumber(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		if a.info == nil {
			return nil, budErrorf("ResourceNotFoundException", "No contact information found for account %s.", a.id)
		}
		if a.phoneStatus != "PENDING" {
			return nil, budErrorf("ConflictException", "No phone number verification is in progress; call SendPhoneNumberVerification first.")
		}
		if !strings.EqualFold(budStr(in, "Otp"), acctOTP(a.id, budStr(a.info, "PhoneNumber"))) {
			return nil, acctValidation("The one-time password is not valid.")
		}
		a.phoneStatus = "VERIFIED"
		return map[string]any{"Status": a.phoneStatus}, nil
	})
}

// ---- primary email ----

func (s *Server) acctGetPrimaryEmail(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		return map[string]any{"PrimaryEmail": a.email}, nil
	})
}

func (s *Server) acctStartPrimaryEmailUpdate(in map[string]any) (map[string]any, *budErr) {
	email := strings.TrimSpace(budStr(in, "PrimaryEmail"))
	if at := strings.Index(email, "@"); at < 1 || at == len(email)-1 {
		return nil, acctValidation("PrimaryEmail must be an email address.")
	}
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		if s.account.emailInUse(email) {
			return nil, budErrorf("ConflictException", "The email address %s is already in use by an AWS account.", email)
		}
		a.emailUpdate = &acctEmailUpdate{email: email, status: "PENDING", updated: nowUTC()}
		return map[string]any{"Status": "PENDING"}, nil
	})
}

func (s *Server) acctAcceptPrimaryEmailUpdate(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		u := a.emailUpdate
		if u == nil || u.email != strings.TrimSpace(budStr(in, "PrimaryEmail")) {
			return nil, budErrorf("ResourceNotFoundException", "No primary email update to %s is in progress.", budStr(in, "PrimaryEmail"))
		}
		if u.status != "PENDING" {
			return nil, budErrorf("ConflictException", "The primary email update is already %s.", u.status)
		}
		if !strings.EqualFold(budStr(in, "Otp"), acctOTP(a.id, u.email)) {
			return nil, acctValidation("The one-time password is not valid.")
		}
		a.email, u.status, u.updated, u.seen = u.email, "ACCEPTED", nowUTC(), false
		return map[string]any{"Status": u.status}, nil
	})
}

func (s *Server) acctGetPrimaryEmailUpdateStatus(in map[string]any) (map[string]any, *budErr) {
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		u := a.emailUpdate
		if u == nil {
			return nil, budErrorf("ResourceNotFoundException", "No primary email update was started for account %s.", a.id)
		}
		status := u.status
		if status == "ACCEPTED" { // reported once, then COMPLETED
			if u.seen {
				u.status, u.updated, status = "COMPLETED", nowUTC(), "COMPLETED"
			}
			u.seen = true
		}
		return map[string]any{"Status": status, "UpdatedAt": budEpoch(u.updated)}, nil
	})
}

// ---- Regions ----

func (s *Server) acctGetRegionOptStatus(in map[string]any) (map[string]any, *budErr) {
	name := budStr(in, "RegionName")
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		status, ok := a.observeRegion(name)
		if !ok {
			return nil, acctRegionInvalid("%s is not a valid Region name.", name)
		}
		return map[string]any{"RegionName": name, "RegionOptStatus": status}, nil
	})
}

func (s *Server) acctListRegions(in map[string]any) (map[string]any, *budErr) {
	want := budStrings(in, "RegionOptStatusContains")
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		var regions []any
		for _, name := range acctRegionNames {
			status, _ := a.observeRegion(name)
			if len(want) == 0 || budContains(want, status) {
				regions = append(regions, map[string]any{"RegionName": name, "RegionOptStatus": status})
			}
		}
		start, end, next, err := budPage(budStr(in, "NextToken"), len(regions), budInt(in, "MaxResults", 20), "ValidationException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"Regions": append([]any{}, regions[start:end]...), "NextToken": next}, nil
	})
}

func (s *Server) acctEnableRegion(in map[string]any) (map[string]any, *budErr) {
	return s.acctSetRegion(in, true)
}

func (s *Server) acctDisableRegion(in map[string]any) (map[string]any, *budErr) {
	return s.acctSetRegion(in, false)
}

// acctSetRegion starts an opt-in (ENABLING) or opt-out (DISABLING). Asking
// for the state a Region is already in or heading to is a no-op; reversing a
// change still in progress is a conflict.
func (s *Server) acctSetRegion(in map[string]any, enable bool) (map[string]any, *budErr) {
	name := budStr(in, "RegionName")
	return s.acctDo(in, func(a *acctAccount) (map[string]any, *budErr) {
		if acctDefaultRegions[name] {
			return nil, acctRegionInvalid("%s is enabled by default and can't be enabled or disabled.", name)
		}
		r := a.regions[name]
		if r == nil {
			return nil, acctRegionInvalid("%s is not a valid opt-in Region name.", name)
		}
		target, moving, opposite := "ENABLED", "ENABLING", "DISABLING"
		if !enable {
			target, moving, opposite = "DISABLED", "DISABLING", "ENABLING"
		}
		switch r.status {
		case target, moving:
			return nil, nil
		case opposite:
			return nil, budErrorf("ConflictException", "%s is %s; wait for it to finish before changing it.", name, r.status)
		}
		r.status, r.seen = moving, false
		return nil, nil
	})
}

func budContains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
