package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"whwriter/backend/internal/agent"
	"whwriter/backend/internal/llm"
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"
)

type ProgressWriter interface {
	io.Writer
	Flush() error
}

type Pipeline struct {
	llm      *llm.Client
	truth    repository.TruthFileRepository
	registry *agent.Registry
}

func New(llmClient *llm.Client, truthRepo repository.TruthFileRepository) *Pipeline {
	return &Pipeline{
		llm:      llmClient,
		truth:    truthRepo,
		registry: agent.NewRegistry(),
	}
}

type WriteChapterInput struct {
	BookID    uint
	ModelID   uint
	UserInput string
	Progress  ProgressWriter
}

type InitBookInput struct {
	BookID   uint
	Progress ProgressWriter
}

type WriteChapterOutput struct {
	ChapterNumber uint
	Title         string
	Content       string
	Memo          string
}

type extractionOptions struct {
	SaveHooks   bool
	SaveSummary bool
}

type auditIssue struct {
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type auditResult struct {
	Passed       bool         `json:"passed"`
	OverallScore int          `json:"overall_score"`
	Issues       []auditIssue `json:"issues"`
	Summary      string       `json:"summary"`
}

type settlerHookOp struct {
	HookID              string `json:"hookId"`
	StartChapter        uint   `json:"startChapter"`
	Type                string `json:"type"`
	Status              string `json:"status"`
	LastAdvancedChapter uint   `json:"lastAdvancedChapter"`
	ExpectedPayoff      string `json:"expectedPayoff"`
	PayoffTiming        string `json:"payoffTiming"`
	Notes               string `json:"notes"`
}

type settlerHookOps struct {
	Upsert  []settlerHookOp `json:"upsert"`
	Mention []string        `json:"mention"`
	Resolve []string        `json:"resolve"`
	Defer   []string        `json:"defer"`
}

type settlerHookCandidate struct {
	Type           string `json:"type"`
	ExpectedPayoff string `json:"expectedPayoff"`
	PayoffTiming   string `json:"payoffTiming"`
	Notes          string `json:"notes"`
}

type settlerChapterSummary struct {
	Chapter    uint   `json:"chapter"`
	Title      string `json:"title"`
	Characters string `json:"characters"`
	Events     string `json:"events"`
	State      string `json:"stateChanges"`
	Hook       string `json:"hookActivity"`
	Mood       string `json:"mood"`
	Type       string `json:"chapterType"`
}

type settlerDelta struct {
	Chapter           uint                   `json:"chapter"`
	CurrentStatePatch map[string]string      `json:"currentStatePatch"`
	HookOps           settlerHookOps         `json:"hookOps"`
	NewHookCandidates []settlerHookCandidate `json:"newHookCandidates"`
	ChapterSummary    settlerChapterSummary  `json:"chapterSummary"`
	Notes             []string               `json:"notes"`
}

func emitProgress(w ProgressWriter, stage, msg string) {
	if w == nil {
		return
	}
	data, _ := json.Marshal(map[string]string{"stage": stage, "message": msg})
	w.Write([]byte("data: " + string(data) + "\n\n"))
	w.Flush()
}

func (p *Pipeline) InitBook(ctx context.Context, in InitBookInput) error {
	emitProgress(in.Progress, "loading", "正在加载书籍信息...")

	book, err := p.truth.GetBook(in.BookID)
	if err != nil {
		return fmt.Errorf("get book: %w", err)
	}

	emitProgress(in.Progress, "planning", "正在初始化基础设定...")

	fallback := buildFallbackArchitectSections(book)
	sections := fallback
	modelID := p.resolveModelID(book.ID, "architect", book.LLMModelID)

	if architectRaw, err := p.runArchitect(ctx, book, modelID); err == nil {
		for name, content := range parseArchitectSections(architectRaw) {
			if strings.TrimSpace(content) != "" {
				sections[name] = content
			}
		}

		_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
			BookID:        book.ID,
			ChapterNumber: 0,
			ArtifactType:  model.ArtifactTrace,
			Content:       architectRaw,
		})
	}

	sections = p.ensureIdentityAnchors(ctx, book, sections, modelID)

	emitProgress(in.Progress, "context", "正在落库基础真相文件...")

	if err := p.seedFoundations(book, sections); err != nil {
		return fmt.Errorf("seed foundations: %w", err)
	}
	p.seedInitialCharacters(book.ID, sections["roles"], sections["book_rules"])
	p.seedInitialHooks(book.ID, sections["pending_hooks"])
	p.saveInitialBookState(book, sections)
	p.saveInitialSnapshot(book, sections)

	_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
		BookID:        book.ID,
		ChapterNumber: 0,
		ArtifactType:  model.ArtifactRuleStack,
		Content:       sections["book_rules"],
	})

	emitProgress(in.Progress, "done", "书籍初始化完成")
	return nil
}

