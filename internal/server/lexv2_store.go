package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	lexV2DefaultRegion    = "us-east-1"
	lexV2DefaultAccountID = "123456789012"
)

type lexV2Store struct {
	mu sync.Mutex

	nextID int64

	bots             map[string]map[string]any
	tags             map[string]map[string]string
	resourcePolicies map[string]map[string]any
}

func newLexV2Store() *lexV2Store {
	now := time.Now().UTC().Format(time.RFC3339)
	seedBotID := "A1B2C3D4E5"
	seedBotName := "stackyard-lex-bot"
	s := &lexV2Store{
		nextID: 2,
		bots: map[string]map[string]any{
			seedBotID: {
				"botId":                   seedBotID,
				"botName":                 seedBotName,
				"description":             "stackyard lexv2 bot",
				"botStatus":               "Available",
				"botType":                 "Bot",
				"idleSessionTTLInSeconds": float64(300),
				"lastUpdatedDateTime":     now,
				"creationDateTime":        now,
				"botArn":                  lexV2BotARN(seedBotID),
				"roleArn":                 "arn:aws:iam::123456789012:role/stackyard-lex-role",
			},
		},
		tags:             map[string]map[string]string{},
		resourcePolicies: map[string]map[string]any{},
	}
	s.tags[lexV2BotARN(seedBotID)] = map[string]string{"stackyard": "true", "env": "coverage"}
	return s
}

