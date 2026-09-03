package agent

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"whwriter/backend/internal/model"
)

type ComposerAgent struct{}

func NewComposerAgent() *ComposerAgent {
	return &ComposerAgent{}
}

func (a *ComposerAgent) Name() string { return "composer" }

func (a *ComposerAgent) SystemPrompt() string {
	return ""
}

type ContextSource struct {
	Source    string `json:"source"`
	Reason    string `json:"reason"`
	Excerpt   string `json:"excerpt,omitempty"`
	Protected bool   `json:"protected"`
}

type ContextPackage struct {
	ChapterNumber   uint            `json:"chapter_number"`
	SelectedContext []ContextSource `json:"selected_context"`
}

type RuleLayer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Precedence int    `json:"precedence"`
	Scope      string `json:"scope"`
}

type ActiveOverride struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Target string `json:"target"`
	Reason string `json:"reason"`
}

type RuleStack struct {
	Layers          []RuleLayer      `json:"layers"`
	Hard            []string         `json:"hard"`
	Soft            []string         `json:"soft"`
	Diagnostic      []string         `json:"diagnostic"`
	ActiveOverrides []ActiveOverride `json:"active_overrides"`
}

type ChapterTrace struct {
	ChapterNumber     uint     `json:"chapter_number"`
	SelectedSources   []string `json:"selected_sources"`
	ProtectedSources  []string `json:"protected_sources"`
	SupportingSources []string `json:"supporting_sources"`
	EstimatedChars    int      `json:"estimated_chars"`
	Notes             []string `json:"notes,omitempty"`
}

type ComposeInput struct {
	Book               *model.Book
	ChapterNumber      uint
	Memo               string
	UserInput          string
	Foundations        []model.BookFoundation
	BookState          *model.BookState
	Characters         []model.Character
	Facts              []model.Fact
	Hooks              []model.Hook
	Summaries          []model.ChapterSummary
	WikiContext        *model.WikiGraphContext
	RetrievedKnowledge []model.KnowledgeSearchResult
	RadarProfiles      []model.RadarTaxonomyProfile
	RadarRules         []model.RadarRule
	PreviousChapter    *model.Chapter
	OriginalChapter    *model.Chapter
	RunType            string
}

type ComposeOutput struct {
	ContextPackage ContextPackage `json:"context_package"`
	RuleStack      RuleStack      `json:"rule_stack"`
	Trace          ChapterTrace   `json:"trace"`
	ContextText    string         `json:"context_text"`
}