func (p *Pipeline) WriteChapter(ctx context.Context, in WriteChapterInput) (*WriteChapterOutput, error) {
	emit := func(stage, msg string) error {
		if in.Progress == nil {
			return nil
		}
		data, _ := json.Marshal(map[string]string{"stage": stage, "message": msg})
		if _, err := in.Progress.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
			return fmt.Errorf("stream write failed at %s: %w", stage, err)
		}
		if err := in.Progress.Flush(); err != nil {
			return fmt.Errorf("stream flush failed at %s: %w", stage, err)
		}
		return nil
	}
	if err := emit("loading", "正在加载书籍信息..."); err != nil {
		return nil, err
	}

	book, err := p.truth.GetBook(in.BookID)
	if err != nil {
		return nil, fmt.Errorf("get book: %w", err)
	}

	chapterNumber, err := p.truth.GetNextChapterNumber(in.BookID)
	if err != nil {
		return nil, fmt.Errorf("get next chapter: %w", err)
	}

	if err := emit("context", "正在构建上下文..."); err != nil {
		return nil, err
	}

	contextPkg, err := p.buildContext(ctx, in.BookID, chapterNumber)
	if err != nil {
		return nil, fmt.Errorf("build context: %w", err)
	}

	_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
		BookID:        in.BookID,
		ChapterNumber: chapterNumber,
		ArtifactType:  model.ArtifactContext,
		Content:       contextPkg,
	})

	if err := emit("planning", "Planner 正在规划本章内容..."); err != nil {
		return nil, err
	}

	plannerModelID := p.resolveModelID(in.BookID, "planner", in.ModelID)

	planner, ok := p.registry.Get("planner")
	if !ok {
		return nil, fmt.Errorf("planner agent not found")
	}

	plannerInput := fmt.Sprintf(plannerBuildInput,
		book.Title,
		book.Genre.Name,
		book.Platform.Name,
		chapterNumber,
		book.ChapterWordCount,
		contextPkg,
		in.UserInput,
	)

	memo, err := p.llm.ChatForAgent(ctx, "planner", plannerModelID, planner.SystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: plannerInput},
	}, 0.7)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}

	_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
		BookID:        in.BookID,
		ChapterNumber: chapterNumber,
		ArtifactType:  model.ArtifactIntent,
		Content:       memo,
	})

	if err := emit("writing", "Writer 正在创作正文..."); err != nil {
		return nil, err
	}

	writerModelID := p.resolveModelID(in.BookID, "writer", in.ModelID)

	writer, ok := p.registry.Get("writer")
	if !ok {
		return nil, fmt.Errorf("writer agent not found")
	}

	writerAgent, ok := writer.(*agent.WriterAgent)
	if !ok {
		return nil, fmt.Errorf("invalid writer agent")
	}

	systemPrompt := writerAgent.BuildSystemPrompt(agent.WriterInput{
		Platform:         book.Platform.Name,
		GenreName:        book.Genre.Name,
		ChapterWordCount: book.ChapterWordCount,
		ChapterNumber:    int(chapterNumber),
		IsGoverned:       true,
	})

	writerInput := fmt.Sprintf(writerBuildInput,
		book.Title,
		chapterNumber,
		memo,
		contextPkg,
	)

	rawOutput, err := p.llm.ChatForAgent(ctx, "writer", writerModelID, systemPrompt, []llm.AgentMessage{
		{Role: "user", Content: writerInput},
	}, 0.8)
	if err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	if err := emit("parsing", "正在解析输出..."); err != nil {
		return nil, err
	}

	sections := parseSections(rawOutput)

	title := strings.TrimSpace(sections["CHAPTER_TITLE"])
	content := strings.TrimSpace(sections["CHAPTER_CONTENT"])
	if content == "" {
		fallbackTitle, fallbackContent := fallbackWriterNarrative(rawOutput)
		if title == "" {
			title = fallbackTitle
		}
		if fallbackContent != "" {
			content = fallbackContent
			sections["CHAPTER_TITLE"] = title
			sections["CHAPTER_CONTENT"] = content
		}
	}

	if title == "" {
		title = fmt.Sprintf("第%d章", chapterNumber)
	}
	if content == "" {
		debugPayload := map[string]string{
			"title":           title,
			"raw_output":      rawOutput,
			"parsed_sections": strings.Join(sortedSectionNames(sections), ", "),
			"pre_write_check": sections["PRE_WRITE_CHECK"],
			"post_settlement": sections["POST_SETTLEMENT"],
			"updated_state":   sections["UPDATED_STATE"],
			"chapter_summary": sections["CHAPTER_SUMMARY"],
			"chapter_content": sections["CHAPTER_CONTENT"],
		}
		if payload, marshalErr := json.Marshal(debugPayload); marshalErr == nil {
			_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
				BookID:        in.BookID,
				ChapterNumber: chapterNumber,
				ArtifactType:  model.ArtifactTrace,
				Content:       string(payload),
			})
		}
		return nil, fmt.Errorf("writer 输出缺少 CHAPTER_CONTENT，未写入章节正文")
	}

	_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
		BookID:        in.BookID,
		ChapterNumber: chapterNumber,
		ArtifactType:  model.ArtifactPlan,
		Content:       sections["PRE_WRITE_CHECK"],
	})

	tracePayload := map[string]interface{}{
		"writer": map[string]interface{}{
			"sections":         sortedSectionNames(sections),
			"pre_write_check":  sections["PRE_WRITE_CHECK"],
			"post_settlement":  sections["POST_SETTLEMENT"],
			"updated_state":    sections["UPDATED_STATE"],
			"chapter_summary":  sections["CHAPTER_SUMMARY"],
			"title":            title,
			"raw_output":       rawOutput,
			"finalized_source": "writer",
		},
	}

	if err := emit("auditing", "Auditor 正在审查章节结构与连续性..."); err != nil {
		return nil, err
	}

	auditResult, auditRaw, err := p.runAuditor(ctx, book, chapterNumber, memo, contextPkg, content, in.ModelID)
	if err == nil {
		tracePayload["audit"] = map[string]interface{}{
			"raw_output": auditRaw,
			"result":     auditResult,
		}
	} else {
		tracePayload["audit"] = map[string]interface{}{
			"error": err.Error(),
		}
	}

	if err == nil && !auditResult.Passed {
		if err := emit("revising", "Reviser 正在根据审稿意见修订章节..."); err != nil {
			return nil, err
		}
		revisedContent, revisedSections, reviserRaw, reviseErr := p.runReviser(ctx, book, chapterNumber, memo, contextPkg, content, auditRaw, in.ModelID)
		if reviseErr == nil && strings.TrimSpace(revisedContent) != "" {
			content = strings.TrimSpace(revisedContent)
			tracePayload["reviser"] = map[string]interface{}{
				"raw_output":    reviserRaw,
				"sections":      sortedSectionNames(revisedSections),
				"fixed_issues":  revisedSections["FIXED_ISSUES"],
				"updated_state": revisedSections["UPDATED_STATE"],
			}
			if strings.TrimSpace(revisedSections["UPDATED_STATE"]) != "" {
				sections["UPDATED_STATE"] = revisedSections["UPDATED_STATE"]
			}
			if strings.TrimSpace(revisedSections["UPDATED_HOOKS"]) != "" {
				sections["UPDATED_HOOKS"] = revisedSections["UPDATED_HOOKS"]
			}
			if strings.TrimSpace(revisedSections["POST_SETTLEMENT"]) != "" {
				sections["POST_SETTLEMENT"] = revisedSections["POST_SETTLEMENT"]
			}
			tracePayload["writer"].(map[string]interface{})["finalized_source"] = "reviser"
		} else {
			reviserErrMsg := "reviser 未产出可用修订结果"
			if reviseErr != nil {
				reviserErrMsg = reviseErr.Error()
			}
			tracePayload["reviser"] = map[string]interface{}{
				"error":      reviserErrMsg,
				"raw_output": reviserRaw,
			}
		}
	}

	if err := emit("polishing", "Polisher 正在润色正文..."); err != nil {
		return nil, err
	}

	polishedContent, polishErr := p.runPolisher(ctx, book, chapterNumber, content, in.ModelID)
	if polishErr == nil && strings.TrimSpace(polishedContent) != "" {
		content = strings.TrimSpace(polishedContent)
		tracePayload["polisher"] = map[string]interface{}{
			"applied": true,
		}
		tracePayload["writer"].(map[string]interface{})["finalized_source"] = "polisher"
	} else if polishErr != nil {
		tracePayload["polisher"] = map[string]interface{}{
			"error": polishErr.Error(),
		}
	}

	ch := &model.Chapter{
		BookID:        in.BookID,
		ChapterNumber: chapterNumber,
		Title:         title,
		Content:       content,
		WordCount:     uint(len([]rune(content))),
		Status:        model.ChapterDraft,
	}
	if err := p.truth.SaveChapter(ch); err != nil {
		return nil, fmt.Errorf("save chapter: %w", err)
	}

	if err := emit("extracting", "Settler 正在结算真相文件增量..."); err != nil {
		return nil, err
	}

	sections["CHAPTER_TITLE"] = title
	sections["CHAPTER_CONTENT"] = content

	settleSections, settleDelta, settlerRaw, settleErr := p.settleTruthFiles(ctx, book, chapterNumber, title, content, writerModelID)
	if settleErr == nil {
		for key, value := range settleSections {
			if strings.TrimSpace(value) != "" {
				sections[key] = value
			}
		}
		p.saveDebugTrace(in.BookID, chapterNumber, "settler_done", map[string]any{
			"sections":  sortedSectionNames(settleSections),
			"has_delta": true,
		})
		tracePayload["settler"] = map[string]any{
			"raw_output": settlerRaw,
			"delta":      settleDelta,
		}
	} else {
		p.saveDebugTrace(in.BookID, chapterNumber, "settler_error", map[string]any{
			"error": settleErr.Error(),
		})
		tracePayload["settler"] = map[string]any{
			"error":      settleErr.Error(),
			"raw_output": settlerRaw,
		}
	}

	p.saveDebugTrace(in.BookID, chapterNumber, "extract_start", map[string]any{
		"save_hooks":   settleErr != nil,
		"save_summary": settleErr != nil,
	})
	p.extractTruthFiles(
		ctx,
		in.BookID,
		chapterNumber,
		fmt.Sprintf("章节标题：%s\n\n章节正文：\n%s", title, content),
		writerModelID,
		extractionOptions{SaveHooks: settleErr != nil, SaveSummary: settleErr != nil},
	)

	if err := emit("snapshot", "正在保存章节快照和运行时产物..."); err != nil {
		return nil, err
	}
	p.saveDebugTrace(in.BookID, chapterNumber, "snapshot_start", nil)

	p.saveChapterSnapshot(in.BookID, chapterNumber, sections)
	if payload, marshalErr := json.Marshal(tracePayload); marshalErr == nil {
		_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
			BookID:        in.BookID,
			ChapterNumber: chapterNumber,
			ArtifactType:  model.ArtifactTrace,
			Content:       string(payload),
		})
	}

	if err := emit("done", fmt.Sprintf("第%d章创作完成", chapterNumber)); err != nil {
		return nil, err
	}

	return &WriteChapterOutput{
		ChapterNumber: chapterNumber,
		Title:         title,
		Content:       content,
		Memo:          memo,
	}, nil
}

func (p *Pipeline) saveDebugTrace(bookID uint, chapterNumber uint, stage string, payload map[string]any) {
	entry := map[string]any{
		"stage": stage,
	}
	if payload != nil {
		entry["payload"] = payload
	}
	if content, err := json.Marshal(entry); err == nil {
		_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
			BookID:        bookID,
			ChapterNumber: chapterNumber,
			ArtifactType:  model.ArtifactTrace,
			Content:       string(content),
		})
	}
}

func (p *Pipeline) resolveModelID(bookID uint, agentName string, fallbackID uint) uint {
	route, err := p.truth.GetAgentModelRoute(bookID, agentName)
	if err == nil && route != nil && route.LLMModelID > 0 {
		return route.LLMModelID
	}
	if fallbackID > 0 {
		return fallbackID
	}
	defaultModel, err := p.llm.GetDefaultModel()
	if err == nil {
		return defaultModel.ID
	}
	return 0
}

func (p *Pipeline) runAuditor(ctx context.Context, book *model.Book, chapterNumber uint, memo, contextPkg, content string, fallbackModelID uint) (auditResult, string, error) {
	auditorAny, ok := p.registry.Get("auditor")
	if !ok {
		return auditResult{}, "", fmt.Errorf("auditor agent not found")
	}
	auditor, ok := auditorAny.(*agent.ContinuityAuditor)
	if !ok {
		return auditResult{}, "", fmt.Errorf("invalid auditor agent")
	}

	modelID := p.resolveModelID(book.ID, "auditor", fallbackModelID)
	systemPrompt := auditor.BuildSystemPrompt(agent.AuditInput{GenreName: book.Genre.Name})
	userPrompt := fmt.Sprintf(`请审查小说《%s》第 %d 章。

## chapter_memo
%s

## 上下文
%s

## 当前正文
%s

请严格按 JSON 输出格式返回审稿结果。`, book.Title, chapterNumber, memo, contextPkg, content)

	raw, err := p.llm.ChatForAgent(ctx, "auditor", modelID, systemPrompt, []llm.AgentMessage{
		{Role: "user", Content: userPrompt},
	}, 0.2)
	if err != nil {
		return auditResult{}, "", err
	}

	var result auditResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &result); err != nil {
		return auditResult{}, raw, fmt.Errorf("parse auditor output: %w", err)
	}
	return result, raw, nil
}

func (p *Pipeline) runReviser(ctx context.Context, book *model.Book, chapterNumber uint, memo, contextPkg, content, auditRaw string, fallbackModelID uint) (string, map[string]string, string, error) {
	reviserAny, ok := p.registry.Get("reviser")
	if !ok {
		return "", nil, "", fmt.Errorf("reviser agent not found")
	}
	reviser, ok := reviserAny.(*agent.ReviserAgent)
	if !ok {
		return "", nil, "", fmt.Errorf("invalid reviser agent")
	}

	modelID := p.resolveModelID(book.ID, "reviser", fallbackModelID)
	systemPrompt := reviser.BuildSystemPrompt(agent.ReviseInput{GenreName: book.Genre.Name})
	userPrompt := fmt.Sprintf(`请根据下面的审稿意见修订小说《%s》第 %d 章。

## 审稿结果
%s

## chapter_memo
%s

## 上下文
%s

## 当前正文
%s`, book.Title, chapterNumber, auditRaw, memo, contextPkg, content)

	raw, err := p.llm.ChatForAgent(ctx, "reviser", modelID, systemPrompt, []llm.AgentMessage{
		{Role: "user", Content: userPrompt},
	}, 0.35)
	if err != nil {
		return "", nil, "", err
	}

	sections := parseSections(raw)
	revisedContent := strings.TrimSpace(sections["REVISED_CONTENT"])
	if revisedContent == "" {
		revisedContent = strings.TrimSpace(sections["CHAPTER_CONTENT"])
	}
	if revisedContent == "" {
		return "", sections, raw, fmt.Errorf("reviser 输出缺少 REVISED_CONTENT")
	}
	return revisedContent, sections, raw, nil
}