func (s *lexV2Store) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	merged := lexV2CloneMap(payload)
	for k, v := range pathParams {
		merged[k] = v
	}
	for k, vals := range query {
		if len(vals) == 0 {
			continue
		}
		merged[k] = vals[len(vals)-1]
	}

	botID := lexV2String(merged, []string{"botId"}, "A1B2C3D4E5")
	botName := lexV2String(merged, []string{"botName", "name"}, "stackyard-lex-bot")
	botVersion := lexV2String(merged, []string{"botVersion"}, "DRAFT")
	localeID := lexV2String(merged, []string{"localeId"}, "en_US")
	intentID := lexV2String(merged, []string{"intentId"}, "I1N2T3E4N5")
	slotID := lexV2String(merged, []string{"slotId"}, "S1L2O3T4ID")
	slotTypeID := lexV2String(merged, []string{"slotTypeId"}, "SL0TTYPE01")
	resourceARN := lexV2String(merged, []string{"resourceARN", "resourceArn"}, lexV2BotAliasARN(botID, "BOTALIAS1"))
	statementID := lexV2String(merged, []string{"statementId"}, "statement-000001")

	bot := s.ensureBotLocked(botID, botName, now)
	s.ensureTagMapLocked(resourceARN)

	s.applyTagMutationsLocked(action, merged, resourceARN)
	s.applyPolicyMutationsLocked(action, merged, resourceARN, statementID, now)

	summary := map[string]any{
		"botId":               bot["botId"],
		"botName":             bot["botName"],
		"description":         bot["description"],
		"botStatus":           bot["botStatus"],
		"lastUpdatedDateTime": bot["lastUpdatedDateTime"],
		"creationDateTime":    bot["creationDateTime"],
	}

	switch action {
	case "CreateBot":
		createdID := fmt.Sprintf("BOT%08d", s.nextID)
		s.nextID++
		createdName := lexV2String(merged, []string{"botName", "name"}, "stackyard-lex-bot")
		created := s.ensureBotLocked(createdID, createdName, now)
		created["description"] = lexV2String(merged, []string{"description"}, "stackyard lexv2 bot")
		created["botStatus"] = "Creating"
		created["lastUpdatedDateTime"] = now
		return map[string]any{"botId": createdID, "botName": created["botName"], "botStatus": created["botStatus"], "creationDateTime": now}
	case "DescribeBot":
		return lexV2CloneMap(bot)
	case "UpdateBot":
		if name := lexV2String(merged, []string{"botName", "name"}, ""); name != "" {
			bot["botName"] = name
		}
		if desc := lexV2String(merged, []string{"description"}, ""); desc != "" {
			bot["description"] = desc
		}
		bot["botStatus"] = "Available"
		bot["lastUpdatedDateTime"] = now
		return map[string]any{"botId": bot["botId"], "botName": bot["botName"], "botStatus": bot["botStatus"], "lastUpdatedDateTime": now}
	case "DeleteBot", "DeleteBotReplica", "DeleteBotVersion", "DeleteBotAlias", "DeleteBotLocale", "DeleteIntent", "DeleteSlot", "DeleteSlotType", "DeleteCustomVocabulary", "DeleteUtterances", "DeleteExport", "DeleteImport", "DeleteTestSet", "DeleteResourcePolicy":
		return map[string]any{}
	case "ListBots":
		return map[string]any{"botSummaries": s.listBotSummariesLocked(), "nextToken": ""}
	case "ListBotVersions":
		return map[string]any{"botVersionSummaries": []any{map[string]any{"botName": bot["botName"], "botVersion": botVersion, "description": bot["description"], "botStatus": "Available"}}, "nextToken": ""}
	case "CreateBotVersion":
		return map[string]any{"botId": botID, "botVersion": "1", "botStatus": "Available", "creationDateTime": now}
	case "DescribeBotVersion":
		return map[string]any{"botId": botID, "botVersion": botVersion, "botName": bot["botName"], "botStatus": "Available", "creationDateTime": bot["creationDateTime"], "lastUpdatedDateTime": now}
	case "CreateBotAlias", "UpdateBotAlias", "DescribeBotAlias":
		aliasID := lexV2String(merged, []string{"botAliasId"}, "BOTALIAS1")
		aliasName := lexV2String(merged, []string{"botAliasName", "name"}, "stackyard-alias")
		return map[string]any{"botAliasId": aliasID, "botAliasName": aliasName, "botAliasStatus": "Available", "botVersion": botVersion, "botId": botID, "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListBotAliases":
		return map[string]any{"botAliasSummaries": []any{map[string]any{"botAliasId": "BOTALIAS1", "botAliasName": "stackyard-alias", "botAliasStatus": "Available", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "ListBotAliasReplicas":
		return map[string]any{"botAliasReplicaSummaries": []any{map[string]any{"botAliasId": "BOTALIAS1", "botAliasStatus": "Available", "replicaRegion": lexV2String(merged, []string{"replicaRegion"}, "us-west-2")}}, "nextToken": ""}
	case "ListBotReplicas":
		return map[string]any{"botReplicaSummaries": []any{map[string]any{"botId": botID, "replicaRegion": "us-west-2", "botReplicaStatus": "Available"}}, "nextToken": ""}
	case "ListBotVersionReplicas":
		return map[string]any{"botVersionReplicaSummaries": []any{map[string]any{"botVersion": botVersion, "replicaRegion": "us-west-2", "botVersionReplicaStatus": "Available"}}, "nextToken": ""}
	case "CreateBotReplica", "DescribeBotReplica":
		return map[string]any{"botId": botID, "replicaRegion": lexV2String(merged, []string{"replicaRegion"}, "us-west-2"), "botReplicaStatus": "Available", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "CreateBotLocale", "DescribeBotLocale", "UpdateBotLocale":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "localeName": "English (US)", "botLocaleStatus": "Built", "nluIntentConfidenceThreshold": 0.4, "creationDateTime": now, "lastUpdatedDateTime": now}
	case "BuildBotLocale":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "botLocaleStatus": "Built"}
	case "ListBotLocales":
		return map[string]any{"botLocaleSummaries": []any{map[string]any{"localeId": localeID, "localeName": "English (US)", "botLocaleStatus": "Built", "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "CreateIntent", "DescribeIntent", "UpdateIntent":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "intentId": intentID, "intentName": "stackyard-intent", "description": "stackyard lex intent", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListIntents":
		return map[string]any{"intentSummaries": []any{map[string]any{"intentId": intentID, "intentName": "stackyard-intent", "description": "stackyard lex intent", "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "CreateSlot", "DescribeSlot", "UpdateSlot":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "intentId": intentID, "slotId": slotID, "slotName": "stackyard-slot", "valueElicitationSetting": map[string]any{}, "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListSlots":
		return map[string]any{"slotSummaries": []any{map[string]any{"slotId": slotID, "slotName": "stackyard-slot", "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "CreateSlotType", "DescribeSlotType", "UpdateSlotType":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "slotTypeId": slotTypeID, "slotTypeName": "stackyard-slot-type", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListSlotTypes":
		return map[string]any{"slotTypeSummaries": []any{map[string]any{"slotTypeId": slotTypeID, "slotTypeName": "stackyard-slot-type", "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "ListBuiltInIntents":
		return map[string]any{"builtInIntentSummaries": []any{map[string]any{"intentSignature": "AMAZON.FallbackIntent", "description": "Fallback intent"}}, "nextToken": ""}
	case "ListBuiltInSlotTypes":
		return map[string]any{"builtInSlotTypeSummaries": []any{map[string]any{"slotTypeSignature": "AMAZON.AlphaNumeric", "description": "Alphanumeric slot type"}}, "nextToken": ""}
	case "BatchCreateCustomVocabularyItem", "BatchUpdateCustomVocabularyItem", "BatchDeleteCustomVocabularyItem":
		return map[string]any{"errors": []any{}, "resources": []any{}}
	case "DescribeCustomVocabularyMetadata":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "customVocabularyStatus": "Ready", "lastUpdatedDateTime": now}
	case "ListCustomVocabularyItems":
		return map[string]any{"botId": botID, "botVersion": botVersion, "localeId": localeID, "nextToken": "", "customVocabularyItems": []any{}}
	case "CreateUploadUrl":
		return map[string]any{"uploadUrl": "https://stackyard.local/lexv2/upload", "importId": fmt.Sprintf("IMP%06d", s.nextID)}
	case "CreateExport", "DescribeExport", "UpdateExport":
		exportID := lexV2String(merged, []string{"exportId"}, fmt.Sprintf("EXP%06d", s.nextID))
		return map[string]any{"exportId": exportID, "resourceSpecification": map[string]any{}, "fileFormat": "LexJson", "exportStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListExports":
		return map[string]any{"exportSummaries": []any{map[string]any{"exportId": "EXP000001", "exportStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "StartImport", "DescribeImport":
		importID := lexV2String(merged, []string{"importId"}, fmt.Sprintf("IMP%06d", s.nextID))
		return map[string]any{"importId": importID, "importStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListImports":
		return map[string]any{"importSummaries": []any{map[string]any{"importId": "IMP000001", "importStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "GenerateBotElement", "DescribeBotResourceGeneration", "StartBotResourceGeneration", "ListBotResourceGenerations":
		generationID := lexV2String(merged, []string{"generationId"}, "GEN000001")
		if action == "ListBotResourceGenerations" {
			return map[string]any{"generationSummaries": []any{map[string]any{"generationId": generationID, "generationStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
		}
		return map[string]any{"generationId": generationID, "generationStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "CreateTestSetDiscrepancyReport", "DescribeTestSetDiscrepancyReport":
		reportID := lexV2String(merged, []string{"testSetDiscrepancyReportId"}, "TSDR000001")
		return map[string]any{"testSetDiscrepancyReportId": reportID, "testSetDiscrepancyReportStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "StartTestSetGeneration", "DescribeTestSetGeneration":
		genID := lexV2String(merged, []string{"testSetGenerationId"}, "TSGEN00001")
		return map[string]any{"testSetGenerationId": genID, "testSetGenerationStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "CreateTestSet", "DescribeTestSet", "UpdateTestSet":
		setID := lexV2String(merged, []string{"testSetId"}, "TSET000001")
		return map[string]any{"testSetId": setID, "testSetName": "stackyard-test-set", "description": "stackyard test set", "status": "Ready", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListTestSets":
		return map[string]any{"testSets": []any{map[string]any{"testSetId": "TSET000001", "testSetName": "stackyard-test-set", "status": "Ready", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "ListTestSetRecords":
		return map[string]any{"testSetRecords": []any{}, "nextToken": ""}
	case "StartTestExecution", "DescribeTestExecution":
		execID := lexV2String(merged, []string{"testExecutionId"}, "TEXEC00001")
		return map[string]any{"testExecutionId": execID, "status": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now, "target": map[string]any{"botId": botID, "botAliasId": "BOTALIAS1", "localeId": localeID}}
	case "ListTestExecutions":
		return map[string]any{"testExecutions": []any{map[string]any{"testExecutionId": "TEXEC00001", "status": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "ListTestExecutionResultItems":
		return map[string]any{"testExecutionResults": []any{}, "nextToken": ""}
	case "GetTestExecutionArtifactsUrl":
		return map[string]any{"downloadUrl": "https://stackyard.local/lexv2/test-artifacts", "testExecutionId": lexV2String(merged, []string{"testExecutionId"}, "TEXEC00001")}
	case "StartBotRecommendation", "StopBotRecommendation", "DescribeBotRecommendation", "UpdateBotRecommendation":
		recID := lexV2String(merged, []string{"botRecommendationId"}, "BREC000001")
		return map[string]any{"botRecommendationId": recID, "botRecommendationStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}
	case "ListBotRecommendations":
		return map[string]any{"botRecommendationSummaries": []any{map[string]any{"botRecommendationId": "BREC000001", "botRecommendationStatus": "Completed", "creationDateTime": now, "lastUpdatedDateTime": now}}, "nextToken": ""}
	case "ListRecommendedIntents":
		return map[string]any{"recommendedIntentSummaries": []any{map[string]any{"intentId": intentID, "intentName": "stackyard-intent", "sampleUtterancesCount": 1}}, "nextToken": ""}
	case "ListSessionAnalyticsData":
		return map[string]any{"botId": botID, "sessions": []any{}, "nextToken": ""}
	case "ListSessionMetrics":
		return map[string]any{"botId": botID, "results": []any{}, "nextToken": ""}
	case "ListIntentMetrics", "ListIntentStageMetrics", "ListUtteranceMetrics":
		return map[string]any{"botId": botID, "results": []any{}, "nextToken": ""}
	case "ListIntentPaths", "ListUtteranceAnalyticsData", "ListAggregatedUtterances", "SearchAssociatedTranscripts":
		return map[string]any{"botId": botID, "results": []any{}, "nextToken": ""}
	case "CreateResourcePolicy", "UpdateResourcePolicy", "DescribeResourcePolicy":
		policy := s.ensurePolicyLocked(resourceARN, now)
		return lexV2CloneMap(policy)
	case "CreateResourcePolicyStatement", "DeleteResourcePolicyStatement":
		policy := s.ensurePolicyLocked(resourceARN, now)
		return lexV2CloneMap(policy)
	case "TagResource", "UntagResource":
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": lexV2StringMapToAnyMap(s.ensureTagMapLocked(resourceARN))}
	default:
		if strings.HasPrefix(action, "List") {
			return map[string]any{"items": []any{}, "nextToken": ""}
		}
		if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get") {
			return map[string]any{"summary": summary}
		}
		return map[string]any{}
	}
}

func (s *lexV2Store) ensureBotLocked(botID, botName, now string) map[string]any {
	botID = strings.TrimSpace(botID)
	if botID == "" {
		botID = "A1B2C3D4E5"
	}
	if bot, ok := s.bots[botID]; ok {
		if name := strings.TrimSpace(botName); name != "" {
			if _, exists := bot["botName"]; !exists {
				bot["botName"] = name
			}
		}
		return bot
	}
	if strings.TrimSpace(botName) == "" {
		botName = "stackyard-lex-bot"
	}
	bot := map[string]any{
		"botId":                   botID,
		"botName":                 botName,
		"description":             "stackyard lexv2 bot",
		"botStatus":               "Available",
		"botType":                 "Bot",
		"idleSessionTTLInSeconds": float64(300),
		"creationDateTime":        now,
		"lastUpdatedDateTime":     now,
		"botArn":                  lexV2BotARN(botID),
		"roleArn":                 "arn:aws:iam::123456789012:role/stackyard-lex-role",
	}
	s.bots[botID] = bot
	return bot
}

func (s *lexV2Store) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = lexV2BotAliasARN("A1B2C3D4E5", "BOTALIAS1")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	tags := map[string]string{"stackyard": "true", "env": "coverage"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *lexV2Store) ensurePolicyLocked(resourceARN, now string) map[string]any {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = lexV2BotAliasARN("A1B2C3D4E5", "BOTALIAS1")
	}
	if policy, ok := s.resourcePolicies[resourceARN]; ok {
		policy["lastUpdatedDateTime"] = now
		return policy
	}
	policy := map[string]any{
		"resourceArn":         resourceARN,
		"policy":              `{"Version":"2012-10-17","Statement":[]}`,
		"revisionId":          "1",
		"creationDateTime":    now,
		"lastUpdatedDateTime": now,
	}
	s.resourcePolicies[resourceARN] = policy
	return policy
}

func (s *lexV2Store) applyTagMutationsLocked(action string, payload map[string]any, resourceARN string) {
	tags := s.ensureTagMapLocked(resourceARN)
	switch action {
	case "TagResource":
		for k, v := range lexV2ExtractTags(payload) {
			tags[k] = v
		}
	case "UntagResource":
		for _, key := range lexV2ExtractTagKeys(payload) {
			delete(tags, key)
		}
	}
}

func (s *lexV2Store) applyPolicyMutationsLocked(action string, payload map[string]any, resourceARN, statementID, now string) {
	switch action {
	case "CreateResourcePolicy", "UpdateResourcePolicy", "CreateResourcePolicyStatement", "DeleteResourcePolicyStatement":
		policy := s.ensurePolicyLocked(resourceARN, now)
		if p := lexV2String(payload, []string{"policy"}, ""); p != "" {
			policy["policy"] = p
		}
		if rid := lexV2String(payload, []string{"revisionId"}, ""); rid != "" {
			policy["revisionId"] = rid
		} else {
			policy["revisionId"] = fmt.Sprintf("%d", s.nextID)
			s.nextID++
		}
		policy["statementId"] = statementID
		policy["lastUpdatedDateTime"] = now
	}
}

func (s *lexV2Store) listBotSummariesLocked() []any {
	ids := make([]string, 0, len(s.bots))
	for id := range s.bots {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]any, 0, len(ids))
	for _, id := range ids {
		bot := s.bots[id]
		out = append(out, map[string]any{
			"botId":               bot["botId"],
			"botName":             bot["botName"],
			"description":         bot["description"],
			"botStatus":           bot["botStatus"],
			"creationDateTime":    bot["creationDateTime"],
			"lastUpdatedDateTime": bot["lastUpdatedDateTime"],
		})
	}
	return out
}

func lexV2ExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	tagsValue := lexV2Any(payload, []string{"tags", "Tags"})
	switch tv := tagsValue.(type) {
	case map[string]any:
		for k, v := range tv {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = fmt.Sprint(v)
		}
	case map[string]string:
		for k, v := range tv {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = v
		}
	}
	return out
}

func lexV2ExtractTagKeys(payload map[string]any) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, keyName := range []string{"tagKeys", "TagKeys"} {
		value, ok := payload[keyName]
		if !ok {
			continue
		}
		switch vv := value.(type) {
		case []any:
			for _, item := range vv {
				key := strings.TrimSpace(fmt.Sprint(item))
				if key == "" {
					continue
				}
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, key)
			}
		case []string:
			for _, item := range vv {
				key := strings.TrimSpace(item)
				if key == "" {
					continue
				}
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, key)
			}
		case string:
			for _, item := range strings.Split(vv, ",") {
				key := strings.TrimSpace(item)
				if key == "" {
					continue
				}
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, key)
			}
		}
	}
	return out
}

func lexV2BotARN(botID string) string {
	botID = strings.TrimSpace(botID)
	if botID == "" {
		botID = "A1B2C3D4E5"
	}
	return fmt.Sprintf("arn:aws:lex:%s:%s:bot/%s", lexV2DefaultRegion, lexV2DefaultAccountID, botID)
}

func lexV2BotAliasARN(botID, aliasID string) string {
	botID = strings.TrimSpace(botID)
	aliasID = strings.TrimSpace(aliasID)
	if botID == "" {
		botID = "A1B2C3D4E5"
	}
	if aliasID == "" {
		aliasID = "BOTALIAS1"
	}
	return fmt.Sprintf("arn:aws:lex:%s:%s:bot-alias/%s/%s", lexV2DefaultRegion, lexV2DefaultAccountID, botID, aliasID)
}

func lexV2String(payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(value))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return def
}

func lexV2Any(payload map[string]any, keys []string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func lexV2CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch vv := v.(type) {
		case map[string]any:
			out[k] = lexV2CloneMap(vv)
		case []any:
			arr := make([]any, len(vv))
			for i, item := range vv {
				if m, ok := item.(map[string]any); ok {
					arr[i] = lexV2CloneMap(m)
				} else {
					arr[i] = item
				}
			}
			out[k] = arr
		default:
			out[k] = vv
		}
	}
	return out
}

func lexV2StringMapToAnyMap(in map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
