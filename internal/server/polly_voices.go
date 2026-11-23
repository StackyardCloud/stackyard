package server

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// pollyVoice is one built-in Polly voice.
type pollyVoice struct {
	id, gender, lang string
	engines          []string
	additional       []string
}

func (v pollyVoice) supports(engine string) bool {
	for _, e := range v.engines {
		if e == engine {
			return true
		}
	}
	return false
}

func (v pollyVoice) speaks(lang string) bool {
	if lang == v.lang {
		return true
	}
	for _, l := range v.additional {
		if l == lang {
			return true
		}
	}
	return false
}

var pollyLanguageNames = map[string]string{
	"arb": "Arabic", "cmn-CN": "Chinese Mandarin", "cy-GB": "Welsh", "da-DK": "Danish", "de-DE": "German",
	"en-AU": "Australian English", "en-GB": "British English", "en-GB-WLS": "Welsh English", "en-IN": "Indian English",
	"en-US": "US English", "es-ES": "Castilian Spanish", "es-MX": "Mexican Spanish", "es-US": "US Spanish",
	"fr-CA": "Canadian French", "fr-FR": "French", "is-IS": "Icelandic", "it-IT": "Italian", "ja-JP": "Japanese",
	"hi-IN": "Hindi", "ko-KR": "Korean", "nb-NO": "Norwegian", "nl-NL": "Dutch", "pl-PL": "Polish",
	"pt-BR": "Brazilian Portuguese", "pt-PT": "Portuguese", "ro-RO": "Romanian", "ru-RU": "Russian", "sv-SE": "Swedish",
	"tr-TR": "Turkish", "en-NZ": "New Zealand English", "en-ZA": "South African English", "ca-ES": "Catalan",
	"de-AT": "Austrian German", "yue-CN": "Chinese Cantonese", "ar-AE": "Gulf Arabic", "fi-FI": "Finnish",
	"en-IE": "Irish English", "nl-BE": "Belgian Dutch", "fr-BE": "Belgian French", "cs-CZ": "Czech",
	"de-CH": "Swiss Standard German", "en-SG": "Singaporean English",
}

// pollyVoiceTable is "Id Gender LanguageCode engines" per line; engines are
// s(tandard) n(eural) l(ong-form) g(enerative).
// ponytail: engine support for the newest voices is approximate; refresh from
// a real DescribeVoices call if exact parity matters.
const pollyVoiceTable = `
Aditi F en-IN s|Amy F en-GB sng|Astrid F sv-SE s|Bianca F it-IT sng|Brian M en-GB sng|Camila F pt-BR sng
Carla F it-IT s|Carmen F ro-RO s|Celine F fr-FR s|Chantal F fr-CA s|Conchita F es-ES s|Cristiano M pt-PT s
Dora F is-IS s|Emma F en-GB sn|Enrique M es-ES s|Ewa F pl-PL s|Filiz F tr-TR s|Gabrielle F fr-CA n
Geraint M en-GB-WLS s|Giorgio M it-IT s|Gwyneth F cy-GB s|Hans M de-DE s|Ines F pt-PT sn|Ivy F en-US sn
Jacek M pl-PL s|Jan M pl-PL s|Joanna F en-US sng|Joey M en-US sn|Justin M en-US sn|Karl M is-IS s
Kendra F en-US sn|Kevin M en-US n|Kimberly F en-US sn|Lea F fr-FR sng|Liv F nb-NO s|Lotte F nl-NL s
Lucia F es-ES sng|Lupe F es-US sng|Mads M da-DK s|Maja F pl-PL s|Marlene F de-DE s|Mathieu M fr-FR s
Matthew M en-US sng|Maxim M ru-RU s|Mia F es-MX sng|Miguel M es-US s|Mizuki F ja-JP s|Naja F da-DK s
Nicole F en-AU s|Olivia F en-AU ng|Penelope F es-US s|Raveena F en-IN s|Ricardo M pt-BR s|Ruben M nl-NL s
Russell M en-AU s|Salli F en-US sng|Seoyeon F ko-KR sng|Takumi M ja-JP sn|Tatyana F ru-RU s|Vicki F de-DE sng
Vitoria F pt-BR sn|Zeina F arb s|Zhiyu F cmn-CN sn|Aria F en-NZ n|Ayanda F en-ZA ng|Arlet F ca-ES n
Hannah F de-AT n|Arthur M en-GB n|Daniel M de-DE ng|Liam M fr-CA n|Pedro M es-US ng|Kajal F en-IN ng
Hiujin F yue-CN n|Laura F nl-NL n|Elin F sv-SE n|Ida F nb-NO n|Suvi F fi-FI n|Ola F pl-PL n
Hala F ar-AE n|Andres M es-MX ng|Sergio M es-ES ng|Remi M fr-FR ng|Adriano M it-IT n|Thiago M pt-BR n
Ruth F en-US nlg|Stephen M en-US ng|Kazuha F ja-JP n|Tomoko F ja-JP n|Niamh F en-IE n|Sofie F da-DK n
Lisa F nl-BE n|Isabelle F fr-BE n|Zayd M ar-AE n|Danielle F en-US nlg|Gregory M en-US nl|Burcu F tr-TR n
Jitka F cs-CZ n|Sabrina F de-CH n|Jasmine F en-SG n|Jihye F ko-KR n|Ambre F fr-FR g|Beatrice F it-IT g
Florian M fr-FR g|Lennart M de-DE g|Lorenzo M it-IT g|Tiffany F en-US g|Arjun M en-IN g|Carlos M es-US g
Carolina F es-ES g|Curtis M en-US g|Hyunduk M ko-KR g|Kenneth M en-US g|Kiara F en-IN g|Leo M en-US g
Nate M en-US g|Tina F de-DE g`