func (p *Pipeline) runPolisher(ctx context.Context, book *model.Book, chapterNumber uint, content string, fallbackModelID uint) (string, error) {
	polisherAny, ok := p.registry.Get("polisher")
	if !ok {
		return "", fmt.Errorf("polisher agent not found")
	}

	modelID := p.resolveModelID(book.ID, "polisher", fallbackModelID)
	userPrompt := fmt.Sprintf(`请润色小说《%s》第 %d 章正文。

要求：
1. 只做文字层优化，不改剧情走向
2. 保持章节信息与人设一致
3. 直接返回润色后的完整正文

## 正文
%s`, book.Title, chapterNumber, content)

	raw, err := p.llm.Chat(ctx, modelID, polisherAny.SystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: userPrompt},
	}, 0.2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(raw), nil
}

func (p *Pipeline) runArchitect(ctx context.Context, book *model.Book, modelID uint) (string, error) {
	architectAny, ok := p.registry.Get("architect")
	if !ok {
		return "", fmt.Errorf("architect agent not found")
	}

	architect, ok := architectAny.(*agent.ArchitectAgent)
	if !ok {
		return "", fmt.Errorf("invalid architect agent")
	}

	systemPrompt := architect.BuildSystemPrompt(agent.ArchitectInput{
		Title:            book.Title,
		Platform:         book.Platform.Name,
		Genre:            book.Genre.Name,
		GenreName:        book.Genre.Name,
		GenreBody:        book.Genre.ProfileMarkdown,
		TargetChapters:   book.TargetChapters,
		ChapterWordCount: book.ChapterWordCount,
		Language:         book.Language,
		ExternalContext:  book.Description,
	})

	userPrompt := fmt.Sprintf(`请为小说《%s》初始化基础设定。

## 书籍简介
%s

## 平台风格
%s

请严格按 system prompt 中的 5 个 SECTION 输出。`, book.Title, defaultString(book.Description, "暂无额外简介，请根据题材与平台风格起盘。"), defaultString(book.Platform.StyleGuide, "保持平台主流网文可读性与节奏感。"))

	return p.llm.Chat(ctx, modelID, systemPrompt, []llm.AgentMessage{
		{Role: "user", Content: userPrompt},
	}, 0.7)
}

func (p *Pipeline) settleTruthFiles(ctx context.Context, book *model.Book, chapterNumber uint, title, content string, fallbackModelID uint) (map[string]string, settlerDelta, string, error) {
	settlerAny, ok := p.registry.Get("settler")
	if !ok {
		return nil, settlerDelta{}, "", fmt.Errorf("settler agent not found")
	}

	settler, ok := settlerAny.(*agent.SettlerAgent)
	if !ok {
		return nil, settlerDelta{}, "", fmt.Errorf("invalid settler agent")
	}

	truthContext, err := p.buildSettlerContext(book.ID)
	if err != nil {
		return nil, settlerDelta{}, "", fmt.Errorf("build settler context: %w", err)
	}

	systemPrompt := settler.BuildSystemPrompt(agent.SettlerInput{
		Title:           book.Title,
		GenreName:       book.Genre.Name,
		Genre:           book.Genre.Name,
		Platform:        book.Platform.Name,
		NumericalSystem: false,
	})

	userPrompt := fmt.Sprintf(`请对小说《%s》第 %d 章做 truth file 增量结算。

## 当前 truth files
%s

## 本章标题
%s

## 本章正文
%s

请严格按 system prompt 里的 === TAG === 格式返回。`, book.Title, chapterNumber, truthContext, title, content)

	modelID := p.resolveModelID(book.ID, "settler", fallbackModelID)
	raw, err := p.llm.ChatForAgent(ctx, "settler", modelID, systemPrompt, []llm.AgentMessage{
		{Role: "user", Content: userPrompt},
	}, 0.2)
	if err != nil {
		return nil, settlerDelta{}, "", err
	}

	sections := parseSections(raw)
	jsonStr := extractJSON(sections["RUNTIME_STATE_DELTA"])
	if jsonStr == "" {
		return sections, settlerDelta{}, raw, fmt.Errorf("settler 输出缺少 RUNTIME_STATE_DELTA")
	}

	var delta settlerDelta
	if err := json.Unmarshal([]byte(jsonStr), &delta); err != nil {
		return sections, settlerDelta{}, raw, fmt.Errorf("parse settler delta: %w", err)
	}

	if err := p.applySettlerDelta(book.ID, chapterNumber, title, sections["POST_SETTLEMENT"], delta); err != nil {
		return sections, delta, raw, err
	}

	if len(delta.CurrentStatePatch) > 0 {
		if payload, err := json.Marshal(delta.CurrentStatePatch); err == nil {
			sections["UPDATED_STATE"] = string(payload)
		}
	}

	return sections, delta, raw, nil
}

func (p *Pipeline) buildSettlerContext(bookID uint) (string, error) {
	foundations, err := p.truth.ListFoundations(bookID)
	if err != nil {
		return "", err
	}
	hooks, _ := p.truth.GetHooks(bookID)
	summaries, _ := p.truth.GetChapterSummaries(bookID)
	snapshots, _ := p.truth.GetChapterSnapshots(bookID)

	var b strings.Builder
	b.WriteString("### Foundations\n")
	for _, f := range foundations {
		b.WriteString(fmt.Sprintf("## %s\n%s\n\n", f.FileType, clipText(strings.TrimSpace(f.Content), 1200)))
	}

	b.WriteString("### Hooks\n")
	if payload, err := json.Marshal(hooks); err == nil {
		b.WriteString(string(payload))
		b.WriteString("\n\n")
	}

	b.WriteString("### Recent Summaries\n")
	start := 0
	if len(summaries) > 5 {
		start = len(summaries) - 5
	}
	if payload, err := json.Marshal(summaries[start:]); err == nil {
		b.WriteString(string(payload))
		b.WriteString("\n\n")
	}

	if len(snapshots) > 0 {
		latest := snapshots[len(snapshots)-1]
		b.WriteString("### Latest Snapshot Current State\n")
		b.WriteString(latest.CurrentStateJSON)
		b.WriteString("\n")
	}

	return b.String(), nil
}