func (a *ComposerAgent) Compose(in ComposeInput) ComposeOutput {
	selected := make([]ContextSource, 0, 16)
	notes := make([]string, 0, 4)
	memoHookIDs := extractHookIDs(in.Memo)

	add := func(source, reason, excerpt string, protected bool, limit int) {
		excerpt = strings.TrimSpace(excerpt)
		if excerpt == "" {
			return
		}
		selected = append(selected, ContextSource{
			Source:    source,
			Reason:    reason,
			Excerpt:   clipComposerText(excerpt, limit),
			Protected: protected,
		})
	}

	add("runtime/chapter_memo", "Planner 产出的本章执行意图，优先于默认卷纲。", in.Memo, true, 4000)
	add("runtime/user_input", "用户输入的本章局部提示，用于在既有剧情约束内调整下一章推进。", in.UserInput, true, 1800)

	for _, foundation := range orderedFoundations(in.Foundations) {
		if !isAlwaysIncludedFoundation(foundation.FileType) {
			continue
		}
		reason := foundationReason(foundation.FileType)
		add("foundation/"+string(foundation.FileType), reason, foundation.Content, isProtectedFoundation(foundation.FileType), 2400)
	}

	if in.BookState != nil {
		add("truth/book_state", "当前可变状态，约束本章地点、目标、冲突和主角状态。", renderBookState(in.BookState), true, 1600)
	}

	hasWikiContext := in.WikiContext != nil && len(in.WikiContext.Entities) > 0
	if hasWikiContext {
		add("wiki/graph_context", "从本章种子实体展开的一跳关系、关联事件与当前有效事实。", RenderWikiGraphContext(in.WikiContext), true, 4800)
	}

	if !hasWikiContext && len(in.Characters) > 0 {
		add("truth/characters", "当前主要人物画像和最近状态，用于防止角色行为漂移。", renderCharacters(in.Characters, in.Memo), true, 3200)
	}

	if !hasWikiContext && len(in.Facts) > 0 {
		add("truth/facts", "长期有效事实，防止设定和关系被重写。", renderFacts(in.Facts, in.Memo), true, 2600)
	}

	if len(in.Hooks) > 0 {
		add("truth/hooks", "活跃伏笔与本章 memo 相关 hook，约束推进、延后和回收。", renderHooks(in.Hooks, memoHookIDs, in.ChapterNumber), true, 3600)
	}

	if len(in.Summaries) > 0 {
		add("truth/recent_summaries", "最近章节摘要，用于保持承接和避免标题/节奏重复。", renderRecentSummaries(in.Summaries, 3), false, 1800)
	}

	if len(in.RetrievedKnowledge) > 0 {
		appendRetrievedKnowledge(&selected, in.RetrievedKnowledge)
	}

	if len(in.RadarProfiles) > 0 {
		for _, profile := range in.RadarProfiles {
			source := "radar/tag_profile/" + strings.TrimSpace(profile.Category)
			if strings.TrimSpace(profile.TagKey) != "" {
				source = "radar/tag_profile/" + profile.TagKey
			}
			add(source, "用户雷达沉淀的番茄官方标签写法画像，用于对齐平台和题材气质。", renderRadarProfile(profile), true, 1400)
		}
	}

	if len(in.RadarRules) > 0 {
		add("radar/rules", "用户雷达提炼的可执行写作规则，按权重和置信度筛选。", renderRadarRules(in.RadarRules), true, 3600)
	}

	if in.PreviousChapter != nil {
		add("chapters/previous_tail", "上一章结尾，保证开章承接。", renderChapterTail(in.PreviousChapter, 700), false, 1400)
	}

	if in.OriginalChapter != nil {
		notes = append(notes, "rewrite_latest")
		add("chapters/original_latest", "重写最后一章时的旧版正文，仅用于对照，不作为不可更改事实。", renderChapterTail(in.OriginalChapter, 2200), true, 2600)
	}

	ruleStack := buildRuleStack(in, memoHookIDs)
	contextPackage := ContextPackage{
		ChapterNumber:   in.ChapterNumber,
		SelectedContext: selected,
	}
	trace := buildTrace(in.ChapterNumber, selected, notes)

	return ComposeOutput{
		ContextPackage: contextPackage,
		RuleStack:      ruleStack,
		Trace:          trace,
		ContextText:    RenderContextPackage(contextPackage, ruleStack),
	}
}

