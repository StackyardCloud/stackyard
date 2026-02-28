package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ekssvc "github.com/stackyard/stackyard/internal/services/eks"
)

type eksError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleEKSRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEKSRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "eks")
	if !ok {
		respondEKSError(w, status, code, msg)
		return true
	}

	if s.handleEKSStage0To2(w, r) {
		return true
	}
	if s.handleEKSStage3To4(w, r) {
		return true
	}
	if s.handleEKSStage5(w, r) {
		return true
	}
	if s.handleEKSStage6(w, r) {
		return true
	}

	respondEKSError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func (s *Server) handleEKSStage0To2(w http.ResponseWriter, r *http.Request) bool {
	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 1 && segments[0] == "cluster-versions" && r.Method == http.MethodGet {
		respondEKSJSON(w, http.StatusOK, map[string]any{
			"clusterVersions": []map[string]any{
				{
					"clusterVersion": "1.29",
					"defaultVersion": true,
					"status":         "STANDARD_SUPPORT",
				},
			},
		})
		return true
	}

	if segments[0] != "clusters" {
		return false
	}

	if len(segments) == 1 {
		switch r.Method {
		case http.MethodGet:
			maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
			if err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
				return true
			}
			nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
			if !ok {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
				return true
			}
			clusters := s.eks.ListClusters()
			start, end, outNextToken, err := paginateEKSBounds(len(clusters), nextToken, maxResults)
			if err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
				return true
			}
			out := map[string]any{"clusters": clusters[start:end]}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondEKSJSON(w, http.StatusOK, out)
			return true
		case http.MethodPost:
			var req eksCreateClusterRequest
			if err := decodeEKSBody(r, &req); err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
				return true
			}
			if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.RoleArn) == "" {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "name and roleArn are required")
				return true
			}
			in := ekssvc.CreateClusterInput{
				Name:    strings.TrimSpace(req.Name),
				Version: strings.TrimSpace(req.Version),
				RoleArn: strings.TrimSpace(req.RoleArn),
				Tags:    req.Tags,
			}
			if req.ResourcesVpcConfig != nil {
				in.ResourcesVpcConfig = &ekssvc.ResourcesVpcConfigInput{
					SubnetIDs:            req.ResourcesVpcConfig.SubnetIDs,
					EndpointPublicAccess: req.ResourcesVpcConfig.EndpointPublicAccess,
				}
			}
			cluster, err := s.eks.CreateCluster(in)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"cluster": cluster})
			return true
		default:
			return false
		}
	}

	clusterName, ok := decodeEKSPathSegment(segments[1])
	if !ok {
		respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "cluster name is required")
		return true
	}

	if len(segments) == 2 {
		switch r.Method {
		case http.MethodGet:
			cluster, err := s.eks.DescribeCluster(clusterName)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"cluster": cluster})
			return true
		case http.MethodDelete:
			cluster, err := s.eks.DeleteCluster(clusterName)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"cluster": cluster})
			return true
		default:
			return false
		}
	}

	switch {
	case len(segments) == 3 && segments[2] == "update-config" && r.Method == http.MethodPost:
		var req eksUpdateClusterConfigRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		update, err := s.eks.UpdateClusterConfig(clusterName, ekssvc.UpdateClusterConfigInput{
			ResourcesVpcConfig: req.ResourcesVpcConfig.toServiceInput(),
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 3 && segments[2] == "updates" && r.Method == http.MethodPost:
		var req eksUpdateClusterVersionRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		update, err := s.eks.UpdateClusterVersion(clusterName, ekssvc.UpdateClusterVersionInput{
			Version: strings.TrimSpace(req.Version),
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 3 && segments[2] == "updates" && r.Method == http.MethodGet:
		nodegroupName, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nodegroupName"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nodegroupName")
			return true
		}
		addonName, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("addonName"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid addonName")
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		updateIDs, err := s.eks.ListUpdates(clusterName, nodegroupName, addonName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(updateIDs), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"updateIds": updateIDs[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "updates" && r.Method == http.MethodGet:
		updateID, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "update id is required")
			return true
		}
		nodegroupName, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nodegroupName"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nodegroupName")
			return true
		}
		addonName, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("addonName"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid addonName")
			return true
		}
		update, err := s.eks.DescribeUpdate(clusterName, updateID, nodegroupName, addonName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 3 && segments[2] == "node-groups" && r.Method == http.MethodPost:
		var req eksCreateNodegroupRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		in := ekssvc.CreateNodegroupInput{
			NodegroupName: strings.TrimSpace(req.NodegroupName),
			NodeRole:      strings.TrimSpace(req.NodeRole),
			Subnets:       req.Subnets,
			Version:       strings.TrimSpace(req.Version),
			Tags:          req.Tags,
		}
		nodegroup, err := s.eks.CreateNodegroup(clusterName, in)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"nodegroup": nodegroup})
		return true

	case len(segments) == 3 && segments[2] == "node-groups" && r.Method == http.MethodGet:
		nodegroups, err := s.eks.ListNodegroups(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(nodegroups), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"nodegroups": nodegroups[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "node-groups" && r.Method == http.MethodGet:
		nodegroupName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "nodegroup name is required")
			return true
		}
		nodegroup, err := s.eks.DescribeNodegroup(clusterName, nodegroupName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"nodegroup": nodegroup})
		return true

	case len(segments) == 4 && segments[2] == "node-groups" && r.Method == http.MethodDelete:
		nodegroupName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "nodegroup name is required")
			return true
		}
		nodegroup, err := s.eks.DeleteNodegroup(clusterName, nodegroupName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"nodegroup": nodegroup})
		return true

	case len(segments) == 5 && segments[2] == "node-groups" && segments[4] == "update-config" && r.Method == http.MethodPost:
		nodegroupName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "nodegroup name is required")
			return true
		}
		var req eksUpdateNodegroupConfigRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		update, err := s.eks.UpdateNodegroupConfig(clusterName, nodegroupName, ekssvc.UpdateNodegroupConfigInput{
			Labels: req.Labels,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 5 && segments[2] == "node-groups" && segments[4] == "update-version" && r.Method == http.MethodPost:
		nodegroupName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "nodegroup name is required")
			return true
		}
		var req eksUpdateNodegroupVersionRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		update, err := s.eks.UpdateNodegroupVersion(clusterName, nodegroupName, ekssvc.UpdateNodegroupVersionInput{
			Version: strings.TrimSpace(req.Version),
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 3 && segments[2] == "fargate-profiles" && r.Method == http.MethodPost:
		var req eksCreateFargateProfileRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		in := ekssvc.CreateFargateProfileInput{
			FargateProfileName:  strings.TrimSpace(req.FargateProfileName),
			PodExecutionRoleArn: strings.TrimSpace(req.PodExecutionRoleArn),
			Subnets:             req.Subnets,
			Selectors:           req.toServiceSelectors(),
			Tags:                req.Tags,
		}
		profile, err := s.eks.CreateFargateProfile(clusterName, in)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"fargateProfile": profile})
		return true

	case len(segments) == 3 && segments[2] == "fargate-profiles" && r.Method == http.MethodGet:
		profiles, err := s.eks.ListFargateProfiles(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(profiles), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"fargateProfileNames": profiles[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "fargate-profiles" && r.Method == http.MethodGet:
		profileName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "fargate profile name is required")
			return true
		}
		profile, err := s.eks.DescribeFargateProfile(clusterName, profileName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"fargateProfile": profile})
		return true

	case len(segments) == 4 && segments[2] == "fargate-profiles" && r.Method == http.MethodDelete:
		profileName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "fargate profile name is required")
			return true
		}
		profile, err := s.eks.DeleteFargateProfile(clusterName, profileName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"fargateProfile": profile})
		return true
	}

	return false
}

