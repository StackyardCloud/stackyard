package server

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The Lex V2 models API above is mostly canned responses. These records keep
// the parts the runtime (lexruntimev2) needs to hold a conversation: locales,
// aliases, intents with their sample utterances, and slots with their prompts.
// ponytail: one model per bot+locale, shared by every bot version and alias;
// track per-version snapshots if a test needs DRAFT and v1 to differ.

const lexV2TestAliasID = "TSTALIASID" // every bot has this alias, pointing at DRAFT

type lexV2Locale struct {
	Threshold float64
	Intents   map[string]*lexV2Intent // by intentId
}

type lexV2Intent struct {
	ID         string
	Name       string
	Utterances []string
	SlotOrder  []string // slot IDs in elicitation order
	Slots      map[string]*lexV2Slot
	created    int64
}

type lexV2Slot struct {
	ID       string
	Name     string
	Required bool
	Prompt   string
}

// assignLexV2CreatedIDLocked gives Create* calls a fresh ID unless the caller
// already named one, so two intents (or slots, or aliases) never collide.
func (s *lexV2Store) assignLexV2CreatedIDLocked(action string, merged map[string]any) {
	field, prefix := "", ""
	switch action {
	case "CreateIntent":
		field, prefix = "intentId", "INT"
	case "CreateSlot":
		field, prefix = "slotId", "SLT"
	case "CreateBotAlias":
		field, prefix = "botAliasId", "ALS"
	default:
		return
	}
	if lexV2String(merged, []string{field}, "") != "" {
		return
	}
	merged[field] = fmt.Sprintf("%s%07d", prefix, s.nextID)
	s.nextID++
}

// recordLexV2ModelLocked keeps the runtime-relevant model in step with the
// models API calls.
func (s *lexV2Store) recordLexV2ModelLocked(action string, merged map[string]any) {
	botID := lexV2String(merged, []string{"botId"}, "A1B2C3D4E5")
	localeID := lexV2String(merged, []string{"localeId"}, "en_US")
	switch action {
	case "UpdateBot":
		setLexV2BotTTL(s.bots[botID], merged)
	case "DeleteBot":
		delete(s.bots, botID)
		delete(s.locales, botID)
		delete(s.aliases, botID)
		s.dropLexRuntimeSessionsLocked(botID)
	case "CreateBotLocale", "UpdateBotLocale":
		locale := s.ensureLexV2LocaleLocked(botID, localeID)
		if threshold, ok := lexV2Number(merged["nluIntentConfidenceThreshold"]); ok {
			locale.Threshold = threshold
		}
	case "DeleteBotLocale":
		delete(s.locales[botID], localeID)
	case "CreateBotAlias", "UpdateBotAlias":
		if s.aliases[botID] == nil {
			s.aliases[botID] = map[string]string{}
		}
		aliasID := lexV2String(merged, []string{"botAliasId"}, "")
		s.aliases[botID][aliasID] = lexV2String(merged, []string{"botAliasName"}, aliasID)
	case "DeleteBotAlias":
		delete(s.aliases[botID], lexV2String(merged, []string{"botAliasId"}, ""))
	case "CreateIntent", "UpdateIntent":
		s.recordLexV2IntentLocked(s.ensureLexV2LocaleLocked(botID, localeID), merged)
	case "DeleteIntent":
		if locale := s.locales[botID][localeID]; locale != nil {
			delete(locale.Intents, lexV2String(merged, []string{"intentId"}, ""))
		}
	case "CreateSlot", "UpdateSlot":
		if intent := s.lexV2IntentLocked(botID, localeID, merged); intent != nil {
			recordLexV2Slot(intent, merged)
		}
	case "DeleteSlot":
		if intent := s.lexV2IntentLocked(botID, localeID, merged); intent != nil {
			slotID := lexV2String(merged, []string{"slotId"}, "")
			delete(intent.Slots, slotID)
			intent.SlotOrder = slices.DeleteFunc(intent.SlotOrder, func(id string) bool { return id == slotID })
		}
	}
}