func (p *Pipeline) applySettlerDelta(bookID uint, chapterNumber uint, title, settlement string, delta settlerDelta) error {
	existingHooks, _ := p.truth.GetHooks(bookID)
	hookMap := make(map[string]*model.Hook, len(existingHooks))
	for i := range existingHooks {
		hook := existingHooks[i]
		hookMap[hook.HookID] = &hook
	}

	for _, upsert := range delta.HookOps.Upsert {
		hook, ok := hookMap[upsert.HookID]
		if !ok {
			hook = &model.Hook{
				BookID:       bookID,
				HookID:       upsert.HookID,
				StartChapter: defaultUint(upsert.StartChapter, chapterNumber),
			}
		}
		if hook.StartChapter == 0 {
			hook.StartChapter = defaultUint(upsert.StartChapter, chapterNumber)
		}
		hook.Type = normalizeHookType(upsert.Type)
		hook.Status = normalizeHookStatus(upsert.Status)
		if upsert.LastAdvancedChapter > 0 {
			hook.LastAdvancedChapter = upsert.LastAdvancedChapter
		} else if hook.LastAdvancedChapter == 0 {
			hook.LastAdvancedChapter = chapterNumber
		}
		if strings.TrimSpace(upsert.ExpectedPayoff) != "" {
			hook.ExpectedPayoff = strings.TrimSpace(upsert.ExpectedPayoff)
		}
		if strings.TrimSpace(upsert.PayoffTiming) != "" {
			hook.PayoffTiming = normalizePayoffTiming(upsert.PayoffTiming)
		}
		if strings.TrimSpace(upsert.Notes) != "" {
			hook.Notes = strings.TrimSpace(upsert.Notes)
		}
		if err := p.truth.SaveHook(hook); err != nil {
			return fmt.Errorf("save settled hook %s: %w", hook.HookID, err)
		}
		hookMap[hook.HookID] = hook
	}

	for _, hookID := range delta.HookOps.Resolve {
		hook := hookMap[hookID]
		if hook == nil {
			continue
		}
		hook.Status = model.HookResolved
		hook.LastAdvancedChapter = chapterNumber
		if err := p.truth.SaveHook(hook); err != nil {
			return fmt.Errorf("resolve hook %s: %w", hookID, err)
		}
	}

	for _, hookID := range delta.HookOps.Defer {
		hook := hookMap[hookID]
		if hook == nil {
			continue
		}
		hook.Status = model.HookDeferred
		hook.LastAdvancedChapter = chapterNumber
		if err := p.truth.SaveHook(hook); err != nil {
			return fmt.Errorf("defer hook %s: %w", hookID, err)
		}
	}

	for _, candidate := range delta.NewHookCandidates {
		newID := nextGeneratedHookID(hookMap)
		hook := &model.Hook{
			BookID:              bookID,
			HookID:              newID,
			StartChapter:        chapterNumber,
			Type:                normalizeHookType(candidate.Type),
			Status:              model.HookSeed,
			LastAdvancedChapter: chapterNumber,
			ExpectedPayoff:      strings.TrimSpace(candidate.ExpectedPayoff),
			PayoffTiming:        normalizePayoffTiming(candidate.PayoffTiming),
			Notes:               strings.TrimSpace(candidate.Notes),
		}
		if err := p.truth.SaveHook(hook); err != nil {
			return fmt.Errorf("create new settled hook %s: %w", newID, err)
		}
		hookMap[newID] = hook
	}

	if summary, ok := p.findChapterSummary(bookID, chapterNumber); ok {
		summary.Title = defaultString(strings.TrimSpace(delta.ChapterSummary.Title), defaultString(strings.TrimSpace(title), summary.Title))
		summary.CharactersAppeared = strings.TrimSpace(delta.ChapterSummary.Characters)
		summary.KeyEvents = strings.TrimSpace(delta.ChapterSummary.Events)
		summary.StateChanges = strings.TrimSpace(delta.ChapterSummary.State)
		summary.HookActivity = strings.TrimSpace(delta.ChapterSummary.Hook)
		summary.Mood = strings.TrimSpace(delta.ChapterSummary.Mood)
		summary.ChapterType = strings.TrimSpace(delta.ChapterSummary.Type)
		if err := p.truth.SaveChapterSummary(summary); err != nil {
			return fmt.Errorf("update chapter summary: %w", err)
		}
	} else if strings.TrimSpace(delta.ChapterSummary.Events) != "" || strings.TrimSpace(delta.ChapterSummary.Title) != "" {
		if err := p.truth.SaveChapterSummary(&model.ChapterSummary{
			BookID:             bookID,
			ChapterNumber:      chapterNumber,
			Title:              defaultString(strings.TrimSpace(delta.ChapterSummary.Title), title),
			CharactersAppeared: strings.TrimSpace(delta.ChapterSummary.Characters),
			KeyEvents:          strings.TrimSpace(delta.ChapterSummary.Events),
			StateChanges:       strings.TrimSpace(delta.ChapterSummary.State),
			HookActivity:       strings.TrimSpace(delta.ChapterSummary.Hook),
			Mood:               strings.TrimSpace(delta.ChapterSummary.Mood),
			ChapterType:        strings.TrimSpace(delta.ChapterSummary.Type),
		}); err != nil {
			return fmt.Errorf("save chapter summary: %w", err)
		}
	}

	if len(delta.CurrentStatePatch) > 0 {
		state, err := p.truth.GetBookState(bookID)
		if err != nil {
			return fmt.Errorf("get book state: %w", err)
		}
		if state == nil {
			state = &model.BookState{BookID: bookID}
		}
		state.CurrentChapter = chapterNumber
		if state.ProtagonistName == "" {
			characters, _ := p.truth.GetCharacters(bookID)
			for _, c := range characters {
				if c.RoleType == model.CharacterProtagonist {
					state.ProtagonistName = c.Name
					break
				}
			}
		}
		if v := strings.TrimSpace(delta.CurrentStatePatch["currentLocation"]); v != "" {
			state.CurrentLocation = v
		}
		if v := strings.TrimSpace(delta.CurrentStatePatch["protagonistState"]); v != "" {
			state.ProtagonistState = v
		}
		if v := strings.TrimSpace(delta.CurrentStatePatch["currentGoal"]); v != "" {
			state.CurrentGoal = v
		}
		if v := strings.TrimSpace(delta.CurrentStatePatch["currentConstraint"]); v != "" {
			state.CurrentConstraint = v
		}
		if v := strings.TrimSpace(delta.CurrentStatePatch["currentAlliances"]); v != "" {
			state.CurrentAlliances = v
		}
		if v := strings.TrimSpace(delta.CurrentStatePatch["currentConflict"]); v != "" {
			state.CurrentConflict = v
		}
		state.SituationSummary = buildSituationSummary(state)
		state.SourceChapter = chapterNumber
		if err := p.truth.SaveBookState(state); err != nil {
			return fmt.Errorf("save book state: %w", err)
		}
		if err := p.upsertFoundation(bookID, model.FoundationCurrentFocus, buildCurrentFocusDelta(chapterNumber, delta.CurrentStatePatch)); err != nil {
			return fmt.Errorf("update current_focus: %w", err)
		}
	}
	auditDrift := buildAuditDrift(settlement, delta.Notes)
	if strings.TrimSpace(auditDrift) != "" {
		if err := p.upsertFoundation(bookID, model.FoundationAuditDrift, auditDrift); err != nil {
			return fmt.Errorf("update audit_drift: %w", err)
		}
	}
	return nil
}

func (p *Pipeline) findChapterSummary(bookID uint, chapterNumber uint) (*model.ChapterSummary, bool) {
	summaries, err := p.truth.GetChapterSummaries(bookID)
	if err != nil {
		return nil, false
	}
	for i := range summaries {
		if summaries[i].ChapterNumber == chapterNumber {
			return &summaries[i], true
		}
	}
	return nil, false
}

func (p *Pipeline) upsertFoundation(bookID uint, fileType model.FoundationFileType, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	existing, err := p.truth.GetFoundation(bookID, fileType)
	if err == nil && existing != nil {
		existing.Content = content
		return p.truth.SaveFoundation(existing)
	}
	return p.truth.SaveFoundation(&model.BookFoundation{
		BookID:   bookID,
		FileType: fileType,
		Content:  content,
	})
}

func (p *Pipeline) extractTruthFiles(ctx context.Context, bookID uint, chapterNumber uint, rawOutput string, modelID uint, opts extractionOptions) {
	p.saveDebugTrace(bookID, chapterNumber, "extract_llm_start", map[string]any{
		"model_id":     modelID,
		"raw_len":      len(rawOutput),
		"save_hooks":   opts.SaveHooks,
		"save_summary": opts.SaveSummary,
	})
	extractPrompt := `你是这本小说的设定管理员。请从以下章节输出中提取关键信息，以JSON格式返回。

## 输出格式
{
  "characters": [{"name": "角色名", "role_type": "protagonist|major|minor", "profile": "一句话简介"}],
  "durable_facts": [{"subject": "主体", "predicate": "关系/属性", "object": "客体/值", "category": "identity|resource|item|rule|relationship"}],
  "hooks": [{"hook_id": "H01", "type": "plot|conflict|item|mystery|character", "description": "伏笔描述"}],
  "evidence_notes": [{"title": "线索标题", "kind": "clue|document|observation", "content": "章节中出现的具体细节、证据或文本内容"}],
  "summary": {"title": "章节标题", "characters_appeared": "角色1,角色2", "key_events": "关键事件", "state_changes": "状态变化", "hook_activity": "伏笔动态", "mood": "情绪基调", "chapter_type": "过渡|冲突|高潮|收束"}
}

## durable_facts 规则
- 这里只保留中长期有效、未来多章仍应成立的事实
- 只允许 5 类：identity、resource、item、rule、relationship
- 当前状态类信息（当前位置、主角状态、当前目标、当前限制、当前敌我、当前冲突）不要写入 durable_facts，这些由状态卡单独维护
- 章节细节类信息不要写入 durable_facts，例如：某页古书的具体内容、屋内气味、窗外脚步声、光线、某句原文、暂时性观察
- 这类细节若有价值，应写入 evidence_notes；若形成持续未解问题，应写入 hooks
- 如果无法确定一条信息能持续至少 3 章，就不要放进 durable_facts

## 章节输出
` + rawOutput

	result, err := p.llm.Chat(ctx, modelID, "你是小说设定提取专家，只返回JSON。", []llm.AgentMessage{
		{Role: "user", Content: extractPrompt},
	}, 0.3)
	if err != nil {
		p.saveDebugTrace(bookID, chapterNumber, "extract_llm_error", map[string]any{
			"error": err.Error(),
		})
		return
	}
	p.saveDebugTrace(bookID, chapterNumber, "extract_llm_done", map[string]any{
		"result_len": len(result),
	})

	jsonStr := extractJSON(result)
	if jsonStr == "" {
		p.saveDebugTrace(bookID, chapterNumber, "extract_json_missing", map[string]any{
			"result_preview": clipText(result, 400),
		})
		return
	}

	var extracted struct {
		Characters []struct {
			Name     string `json:"name"`
			RoleType string `json:"role_type"`
			Profile  string `json:"profile"`
		} `json:"characters"`
		DurableFacts []struct {
			Subject   string `json:"subject"`
			Predicate string `json:"predicate"`
			Object    string `json:"object"`
			Category  string `json:"category"`
		} `json:"durable_facts"`
		Hooks []struct {
			HookID      string `json:"hook_id"`
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"hooks"`
		EvidenceNotes []struct {
			Title   string `json:"title"`
			Kind    string `json:"kind"`
			Content string `json:"content"`
		} `json:"evidence_notes"`
		Summary struct {
			Title              string `json:"title"`
			CharactersAppeared string `json:"characters_appeared"`
			KeyEvents          string `json:"key_events"`
			StateChanges       string `json:"state_changes"`
			HookActivity       string `json:"hook_activity"`
			Mood               string `json:"mood"`
			ChapterType        string `json:"chapter_type"`
		} `json:"summary"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &extracted); err != nil {
		p.saveDebugTrace(bookID, chapterNumber, "extract_json_parse_error", map[string]any{
			"error": err.Error(),
		})
		return
	}

	for _, c := range extracted.Characters {
		if c.Name == "" {
			continue
		}
		rt := model.CharacterMinor
		if c.RoleType == "protagonist" {
			rt = model.CharacterProtagonist
		} else if c.RoleType == "major" {
			rt = model.CharacterMajor
		}
		_ = p.truth.SaveCharacter(&model.Character{
			BookID:          bookID,
			Name:            c.Name,
			RoleType:        rt,
			Profile:         c.Profile,
			SourceChapter:   chapterNumber,
			LastSeenChapter: chapterNumber,
		})
	}

	for _, f := range extracted.DurableFacts {
		if f.Subject == "" {
			continue
		}
		_ = p.truth.SaveFact(&model.Fact{
			BookID:           bookID,
			Subject:          f.Subject,
			Predicate:        f.Predicate,
			Object:           f.Object,
			Category:         f.Category,
			ValidFromChapter: chapterNumber,
			SourceChapter:    chapterNumber,
		})
	}

	if len(extracted.EvidenceNotes) > 0 {
		evidenceJSON, _ := json.Marshal(extracted.EvidenceNotes)
		_ = p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
			BookID:        bookID,
			ChapterNumber: chapterNumber,
			ArtifactType:  model.ArtifactEvidence,
			Content:       string(evidenceJSON),
		})
	}

	if opts.SaveHooks {
		for _, h := range extracted.Hooks {
			if h.HookID == "" {
				continue
			}
			_ = p.truth.SaveHook(&model.Hook{
				BookID:       bookID,
				HookID:       h.HookID,
				StartChapter: chapterNumber,
				Type:         normalizeHookType(h.Type),
				Status:       model.HookSeed,
				Notes:        h.Description,
			})
		}
	}

	if opts.SaveSummary && extracted.Summary.KeyEvents != "" {
		_ = p.truth.SaveChapterSummary(&model.ChapterSummary{
			BookID:             bookID,
			ChapterNumber:      chapterNumber,
			Title:              extracted.Summary.Title,
			CharactersAppeared: extracted.Summary.CharactersAppeared,
			KeyEvents:          extracted.Summary.KeyEvents,
			StateChanges:       extracted.Summary.StateChanges,
			HookActivity:       extracted.Summary.HookActivity,
			Mood:               extracted.Summary.Mood,
			ChapterType:        extracted.Summary.ChapterType,
		})
	}

	p.saveDebugTrace(bookID, chapterNumber, "extract_done", map[string]any{
		"characters": len(extracted.Characters),
		"facts":      len(extracted.DurableFacts),
		"hooks":      len(extracted.Hooks),
		"evidence":   len(extracted.EvidenceNotes),
	})
}