func (s *Server) handleEKSStage3To4(w http.ResponseWriter, r *http.Request) bool {
	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 2 && segments[0] == "addons" && segments[1] == "supported-versions" && r.Method == http.MethodGet {
		addonName, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("addonName"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid addonName")
			return true
		}
		versions := s.eks.DescribeAddonVersions(addonName)
		type addonVersionView struct {
			AddonVersion string   `json:"addonVersion"`
			Architecture []string `json:"architecture,omitempty"`
		}
		type addonView struct {
			AddonName     string             `json:"addonName"`
			AddonVersions []addonVersionView `json:"addonVersions"`
		}
		out := make([]addonView, 0, len(versions))
		for _, version := range versions {
			out = append(out, addonView{
				AddonName: version.AddonName,
				AddonVersions: []addonVersionView{
					{
						AddonVersion: version.AddonVersion,
						Architecture: version.Architecture,
					},
				},
			})
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"addons": out})
		return true
	}

	if len(segments) == 2 && segments[0] == "addons" && segments[1] == "configuration-schemas" &&
		(r.Method == http.MethodPost || r.Method == http.MethodGet) {
		var req eksDescribeAddonConfigurationRequest
		if r.Method == http.MethodPost {
			if err := decodeEKSBody(r, &req); err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
				return true
			}
		} else {
			req.AddonName = strings.TrimSpace(r.URL.Query().Get("addonName"))
			req.AddonVersion = strings.TrimSpace(r.URL.Query().Get("addonVersion"))
		}
		addonName, addonVersion := s.eks.DescribeAddonConfiguration(req.AddonName, req.AddonVersion)
		respondEKSJSON(w, http.StatusOK, map[string]any{
			"addonName":           addonName,
			"addonVersion":        addonVersion,
			"configurationSchema": "{}",
		})
		return true
	}

	if len(segments) == 1 && segments[0] == "access-policies" && r.Method == http.MethodGet {
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		accessPolicies := s.eks.ListAccessPolicies()
		start, end, outNextToken, err := paginateEKSBounds(len(accessPolicies), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"accessPolicies": accessPolicies[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true
	}

	if segments[0] != "clusters" || len(segments) < 2 {
		return false
	}

	clusterName, ok := decodeEKSPathSegment(segments[1])
	if !ok {
		respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "cluster name is required")
		return true
	}

	switch {
	case len(segments) == 3 && segments[2] == "addons" && r.Method == http.MethodPost:
		var req eksCreateAddonRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		addon, err := s.eks.CreateAddon(clusterName, ekssvc.CreateAddonInput{
			AddonName:             strings.TrimSpace(req.AddonName),
			AddonVersion:          strings.TrimSpace(req.AddonVersion),
			ServiceAccountRoleArn: strings.TrimSpace(req.ServiceAccountRoleArn),
			ConfigurationValues:   req.ConfigurationValues,
			Tags:                  req.Tags,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"addon": addon})
		return true

	case len(segments) == 3 && segments[2] == "addons" && r.Method == http.MethodGet:
		addons, err := s.eks.ListAddons(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(addons), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"addons": addons[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "addons" && r.Method == http.MethodGet:
		addonName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "addon name is required")
			return true
		}
		addon, err := s.eks.DescribeAddon(clusterName, addonName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"addon": addon})
		return true

	case len(segments) == 5 && segments[2] == "addons" && segments[4] == "update" && r.Method == http.MethodPost:
		addonName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "addon name is required")
			return true
		}
		var req eksUpdateAddonRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		update, err := s.eks.UpdateAddon(clusterName, addonName, ekssvc.UpdateAddonInput{
			AddonVersion:          strings.TrimSpace(req.AddonVersion),
			ServiceAccountRoleArn: strings.TrimSpace(req.ServiceAccountRoleArn),
			ConfigurationValues:   req.ConfigurationValues,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 4 && segments[2] == "addons" && r.Method == http.MethodDelete:
		addonName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "addon name is required")
			return true
		}
		addon, err := s.eks.DeleteAddon(clusterName, addonName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"addon": addon})
		return true

	case len(segments) == 3 && segments[2] == "identity-provider-configs" && r.Method == http.MethodGet:
		configs, err := s.eks.ListIdentityProviderConfigs(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(configs), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"identityProviderConfigs": configs[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "identity-provider-configs" && segments[3] == "associate" && r.Method == http.MethodPost:
		var req eksAssociateIdentityProviderConfigRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		var oidc *ekssvc.OIDCIdentityProviderConfig
		if req.Oidc != nil {
			oidc = &ekssvc.OIDCIdentityProviderConfig{
				IdentityProviderConfigName: strings.TrimSpace(req.Oidc.IdentityProviderConfigName),
				IssuerURL:                  strings.TrimSpace(req.Oidc.IssuerURL),
				ClientID:                   strings.TrimSpace(req.Oidc.ClientID),
				UsernameClaim:              strings.TrimSpace(req.Oidc.UsernameClaim),
				UsernamePrefix:             strings.TrimSpace(req.Oidc.UsernamePrefix),
				GroupsClaim:                strings.TrimSpace(req.Oidc.GroupsClaim),
				GroupsPrefix:               strings.TrimSpace(req.Oidc.GroupsPrefix),
				RequiredClaims:             req.Oidc.RequiredClaims,
				Tags:                       req.Oidc.Tags,
			}
		}
		update, err := s.eks.AssociateIdentityProviderConfig(clusterName, ekssvc.AssociateIdentityProviderConfigInput{
			OIDC: oidc,
			Tags: req.Tags,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 4 && segments[2] == "identity-provider-configs" && segments[3] == "disassociate" && r.Method == http.MethodPost:
		var req eksDisassociateIdentityProviderConfigRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		if req.IdentityProviderConfig == nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "identityProviderConfig is required")
			return true
		}
		update, err := s.eks.DisassociateIdentityProviderConfig(
			clusterName,
			strings.TrimSpace(req.IdentityProviderConfig.Type),
			strings.TrimSpace(req.IdentityProviderConfig.Name),
		)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 4 && segments[2] == "identity-provider-configs" && segments[3] == "describe" && r.Method == http.MethodPost:
		var req eksDescribeIdentityProviderConfigRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		if req.IdentityProviderConfig == nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "identityProviderConfig is required")
			return true
		}
		config, err := s.eks.DescribeIdentityProviderConfig(
			clusterName,
			strings.TrimSpace(req.IdentityProviderConfig.Type),
			strings.TrimSpace(req.IdentityProviderConfig.Name),
		)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{
			"identityProviderConfig": map[string]any{
				"oidc": config,
			},
		})
		return true

	case len(segments) == 3 && segments[2] == "access-entries" && r.Method == http.MethodPost:
		var req eksCreateAccessEntryRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		entry, err := s.eks.CreateAccessEntry(clusterName, ekssvc.CreateAccessEntryInput{
			PrincipalArn:     strings.TrimSpace(req.PrincipalArn),
			Type:             strings.TrimSpace(req.Type),
			Username:         strings.TrimSpace(req.Username),
			KubernetesGroups: req.KubernetesGroups,
			Tags:             req.Tags,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"accessEntry": entry})
		return true

	case len(segments) == 3 && segments[2] == "access-entries" && r.Method == http.MethodGet:
		associatedPolicyArn, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("associatedPolicyArn"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid associatedPolicyArn")
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		entries, err := s.eks.ListAccessEntries(clusterName, associatedPolicyArn)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(entries), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"accessEntries": entries[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "access-entries" && r.Method == http.MethodGet:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		entry, err := s.eks.DescribeAccessEntry(clusterName, principalArn)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"accessEntry": entry})
		return true

	case len(segments) == 4 && segments[2] == "access-entries" && r.Method == http.MethodPost:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		var req eksUpdateAccessEntryRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		var username *string
		if req.Username != nil {
			trimmed := strings.TrimSpace(*req.Username)
			username = &trimmed
		}
		entry, err := s.eks.UpdateAccessEntry(clusterName, principalArn, ekssvc.UpdateAccessEntryInput{
			Username:         username,
			KubernetesGroups: req.KubernetesGroups,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"accessEntry": entry})
		return true

	case len(segments) == 4 && segments[2] == "access-entries" && r.Method == http.MethodDelete:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		if err := s.eks.DeleteAccessEntry(clusterName, principalArn); err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{})
		return true

	case len(segments) == 5 && segments[2] == "access-entries" && segments[4] == "access-policies" && r.Method == http.MethodPost:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		var req eksAssociateAccessPolicyRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		policy, err := s.eks.AssociateAccessPolicy(clusterName, principalArn, ekssvc.AssociateAccessPolicyInput{
			PolicyArn: strings.TrimSpace(req.PolicyArn),
			AccessScope: ekssvc.AccessScope{
				Type:       strings.TrimSpace(req.AccessScope.Type),
				Namespaces: req.AccessScope.Namespaces,
			},
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{
			"clusterName":            clusterName,
			"principalArn":           principalArn,
			"associatedAccessPolicy": policy,
		})
		return true

	case len(segments) == 5 && segments[2] == "access-entries" && segments[4] == "access-policies" && r.Method == http.MethodGet:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		policies, err := s.eks.ListAssociatedAccessPolicies(clusterName, principalArn)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(policies), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{
			"clusterName":              clusterName,
			"principalArn":             principalArn,
			"associatedAccessPolicies": policies[start:end],
		}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 5 && segments[2] == "access-entries" && segments[4] == "access-policies" && r.Method == http.MethodDelete:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		policyArn, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("policyArn"))
		if !ok || strings.TrimSpace(policyArn) == "" {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "policyArn is required")
			return true
		}
		if err := s.eks.DisassociateAccessPolicy(clusterName, principalArn, policyArn); err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{})
		return true

	case len(segments) == 6 && segments[2] == "access-entries" && segments[4] == "access-policies" && r.Method == http.MethodDelete:
		principalArn, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "principalArn is required")
			return true
		}
		policyArn, ok := decodeEKSPathSegment(segments[5])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "policyArn is required")
			return true
		}
		if err := s.eks.DisassociateAccessPolicy(clusterName, principalArn, policyArn); err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	return false
}