func (s *lexV2Store) ensureLexV2LocaleLocked(botID, localeID string) *lexV2Locale {
	if s.locales[botID] == nil {
		s.locales[botID] = map[string]*lexV2Locale{}
	}
	locale := s.locales[botID][localeID]
	if locale == nil {
		locale = &lexV2Locale{Threshold: 0.4, Intents: map[string]*lexV2Intent{}}
		s.locales[botID][localeID] = locale
	}
	return locale
}

func (s *lexV2Store) lexV2IntentLocked(botID, localeID string, merged map[string]any) *lexV2Intent {
	locale := s.locales[botID][localeID]
	if locale == nil {
		return nil
	}
	return locale.Intents[lexV2String(merged, []string{"intentId"}, "")]
}

func (s *lexV2Store) recordLexV2IntentLocked(locale *lexV2Locale, merged map[string]any) {
	intentID := lexV2String(merged, []string{"intentId"}, "")
	intent := locale.Intents[intentID]
	if intent == nil {
		intent = &lexV2Intent{ID: intentID, Slots: map[string]*lexV2Slot{}, created: s.nextID}
		s.nextID++
		locale.Intents[intentID] = intent
	}
	if name := lexV2String(merged, []string{"intentName"}, ""); name != "" {
		intent.Name = name
	}
	if raw, ok := merged["sampleUtterances"].([]any); ok {
		intent.Utterances = intent.Utterances[:0]
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				if u := lexV2String(m, []string{"utterance"}, ""); u != "" {
					intent.Utterances = append(intent.Utterances, u)
				}
			}
		}
	}
	if raw, ok := merged["slotPriorities"].([]any); ok {
		type prio struct {
			n  float64
			id string
		}
		var prios []prio
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				n, _ := lexV2Number(m["priority"])
				prios = append(prios, prio{n, lexV2String(m, []string{"slotId"}, "")})
			}
		}
		sort.SliceStable(prios, func(i, j int) bool { return prios[i].n < prios[j].n })
		order := make([]string, 0, len(prios))
		for _, p := range prios {
			order = append(order, p.id)
		}
		// Slots created before the priorities were set keep their place at the end.
		for _, id := range intent.SlotOrder {
			if !slices.Contains(order, id) {
				order = append(order, id)
			}
		}
		intent.SlotOrder = order
	}
}

func recordLexV2Slot(intent *lexV2Intent, merged map[string]any) {
	slotID := lexV2String(merged, []string{"slotId"}, "")
	slot := intent.Slots[slotID]
	if slot == nil {
		slot = &lexV2Slot{ID: slotID}
		intent.Slots[slotID] = slot
	}
	if !slices.Contains(intent.SlotOrder, slotID) {
		intent.SlotOrder = append(intent.SlotOrder, slotID)
	}
	if name := lexV2String(merged, []string{"slotName"}, ""); name != "" {
		slot.Name = name
	}
	setting, _ := merged["valueElicitationSetting"].(map[string]any)
	if setting == nil {
		return
	}
	slot.Required = lexV2String(setting, []string{"slotConstraint"}, "") == "Required"
	spec, _ := setting["promptSpecification"].(map[string]any)
	groups, _ := spec["messageGroups"].([]any)
	if len(groups) == 0 {
		return
	}
	group, _ := groups[0].(map[string]any)
	message, _ := group["message"].(map[string]any)
	plain, _ := message["plainTextMessage"].(map[string]any)
	slot.Prompt = lexV2String(plain, []string{"value"}, "")
}

func lexV2Number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case interface{ String() string }: // json.Number
		f, err := strconv.ParseFloat(n.String(), 64)
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	}
	return 0, false
}

func setLexV2BotTTL(bot map[string]any, merged map[string]any) {
	if ttl, ok := lexV2Number(merged["idleSessionTTLInSeconds"]); ok && ttl > 0 {
		bot["idleSessionTTLInSeconds"] = ttl
	}
}