func (p *Pipeline) saveChapterSnapshot(bookID uint, chapterNumber uint, sections map[string]string) {
	characters, _ := p.truth.GetCharacters(bookID)
	facts, _ := p.truth.GetFacts(bookID)
	hooks, _ := p.truth.GetHooks(bookID)
	summaries, _ := p.truth.GetChapterSummaries(bookID)
	bookState, _ := p.truth.GetBookState(bookID)

	currentState := map[string]interface{}{
		"chapter_number":   chapterNumber,
		"characters_count": len(characters),
		"facts_count":      len(facts),
		"hooks_count":      len(hooks),
		"summaries_count":  len(summaries),
	}
	if stateSection := sections["UPDATED_STATE"]; stateSection != "" {
		currentState["writer_state"] = stateSection
	}
	currentStateJSON, _ := json.Marshal(currentState)
	bookStateJSON, _ := json.Marshal(bookState)
	charactersJSON, _ := json.Marshal(characters)
	factsJSON, _ := json.Marshal(facts)

	hooksJSON, _ := json.Marshal(hooks)
	summariesJSON, _ := json.Marshal(summaries)

	manifest := map[string]interface{}{
		"book_id":            bookID,
		"chapter_number":     chapterNumber,
		"characters_count":   len(characters),
		"facts_count":        len(facts),
		"active_hooks_count": countActiveHooks(hooks),
		"total_hooks_count":  len(hooks),
		"sections_saved":     []string{"UPDATED_STATE", "UPDATED_HOOKS", "CHAPTER_SUMMARY"},
	}
	manifestJSON, _ := json.Marshal(manifest)

	_ = p.truth.SaveChapterSnapshot(&model.ChapterSnapshot{
		BookID:               bookID,
		ChapterNumber:        chapterNumber,
		CurrentStateJSON:     string(currentStateJSON),
		BookStateJSON:        string(bookStateJSON),
		CharactersJSON:       string(charactersJSON),
		FactsJSON:            string(factsJSON),
		HooksJSON:            string(hooksJSON),
		ChapterSummariesJSON: string(summariesJSON),
		ManifestJSON:         string(manifestJSON),
	})
}