func (s *Server) handleEKSStage5(w http.ResponseWriter, r *http.Request) bool {
	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 1 && segments[0] == "eks-anywhere-subscriptions" {
		switch r.Method {
		case http.MethodPost:
			var req eksCreateEksAnywhereSubscriptionRequest
			if err := decodeEKSBody(r, &req); err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
				return true
			}
			in := ekssvc.CreateEksAnywhereSubscriptionInput{
				Name: strings.TrimSpace(req.Name),
				Term: ekssvc.EksAnywhereSubscriptionTerm{
					Duration: req.Term.Duration,
					Unit:     strings.TrimSpace(req.Term.Unit),
				},
				LicenseQuantity: req.LicenseQuantity,
				LicenseType:     strings.TrimSpace(req.LicenseType),
				AutoRenew:       req.AutoRenew,
				Tags:            req.Tags,
			}
			subscription, err := s.eks.CreateEksAnywhereSubscription(in)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"subscription": subscription})
			return true
		case http.MethodGet:
			maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
			if err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
				return true
			}
			nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
			if !ok {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
				return true
			}
			statusFilter := map[string]bool{}
			for _, value := range r.URL.Query()["includeStatus"] {
				for _, entry := range strings.Split(value, ",") {
					status := strings.TrimSpace(entry)
					if status == "" {
						continue
					}
					statusFilter[strings.ToUpper(status)] = true
				}
			}
			subscriptions := s.eks.ListEksAnywhereSubscriptions()
			if len(statusFilter) > 0 {
				filtered := make([]ekssvc.EksAnywhereSubscription, 0, len(subscriptions))
				for _, subscription := range subscriptions {
					if statusFilter[strings.ToUpper(subscription.Status)] {
						filtered = append(filtered, subscription)
					}
				}
				subscriptions = filtered
			}
			start, end, outNextToken, err := paginateEKSBounds(len(subscriptions), nextToken, maxResults)
			if err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
				return true
			}
			out := map[string]any{"subscriptions": subscriptions[start:end]}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondEKSJSON(w, http.StatusOK, out)
			return true
		default:
			return false
		}
	}

	if len(segments) == 2 && segments[0] == "eks-anywhere-subscriptions" {
		subscriptionID, ok := decodeEKSPathSegment(segments[1])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "subscription id is required")
			return true
		}
		switch r.Method {
		case http.MethodGet:
			subscription, err := s.eks.DescribeEksAnywhereSubscription(subscriptionID)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"subscription": subscription})
			return true
		case http.MethodPost:
			var req eksUpdateEksAnywhereSubscriptionRequest
			if err := decodeEKSBody(r, &req); err != nil {
				respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
				return true
			}
			subscription, err := s.eks.UpdateEksAnywhereSubscription(subscriptionID, ekssvc.UpdateEksAnywhereSubscriptionInput{
				AutoRenew: req.AutoRenew,
				Tags:      req.Tags,
			})
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"subscription": subscription})
			return true
		case http.MethodDelete:
			subscription, err := s.eks.DeleteEksAnywhereSubscription(subscriptionID)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"subscription": subscription})
			return true
		default:
			return false
		}
	}

	if len(segments) == 1 && segments[0] == "cluster-registrations" && r.Method == http.MethodPost {
		var req eksRegisterClusterRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		cluster, err := s.eks.RegisterCluster(ekssvc.RegisterClusterInput{
			Name:           strings.TrimSpace(req.Name),
			ConnectorRole:  strings.TrimSpace(req.ConnectorConfig.RoleArn),
			ConnectorCloud: strings.TrimSpace(req.ConnectorConfig.Provider),
			Tags:           req.Tags,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"cluster": cluster})
		return true
	}

	if len(segments) == 2 && segments[0] == "cluster-registrations" && r.Method == http.MethodDelete {
		clusterName, ok := decodeEKSPathSegment(segments[1])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "cluster name is required")
			return true
		}
		cluster, err := s.eks.DeregisterCluster(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"cluster": cluster})
		return true
	}

	if segments[0] != "clusters" || len(segments) < 2 {
		return false
	}

	clusterName, ok := decodeEKSPathSegment(segments[1])
	if !ok {
		respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "cluster name is required")
		return true
	}

	switch {
	case len(segments) == 3 && segments[2] == "capabilities" && r.Method == http.MethodPost:
		var req eksCreateCapabilityRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		capability, err := s.eks.CreateCapability(clusterName, ekssvc.CreateCapabilityInput{
			CapabilityName: strings.TrimSpace(req.CapabilityName),
			Tags:           req.Tags,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"capability": capability})
		return true

	case len(segments) == 3 && segments[2] == "capabilities" && r.Method == http.MethodGet:
		capabilities, err := s.eks.ListCapabilities(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(capabilities), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"capabilities": capabilities[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "capabilities" && r.Method == http.MethodGet:
		capabilityName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "capability name is required")
			return true
		}
		capability, err := s.eks.DescribeCapability(clusterName, capabilityName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"capability": capability})
		return true

	case len(segments) == 4 && segments[2] == "capabilities" && r.Method == http.MethodPost:
		capabilityName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "capability name is required")
			return true
		}
		var req eksUpdateCapabilityRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		update, err := s.eks.UpdateCapability(clusterName, capabilityName, ekssvc.UpdateCapabilityInput{Tags: req.Tags})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true

	case len(segments) == 4 && segments[2] == "capabilities" && r.Method == http.MethodDelete:
		capabilityName, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "capability name is required")
			return true
		}
		capability, err := s.eks.DeleteCapability(clusterName, capabilityName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"capability": capability})
		return true

	case len(segments) == 3 && segments[2] == "insights" && r.Method == http.MethodPost:
		var req eksListInsightsRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		filter := req.Filter
		insights, err := s.eks.ListInsights(clusterName, ekssvc.ListInsightsInput{
			Categories:         filter.Categories,
			KubernetesVersions: filter.KubernetesVersions,
			Statuses:           filter.Statuses,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(insights), req.NextToken, req.MaxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"insights": insights[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "insights" && r.Method == http.MethodGet:
		insightID, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "insight id is required")
			return true
		}
		insight, err := s.eks.DescribeInsight(clusterName, insightID)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"insight": insight})
		return true

	case len(segments) == 3 && segments[2] == "insights-refresh" && r.Method == http.MethodPost:
		refresh, err := s.eks.StartInsightsRefresh(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"insightsRefresh": refresh})
		return true

	case len(segments) == 3 && segments[2] == "insights-refresh" && r.Method == http.MethodGet:
		refresh, err := s.eks.DescribeInsightsRefresh(clusterName)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"insightsRefresh": refresh})
		return true

	case len(segments) == 3 && segments[2] == "pod-identity-associations" && r.Method == http.MethodPost:
		var req eksCreatePodIdentityAssociationRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		association, err := s.eks.CreatePodIdentityAssociation(clusterName, ekssvc.CreatePodIdentityAssociationInput{
			Namespace:      strings.TrimSpace(req.Namespace),
			ServiceAccount: strings.TrimSpace(req.ServiceAccount),
			RoleArn:        strings.TrimSpace(req.RoleArn),
			Tags:           req.Tags,
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"association": association})
		return true

	case len(segments) == 3 && segments[2] == "pod-identity-associations" && r.Method == http.MethodGet:
		namespace, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("namespace"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid namespace")
			return true
		}
		serviceAccount, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("serviceAccount"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid serviceAccount")
			return true
		}
		maxResults, err := parseOptionalEKSMaxResults(r.URL.Query().Get("maxResults"))
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid maxResults")
			return true
		}
		nextToken, ok := decodeOptionalEKSQueryValue(r.URL.Query().Get("nextToken"))
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		associations, err := s.eks.ListPodIdentityAssociations(clusterName, namespace, serviceAccount)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		start, end, outNextToken, err := paginateEKSBounds(len(associations), nextToken, maxResults)
		if err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid nextToken")
			return true
		}
		out := map[string]any{"associations": associations[start:end]}
		if outNextToken != "" {
			out["nextToken"] = outNextToken
		}
		respondEKSJSON(w, http.StatusOK, out)
		return true

	case len(segments) == 4 && segments[2] == "pod-identity-associations" && r.Method == http.MethodGet:
		associationID, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "association id is required")
			return true
		}
		association, err := s.eks.DescribePodIdentityAssociation(clusterName, associationID)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"association": association})
		return true

	case len(segments) == 4 && segments[2] == "pod-identity-associations" && r.Method == http.MethodPost:
		associationID, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "association id is required")
			return true
		}
		var req eksUpdatePodIdentityAssociationRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		association, err := s.eks.UpdatePodIdentityAssociation(clusterName, associationID, ekssvc.UpdatePodIdentityAssociationInput{
			RoleArn: strings.TrimSpace(req.RoleArn),
		})
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"association": association})
		return true

	case len(segments) == 4 && segments[2] == "pod-identity-associations" && r.Method == http.MethodDelete:
		associationID, ok := decodeEKSPathSegment(segments[3])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "association id is required")
			return true
		}
		association, err := s.eks.DeletePodIdentityAssociation(clusterName, associationID)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"association": association})
		return true
	}

	return false
}

