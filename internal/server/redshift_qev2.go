package server

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// redshiftQev2MaxApplications bounds the Query Editor v2 IDC applications a
// single emulator instance stores.
const redshiftQev2MaxApplications = 1000

type redshiftQev2IdcApplication struct {
	Name           string
	ARN            string
	IdcInstanceArn string
	DisplayName    string
	Tags           []redshiftTagXML
}

type redshiftQev2IdcApplicationXML struct {
	IdcInstanceArn           string           `xml:"IdcInstanceArn"`
	Qev2IdcApplicationName   string           `xml:"Qev2IdcApplicationName"`
	Qev2IdcApplicationArn    string           `xml:"Qev2IdcApplicationArn"`
	IdcManagedApplicationArn string           `xml:"IdcManagedApplicationArn"`
	IdcOnboardStatus         string           `xml:"IdcOnboardStatus"`
	IdcDisplayName           string           `xml:"IdcDisplayName"`
	Tags                     []redshiftTagXML `xml:"Tags>Tag,omitempty"`
}

func redshiftQev2IdcApplicationArn(name string) string {
	return fmt.Sprintf("arn:aws:redshift:us-east-1:123456789012:qev2idcapplication:%s", name)
}

func (app *redshiftQev2IdcApplication) toXML() redshiftQev2IdcApplicationXML {
	return redshiftQev2IdcApplicationXML{
		IdcInstanceArn:           app.IdcInstanceArn,
		Qev2IdcApplicationName:   app.Name,
		Qev2IdcApplicationArn:    app.ARN,
		IdcManagedApplicationArn: app.IdcInstanceArn + "/application/" + app.Name,
		IdcOnboardStatus:         "Completed",
		IdcDisplayName:           app.DisplayName,
		Tags:                     append([]redshiftTagXML(nil), app.Tags...),
	}
}

// respondRedshiftQev2Result writes <Action>Response/<Action>Result carrying the
// given result members.
func respondRedshiftQev2Result(w http.ResponseWriter, action string, result any) {
	respondRedshiftXML(w, http.StatusOK, struct {
		XMLName          xml.Name                 `xml:""`
		XMLNS            string                   `xml:"xmlns,attr"`
		Result           any                      `xml:""`
		ResponseMetadata redshiftResponseMetadata `xml:"ResponseMetadata"`
	}{
		XMLName:          xml.Name{Local: action + "Response"},
		XMLNS:            redshiftNamespace,
		Result:           result,
		ResponseMetadata: redshiftResponseMetadata{RequestID: "stackyard-request"},
	})
}

type redshiftQev2SingleResult struct {
	XMLName            xml.Name
	Qev2IdcApplication redshiftQev2IdcApplicationXML `xml:"Qev2IdcApplication"`
}

func (s *Server) handleRedshiftCreateQev2IdcApplication(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.Form.Get("Qev2IdcApplicationName"))
	instance := strings.TrimSpace(r.Form.Get("IdcInstanceArn"))
	display := strings.TrimSpace(r.Form.Get("IdcDisplayName"))
	for field, value := range map[string]string{"Qev2IdcApplicationName": name, "IdcInstanceArn": instance, "IdcDisplayName": display} {
		if value == "" {
			respondRedshiftErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", field+" is required")
			return
		}
	}
	tags, err := redshiftParseTags(r.Form)
	if err != nil {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}
	arn := redshiftQev2IdcApplicationArn(name)
	s.redshiftMu.Lock()
	defer s.redshiftMu.Unlock()
	if _, exists := s.redshift.qev2Applications[arn]; exists {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "Qev2IdcApplicationAlreadyExists", "qev2 idc application already exists")
		return
	}
	if len(s.redshift.qev2Applications) >= redshiftQev2MaxApplications {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "UnsupportedOperation", "qev2 idc application limit reached")
		return
	}
	app := &redshiftQev2IdcApplication{Name: name, ARN: arn, IdcInstanceArn: instance, DisplayName: display, Tags: tags}
	s.redshift.qev2Applications[arn] = app
	respondRedshiftQev2Result(w, "CreateQev2IdcApplication", redshiftQev2SingleResult{
		XMLName:            xml.Name{Local: "CreateQev2IdcApplicationResult"},
		Qev2IdcApplication: app.toXML(),
	})
}

func (s *Server) handleRedshiftDescribeQev2IdcApplications(w http.ResponseWriter, r *http.Request) {
	filter := strings.TrimSpace(r.Form.Get("Qev2IdcApplicationArn"))
	s.redshiftMu.Lock()
	defer s.redshiftMu.Unlock()
	items := make([]redshiftQev2IdcApplicationXML, 0, len(s.redshift.qev2Applications))
	if filter != "" {
		app, ok := s.redshift.qev2Applications[filter]
		if !ok {
			respondRedshiftErrorXML(w, http.StatusBadRequest, "Qev2IdcApplicationNotExists", "qev2 idc application not found")
			return
		}
		items = append(items, app.toXML())
	} else {
		for _, app := range s.redshift.qev2Applications {
			items = append(items, app.toXML())
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Qev2IdcApplicationArn < items[j].Qev2IdcApplicationArn })
	}
	respondRedshiftQev2Result(w, "DescribeQev2IdcApplications", struct {
		XMLName             xml.Name                        `xml:"DescribeQev2IdcApplicationsResult"`
		Qev2IdcApplications []redshiftQev2IdcApplicationXML `xml:"Qev2IdcApplications>member"`
	}{Qev2IdcApplications: items})
}

func (s *Server) handleRedshiftModifyQev2IdcApplication(w http.ResponseWriter, r *http.Request) {
	arn := strings.TrimSpace(r.Form.Get("Qev2IdcApplicationArn"))
	if arn == "" {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Qev2IdcApplicationArn is required")
		return
	}
	display := strings.TrimSpace(r.Form.Get("IdcDisplayName"))
	s.redshiftMu.Lock()
	defer s.redshiftMu.Unlock()
	app, ok := s.redshift.qev2Applications[arn]
	if !ok {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "Qev2IdcApplicationNotExists", "qev2 idc application not found")
		return
	}
	if display != "" {
		app.DisplayName = display
	}
	respondRedshiftQev2Result(w, "ModifyQev2IdcApplication", redshiftQev2SingleResult{
		XMLName:            xml.Name{Local: "ModifyQev2IdcApplicationResult"},
		Qev2IdcApplication: app.toXML(),
	})
}

func (s *Server) handleRedshiftDeleteQev2IdcApplication(w http.ResponseWriter, r *http.Request) {
	arn := strings.TrimSpace(r.Form.Get("Qev2IdcApplicationArn"))
	if arn == "" {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Qev2IdcApplicationArn is required")
		return
	}
	s.redshiftMu.Lock()
	defer s.redshiftMu.Unlock()
	if _, ok := s.redshift.qev2Applications[arn]; !ok {
		respondRedshiftErrorXML(w, http.StatusBadRequest, "Qev2IdcApplicationNotExists", "qev2 idc application not found")
		return
	}
	delete(s.redshift.qev2Applications, arn)
	respondRedshiftQev2Result(w, "DeleteQev2IdcApplication", struct {
		XMLName xml.Name `xml:"DeleteQev2IdcApplicationResult"`
	}{})
}

// respondRedshiftNamespaceStatus answers Register/DeregisterNamespace, whose
// output is just the registration Status.
func respondRedshiftNamespaceStatus(w http.ResponseWriter, action, status string) {
	respondRedshiftQev2Result(w, action, struct {
		XMLName xml.Name
		Status  string `xml:"Status"`
	}{XMLName: xml.Name{Local: action + "Result"}, Status: status})
}
