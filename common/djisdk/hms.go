package djisdk

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zeromicro/go-zero/core/logx"
)

//go:embed hms.json
var embeddedHmsDictionary []byte

var hmsReplacementPattern = regexp.MustCompile(`%(?:component_index|index|battery_index|dock_cover_index|charging_rod_index|alarmid|gimbal_index|lidar_index|lte_index|[0-9]+\$[sdf]|[sd])`)
var hmsPlaceholderPattern = regexp.MustCompile(`%(?:[0-9]+\$)?(?:\.[0-9]+)?[A-Za-z_]+`)

// HmsConfig controls HMS localization. The embedded dictionary is used when DictionaryPath is empty.
type HmsConfig struct {
	Language       string `json:",default=zh"`
	DictionaryPath string `json:",optional"`
}

// HmsResolveResult describes the dictionary entry and rendered HMS message selected for an item.
type HmsResolveResult struct {
	Key      string
	Language string
	Template string
	Message  string
}

// HmsResolver resolves typed HMS items against an immutable localization dictionary.
type HmsResolver struct {
	language   string
	dictionary map[string]map[string]string
}

// NewHmsResolver loads and validates the configured HMS dictionary.
func NewHmsResolver(cfg HmsConfig) (*HmsResolver, error) {
	data := embeddedHmsDictionary
	if cfg.DictionaryPath != "" {
		var err error
		data, err = os.ReadFile(cfg.DictionaryPath)
		if err != nil {
			return nil, fmt.Errorf("read HMS dictionary %q: %w", cfg.DictionaryPath, err)
		}
	}

	var dictionary map[string]map[string]string
	if err := json.Unmarshal(data, &dictionary); err != nil {
		return nil, fmt.Errorf("decode HMS dictionary: %w", err)
	}
	if len(dictionary) == 0 {
		return nil, fmt.Errorf("decode HMS dictionary: dictionary is empty")
	}

	language := strings.ToLower(strings.TrimSpace(cfg.Language))
	if language == "" {
		language = "zh"
	}
	return &HmsResolver{language: language, dictionary: dictionary}, nil
}

// MustNewHmsResolver loads an HMS dictionary and terminates startup when it is invalid.
func MustNewHmsResolver(cfg HmsConfig) *HmsResolver {
	resolver, err := NewHmsResolver(cfg)
	logx.Must(err)
	return resolver
}

// Resolve selects and renders the best localized HMS message for item.
func (r *HmsResolver) Resolve(item HmsItem) HmsResolveResult {
	key, translations := r.findTranslations(item)
	if translations == nil {
		return HmsResolveResult{
			Language: r.language,
			Message:  unknownHmsMessage(r.language, item.Code),
		}
	}

	template := strings.TrimSpace(translations[r.language])
	if template == "" {
		return HmsResolveResult{
			Key:      key,
			Language: r.language,
			Message:  unknownHmsMessage(r.language, item.Code),
		}
	}
	message, unresolved := renderHmsTemplate(key, template, item.Args, r.language)
	if len(unresolved) > 0 {
		logx.Errorf("[dji-sdk] unresolved HMS placeholders: key=%s code=%s placeholders=%s", key, item.Code, strings.Join(unresolved, ","))
	}
	return HmsResolveResult{Key: key, Language: r.language, Template: template, Message: message}
}

func (r *HmsResolver) findTranslations(item HmsItem) (string, map[string]string) {
	deviceType, err := ParseDeviceType(item.DeviceType)
	if err != nil {
		return "", nil
	}
	prefix := hmsTipPrefix(deviceType.Domain)
	if prefix == "" {
		return "", nil
	}
	baseKey := prefix + "_tip_" + item.Code
	if deviceType.Domain == DeviceDomainAircraft && item.InTheSky == 1 {
		if translations, ok := r.dictionary[baseKey+"_in_the_sky"]; ok {
			return baseKey + "_in_the_sky", translations
		}
	}
	if translations, ok := r.dictionary[baseKey]; ok {
		return baseKey, translations
	}
	return "", nil
}

func hmsTipPrefix(domain DeviceDomain) string {
	switch domain {
	case DeviceDomainAircraft:
		return "fpv"
	case DeviceDomainDock:
		return "dock"
	default:
		return ""
	}
}

func renderHmsTemplate(_ string, template string, args HmsArgs, language string) (string, []string) {
	replacements := make(map[string]string)
	if value, ok := args.Int("component_index"); ok {
		component := strconv.Itoa(value + 1)
		replacements["%component_index"] = component
	}
	if value, ok := args.Int("sensor_index"); ok {
		index := strconv.Itoa(value + 1)
		replacements["%index"] = index
		replacements["%battery_index"] = localizedSide(value, language)
		replacements["%dock_cover_index"] = localizedSide(value, language)
		if direction, ok := localizedDirection(value, language); ok {
			replacements["%charging_rod_index"] = direction
		}
	}
	if value, ok := args.String("alarmid"); ok {
		replacements["%alarmid"] = value
	}
	if value, ok := args.Int("gimbal_index"); ok {
		replacements["%gimbal_index"] = strconv.Itoa(value)
	}
	if value, ok := args.Int("lidar_index"); ok {
		replacements["%lidar_index"] = strconv.Itoa(value)
	}
	if value, ok := args.Int("lte_index"); ok {
		replacements["%lte_index"] = strconv.Itoa(value)
	}

	message := hmsReplacementPattern.ReplaceAllStringFunc(template, func(placeholder string) string {
		if replacement, ok := replacements[placeholder]; ok {
			return replacement
		}
		return placeholder
	})
	unresolved := hmsPlaceholderPattern.FindAllString(message, -1)
	return message, slice.Unique(unresolved)
}

func localizedSide(index int, language string) string {
	if language == "zh" {
		if index == 0 {
			return "左"
		}
		return "右"
	}
	if index == 0 {
		return "left"
	}
	return "right"
}

func localizedDirection(index int, language string) (string, bool) {
	if index < 0 || index > 3 {
		return "", false
	}
	if language == "zh" {
		return []string{"前", "后", "左", "右"}[index], true
	}
	return []string{"front", "rear", "left", "right"}[index], true
}

func unknownHmsMessage(language, code string) string {
	if language == "zh" || language == "" {
		return fmt.Sprintf("未知 HMS 告警（%s）", code)
	}
	return fmt.Sprintf("Unknown HMS alert (%s)", code)
}