var pollyVoices, pollyVoiceByID = func() ([]pollyVoice, map[string]pollyVoice) {
	names := map[byte]string{'s': "standard", 'n': "neural", 'l': "long-form", 'g': "generative"}
	var list []pollyVoice
	byID := map[string]pollyVoice{}
	for _, entry := range strings.FieldsFunc(pollyVoiceTable, func(r rune) bool { return r == '|' || r == '\n' }) {
		f := strings.Fields(entry)
		v := pollyVoice{id: f[0], gender: map[string]string{"F": "Female", "M": "Male"}[f[1]], lang: f[2]}
		for i := 0; i < len(f[3]); i++ {
			v.engines = append(v.engines, names[f[3][i]])
		}
		if v.id == "Aditi" || v.id == "Kajal" {
			v.additional = []string{"hi-IN"}
		}
		list = append(list, v)
		byID[v.id] = v
	}
	sort.Slice(list, func(i, j int) bool { return list[i].id < list[j].id })
	return list, byID
}()

func (v pollyVoice) describe() map[string]any {
	out := map[string]any{
		"Id": v.id, "Name": v.id, "Gender": v.gender, "LanguageCode": v.lang,
		"LanguageName": pollyLanguageNames[v.lang], "SupportedEngines": pollyAnyList(v.engines),
	}
	if len(v.additional) > 0 {
		out["AdditionalLanguageCodes"] = pollyAnyList(v.additional)
	}
	return out
}

func pollyAnyList(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

const pollyVoicePageSize = 100

// pollyDescribeVoices lists built-in voices filtered by engine and language.
func (s *Server) pollyDescribeVoices(q url.Values) (map[string]any, *pollyError) {
	engine, lang := q.Get("Engine"), q.Get("LanguageCode")
	if engine != "" {
		if err := pollyCheck("Engine", "Engine", engine); err != nil {
			return nil, err
		}
	}
	if lang != "" {
		if err := pollyCheck("LanguageCode", "LanguageCode", lang); err != nil {
			return nil, err
		}
	}
	include := q.Get("IncludeAdditionalLanguageCodes")
	if include != "" && include != "true" && include != "false" {
		return nil, pollyValidation("IncludeAdditionalLanguageCodes must be true or false.")
	}
	start, err := pollyToken(q.Get("NextToken"))
	if err != nil {
		return nil, err
	}
	var matched []any
	for _, v := range pollyVoices {
		if engine != "" && !v.supports(engine) {
			continue
		}
		if lang != "" && v.lang != lang && !(include == "true" && v.speaks(lang)) {
			continue
		}
		matched = append(matched, v.describe())
	}
	return pollyPage("Voices", matched, start, pollyVoicePageSize)
}

// pollyPage slices items at an offset token.
func pollyPage(key string, items []any, start, size int) (map[string]any, *pollyError) {
	if start > len(items) {
		return nil, pollyBad("InvalidNextTokenException", "The NextToken is invalid.")
	}
	end := min(start+size, len(items))
	out := map[string]any{key: append([]any{}, items[start:end]...)}
	if end < len(items) {
		out["NextToken"] = strconv.Itoa(end)
	}
	return out, nil
}