func (s *Server) handleEKSStage6(w http.ResponseWriter, r *http.Request) bool {
	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		return false
	}

	if len(segments) == 2 && segments[0] == "tags" {
		resourceARN, ok := decodeEKSPathSegment(segments[1])
		if !ok {
			respondEKSError(w, http.StatusBadRequest, "BadRequestException", "resourceArn is required")
			return true
		}
		switch r.Method {
		case http.MethodGet:
			tags, err := s.eks.ListTagsForResource(resourceARN)
			if err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{"tags": tags})
			return true
		case http.MethodPost:
			var req eksTagResourceRequest
			if err := decodeEKSBody(r, &req); err != nil {
				respondEKSError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
				return true
			}
			if err := s.eks.TagResource(resourceARN, req.Tags); err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{})
			return true
		case http.MethodDelete:
			var tagKeys []string
			for _, value := range r.URL.Query()["tagKeys"] {
				for _, key := range strings.Split(value, ",") {
					trimmed := strings.TrimSpace(key)
					if trimmed == "" {
						continue
					}
					tagKeys = append(tagKeys, trimmed)
				}
			}
			if err := s.eks.UntagResource(resourceARN, tagKeys); err != nil {
				respondEKSErrorForErr(w, err)
				return true
			}
			respondEKSJSON(w, http.StatusOK, map[string]any{})
			return true
		default:
			return false
		}
	}

	if segments[0] != "clusters" || len(segments) < 2 {
		return false
	}
	clusterName, ok := decodeEKSPathSegment(segments[1])
	if !ok {
		respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "cluster name is required")
		return true
	}

	if len(segments) == 4 && segments[2] == "encryption-config" && segments[3] == "associate" && r.Method == http.MethodPost {
		var req eksAssociateEncryptionConfigRequest
		if err := decodeEKSBody(r, &req); err != nil {
			respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", "invalid JSON body")
			return true
		}
		entries := make([]ekssvc.EncryptionConfigEntry, 0, len(req.EncryptionConfig))
		for _, config := range req.EncryptionConfig {
			entries = append(entries, ekssvc.EncryptionConfigEntry{
				Resources: config.Resources,
				Provider: ekssvc.EncryptionProvider{
					KeyArn: strings.TrimSpace(config.Provider.KeyArn),
				},
			})
		}
		update, err := s.eks.AssociateEncryptionConfig(clusterName, entries)
		if err != nil {
			respondEKSErrorForErr(w, err)
			return true
		}
		respondEKSJSON(w, http.StatusOK, map[string]any{"update": update})
		return true
	}

	return false
}

func isEKSRESTCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "eks" {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	isEKSHost := strings.Contains(host, ".eks.") || strings.HasPrefix(host, "eks.")

	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = "/"
	}

	prefixes := []string{
		"/access-entries",
		"/access-policies",
		"/addons",
		"/capabilities",
		"/cluster-registrations",
		"/clusters",
		"/eks-anywhere-subscriptions",
		"/fargate-profiles",
		"/identity-provider-configs",
		"/insights",
		"/insights-refresh",
		"/nodegroups",
		"/pod-identity-associations",
		"/registered-clusters",
		"/updates",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	if strings.HasPrefix(path, "/tags") && (service == "eks" || isEKSHost) {
		return true
	}

	if service == "eks" {
		return true
	}

	return isEKSHost
}

func respondEKSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondEKSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondEKSJSON(w, status, eksError{Type: code, Message: msg})
}

func respondEKSErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ekssvc.ErrInvalidParameter):
		respondEKSError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
	case errors.Is(err, ekssvc.ErrClusterAlreadyExists),
		errors.Is(err, ekssvc.ErrNodegroupAlreadyExists),
		errors.Is(err, ekssvc.ErrFargateProfileExists),
		errors.Is(err, ekssvc.ErrAddonAlreadyExists),
		errors.Is(err, ekssvc.ErrIdentityProviderExists),
		errors.Is(err, ekssvc.ErrAccessEntryAlreadyExists),
		errors.Is(err, ekssvc.ErrCapabilityAlreadyExists),
		errors.Is(err, ekssvc.ErrPodIdentityExists),
		errors.Is(err, ekssvc.ErrSubscriptionExists):
		respondEKSError(w, http.StatusConflict, "ResourceInUseException", err.Error())
	case errors.Is(err, ekssvc.ErrClusterNotFound),
		errors.Is(err, ekssvc.ErrNodegroupNotFound),
		errors.Is(err, ekssvc.ErrFargateProfileNotFound),
		errors.Is(err, ekssvc.ErrUpdateNotFound),
		errors.Is(err, ekssvc.ErrAddonNotFound),
		errors.Is(err, ekssvc.ErrCapabilityNotFound),
		errors.Is(err, ekssvc.ErrIdentityProviderNotFound),
		errors.Is(err, ekssvc.ErrAccessEntryNotFound),
		errors.Is(err, ekssvc.ErrAssociatedPolicyNotFound),
		errors.Is(err, ekssvc.ErrInsightNotFound),
		errors.Is(err, ekssvc.ErrPodIdentityNotFound),
		errors.Is(err, ekssvc.ErrSubscriptionNotFound):
		respondEKSError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
	case errors.Is(err, ekssvc.ErrTagNotFound):
		respondEKSError(w, http.StatusNotFound, "NotFoundException", err.Error())
	case errors.Is(err, ekssvc.ErrAccessPolicyNotFound):
		respondEKSError(w, http.StatusNotFound, "InvalidRequestException", err.Error())
	default:
		respondEKSError(w, http.StatusBadRequest, "ClientException", err.Error())
	}
}