func (p *Pipeline) seedFoundations(book *model.Book, sections map[string]string) error {
	foundations := []model.BookFoundation{
		{
			BookID:   book.ID,
			FileType: model.FoundationStoryFrame,
			Content:  strings.TrimSpace(sections["story_frame"]),
		},
		{
			BookID:   book.ID,
			FileType: model.FoundationVolumeMap,
			Content:  strings.TrimSpace(sections["volume_map"]),
		},
		{
			BookID:   book.ID,
			FileType: model.FoundationBookRules,
			Content:  strings.TrimSpace(sections["book_rules"]),
		},
		{
			BookID:   book.ID,
			FileType: model.FoundationAuthorIntent,
			Content:  buildAuthorIntent(book),
		},
		{
			BookID:   book.ID,
			FileType: model.FoundationStyleGuide,
			Content:  buildStyleGuide(book, sections),
		},
		{
			BookID:   book.ID,
			FileType: model.FoundationCurrentFocus,
			Content:  buildCurrentFocus(book),
		},
	}

	for _, foundation := range foundations {
		if strings.TrimSpace(foundation.Content) == "" {
			continue
		}
		f := foundation
		if err := p.truth.SaveFoundation(&f); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pipeline) seedInitialCharacters(bookID uint, rolesSection string, bookRules string) {
	protagonistName := extractProtagonistName(bookRules)
	for i, role := range parseRoleBlocks(rolesSection) {
		if role.Name == "" {
			continue
		}

		name := strings.TrimSpace(role.Name)
		roleType := normalizeRoleType(role.Tier)
		if i == 0 || strings.Contains(role.Body, "## 主角弧线") {
			roleType = model.CharacterProtagonist
		}
		if roleType == model.CharacterProtagonist && isPlaceholderRoleName(name) && !isPlaceholderRoleName(protagonistName) {
			name = protagonistName
		}

		parts := parseMarkdownSections(role.Body)
		profile := buildRoleProfileSummary(parts)
		if profile == "" {
			profile = clipText(role.Body, 2000)
		}

		_ = p.truth.SaveCharacter(&model.Character{
			BookID:              bookID,
			Name:                name,
			RoleType:            roleType,
			CoreTags:            parts["核心标签"],
			ContrastDetails:     parts["反差细节"],
			Backstory:           parts["人物小传（过往经历）"],
			CharacterArc:        parts["主角弧线（起点 → 终点 → 代价）"],
			CurrentStatus:       firstNonEmpty(parts["当前现状（第 0 章初始状态）"], parts["当前现状"]),
			RelationshipNetwork: parts["关系网络"],
			InnerDrive:          parts["内在驱动"],
			GrowthArc:           parts["成长弧光"],
			Profile:             profile,
			IsPlaceholder:       isPlaceholderRoleName(name),
			SourceChapter:       0,
			LastSeenChapter:     0,
		})
	}
}

func (p *Pipeline) seedInitialHooks(bookID uint, hookSection string) {
	for _, row := range parseMarkdownTableRows(hookSection) {
		if len(row) < 4 {
			continue
		}

		hookID := strings.TrimSpace(row[0])
		if hookID == "" || strings.EqualFold(hookID, "hook_id") {
			continue
		}

		startChapter := parseUintCell(cellAt(row, 1))
		lastAdvanced := parseUintCell(cellAt(row, 4))
		payoffVolume := parseOptionalUint(cellAt(row, 10))

		hook := model.Hook{
			BookID:              bookID,
			HookID:              hookID,
			StartChapter:        startChapter,
			Type:                normalizeHookType(cellAt(row, 2)),
			Status:              normalizeHookStatus(cellAt(row, 3)),
			LastAdvancedChapter: lastAdvanced,
			ExpectedPayoff:      strings.TrimSpace(cellAt(row, 5)),
			PayoffTiming:        normalizePayoffTiming(cellAt(row, 6)),
			UpstreamDependency:  strings.TrimSpace(cellAt(row, 7)),
			IsCritical:          parseBoolCell(cellAt(row, 9)),
			HalfLife:            payoffVolume,
			Notes:               strings.TrimSpace(cellAt(row, len(row)-1)),
		}
		_ = p.truth.SaveHook(&hook)
	}
}

func (p *Pipeline) saveInitialBookState(book *model.Book, sections map[string]string) {
	characters, _ := p.truth.GetCharacters(book.ID)
	protagonist := ""
	currentStatus := ""
	for _, c := range characters {
		if c.RoleType == model.CharacterProtagonist {
			protagonist = c.Name
			currentStatus = c.CurrentStatus
			break
		}
	}

	state := &model.BookState{
		BookID:           book.ID,
		CurrentChapter:   0,
		ProtagonistName:  protagonist,
		SituationSummary: clipText(firstNonEmpty(currentStatus, sections["story_frame"]), 500),
		CurrentGoal:      "完成开篇三章，建立主线冲突与读者期待",
		ProtagonistState: clipText(currentStatus, 240),
		SourceChapter:    0,
	}
	state.SituationSummary = buildSituationSummary(state)
	_ = p.truth.SaveBookState(state)
}

func (p *Pipeline) saveInitialSnapshot(book *model.Book, sections map[string]string) {
	characters, _ := p.truth.GetCharacters(book.ID)
	facts, _ := p.truth.GetFacts(book.ID)
	hooks, _ := p.truth.GetHooks(book.ID)
	bookState, _ := p.truth.GetBookState(book.ID)

	protagonist := ""
	for _, c := range characters {
		if c.RoleType == model.CharacterProtagonist {
			protagonist = c.Name
			break
		}
	}

	currentState := map[string]interface{}{
		"phase":              "initialized",
		"book_title":         book.Title,
		"protagonist":        protagonist,
		"world_anchor":       clipText(strings.TrimSpace(sections["story_frame"]), 300),
		"characters_count":   len(characters),
		"active_hooks_count": countActiveHooks(hooks),
	}
	if bookState != nil {
		currentState["current_chapter"] = bookState.CurrentChapter
		currentState["situation_summary"] = bookState.SituationSummary
		currentState["current_location"] = bookState.CurrentLocation
		currentState["protagonist_state"] = bookState.ProtagonistState
		currentState["current_goal"] = bookState.CurrentGoal
		currentState["current_constraint"] = bookState.CurrentConstraint
		currentState["current_alliances"] = bookState.CurrentAlliances
		currentState["current_conflict"] = bookState.CurrentConflict
	} else {
		currentState["current_goal"] = "完成开篇三章，建立主线冲突与读者期待"
	}
	currentStateJSON, _ := json.Marshal(currentState)
	bookStateJSON, _ := json.Marshal(bookState)
	charactersJSON, _ := json.Marshal(characters)
	factsJSON, _ := json.Marshal(facts)
	hooksJSON, _ := json.Marshal(hooks)
	summariesJSON, _ := json.Marshal([]model.ChapterSummary{})
	manifestJSON, _ := json.Marshal(map[string]interface{}{
		"book_id":            book.ID,
		"chapter_number":     0,
		"initialized":        true,
		"characters_count":   len(characters),
		"total_hooks_count":  len(hooks),
		"active_hooks_count": countActiveHooks(hooks),
		"foundations": []string{
			string(model.FoundationStoryFrame),
			string(model.FoundationVolumeMap),
			string(model.FoundationBookRules),
			string(model.FoundationAuthorIntent),
			string(model.FoundationStyleGuide),
			string(model.FoundationCurrentFocus),
		},
	})

	_ = p.truth.SaveChapterSnapshot(&model.ChapterSnapshot{
		BookID:               book.ID,
		ChapterNumber:        0,
		CurrentStateJSON:     string(currentStateJSON),
		BookStateJSON:        string(bookStateJSON),
		CharactersJSON:       string(charactersJSON),
		FactsJSON:            string(factsJSON),
		HooksJSON:            string(hooksJSON),
		ChapterSummariesJSON: string(summariesJSON),
		ManifestJSON:         string(manifestJSON),
	})
}

func countActiveHooks(hooks []model.Hook) int {
	count := 0
	for _, h := range hooks {
		if h.Status != model.HookResolved && h.Status != model.HookStale {
			count++
		}
	}
	return count
}

func (p *Pipeline) buildContext(ctx context.Context, bookID uint, chapterNumber uint) (string, error) {
	var b strings.Builder

	foundations := []model.FoundationFileType{
		model.FoundationStoryFrame,
		model.FoundationVolumeMap,
		model.FoundationBookRules,
		model.FoundationAuthorIntent,
		model.FoundationStyleGuide,
		model.FoundationCurrentFocus,
	}
	for _, ft := range foundations {
		f, err := p.truth.GetFoundation(bookID, ft)
		if err == nil && f != nil && f.Content != "" {
			b.WriteString(fmt.Sprintf("## %s\n%s\n\n", ft, f.Content))
		}
	}

	if state, err := p.truth.GetBookState(bookID); err == nil && state != nil {
		b.WriteString("## 当前状态\n")
		if state.ProtagonistName != "" {
			b.WriteString(fmt.Sprintf("- 主角：%s\n", state.ProtagonistName))
		}
		if state.SituationSummary != "" {
			b.WriteString(fmt.Sprintf("- 状态摘要：%s\n", state.SituationSummary))
		}
		if state.CurrentLocation != "" {
			b.WriteString(fmt.Sprintf("- 当前位置：%s\n", state.CurrentLocation))
		}
		if state.ProtagonistState != "" {
			b.WriteString(fmt.Sprintf("- 主角状态：%s\n", state.ProtagonistState))
		}
		if state.CurrentGoal != "" {
			b.WriteString(fmt.Sprintf("- 当前目标：%s\n", state.CurrentGoal))
		}
		if state.CurrentConstraint != "" {
			b.WriteString(fmt.Sprintf("- 当前限制：%s\n", state.CurrentConstraint))
		}
		if state.CurrentAlliances != "" {
			b.WriteString(fmt.Sprintf("- 当前敌我：%s\n", state.CurrentAlliances))
		}
		if state.CurrentConflict != "" {
			b.WriteString(fmt.Sprintf("- 当前冲突：%s\n", state.CurrentConflict))
		}
		b.WriteString("\n")
	}

	characters, _ := p.truth.GetCharacters(bookID)
	if len(characters) > 0 {
		b.WriteString("## 人物\n")
		for _, c := range characters {
			b.WriteString(fmt.Sprintf("- %s (%s): %s\n", c.Name, c.RoleType, buildCharacterContext(c)))
		}
		b.WriteString("\n")
	}

	facts, _ := p.truth.GetFacts(bookID)
	if len(facts) > 0 {
		b.WriteString("## 已确立事实\n")
		for _, f := range facts {
			b.WriteString(fmt.Sprintf("- %s %s %s (自第%d章)\n", f.Subject, f.Predicate, f.Object, f.ValidFromChapter))
		}
		b.WriteString("\n")
	}

	hooks, _ := p.truth.GetHooks(bookID)
	if len(hooks) > 0 {
		b.WriteString("## 活跃伏笔\n")
		for _, h := range hooks {
			if h.Status != model.HookResolved && h.Status != model.HookStale {
				b.WriteString(fmt.Sprintf("- %s [%s] 状态:%s 始于第%d章\n", h.HookID, h.Type, h.Status, h.StartChapter))
			}
		}
		b.WriteString("\n")
	}

	summaries, _ := p.truth.GetChapterSummaries(bookID)
	if len(summaries) > 0 {
		b.WriteString("## 最近章节摘要\n")
		start := 0
		if len(summaries) > 5 {
			start = len(summaries) - 5
		}
		for _, s := range summaries[start:] {
			b.WriteString(fmt.Sprintf("- 第%d章 %s: %s\n", s.ChapterNumber, s.Title, s.KeyEvents))
		}
		b.WriteString("\n")
	}

	prevChapter, err := p.truth.GetChapter(bookID, chapterNumber-1)
	if err == nil && prevChapter != nil {
		b.WriteString("## 上一章结尾\n")
		runes := []rune(prevChapter.Content)
		if len(runes) > 500 {
			runes = runes[len(runes)-500:]
		}
		b.WriteString(string(runes))
		b.WriteString("\n")
	}

	return b.String(), nil
}

type roleBlock struct {
	Tier string
	Name string
	Body string
}

type roleAnchorResponse struct {
	ProtagonistName string `json:"protagonist_name"`
	MajorAntagonist string `json:"major_antagonist_name"`
	MajorAllyName   string `json:"major_ally_name"`
}

func parseSections(raw string) map[string]string {
	raw = normalizeEscapedMultiline(raw)
	sections := make(map[string]string)
	// Support bare section names like `CHAPTER_TITLE`, markdown headings like
	// `# CHAPTER_CONTENT`, and the stricter `# === CHAPTER_CONTENT ===` form.
	re := regexp.MustCompile(`(?m)^\s*(?:#+\s*)?(?:===\s*)?([A-Z][A-Z0-9_]+)(?:\s*===)?\s*$`)
	matches := re.FindAllStringSubmatchIndex(raw, -1)

	for i, match := range matches {
		name := raw[match[2]:match[3]]
		contentStart := match[1]
		var contentEnd int
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		} else {
			contentEnd = len(raw)
		}
		sections[name] = strings.TrimSpace(raw[contentStart:contentEnd])
	}

	return sections
}

func normalizeEscapedMultiline(raw string) string {
	if strings.Contains(raw, "\n") {
		return raw
	}
	if strings.Contains(raw, `\r\n`) {
		raw = strings.ReplaceAll(raw, `\r\n`, "\n")
	}
	if strings.Contains(raw, `\n`) {
		raw = strings.ReplaceAll(raw, `\n`, "\n")
	}
	if strings.Contains(raw, `\t`) {
		raw = strings.ReplaceAll(raw, `\t`, "\t")
	}
	return raw
}

func fallbackWriterNarrative(raw string) (string, string) {
	raw = strings.TrimSpace(normalizeEscapedMultiline(raw))
	if raw == "" {
		return "", ""
	}

	lines := strings.Split(raw, "\n")
	firstContentLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContentLine = i
			break
		}
	}
	if firstContentLine == -1 {
		return "", ""
	}

	var title string
	start := firstContentLine
	firstLine := strings.TrimSpace(lines[firstContentLine])
	if strings.HasPrefix(firstLine, "#") {
		title = strings.TrimSpace(strings.TrimLeft(firstLine, "#"))
		title = regexp.MustCompile(`^第[0-9零一二三四五六七八九十百千两]+章[：:\s-]*`).ReplaceAllString(title, "")
		start = firstContentLine + 1
		for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
			start++
		}
	}

	content := strings.TrimSpace(strings.Join(lines[start:], "\n"))
	if content == "" {
		return title, ""
	}
	return title, content
}

