package server

import (
	"net/http"
	"sort"
	"strings"
)

func (s *Server) handleCognitoUserPoolsStage3Action(w http.ResponseWriter, action string, payload map[string]any) bool {
	switch action {
	case "AdminCreateUser":
		attributes, ok := cognitoUserPoolsAttributes(payload["UserAttributes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserAttributes")
			return true
		}
		deliveryMediums, ok := cognitoUserPoolsStringSlice(payload["DesiredDeliveryMediums"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid DesiredDeliveryMediums")
			return true
		}
		forceAliasCreation, ok := cognitoUserPoolsBool(payload["ForceAliasCreation"])
		if payload["ForceAliasCreation"] != nil && !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ForceAliasCreation")
			return true
		}
		record, err := s.cognitouserpools.AdminCreateUser(cognitoUserPoolsAdminCreateUserInput{
			UserPoolID:         cognitoUserPoolsString(payload["UserPoolId"]),
			Username:           cognitoUserPoolsString(payload["Username"]),
			TemporaryPassword:  cognitoUserPoolsString(payload["TemporaryPassword"]),
			UserAttributes:     attributes,
			DesiredDelivery:    deliveryMediums,
			ForceAliasCreation: forceAliasCreation,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"User": cognitoUserPoolsAdminUserPayload(record)})
		return true

	case "AdminGetUser":
		record, err := s.cognitouserpools.AdminGetUser(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsAdminGetUserPayload(record))
		return true

	case "ListUsers":
		limit, ok := cognitoUserPoolsIntDefault(payload, "Limit", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListUsers(
			cognitoUserPoolsString(payload["UserPoolId"]),
			limit,
			cognitoUserPoolsString(payload["PaginationToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Users": cognitoUserPoolsUsersPayload(records)}
		if nextToken != "" {
			out["PaginationToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "AdminUpdateUserAttributes":
		attributes, ok := cognitoUserPoolsAttributes(payload["UserAttributes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserAttributes")
			return true
		}
		if err := s.cognitouserpools.AdminUpdateUserAttributes(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			attributes,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminDeleteUserAttributes":
		attributeNames, ok := cognitoUserPoolsStringSlice(payload["UserAttributeNames"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserAttributeNames")
			return true
		}
		if err := s.cognitouserpools.AdminDeleteUserAttributes(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			attributeNames,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminDeleteUser":
		if err := s.cognitouserpools.AdminDeleteUser(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminDisableUser":
		if err := s.cognitouserpools.AdminDisableUser(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminEnableUser":
		if err := s.cognitouserpools.AdminEnableUser(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminSetUserPassword":
		permanent, ok := cognitoUserPoolsBool(payload["Permanent"])
		if payload["Permanent"] != nil && !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Permanent")
			return true
		}
		if err := s.cognitouserpools.AdminSetUserPassword(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["Password"]),
			permanent,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminResetUserPassword":
		destination, err := s.cognitouserpools.AdminResetUserPassword(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{
			"CodeDeliveryDetails": map[string]any{
				"Destination":    destination,
				"DeliveryMedium": "EMAIL",
				"AttributeName":  "email",
			},
		})
		return true

	case "AdminConfirmSignUp":
		if err := s.cognitouserpools.AdminConfirmSignUp(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminUserGlobalSignOut":
		if err := s.cognitouserpools.AdminUserGlobalSignOut(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateGroup":
		precedence, precedenceSet, ok := cognitoUserPoolsIntPointer(payload, "Precedence")
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Precedence")
			return true
		}
		record, err := s.cognitouserpools.CreateGroup(cognitoUserPoolsCreateGroupInput{
			UserPoolID:  cognitoUserPoolsString(payload["UserPoolId"]),
			GroupName:   cognitoUserPoolsString(payload["GroupName"]),
			Description: cognitoUserPoolsString(payload["Description"]),
			RoleARN:     cognitoUserPoolsString(payload["RoleArn"]),
			Precedence:  precedence,
		})
		if !precedenceSet {
			record.Precedence = nil
		}
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Group": cognitoUserPoolsGroupPayload(record)})
		return true

	case "GetGroup":
		record, err := s.cognitouserpools.GetGroup(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["GroupName"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Group": cognitoUserPoolsGroupPayload(record)})
		return true

	case "UpdateGroup":
		precedence, precedenceSet, ok := cognitoUserPoolsIntPointer(payload, "Precedence")
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Precedence")
			return true
		}
		input := cognitoUserPoolsUpdateGroupInput{
			UserPoolID: cognitoUserPoolsString(payload["UserPoolId"]),
			GroupName:  cognitoUserPoolsString(payload["GroupName"]),
		}
		if raw, exists := cognitoUserPoolsField(payload, "Description"); exists {
			input.DescriptionSet = true
			input.Description = cognitoUserPoolsString(raw)
		}
		if raw, exists := cognitoUserPoolsField(payload, "RoleArn"); exists {
			input.RoleARNSet = true
			input.RoleARN = cognitoUserPoolsString(raw)
		}
		if precedenceSet {
			input.PrecedenceSet = true
			input.Precedence = precedence
		}
		record, err := s.cognitouserpools.UpdateGroup(input)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Group": cognitoUserPoolsGroupPayload(record)})
		return true

	case "DeleteGroup":
		if err := s.cognitouserpools.DeleteGroup(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["GroupName"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListGroups":
		limit, ok := cognitoUserPoolsIntDefault(payload, "Limit", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListGroups(
			cognitoUserPoolsString(payload["UserPoolId"]),
			limit,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Groups": cognitoUserPoolsGroupsPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "AdminAddUserToGroup":
		if err := s.cognitouserpools.AdminAddUserToGroup(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["GroupName"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminRemoveUserFromGroup":
		if err := s.cognitouserpools.AdminRemoveUserFromGroup(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["GroupName"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminListGroupsForUser":
		limit, ok := cognitoUserPoolsIntDefault(payload, "Limit", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		records, nextToken, err := s.cognitouserpools.AdminListGroupsForUser(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			limit,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Groups": cognitoUserPoolsGroupsPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "ListUsersInGroup":
		limit, ok := cognitoUserPoolsIntDefault(payload, "Limit", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListUsersInGroup(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["GroupName"]),
			limit,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Users": cognitoUserPoolsUsersPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "AdminGetDevice":
		record, err := s.cognitouserpools.AdminGetDevice(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Device": cognitoUserPoolsDevicePayload(record)})
		return true

	case "AdminListDevices":
		limit, ok := cognitoUserPoolsIntDefault(payload, "Limit", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		records, nextToken, err := s.cognitouserpools.AdminListDevices(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			limit,
			cognitoUserPoolsString(payload["PaginationToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Devices": cognitoUserPoolsDevicesPayload(records)}
		if nextToken != "" {
			out["PaginationToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "AdminUpdateDeviceStatus":
		if err := s.cognitouserpools.AdminUpdateDeviceStatus(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
			cognitoUserPoolsString(payload["DeviceRememberedStatus"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminForgetDevice":
		if err := s.cognitouserpools.AdminForgetDevice(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateUserImportJob":
		record, err := s.cognitouserpools.CreateUserImportJob(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["JobName"]),
			cognitoUserPoolsString(payload["CloudWatchLogsRoleArn"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserImportJob": cognitoUserPoolsImportJobPayload(record)})
		return true

	case "DescribeUserImportJob":
		record, err := s.cognitouserpools.DescribeUserImportJob(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["JobId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserImportJob": cognitoUserPoolsImportJobPayload(record)})
		return true

	case "ListUserImportJobs":
		maxResults, ok := cognitoUserPoolsIntDefault(payload, "MaxResults", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListUserImportJobs(
			cognitoUserPoolsString(payload["UserPoolId"]),
			maxResults,
			cognitoUserPoolsString(payload["PaginationToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"UserImportJobs": cognitoUserPoolsImportJobsPayload(records)}
		if nextToken != "" {
			out["PaginationToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "StartUserImportJob":
		record, err := s.cognitouserpools.StartUserImportJob(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["JobId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserImportJob": cognitoUserPoolsImportJobPayload(record)})
		return true

	case "StopUserImportJob":
		record, err := s.cognitouserpools.StopUserImportJob(
			cognitoUserPoolsString(payload["UserPoolId"]),
			cognitoUserPoolsString(payload["JobId"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserImportJob": cognitoUserPoolsImportJobPayload(record)})
		return true

	case "GetCSVHeader":
		header, err := s.cognitouserpools.GetCSVHeader(cognitoUserPoolsString(payload["UserPoolId"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"CSVHeader": header})
		return true
	}

	return false
}

func (s *Server) handleCognitoUserPoolsStage4Action(w http.ResponseWriter, action string, payload map[string]any) bool {
	switch action {
	case "SignUp":
		attributes, ok := cognitoUserPoolsAttributes(payload["UserAttributes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserAttributes")
			return true
		}
		record, userConfirmed, err := s.cognitouserpools.SignUp(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["Password"]),
			attributes,
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{
			"UserConfirmed": userConfirmed,
			"UserSub":       record.Sub,
		}
		if !userConfirmed {
			out["CodeDeliveryDetails"] = map[string]any{
				"Destination":    cognitoUserPoolsDeliveryDestination(record),
				"DeliveryMedium": "EMAIL",
				"AttributeName":  "email",
			}
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "ConfirmSignUp":
		if err := s.cognitouserpools.ConfirmSignUp(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["ConfirmationCode"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ResendConfirmationCode":
		destination, err := s.cognitouserpools.ResendConfirmationCode(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["Username"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{
			"CodeDeliveryDetails": map[string]any{
				"Destination":    destination,
				"DeliveryMedium": "EMAIL",
				"AttributeName":  "email",
			},
		})
		return true

	case "InitiateAuth":
		authParams, ok := cognitoUserPoolsStringMap(payload["AuthParameters"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid AuthParameters")
			return true
		}
		result, err := s.cognitouserpools.InitiateAuth(cognitoUserPoolsInitiateAuthInput{
			ClientID:       cognitoUserPoolsString(payload["ClientId"]),
			AuthFlow:       cognitoUserPoolsString(payload["AuthFlow"]),
			AuthParameters: authParams,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsAuthFlowPayload(result))
		return true

	case "AdminInitiateAuth":
		authParams, ok := cognitoUserPoolsStringMap(payload["AuthParameters"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid AuthParameters")
			return true
		}
		result, err := s.cognitouserpools.InitiateAuth(cognitoUserPoolsInitiateAuthInput{
			UserPoolID:     cognitoUserPoolsString(payload["UserPoolId"]),
			ClientID:       cognitoUserPoolsString(payload["ClientId"]),
			AuthFlow:       cognitoUserPoolsString(payload["AuthFlow"]),
			AuthParameters: authParams,
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsAuthFlowPayload(result))
		return true

	case "RespondToAuthChallenge":
		responses, ok := cognitoUserPoolsStringMap(payload["ChallengeResponses"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ChallengeResponses")
			return true
		}
		result, err := s.cognitouserpools.RespondToAuthChallenge(cognitoUserPoolsRespondAuthChallengeInput{
			ClientID:           cognitoUserPoolsString(payload["ClientId"]),
			ChallengeName:      cognitoUserPoolsString(payload["ChallengeName"]),
			ChallengeResponses: responses,
			Session:            cognitoUserPoolsString(payload["Session"]),
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsAuthFlowPayload(result))
		return true

	case "AdminRespondToAuthChallenge":
		responses, ok := cognitoUserPoolsStringMap(payload["ChallengeResponses"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid ChallengeResponses")
			return true
		}
		result, err := s.cognitouserpools.RespondToAuthChallenge(cognitoUserPoolsRespondAuthChallengeInput{
			UserPoolID:         cognitoUserPoolsString(payload["UserPoolId"]),
			ClientID:           cognitoUserPoolsString(payload["ClientId"]),
			ChallengeName:      cognitoUserPoolsString(payload["ChallengeName"]),
			ChallengeResponses: responses,
			Session:            cognitoUserPoolsString(payload["Session"]),
		})
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsAuthFlowPayload(result))
		return true

	case "ForgotPassword":
		destination, err := s.cognitouserpools.ForgotPassword(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["Username"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{
			"CodeDeliveryDetails": map[string]any{
				"Destination":    destination,
				"DeliveryMedium": "EMAIL",
				"AttributeName":  "email",
			},
		})
		return true

	case "ConfirmForgotPassword":
		if err := s.cognitouserpools.ConfirmForgotPassword(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["Username"]),
			cognitoUserPoolsString(payload["ConfirmationCode"]),
			cognitoUserPoolsString(payload["Password"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ChangePassword":
		if err := s.cognitouserpools.ChangePassword(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["PreviousPassword"]),
			cognitoUserPoolsString(payload["ProposedPassword"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetUser":
		record, err := s.cognitouserpools.GetUser(cognitoUserPoolsString(payload["AccessToken"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsUserPayload(record))
		return true

	case "DeleteUser":
		if err := s.cognitouserpools.DeleteUser(cognitoUserPoolsString(payload["AccessToken"])); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UpdateUserAttributes":
		attributes, ok := cognitoUserPoolsAttributes(payload["UserAttributes"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserAttributes")
			return true
		}
		requiresVerification, err := s.cognitouserpools.UpdateUserAttributes(
			cognitoUserPoolsString(payload["AccessToken"]),
			attributes,
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{}
		if len(requiresVerification) > 0 {
			details := make([]map[string]any, 0, len(requiresVerification))
			for _, name := range requiresVerification {
				details = append(details, map[string]any{
					"AttributeName":  name,
					"DeliveryMedium": "EMAIL",
					"Destination":    "***",
				})
			}
			out["CodeDeliveryDetailsList"] = details
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "DeleteUserAttributes":
		attributeNames, ok := cognitoUserPoolsStringSlice(payload["UserAttributeNames"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid UserAttributeNames")
			return true
		}
		if err := s.cognitouserpools.DeleteUserAttributes(
			cognitoUserPoolsString(payload["AccessToken"]),
			attributeNames,
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetUserAttributeVerificationCode":
		destination, err := s.cognitouserpools.GetUserAttributeVerificationCode(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["AttributeName"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{
			"CodeDeliveryDetails": map[string]any{
				"Destination":    destination,
				"DeliveryMedium": "EMAIL",
				"AttributeName":  cognitoUserPoolsString(payload["AttributeName"]),
			},
		})
		return true

	case "VerifyUserAttribute":
		if err := s.cognitouserpools.VerifyUserAttribute(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["AttributeName"]),
			cognitoUserPoolsString(payload["Code"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetTokensFromRefreshToken":
		result, err := s.cognitouserpools.GetTokensFromRefreshToken(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["RefreshToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"AuthenticationResult": cognitoUserPoolsAuthResultPayload(result)})
		return true

	case "RevokeToken":
		if err := s.cognitouserpools.RevokeToken(
			cognitoUserPoolsString(payload["ClientId"]),
			cognitoUserPoolsString(payload["Token"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GlobalSignOut":
		if err := s.cognitouserpools.GlobalSignOut(cognitoUserPoolsString(payload["AccessToken"])); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ConfirmDevice":
		if err := s.cognitouserpools.ConfirmDevice(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
			cognitoUserPoolsString(payload["DeviceName"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"UserConfirmationNecessary": false})
		return true

	case "GetDevice":
		record, err := s.cognitouserpools.GetDeviceByAccessToken(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Device": cognitoUserPoolsDevicePayload(record)})
		return true

	case "ListDevices":
		limit, ok := cognitoUserPoolsIntDefault(payload, "Limit", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListDevicesByAccessToken(
			cognitoUserPoolsString(payload["AccessToken"]),
			limit,
			cognitoUserPoolsString(payload["PaginationToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Devices": cognitoUserPoolsDevicesPayload(records)}
		if nextToken != "" {
			out["PaginationToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "UpdateDeviceStatus":
		if err := s.cognitouserpools.UpdateDeviceStatusByAccessToken(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
			cognitoUserPoolsString(payload["DeviceRememberedStatus"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ForgetDevice":
		if err := s.cognitouserpools.ForgetDeviceByAccessToken(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["DeviceKey"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	return false
}

func (s *Server) handleCognitoUserPoolsStage5Action(w http.ResponseWriter, action string, payload map[string]any) bool {
	switch action {
	case "AssociateSoftwareToken":
		secret, session, err := s.cognitouserpools.AssociateSoftwareToken(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["Session"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"SecretCode": secret, "Session": session})
		return true

	case "VerifySoftwareToken":
		status, session, err := s.cognitouserpools.VerifySoftwareToken(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["Session"]),
			cognitoUserPoolsString(payload["UserCode"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Status": status, "Session": session})
		return true

	case "SetUserMFAPreference":
		enabled, preferred, ok := cognitoUserPoolsMFASettingPointers(payload["SoftwareTokenMfaSettings"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid SoftwareTokenMfaSettings")
			return true
		}
		if err := s.cognitouserpools.SetUserMFAPreference(cognitoUserPoolsSetMFAPreferenceInput{
			ByAccessToken:     cognitoUserPoolsString(payload["AccessToken"]),
			SoftwareEnabled:   enabled,
			SoftwarePreferred: preferred,
		}); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "AdminSetUserMFAPreference":
		enabled, preferred, ok := cognitoUserPoolsMFASettingPointers(payload["SoftwareTokenMfaSettings"])
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid SoftwareTokenMfaSettings")
			return true
		}
		if err := s.cognitouserpools.SetUserMFAPreference(cognitoUserPoolsSetMFAPreferenceInput{
			UserPoolID:        cognitoUserPoolsString(payload["UserPoolId"]),
			Username:          cognitoUserPoolsString(payload["Username"]),
			SoftwareEnabled:   enabled,
			SoftwarePreferred: preferred,
		}); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetUserPoolMfaConfig":
		record, err := s.cognitouserpools.GetUserPoolMFAConfig(cognitoUserPoolsString(payload["UserPoolId"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsUserPoolMFAConfigPayload(record))
		return true

	case "SetUserPoolMfaConfig":
		enabled, enabledSet, ok := cognitoUserPoolsNestedBool(payload, "SoftwareTokenMfaConfiguration", "Enabled")
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid SoftwareTokenMfaConfiguration")
			return true
		}
		input := cognitoUserPoolsSetPoolMFAConfigInput{
			UserPoolID:       cognitoUserPoolsString(payload["UserPoolId"]),
			MFAConfiguration: cognitoUserPoolsString(payload["MfaConfiguration"]),
		}
		if enabledSet {
			input.SoftwareTokenMFAEnabled = &enabled
		}
		if raw, exists := cognitoUserPoolsField(payload, "WebAuthnConfiguration"); exists {
			cfg, ok := raw.(map[string]any)
			if !ok {
				respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid WebAuthnConfiguration")
				return true
			}
			if rawRPID, exists := cognitoUserPoolsField(cfg, "RelyingPartyId"); exists {
				input.WebAuthnRelyingPartyIDSet = true
				input.WebAuthnRelyingPartyID = cognitoUserPoolsString(rawRPID)
			}
			if rawUserVerification, exists := cognitoUserPoolsField(cfg, "UserVerification"); exists {
				input.WebAuthnUserVerificationSet = true
				input.WebAuthnUserVerification = cognitoUserPoolsString(rawUserVerification)
			}
		}
		record, err := s.cognitouserpools.SetUserPoolMFAConfig(input)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, cognitoUserPoolsUserPoolMFAConfigPayload(record))
		return true

	case "StartWebAuthnRegistration":
		result, err := s.cognitouserpools.StartWebAuthnRegistration(cognitoUserPoolsString(payload["AccessToken"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{
			"CredentialCreationOptions": map[string]any{
				"Challenge": result.Challenge,
			},
			"Session": result.Session,
		})
		return true

	case "CompleteWebAuthnRegistration":
		record, err := s.cognitouserpools.CompleteWebAuthnRegistration(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["Session"]),
			cognitoUserPoolsString(payload["Credential"]),
			cognitoUserPoolsString(payload["FriendlyName"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{"Credential": cognitoUserPoolsWebAuthnCredentialPayload(record)})
		return true

	case "ListWebAuthnCredentials":
		maxResults, ok := cognitoUserPoolsIntDefault(payload, "MaxResults", 60)
		if !ok {
			respondCognitoUserPoolsError(w, http.StatusBadRequest, "ValidationException", "invalid MaxResults")
			return true
		}
		records, nextToken, err := s.cognitouserpools.ListWebAuthnCredentials(
			cognitoUserPoolsString(payload["AccessToken"]),
			maxResults,
			cognitoUserPoolsString(payload["NextToken"]),
		)
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Credentials": cognitoUserPoolsWebAuthnCredentialsPayload(records)}
		if nextToken != "" {
			out["NextToken"] = nextToken
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true

	case "DeleteWebAuthnCredential":
		if err := s.cognitouserpools.DeleteWebAuthnCredential(
			cognitoUserPoolsString(payload["AccessToken"]),
			cognitoUserPoolsString(payload["CredentialId"]),
		); err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, map[string]any{})
		return true

	case "GetUserAuthFactors":
		factors, preferred, err := s.cognitouserpools.GetUserAuthFactors(cognitoUserPoolsString(payload["AccessToken"]))
		if err != nil {
			respondCognitoUserPoolsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"ConfiguredUserAuthFactors": factors}
		if preferred != "" {
			out["PreferredMfaSetting"] = preferred
		}
		respondCognitoUserPoolsJSON(w, http.StatusOK, out)
		return true
	}
	return false
}

func cognitoUserPoolsAttributes(value any) (map[string]string, bool) {
	if value == nil {
		return map[string]string{}, true
	}
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		name := strings.TrimSpace(cognitoUserPoolsString(obj["Name"]))
		if name == "" {
			return nil, false
		}
		out[name] = cognitoUserPoolsString(obj["Value"])
	}
	return out, true
}

func cognitoUserPoolsBool(value any) (bool, bool) {
	if value == nil {
		return false, false
	}
	parsed, ok := value.(bool)
	if !ok {
		return false, false
	}
	return parsed, true
}

func cognitoUserPoolsIntPointer(payload map[string]any, key string) (*int, bool, bool) {
	raw, exists := cognitoUserPoolsField(payload, key)
	if !exists {
		return nil, false, true
	}
	if raw == nil {
		return nil, true, true
	}
	value, ok := cognitoUserPoolsInt(raw)
	if !ok {
		return nil, false, false
	}
	copied := value
	return &copied, true, true
}

func cognitoUserPoolsNestedBool(payload map[string]any, objectKey, fieldKey string) (bool, bool, bool) {
	raw, exists := cognitoUserPoolsField(payload, objectKey)
	if !exists {
		return false, false, true
	}
	if raw == nil {
		return false, false, true
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return false, false, false
	}
	rawField, exists := cognitoUserPoolsField(obj, fieldKey)
	if !exists {
		return false, false, true
	}
	value, ok := cognitoUserPoolsBool(rawField)
	if !ok {
		return false, false, false
	}
	return value, true, true
}

func cognitoUserPoolsMFASettingPointers(value any) (*bool, *bool, bool) {
	if value == nil {
		return nil, nil, true
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, nil, false
	}
	var enabled *bool
	var preferred *bool
	if raw, exists := cognitoUserPoolsField(obj, "Enabled"); exists {
		parsed, ok := cognitoUserPoolsBool(raw)
		if !ok {
			return nil, nil, false
		}
		enabled = &parsed
	}
	if raw, exists := cognitoUserPoolsField(obj, "PreferredMfa"); exists {
		parsed, ok := cognitoUserPoolsBool(raw)
		if !ok {
			return nil, nil, false
		}
		preferred = &parsed
	}
	return enabled, preferred, true
}

func cognitoUserPoolsAdminGetUserPayload(record cognitoUserPoolsUserRecord) map[string]any {
	out := cognitoUserPoolsUserPayload(record)
	out["Enabled"] = record.Enabled
	out["UserStatus"] = record.UserStatus
	out["Username"] = record.Username
	out["UserCreateDate"] = float64(record.CreatedAt.Unix())
	out["UserLastModifiedDate"] = float64(record.UpdatedAt.Unix())
	return out
}

func cognitoUserPoolsAdminUserPayload(record cognitoUserPoolsUserRecord) map[string]any {
	return map[string]any{
		"Username":             record.Username,
		"Attributes":           cognitoUserPoolsAttributesPayload(record.Attributes),
		"Enabled":              record.Enabled,
		"UserStatus":           record.UserStatus,
		"UserCreateDate":       float64(record.CreatedAt.Unix()),
		"UserLastModifiedDate": float64(record.UpdatedAt.Unix()),
	}
}

func cognitoUserPoolsUsersPayload(records []cognitoUserPoolsUserRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsAdminUserPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["Username"].(string)
		right, _ := out[j]["Username"].(string)
		return strings.ToLower(left) < strings.ToLower(right)
	})
	return out
}

func cognitoUserPoolsAttributesPayload(attributes map[string]string) []map[string]any {
	if len(attributes) == 0 {
		return []map[string]any{}
	}
	names := make([]string, 0, len(attributes))
	for name := range attributes {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, map[string]any{"Name": name, "Value": attributes[name]})
	}
	return out
}

func cognitoUserPoolsUserPayload(record cognitoUserPoolsUserRecord) map[string]any {
	out := map[string]any{
		"Username":       record.Username,
		"UserAttributes": cognitoUserPoolsAttributesPayload(record.Attributes),
	}
	if record.PreferredMFA != "" {
		out["PreferredMfaSetting"] = record.PreferredMFA
	}
	mfa := make([]string, 0)
	if record.SoftwareTokenEnabled && record.SoftwareTokenVerified {
		mfa = append(mfa, "SOFTWARE_TOKEN_MFA")
	}
	if len(mfa) > 0 {
		out["UserMFASettingList"] = mfa
	}
	return out
}

func cognitoUserPoolsGroupPayload(record cognitoUserPoolsGroupRecord) map[string]any {
	out := map[string]any{
		"GroupName":        record.GroupName,
		"Description":      record.Description,
		"RoleArn":          record.RoleARN,
		"CreationDate":     float64(record.CreatedAt.Unix()),
		"LastModifiedDate": float64(record.LastModifiedAt.Unix()),
		"UserPoolId":       record.UserPoolID,
	}
	if record.Precedence != nil {
		out["Precedence"] = *record.Precedence
	}
	return out
}

func cognitoUserPoolsGroupsPayload(records []cognitoUserPoolsGroupRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsGroupPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["GroupName"].(string)
		right, _ := out[j]["GroupName"].(string)
		return strings.ToLower(left) < strings.ToLower(right)
	})
	return out
}

func cognitoUserPoolsDevicePayload(record cognitoUserPoolsDeviceRecord) map[string]any {
	out := map[string]any{
		"DeviceKey":                   record.DeviceKey,
		"DeviceAttributes":            []map[string]any{},
		"DeviceCreateDate":            float64(record.DeviceCreateDate.Unix()),
		"DeviceLastModifiedDate":      float64(record.DeviceLastModifiedDate.Unix()),
		"DeviceLastAuthenticatedDate": float64(record.DeviceLastAuthenticatedAt.Unix()),
	}
	if record.DeviceRememberedStatus != "" {
		out["DeviceRememberedStatus"] = strings.ToLower(record.DeviceRememberedStatus)
	}
	if record.DeviceName != "" {
		out["DeviceAttributes"] = []map[string]any{{"Name": "device_name", "Value": record.DeviceName}}
	}
	return out
}

func cognitoUserPoolsDevicesPayload(records []cognitoUserPoolsDeviceRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsDevicePayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["DeviceKey"].(string)
		right, _ := out[j]["DeviceKey"].(string)
		return left < right
	})
	return out
}

func cognitoUserPoolsImportJobPayload(record cognitoUserPoolsImportJobRecord) map[string]any {
	out := map[string]any{
		"JobName":               record.JobName,
		"JobId":                 record.JobID,
		"UserPoolId":            record.UserPoolID,
		"PreSignedUrl":          record.PreSignedURL,
		"Status":                record.Status,
		"CloudWatchLogsRoleArn": record.CloudWatchLogsRoleArn,
		"CreationDate":          float64(record.CreatedAt.Unix()),
		"LastModifiedDate":      float64(record.LastModifiedAt.Unix()),
	}
	if record.StartedAt != nil {
		out["StartDate"] = float64(record.StartedAt.Unix())
	}
	if record.CompletedAt != nil {
		out["CompletionDate"] = float64(record.CompletedAt.Unix())
	}
	return out
}

func cognitoUserPoolsImportJobsPayload(records []cognitoUserPoolsImportJobRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsImportJobPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["JobId"].(string)
		right, _ := out[j]["JobId"].(string)
		return left < right
	})
	return out
}

func cognitoUserPoolsAuthResultPayload(result cognitoUserPoolsAuthResult) map[string]any {
	out := map[string]any{
		"AccessToken": result.AccessToken,
		"IdToken":     result.IDToken,
		"ExpiresIn":   result.ExpiresIn,
		"TokenType":   result.TokenType,
	}
	if result.RefreshToken != "" {
		out["RefreshToken"] = result.RefreshToken
	}
	return out
}

func cognitoUserPoolsAuthFlowPayload(result cognitoUserPoolsAuthFlowResult) map[string]any {
	out := map[string]any{}
	if result.AuthenticationResult != nil {
		out["AuthenticationResult"] = cognitoUserPoolsAuthResultPayload(*result.AuthenticationResult)
	}
	if result.ChallengeName != "" {
		out["ChallengeName"] = result.ChallengeName
	}
	if len(result.ChallengeParameters) > 0 {
		out["ChallengeParameters"] = cloneStringMap(result.ChallengeParameters)
	}
	if result.Session != "" {
		out["Session"] = result.Session
	}
	return out
}

func cognitoUserPoolsUserPoolMFAConfigPayload(record cognitoUserPoolsUserPoolRecord) map[string]any {
	out := map[string]any{
		"MfaConfiguration": record.MFAConfiguration,
		"SoftwareTokenMfaConfiguration": map[string]any{
			"Enabled": record.SoftwareTokenMFAEnabled,
		},
	}
	if record.WebAuthnRelyingPartyID != "" || record.WebAuthnUserVerification != "" {
		out["WebAuthnConfiguration"] = map[string]any{
			"RelyingPartyId":   record.WebAuthnRelyingPartyID,
			"UserVerification": record.WebAuthnUserVerification,
		}
	}
	return out
}

func cognitoUserPoolsWebAuthnCredentialPayload(record cognitoUserPoolsWebAuthnCredentialRecord) map[string]any {
	return map[string]any{
		"CredentialId": record.CredentialID,
		"FriendlyName": record.FriendlyName,
		"CreatedAt":    float64(record.CreatedAt.Unix()),
	}
}

func cognitoUserPoolsWebAuthnCredentialsPayload(records []cognitoUserPoolsWebAuthnCredentialRecord) []map[string]any {
	if len(records) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, cognitoUserPoolsWebAuthnCredentialPayload(record))
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i]["CredentialId"].(string)
		right, _ := out[j]["CredentialId"].(string)
		return left < right
	})
	return out
}
