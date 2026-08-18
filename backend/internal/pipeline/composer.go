package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"whwriter/backend/internal/agent"
	"whwriter/backend/internal/model"
)

func (p *Pipeline) composeChapterContext(book *model.Book, chapterNumber uint, memo, userInput string, runType model.ChapterWriteRunType, original *model.Chapter) (*agent.ComposeOutput, error) {
	composerAny, ok := p.registry.Get("composer")
	if !ok {
		return nil, fmt.Errorf("composer agent not found")
	}
	composer, ok := composerAny.(*agent.ComposerAgent)
	if !ok {
		return nil, fmt.Errorf("invalid composer agent")
	}

	foundations, err := p.truth.ListFoundations(book.ID)
	if err != nil {
		return nil, fmt.Errorf("list foundations: %w", err)
	}
	bookState, err := p.truth.GetBookState(book.ID)
	if err != nil {
		return nil, fmt.Errorf("get book state: %w", err)
	}
	characters, err := p.truth.GetCharacters(book.ID)
	if err != nil {
		return nil, fmt.Errorf("get characters: %w", err)
	}
	facts, err := p.truth.GetFacts(book.ID)
	if err != nil {
		return nil, fmt.Errorf("get facts: %w", err)
	}
	hooks, err := p.truth.GetHooks(book.ID)
	if err != nil {
		return nil, fmt.Errorf("get hooks: %w", err)
	}
	summaries, err := p.truth.GetChapterSummaries(book.ID)
	if err != nil {
		return nil, fmt.Errorf("get chapter summaries: %w", err)
	}
	radarProfiles, radarRules := p.loadRadarContext(book)

	var previous *model.Chapter
	if chapterNumber > 1 {
		if ch, err := p.truth.GetChapter(book.ID, chapterNumber-1); err == nil {
			previous = ch
		}
	}

	output := composer.Compose(agent.ComposeInput{
		Book:            book,
		ChapterNumber:   chapterNumber,
		Memo:            memo,
		UserInput:       userInput,
		Foundations:     foundations,
		BookState:       bookState,
		Characters:      characters,
		Facts:           facts,
		Hooks:           hooks,
		Summaries:       summaries,
		RadarProfiles:   radarProfiles,
		RadarRules:      radarRules,
		PreviousChapter: previous,
		OriginalChapter: original,
		RunType:         string(runType),
	})
	return &output, nil
}

func (p *Pipeline) loadRadarContext(book *model.Book) ([]model.RadarTaxonomyProfile, []model.RadarRule) {
	if p.radar == nil || book == nil {
		return nil, nil
	}
	platform := model.RadarPlatformFanqie
	if !strings.Contains(strings.ToLower(book.Platform.Name), "番茄") && !strings.Contains(strings.ToLower(book.Platform.Name), "fanqie") {
		return nil, nil
	}
	setting, _ := p.radar.GetBookSetting(book.ID)
	var tags []string
	if setting != nil {
		if strings.TrimSpace(setting.Platform) != "" {
			platform = setting.Platform
		}
		tags = parseRadarTags(setting.TagsJSON)
		if strings.TrimSpace(setting.Category) != "" {
			tags = append([]string{strings.TrimSpace(setting.Category)}, tags...)
		}
	}
	tags = uniqueRadarTags(tags)
	if len(tags) == 0 {
		return nil, nil
	}
	profiles, _ := p.radar.ListActiveTaxonomyProfiles(platform, "", tags)
	rules, _ := p.radar.ListActiveRules(platform, "", tags, 30)
	return profiles, rules
}

func parseRadarTags(raw string) []string {
	var tags []string
	if strings.TrimSpace(raw) == "" {
		return tags
	}
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

func uniqueRadarTags(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == "other_pending" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func marshalArtifactPayload(v any) string {
	payload, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(payload)
}