func sortedSectionNames(sections map[string]string) []string {
	names := make([]string, 0, len(sections))
	for name := range sections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func parseArchitectSections(raw string) map[string]string {
	sections := make(map[string]string)
	re := regexp.MustCompile(`(?m)^=== SECTION: ([a-z_]+) ===\s*$`)
	matches := re.FindAllStringSubmatchIndex(raw, -1)

	for i, match := range matches {
		name := raw[match[2]:match[3]]
		contentStart := match[1]
		contentEnd := len(raw)
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		}
		sections[name] = strings.TrimSpace(raw[contentStart:contentEnd])
	}

	return sections
}

func parseRoleBlocks(raw string) []roleBlock {
	chunks := strings.Split(raw, "---ROLE---")
	roles := make([]roleBlock, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}

		parts := strings.SplitN(chunk, "---CONTENT---", 2)
		if len(parts) != 2 {
			continue
		}

		meta := strings.TrimSpace(parts[0])
		body := strings.TrimSpace(parts[1])
		var role roleBlock

		for _, line := range strings.Split(meta, "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "tier:"):
				role.Tier = strings.TrimSpace(strings.TrimPrefix(line, "tier:"))
			case strings.HasPrefix(line, "name:"):
				role.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			}
		}

		if role.Name == "" {
			continue
		}
		role.Body = body
		roles = append(roles, role)
	}
	return roles
}

func (p *Pipeline) ensureIdentityAnchors(ctx context.Context, book *model.Book, sections map[string]string, modelID uint) map[string]string {
	rolesSection := sections["roles"]
	bookRules := sections["book_rules"]
	roleBlocks := parseRoleBlocks(rolesSection)
	if len(roleBlocks) == 0 {
		return sections
	}

	protagonistName := extractProtagonistName(bookRules)
	needsRepair := isPlaceholderRoleName(protagonistName)
	for i, role := range roleBlocks {
		isLead := i == 0 || strings.Contains(role.Body, "## 主角弧线")
		if isLead && isPlaceholderRoleName(role.Name) {
			needsRepair = true
			break
		}
	}
	if !needsRepair {
		return sections
	}

	anchors, err := p.generateRoleAnchors(ctx, book, sections, modelID)
	if err != nil {
		return sections
	}
	sections["roles"] = replacePlaceholderRoleNames(rolesSection, anchors)
	if !isPlaceholderRoleName(anchors.ProtagonistName) {
		sections["book_rules"] = replaceProtagonistNameInBookRules(bookRules, anchors.ProtagonistName)
	}
	return sections
}

func (p *Pipeline) generateRoleAnchors(ctx context.Context, book *model.Book, sections map[string]string, fallbackModelID uint) (roleAnchorResponse, error) {
	modelID := p.resolveModelID(book.ID, "architect", fallbackModelID)
	prompt := fmt.Sprintf(`你正在修复小说初始化设定中的占位角色名。

小说标题：%s
简介：%s

## story_frame
%s

## roles
%s

请只返回 JSON：
{
  "protagonist_name": "真实主角名",
  "major_antagonist_name": "主要对手名",
  "major_ally_name": "主要协作者名"
}

要求：
1. 必须是具体可用的人名或称号，不能再返回“主角”“主要对手”“主要协作者”
2. 要符合题材和故事气质
3. 只返回 JSON，不要解释`, book.Title, defaultString(book.Description, "暂无额外简介"), clipText(sections["story_frame"], 1000), clipText(sections["roles"], 1500))

	raw, err := p.llm.Chat(ctx, modelID, "你是小说角色命名编辑，只返回 JSON。", []llm.AgentMessage{
		{Role: "user", Content: prompt},
	}, 0.2)
	if err != nil {
		return roleAnchorResponse{}, err
	}

	var anchors roleAnchorResponse
	if err := json.Unmarshal([]byte(extractJSON(raw)), &anchors); err != nil {
		return roleAnchorResponse{}, err
	}
	return anchors, nil
}

func parseMarkdownSections(raw string) map[string]string {
	sections := make(map[string]string)
	re := regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	matches := re.FindAllStringSubmatchIndex(raw, -1)
	for i, match := range matches {
		name := strings.TrimSpace(raw[match[2]:match[3]])
		start := match[1]
		end := len(raw)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		sections[name] = strings.TrimSpace(raw[start:end])
	}
	return sections
}

func buildRoleProfileSummary(parts map[string]string) string {
	chunks := []string{
		strings.TrimSpace(parts["核心标签"]),
		strings.TrimSpace(parts["反差细节"]),
		strings.TrimSpace(firstNonEmpty(parts["当前现状（第 0 章初始状态）"], parts["当前现状"])),
		strings.TrimSpace(parts["内在驱动"]),
	}
	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != "" {
			out = append(out, chunk)
		}
	}
	return clipText(strings.Join(out, "\n\n"), 2000)
}

func buildCharacterContext(c model.Character) string {
	parts := []string{
		strings.TrimSpace(c.CoreTags),
		strings.TrimSpace(c.CurrentStatus),
		strings.TrimSpace(c.InnerDrive),
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return c.Profile
	}
	return clipText(strings.Join(filtered, "；"), 500)
}