func splitPathSegments(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func decodeEKSPathSegment(value string) (string, bool) {
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

func decodeOptionalEKSQueryValue(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", true
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", false
	}
	return decoded, true
}

func decodeEKSBody(r *http.Request, out any) error {
	body, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func parseOptionalEKSMaxResults(value string) (*int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 || parsed > 100 {
		return nil, errors.New("invalid maxResults")
	}
	return &parsed, nil
}

func paginateEKSBounds(total int, nextToken string, maxResults *int) (int, int, string, error) {
	start := 0
	if strings.TrimSpace(nextToken) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(nextToken))
		if err != nil || parsed < 0 || parsed > total {
			return 0, 0, "", errors.New("invalid nextToken")
		}
		start = parsed
	}
	end := total
	if maxResults != nil {
		if *maxResults <= 0 || *maxResults > 100 {
			return 0, 0, "", errors.New("invalid maxResults")
		}
		end = start + *maxResults
		if end > total {
			end = total
		}
	}
	outNextToken := ""
	if end < total {
		outNextToken = strconv.Itoa(end)
	}
	return start, end, outNextToken, nil
}

type eksResourcesVpcConfigRequest struct {
	SubnetIDs            []string `json:"subnetIds,omitempty"`
	EndpointPublicAccess *bool    `json:"endpointPublicAccess,omitempty"`
}