func RenderContextPackage(pkg ContextPackage, stack RuleStack) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# ContextPackage for Chapter %d\n\n", pkg.ChapterNumber))
	b.WriteString("## RuleStack\n")
	if len(stack.Hard) > 0 {
		b.WriteString("- Hard: " + strings.Join(stack.Hard, ", ") + "\n")
	}
	if len(stack.Soft) > 0 {
		b.WriteString("- Soft: " + strings.Join(stack.Soft, ", ") + "\n")
	}
	if len(stack.ActiveOverrides) > 0 {
		b.WriteString("- Active Overrides:\n")
		for _, override := range stack.ActiveOverrides {
			b.WriteString(fmt.Sprintf("  - %s -> %s %s: %s\n", override.From, override.To, override.Target, override.Reason))
		}
	}
	b.WriteString("\n## Selected Context\n")
	for _, source := range pkg.SelectedContext {
		b.WriteString(fmt.Sprintf("\n### %s\n", source.Source))
		b.WriteString("Reason: " + source.Reason + "\n")
		if source.Protected {
			b.WriteString("Tier: protected\n")
		} else {
			b.WriteString("Tier: supporting\n")
		}
		if strings.TrimSpace(source.Excerpt) != "" {
			b.WriteString(source.Excerpt)
			if !strings.HasSuffix(source.Excerpt, "\n") {
				b.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func buildRuleStack(in ComposeInput, memoHookIDs map[string]struct{}) RuleStack {
	overrides := make([]ActiveOverride, 0, 2+len(memoHookIDs))
	if strings.TrimSpace(in.UserInput) != "" {
		overrides = append(overrides, ActiveOverride{
			From:   "L4",
			To:     "L3",
			Target: fmt.Sprintf("chapter:%d/user_input", in.ChapterNumber),
			Reason: "用户本章指令覆盖默认规划层。",
		})
	}
	if len(memoHookIDs) > 0 {
		ids := make([]string, 0, len(memoHookIDs))
		for id := range memoHookIDs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		overrides = append(overrides, ActiveOverride{
			From:   "L4",
			To:     "L3",
			Target: fmt.Sprintf("chapter:%d/hook_agenda", in.ChapterNumber),
			Reason: "chapter_memo 显式点名 hook：" + strings.Join(ids, ", "),
		})
	}

	return RuleStack{
		Layers: []RuleLayer{
			{ID: "L1", Name: "hard_facts", Precedence: 100, Scope: "global"},
			{ID: "L2", Name: "author_intent", Precedence: 80, Scope: "book"},
			{ID: "L3", Name: "planning", Precedence: 60, Scope: "arc"},
			{ID: "L4", Name: "current_task", Precedence: 90, Scope: "local"},
		},
		Hard:            []string{"foundations/story_frame", "foundations/book_rules", "truth/book_state", "wiki/graph_context", "truth/hooks"},
		Soft:            []string{"foundations/author_intent", "foundations/style_guide", "foundations/current_focus", "truth/recent_summaries", "chapters/previous_tail"},
		Diagnostic:      []string{"continuity_audit", "revision_gate", "settler_validation"},
		ActiveOverrides: overrides,
	}
}

func RenderWikiGraphContext(graph *model.WikiGraphContext) string {
	if graph == nil {
		return ""
	}
	seedIDs := make(map[uint]struct{}, len(graph.Seeds))
	for _, seed := range graph.Seeds {
		seedIDs[seed.ID] = struct{}{}
	}

	var b strings.Builder
	if len(graph.Entities) > 0 {
		b.WriteString("### 实体\n")
		for _, entity := range graph.Entities {
			role := "邻接"
			if _, ok := seedIDs[entity.ID]; ok {
				role = "种子"
			}
			fmt.Fprintf(&b, "- [%s/%s] %s", role, entity.EntityType, entity.CanonicalName)
			if strings.TrimSpace(entity.Summary) != "" {
				fmt.Fprintf(&b, "：%s", clipComposerText(entity.Summary, 320))
			}
			b.WriteByte('\n')
		}
	}
	if len(graph.Relations) > 0 {
		b.WriteString("\n### 当前有效关系\n")
		for _, relation := range graph.Relations {
			object := relation.ObjectName
			if strings.TrimSpace(object) == "" {
				object = relation.ObjectLiteral
			}
			fmt.Fprintf(&b, "- %s --%s--> %s（第%d章起",
				relation.SubjectName, relation.Predicate, object, relation.ValidFromChapter)
			if relation.ValidUntilChapter != nil {
				fmt.Fprintf(&b, "，至第%d章", *relation.ValidUntilChapter)
			}
			b.WriteString("）\n")
		}
	}
	if len(graph.Events) > 0 {
		b.WriteString("\n### 关联事件\n")
		for _, event := range graph.Events {
			fmt.Fprintf(&b, "- 第%d章 %s：%s", event.ChapterNumber, event.Title, clipComposerText(event.Summary, 320))
			if strings.TrimSpace(event.Consequence) != "" {
				fmt.Fprintf(&b, "；后果：%s", clipComposerText(event.Consequence, 220))
			}
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}

func buildTrace(chapterNumber uint, selected []ContextSource, notes []string) ChapterTrace {
	trace := ChapterTrace{
		ChapterNumber:     chapterNumber,
		SelectedSources:   make([]string, 0, len(selected)),
		ProtectedSources:  make([]string, 0, len(selected)),
		SupportingSources: make([]string, 0, len(selected)),
		Notes:             notes,
	}
	for _, source := range selected {
		trace.SelectedSources = append(trace.SelectedSources, source.Source)
		trace.EstimatedChars += len([]rune(source.Excerpt))
		if source.Protected {
			trace.ProtectedSources = append(trace.ProtectedSources, source.Source)
		} else {
			trace.SupportingSources = append(trace.SupportingSources, source.Source)
		}
	}
	return trace
}

func orderedFoundations(foundations []model.BookFoundation) []model.BookFoundation {
	out := append([]model.BookFoundation(nil), foundations...)
	priority := map[model.FoundationFileType]int{
		model.FoundationStoryFrame:   0,
		model.FoundationBookRules:    1,
		model.FoundationAuthorIntent: 2,
		model.FoundationStyleGuide:   3,
		model.FoundationCurrentFocus: 4,
		model.FoundationAuditDrift:   5,
		model.FoundationVolumeMap:    6,
	}
	sort.SliceStable(out, func(i, j int) bool {
		return priority[out[i].FileType] < priority[out[j].FileType]
	})
	return out
}

func foundationReason(fileType model.FoundationFileType) string {
	switch fileType {
	case model.FoundationStoryFrame:
		return "故事框架和世界约束，是硬护栏。"
	case model.FoundationBookRules:
		return "本书规则、禁令和主角约束，是硬护栏。"
	case model.FoundationAuthorIntent:
		return "长期作者意图，约束大方向。"
	case model.FoundationStyleGuide:
		return "平台与文风约束。"
	case model.FoundationCurrentFocus:
		return "最近 1-3 章的推进焦点。"
	case model.FoundationAuditDrift:
		return "前序章节遗留的审计偏移提醒。"
	case model.FoundationVolumeMap:
		return "卷级规划，仅作为软约束。"
	default:
		return "基础设定文件。"
	}
}

func isProtectedFoundation(fileType model.FoundationFileType) bool {
	switch fileType {
	case model.FoundationStoryFrame, model.FoundationBookRules, model.FoundationAuthorIntent, model.FoundationCurrentFocus, model.FoundationAuditDrift:
		return true
	default:
		return false
	}
}

func isAlwaysIncludedFoundation(fileType model.FoundationFileType) bool {
	switch fileType {
	case model.FoundationBookRules,
		model.FoundationAuthorIntent,
		model.FoundationCurrentFocus,
		model.FoundationAuditDrift:
		return true
	default:
		return false
	}
}

func appendRetrievedKnowledge(selected *[]ContextSource, results []model.KnowledgeSearchResult) {
	const budget = 3200
	remaining := budget
	seen := make(map[string]struct{}, len(results))

	for _, result := range results {
		if remaining <= 0 {
			return
		}
		key := fmt.Sprintf("%s/%s/%d", result.SourceType, result.SourceID, result.ChunkIndex)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		limit := 600
		switch result.SourceType {
		case model.KnowledgeSourceFoundation:
			limit = 800
		case model.KnowledgeSourceCharacter:
			limit = 680
		}
		if limit > remaining {
			limit = remaining
		}
		excerpt := clipComposerText(result.Content, limit)
		if strings.TrimSpace(excerpt) == "" {
			continue
		}
		*selected = append(*selected, ContextSource{
			Source:    fmt.Sprintf("retrieval/%s/%s#%d", result.SourceType, result.SourceID, result.ChunkIndex),
			Reason:    fmt.Sprintf("BM25 检索命中：%s。", strings.TrimSpace(result.Title)),
			Excerpt:   excerpt,
			Protected: false,
		})
		remaining -= len([]rune(excerpt))
	}
}

func renderBookState(state *model.BookState) string {
	lines := []string{
		fmt.Sprintf("- current_chapter: %d", state.CurrentChapter),
		"- protagonist: " + state.ProtagonistName,
		"- situation: " + state.SituationSummary,
		"- location: " + state.CurrentLocation,
		"- protagonist_state: " + state.ProtagonistState,
		"- goal: " + state.CurrentGoal,
		"- constraint: " + state.CurrentConstraint,
		"- alliances: " + state.CurrentAlliances,
		"- conflict: " + state.CurrentConflict,
	}
	return strings.Join(nonEmptyLines(lines), "\n")
}

func renderCharacters(characters []model.Character, memo string) string {
	out := append([]model.Character(nil), characters...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RoleType != out[j].RoleType {
			return characterPriority(out[i].RoleType) < characterPriority(out[j].RoleType)
		}
		return out[i].LastSeenChapter > out[j].LastSeenChapter
	})
	lines := make([]string, 0, len(out))
	optionalCount := 0
	for _, c := range out {
		context := firstNonEmptyComposer(c.Profile, c.CurrentStatus, c.CoreTags, c.InnerDrive)
		if context == "" {
			continue
		}
		required := c.RoleType == model.CharacterProtagonist || strings.Contains(memo, c.Name)
		if !required && (c.RoleType == model.CharacterMinor || optionalCount >= 3) {
			continue
		}
		if !required {
			optionalCount++
		}
		lines = append(lines, fmt.Sprintf("- %s (%s, last_seen=%d): %s", c.Name, c.RoleType, c.LastSeenChapter, clipComposerText(context, 360)))
	}
	return strings.Join(lines, "\n")
}

func renderFacts(facts []model.Fact, memo string) string {
	out := append([]model.Fact(nil), facts...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Subject != out[j].Subject {
			return out[i].Subject < out[j].Subject
		}
		return out[i].ValidFromChapter > out[j].ValidFromChapter
	})
	lines := make([]string, 0, len(out))
	optionalCount := 0
	for _, fact := range out {
		required := fact.Category == "rule" || factMentionedInMemo(fact, memo)
		if !required && optionalCount >= 4 {
			continue
		}
		if !required {
			optionalCount++
		}
		lines = append(lines, fmt.Sprintf("- %s %s %s (category=%s, from=%d)", fact.Subject, fact.Predicate, fact.Object, fact.Category, fact.ValidFromChapter))
	}
	return strings.Join(lines, "\n")
}

func renderHooks(hooks []model.Hook, memoHookIDs map[string]struct{}, chapterNumber uint) string {
	active := make([]model.Hook, 0, len(hooks))
	for _, hook := range hooks {
		if hook.Status == model.HookResolved || hook.Status == model.HookStale {
			continue
		}
		active = append(active, hook)
	}
	sort.SliceStable(active, func(i, j int) bool {
		_, leftReferenced := memoHookIDs[active[i].HookID]
		_, rightReferenced := memoHookIDs[active[j].HookID]
		if leftReferenced != rightReferenced {
			return leftReferenced
		}
		if active[i].IsCritical != active[j].IsCritical {
			return active[i].IsCritical
		}
		return active[i].LastAdvancedChapter > active[j].LastAdvancedChapter
	})
	lines := make([]string, 0, len(active))
	optionalCount := 0
	for _, hook := range active {
		_, referenced := memoHookIDs[hook.HookID]
		age := int(chapterNumber) - int(hook.LastAdvancedChapter)
		required := referenced || hook.IsCritical || age >= 5
		if !required && optionalCount >= 4 {
			continue
		}
		if !required {
			optionalCount++
		}
		lines = append(lines, fmt.Sprintf("- %s [%s] status=%s start=%d last=%d age=%d payoff=%s notes=%s",
			hook.HookID, hook.Type, hook.Status, hook.StartChapter, hook.LastAdvancedChapter, age,
			clipComposerText(hook.ExpectedPayoff, 120), clipComposerText(hook.Notes, 180)))
	}
	return strings.Join(lines, "\n")
}

func renderRecentSummaries(summaries []model.ChapterSummary, keep int) string {
	out := append([]model.ChapterSummary(nil), summaries...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ChapterNumber < out[j].ChapterNumber
	})
	if len(out) > keep {
		out = out[len(out)-keep:]
	}
	lines := make([]string, 0, len(out))
	for _, summary := range out {
		lines = append(lines, fmt.Sprintf("- 第%d章 %s: %s | state=%s | hook=%s",
			summary.ChapterNumber, summary.Title,
			clipComposerText(summary.KeyEvents, 220),
			clipComposerText(summary.StateChanges, 160),
			clipComposerText(summary.HookActivity, 160)))
	}
	return strings.Join(lines, "\n")
}

func renderRadarProfile(profile model.RadarTaxonomyProfile) string {
	parts := []string{
		strings.TrimSpace(profile.WriterBrief),
		strings.TrimSpace(profile.ProfileSummary),
	}
	if profile.TagKey != "" {
		parts = append([]string{"标签：" + profile.TagKey}, parts...)
	}
	return strings.Join(nonEmptyLines(parts), "\n\n")
}

func renderRadarRules(rules []model.RadarRule) string {
	out := append([]model.RadarRule(nil), rules...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Weight != out[j].Weight {
			return out[i].Weight > out[j].Weight
		}
		return out[i].Confidence > out[j].Confidence
	})
	lines := make([]string, 0, len(out))
	for _, rule := range out {
		lines = append(lines, fmt.Sprintf("- [%s/%s] %s：%s", rule.RuleType, rule.Category, rule.Title, rule.Content))
	}
	return strings.Join(lines, "\n")
}

func renderChapterTail(chapter *model.Chapter, max int) string {
	content := strings.TrimSpace(chapter.Content)
	runes := []rune(content)
	if len(runes) > max {
		content = string(runes[len(runes)-max:])
	}
	return fmt.Sprintf("标题：%s\n%s", strings.TrimSpace(chapter.Title), content)
}

func extractHookIDs(memo string) map[string]struct{} {
	result := make(map[string]struct{})
	re := regexp.MustCompile(`\b[Hh]\d{2,}\b`)
	for _, match := range re.FindAllString(memo, -1) {
		result[strings.ToUpper(match)] = struct{}{}
	}
	return result
}

func factMentionedInMemo(fact model.Fact, memo string) bool {
	return strings.Contains(memo, fact.Subject) ||
		strings.Contains(memo, fact.Predicate) ||
		strings.Contains(memo, fact.Object)
}

func characterPriority(role model.CharacterRoleType) int {
	switch role {
	case model.CharacterProtagonist:
		return 0
	case model.CharacterMajor:
		return 1
	default:
		return 2
	}
}

func nonEmptyLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasSuffix(line, ": ") {
			out = append(out, line)
		}
	}
	return out
}

func clipComposerText(raw string, max int) string {
	raw = strings.TrimSpace(raw)
	if max <= 0 {
		return raw
	}
	runes := []rune(raw)
	if len(runes) <= max {
		return raw
	}
	return string(runes[:max]) + "\n（已截断）"
}

func firstNonEmptyComposer(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