func extractProtagonistName(bookRules string) string {
	re := regexp.MustCompile(`(?s)protagonist:\s*.*?name:\s*"?([^\n"]+)"?`)
	match := re.FindStringSubmatch(bookRules)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func replaceProtagonistNameInBookRules(bookRules, protagonistName string) string {
	if strings.TrimSpace(protagonistName) == "" {
		return bookRules
	}
	re := regexp.MustCompile(`(?m)^(\s*name:\s*).*$`)
	lines := strings.Split(bookRules, "\n")
	inProtagonist := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "protagonist:" {
			inProtagonist = true
			continue
		}
		if inProtagonist {
			if strings.HasPrefix(trimmed, "name:") {
				lines[i] = re.ReplaceAllString(line, `${1}"`+protagonistName+`"`)
				break
			}
			if trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func replacePlaceholderRoleNames(rolesSection string, anchors roleAnchorResponse) string {
	replacements := map[string]string{
		"主角":    strings.TrimSpace(anchors.ProtagonistName),
		"主要对手":  strings.TrimSpace(anchors.MajorAntagonist),
		"主要协作者": strings.TrimSpace(anchors.MajorAllyName),
		"主要盟友":  strings.TrimSpace(anchors.MajorAllyName),
	}
	for placeholder, name := range replacements {
		if isPlaceholderRoleName(name) || name == "" {
			continue
		}
		rolesSection = strings.Replace(rolesSection, "name: "+placeholder, "name: "+name, 1)
	}
	return rolesSection
}

func isPlaceholderRoleName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	placeholders := []string{"主角", "主要对手", "主要协作者", "主要盟友", "次要角色", "<角色名>", "<下一个主要角色>", "<次要角色名>"}
	for _, placeholder := range placeholders {
		if name == placeholder {
			return true
		}
	}
	return strings.Contains(name, "<") || strings.Contains(name, ">")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseMarkdownTableRows(raw string) [][]string {
	lines := strings.Split(raw, "\n")
	rows := make([][]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}

		cells := splitMarkdownRow(line)
		if len(cells) == 0 {
			continue
		}

		allSeparator := true
		for _, cell := range cells {
			trimmed := strings.TrimSpace(cell)
			if strings.Trim(trimmed, "-: ") != "" {
				allSeparator = false
				break
			}
		}
		if allSeparator {
			continue
		}

		rows = append(rows, cells)
	}
	return rows
}

func splitMarkdownRow(line string) []string {
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func extractJSON(s string) string {
	re := regexp.MustCompile("(?s)\\{.*\\}")
	match := re.FindString(s)
	return match
}

func buildFallbackArchitectSections(book *model.Book) map[string]string {
	storyFrame := fmt.Sprintf(`## 主题与基调
《%s》是一部面向%s平台读者的%s故事。故事的起点来自“%s”，整本书要持续放大这个核心矛盾，让主角在一次次选择里暴露欲望、代价和底线。整体基调强调可读性、推进感与连续钩子，优先保证读者能快速进入主线。

## 核心冲突
前台故事是主角围绕当前目标展开的连续行动，后台故事是隐藏在主线背后的更大因果链。每一卷都要让前台矛盾解决一层，同时把后台真相再撬开一点，避免写成单元事件堆砌。

## 世界观底色
这本书的世界规则必须稳定，人物做出的每个决定都要符合既有设定与自身利益。环境描写服务冲突，情绪描写服务人物，所有伏笔都要能在后续章节里推进或回收。

## 终局方向
全书最终要把主角送到一个可以被外部观察者确认的终局位置，同时兑现开篇埋下的主问题。当前先聚焦前 3 章：建立主角、主线冲突和第一组核心期待。`, book.Title, defaultString(book.Platform.Name, "目标"), defaultString(book.Genre.Name, "网络小说"), defaultString(book.Description, "主角被卷入一场必须解决的困局"))

	volumeMap := fmt.Sprintf(`## 卷一：开局起盘
卷一的任务是让读者迅速理解主角当前处境、最迫切目标和主要阻力，至少建立一条主线与一条暗线。前三章必须给出可见的短期目标，并在章尾持续留钩。

## 卷二：冲突升级
在完成开局立人之后，卷二要扩大主角的行动半径，让代价与收益同时提升。新角色和新资源必须为主线服务，不能变成散点信息。

## 卷三以后：持续放大
后续每一卷都围绕主线目标拆解阶段成果，让前台故事和后台故事同时推进。高潮之后必须写出影响，不能上一章爆发、下一章一切如常。

## 节奏原则
每 3 到 5 章推进一个小目标，每章结尾都要保留继续阅读的理由。日常段落必须承担功能，不能做纯填充。`)

	bookRules := fmt.Sprintf(`---
version: "1.0"
protagonist:
  name: "主角"
  personalityLock: ["目标明确", "行动驱动", "有代价意识"]
  behavioralConstraints: ["不做无因决策", "不脱离世界规则", "不无故降智"]
genreLock:
  primary: "%s"
  forbidden: ["无关文风混入", "脱离平台节奏", "纯说明文叙事"]
prohibitions:
  - "禁止角色行为脱离自身利益"
  - "禁止用旁白替代冲突"
  - "禁止无回收伏笔堆积"
chapterTypesOverride: []
fatigueWordsOverride: []
additionalAuditDimensions: []
enableFullCastTracking: true
---`, defaultString(book.Genre.Name, "通用网文"))

	roles := `---ROLE---
tier: major
name: 主角
---CONTENT---
## 核心标签
这是故事当前的第一视角核心人物，他的行动会直接决定主线推进速度与读者情绪节奏。

## 反差细节
主角表面上被现实压力推动，但内心始终保留一条不会轻易妥协的底线，这会成为后续冲突的核心来源。

## 人物小传（过往经历）
主角在故事开始前已经积累了一段足以塑造性格的过去经历，这段过去决定了他面对风险时优先保护什么、优先放弃什么。

## 主角弧线（起点 → 终点 → 代价）
主角会从被动应对局势的人，成长为能主动定义局势的人。为了完成这次位移，他必须付出关系、资源或认知上的真实代价。

## 当前现状（第 0 章初始状态）
故事开场时，主角正处于一段需要尽快破局的处境里，短期目标明确，但可用资源有限。

## 关系网络
主角与周围人的关系尚未稳定，合作和对立都会随着主线推进快速变化。

## 内在驱动
他真正想要的不是抽象的成功，而是解决眼下最现实、最压迫他的那个问题。

## 成长弧光
随着剧情推进，主角必须在保留底线和争取结果之间不断做出选择。

---ROLE---
tier: major
name: 主要对手
---CONTENT---
## 核心标签
这名角色是前期最主要的阻力来源，他代表主角当前阶段无法绕开的外部压力。

## 反差细节
对手并不是为了作恶而作恶，他有自己的利益与逻辑。

## 当前现状
对手已经占据某种优势地位，并准备在主线冲突里率先出手。

## 与主角关系
双方的关系会随着信息与筹码变化不断升级。`

	pendingHooks := `| hook_id | 起始章节 | 类型 | 状态 | 最近推进 | 预期回收 | 回收节奏 | 上游依赖 | 回收卷 | 核心 | 半衰期 | 备注 |
|---------|----------|------|------|----------|----------|----------|----------|--------|------|--------|------|
| H001 | 0 | plot | seed | 0 | 揭开开篇主问题的第一层真相 | near-term | 无 | 第1卷中段 | true | 10 | 开篇主线钩子 |
| H002 | 0 | mystery | seed | 0 | 曝光幕后压力来源 | slow-burn | H001 | 第2卷后段 | true | 30 | 后台故事初始暗桩 |`

	return map[string]string{
		"story_frame":   storyFrame,
		"volume_map":    volumeMap,
		"book_rules":    bookRules,
		"roles":         roles,
		"pending_hooks": pendingHooks,
	}
}

func buildAuthorIntent(book *model.Book) string {
	if strings.TrimSpace(book.Description) != "" {
		return fmt.Sprintf("## 创作目标\n%s\n\n## 当前优先级\n先完成开篇三章的角色立住、主线冲突落地和持续追读钩子。", strings.TrimSpace(book.Description))
	}
	return "## 创作目标\n本书以持续主线推进、明确人物动机和稳定节奏为第一优先级。\n\n## 当前优先级\n先完成开篇三章的角色立住、主线冲突落地和持续追读钩子。"
}

func buildStyleGuide(book *model.Book, sections map[string]string) string {
	if strings.TrimSpace(book.Platform.StyleGuide) != "" {
		return strings.TrimSpace(book.Platform.StyleGuide)
	}
	if strings.TrimSpace(sections["story_frame"]) != "" {
		return "## 风格指南\n保持网文节奏感、强行动性和移动端可读性。\n\n" + clipText(strings.TrimSpace(sections["story_frame"]), 500)
	}
	return "## 风格指南\n保持网文节奏感、强行动性和移动端可读性。"
}

func buildCurrentFocus(book *model.Book) string {
	return fmt.Sprintf(`## 当前目标
- 完成《%s》的开篇三章
- 第一章建立主角处境与主线冲突
- 第二章让主角做出第一次关键行动
- 第三章给出清晰的短期目标与下一阶段钩子

## 执行约束
- 每章都要留下继续阅读的理由
- 日常段落必须承担功能
- 伏笔必须可追踪、可推进、可回收`, book.Title)
}

func normalizeRoleType(raw string) model.CharacterRoleType {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "protagonist", "主角":
		return model.CharacterProtagonist
	case "major", "主要", "重要":
		return model.CharacterMajor
	default:
		return model.CharacterMinor
	}
}

func normalizeHookType(raw string) model.HookType {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "conflict", "冲突":
		return model.HookConflict
	case "item", "道具", "物品":
		return model.HookItem
	case "mystery", "悬疑", "谜题":
		return model.HookMystery
	case "character", "人物":
		return model.HookCharacter
	default:
		return model.HookPlot
	}
}

func normalizeHookStatus(raw string) model.HookStatus {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "open", "开放":
		return model.HookOpen
	case "progressing", "推进中", "advanced":
		return model.HookProgressing
	case "resolved", "已回收", "回收":
		return model.HookResolved
	case "deferred", "延后":
		return model.HookDeferred
	case "stale", "过期":
		return model.HookStale
	default:
		return model.HookSeed
	}
}

func normalizePayoffTiming(raw string) model.PayoffTiming {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "immediate", "立即":
		return model.PayoffImmediate
	case "near-term", "近期":
		return model.PayoffNearTerm
	case "mid-arc", "mid-term", "中程", "中期":
		return model.PayoffMidTerm
	default:
		return model.PayoffSlowBurn
	}
}

func parseUintCell(raw string) uint {
	v, _ := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	return uint(v)
}

func defaultUint(v uint, fallback uint) uint {
	if v == 0 {
		return fallback
	}
	return v
}

func parseOptionalUint(raw string) *uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "无" || raw == "-" {
		return nil
	}
	v := parseUintCell(raw)
	if v == 0 {
		return nil
	}
	return &v
}

func parseBoolCell(raw string) bool {
	v := strings.ToLower(strings.TrimSpace(raw))
	return v == "true" || v == "yes" || v == "是" || v == "1"
}

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func clipText(raw string, max int) string {
	if len([]rune(raw)) <= max {
		return raw
	}
	return string([]rune(raw)[:max])
}

func nextGeneratedHookID(existing map[string]*model.Hook) string {
	re := regexp.MustCompile(`^H(\d+)$`)
	maxVal := 0
	for hookID := range existing {
		match := re.FindStringSubmatch(strings.TrimSpace(hookID))
		if len(match) != 2 {
			continue
		}
		n, err := strconv.Atoi(match[1])
		if err == nil && n > maxVal {
			maxVal = n
		}
	}
	return fmt.Sprintf("H%03d", maxVal+1)
}

func buildCurrentFocusDelta(chapterNumber uint, patch map[string]string) string {
	keys := []string{
		"currentLocation",
		"protagonistState",
		"currentGoal",
		"currentConstraint",
		"currentAlliances",
		"currentConflict",
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## 第 %d 章后当前焦点\n", chapterNumber))
	for _, key := range keys {
		value := strings.TrimSpace(patch[key])
		if value == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s：%s\n", key, value))
	}

	if b.Len() == 0 {
		return ""
	}
	return strings.TrimSpace(b.String())
}

func buildSituationSummary(state *model.BookState) string {
	if state == nil {
		return ""
	}
	parts := []string{
		strings.TrimSpace(state.ProtagonistState),
		strings.TrimSpace(state.CurrentLocation),
		strings.TrimSpace(state.CurrentGoal),
		strings.TrimSpace(state.CurrentConstraint),
		strings.TrimSpace(state.CurrentConflict),
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return strings.TrimSpace(state.SituationSummary)
	}
	return clipText(strings.Join(filtered, "；"), 500)
}

func buildAuditDrift(settlement string, notes []string) string {
	settlement = strings.TrimSpace(settlement)
	if settlement == "" && len(notes) == 0 {
		return ""
	}

	var b strings.Builder
	if settlement != "" {
		b.WriteString("## 本章结算摘要\n")
		b.WriteString(settlement)
		b.WriteString("\n")
	}
	if len(notes) > 0 {
		b.WriteString("\n## 结算备注\n")
		for _, note := range notes {
			note = strings.TrimSpace(note)
			if note == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(note)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

const plannerBuildInput = `你正在创作小说《%s》（题材：%s，平台：%s）。

这是第 %d 章，每章目标 %d 字。

## 当前上下文
%s

## 用户指令
%s

请为本章生成 chapter_memo。`

const writerBuildInput = `你正在创作小说《%s》。

这是第 %d 章。

## 本章规划 (chapter_memo)
%s

## 当前上下文
%s

请按照规划写出本章内容。

再次提醒：最终答案必须严格包含 ` + "`=== PRE_WRITE_CHECK ===`" + `、` + "`=== CHAPTER_TITLE ===`" + `、` + "`=== CHAPTER_CONTENT ===`" + ` 等 section。
禁止把最终答案写成 ` + "`# 第一章：标题`" + ` 开头的普通 Markdown 文章。
禁止在 ` + "`CHAPTER_CONTENT`" + ` 里重复输出标题。`