func (c *eksResourcesVpcConfigRequest) toServiceInput() *ekssvc.ResourcesVpcConfigInput {
	if c == nil {
		return nil
	}
	return &ekssvc.ResourcesVpcConfigInput{
		SubnetIDs:            c.SubnetIDs,
		EndpointPublicAccess: c.EndpointPublicAccess,
	}
}

type eksCreateClusterRequest struct {
	Name               string                        `json:"name"`
	Version            string                        `json:"version,omitempty"`
	RoleArn            string                        `json:"roleArn"`
	ResourcesVpcConfig *eksResourcesVpcConfigRequest `json:"resourcesVpcConfig,omitempty"`
	Tags               map[string]string             `json:"tags,omitempty"`
}

type eksUpdateClusterConfigRequest struct {
	ResourcesVpcConfig *eksResourcesVpcConfigRequest `json:"resourcesVpcConfig,omitempty"`
}

type eksUpdateClusterVersionRequest struct {
	Version string `json:"version,omitempty"`
}

type eksCreateNodegroupRequest struct {
	NodegroupName string            `json:"nodegroupName"`
	NodeRole      string            `json:"nodeRole"`
	Subnets       []string          `json:"subnets,omitempty"`
	Version       string            `json:"version,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type eksUpdateNodegroupConfigRequest struct {
	Labels map[string]string `json:"labels,omitempty"`
}

type eksUpdateNodegroupVersionRequest struct {
	Version string `json:"version,omitempty"`
}

type eksFargateSelectorRequest struct {
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type eksCreateFargateProfileRequest struct {
	FargateProfileName  string                      `json:"fargateProfileName"`
	PodExecutionRoleArn string                      `json:"podExecutionRoleArn"`
	Subnets             []string                    `json:"subnets,omitempty"`
	Selectors           []eksFargateSelectorRequest `json:"selectors,omitempty"`
	Tags                map[string]string           `json:"tags,omitempty"`
}

type eksCreateAddonRequest struct {
	AddonName             string            `json:"addonName"`
	AddonVersion          string            `json:"addonVersion,omitempty"`
	ServiceAccountRoleArn string            `json:"serviceAccountRoleArn,omitempty"`
	ConfigurationValues   string            `json:"configurationValues,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty"`
}

type eksUpdateAddonRequest struct {
	AddonVersion          string `json:"addonVersion,omitempty"`
	ServiceAccountRoleArn string `json:"serviceAccountRoleArn,omitempty"`
	ConfigurationValues   string `json:"configurationValues,omitempty"`
}

type eksDescribeAddonConfigurationRequest struct {
	AddonName    string `json:"addonName,omitempty"`
	AddonVersion string `json:"addonVersion,omitempty"`
}

type eksOIDCIdentityProviderConfigRequest struct {
	IdentityProviderConfigName string            `json:"identityProviderConfigName"`
	IssuerURL                  string            `json:"issuerUrl,omitempty"`
	ClientID                   string            `json:"clientId,omitempty"`
	UsernameClaim              string            `json:"usernameClaim,omitempty"`
	UsernamePrefix             string            `json:"usernamePrefix,omitempty"`
	GroupsClaim                string            `json:"groupsClaim,omitempty"`
	GroupsPrefix               string            `json:"groupsPrefix,omitempty"`
	RequiredClaims             map[string]string `json:"requiredClaims,omitempty"`
	Tags                       map[string]string `json:"tags,omitempty"`
}

type eksIdentityProviderConfigKeyRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type eksAssociateIdentityProviderConfigRequest struct {
	Oidc *eksOIDCIdentityProviderConfigRequest `json:"oidc,omitempty"`
	Tags map[string]string                     `json:"tags,omitempty"`
}

type eksDisassociateIdentityProviderConfigRequest struct {
	IdentityProviderConfig *eksIdentityProviderConfigKeyRequest `json:"identityProviderConfig,omitempty"`
}

type eksDescribeIdentityProviderConfigRequest struct {
	IdentityProviderConfig *eksIdentityProviderConfigKeyRequest `json:"identityProviderConfig,omitempty"`
}

type eksCreateAccessEntryRequest struct {
	PrincipalArn     string            `json:"principalArn"`
	Type             string            `json:"type,omitempty"`
	Username         string            `json:"username,omitempty"`
	KubernetesGroups []string          `json:"kubernetesGroups,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

type eksUpdateAccessEntryRequest struct {
	Username         *string  `json:"username,omitempty"`
	KubernetesGroups []string `json:"kubernetesGroups,omitempty"`
}

type eksAccessScopeRequest struct {
	Type       string   `json:"type,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
}

type eksAssociateAccessPolicyRequest struct {
	PolicyArn   string                `json:"policyArn"`
	AccessScope eksAccessScopeRequest `json:"accessScope"`
}

type eksCreateCapabilityRequest struct {
	CapabilityName string            `json:"capabilityName"`
	Tags           map[string]string `json:"tags,omitempty"`
}

type eksUpdateCapabilityRequest struct {
	Tags map[string]string `json:"tags,omitempty"`
}

type eksInsightsFilterRequest struct {
	Categories         []string `json:"categories,omitempty"`
	KubernetesVersions []string `json:"kubernetesVersions,omitempty"`
	Statuses           []string `json:"statuses,omitempty"`
}

type eksListInsightsRequest struct {
	Filter     eksInsightsFilterRequest `json:"filter,omitempty"`
	MaxResults *int                     `json:"maxResults,omitempty"`
	NextToken  string                   `json:"nextToken,omitempty"`
}

type eksCreatePodIdentityAssociationRequest struct {
	Namespace      string            `json:"namespace"`
	ServiceAccount string            `json:"serviceAccount"`
	RoleArn        string            `json:"roleArn"`
	Tags           map[string]string `json:"tags,omitempty"`
}

type eksUpdatePodIdentityAssociationRequest struct {
	RoleArn string `json:"roleArn,omitempty"`
}

type eksEksAnywhereSubscriptionTermRequest struct {
	Duration int    `json:"duration,omitempty"`
	Unit     string `json:"unit,omitempty"`
}

type eksCreateEksAnywhereSubscriptionRequest struct {
	Name            string                                `json:"name"`
	Term            eksEksAnywhereSubscriptionTermRequest `json:"term,omitempty"`
	LicenseQuantity int                                   `json:"licenseQuantity,omitempty"`
	LicenseType     string                                `json:"licenseType,omitempty"`
	AutoRenew       bool                                  `json:"autoRenew,omitempty"`
	Tags            map[string]string                     `json:"tags,omitempty"`
}

type eksUpdateEksAnywhereSubscriptionRequest struct {
	AutoRenew *bool             `json:"autoRenew,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
}

type eksRegisterClusterConnectorConfigRequest struct {
	RoleArn  string `json:"roleArn"`
	Provider string `json:"provider"`
}

type eksRegisterClusterRequest struct {
	Name            string                                   `json:"name"`
	ConnectorConfig eksRegisterClusterConnectorConfigRequest `json:"connectorConfig"`
	Tags            map[string]string                        `json:"tags,omitempty"`
}

type eksEncryptionProviderRequest struct {
	KeyArn string `json:"keyArn,omitempty"`
}

type eksEncryptionConfigEntryRequest struct {
	Resources []string                     `json:"resources,omitempty"`
	Provider  eksEncryptionProviderRequest `json:"provider,omitempty"`
}

type eksAssociateEncryptionConfigRequest struct {
	EncryptionConfig []eksEncryptionConfigEntryRequest `json:"encryptionConfig"`
}

type eksTagResourceRequest struct {
	Tags map[string]string `json:"tags"`
}

func (r *eksCreateFargateProfileRequest) toServiceSelectors() []ekssvc.FargateProfileSelector {
	if len(r.Selectors) == 0 {
		return nil
	}
	out := make([]ekssvc.FargateProfileSelector, 0, len(r.Selectors))
	for _, selector := range r.Selectors {
		namespace := strings.TrimSpace(selector.Namespace)
		if namespace == "" {
			continue
		}
		out = append(out, ekssvc.FargateProfileSelector{
			Namespace: namespace,
			Labels:    selector.Labels,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Namespace < out[j].Namespace
	})
	return out
}
