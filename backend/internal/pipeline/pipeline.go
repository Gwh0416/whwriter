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
	"sync"

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
	llm        *llm.Client
	truth      repository.TruthFileRepository
	radar      repository.RadarRepository
	tokenUsage repository.TokenUsageRepository
	registry   *agent.Registry
	runMu      sync.Mutex
	runStops   map[uint]context.CancelFunc
}

func New(llmClient *llm.Client, truthRepo repository.TruthFileRepository, radarRepo repository.RadarRepository, tokenUsageRepo ...repository.TokenUsageRepository) *Pipeline {
	var rr repository.RadarRepository
	if radarRepo != nil {
		rr = radarRepo
	}
	var tokenUsage repository.TokenUsageRepository
	if len(tokenUsageRepo) > 0 {
		tokenUsage = tokenUsageRepo[0]
	}
	return &Pipeline{
		llm:        llmClient,
		truth:      truthRepo,
		radar:      rr,
		tokenUsage: tokenUsage,
		registry:   agent.NewRegistry(),
		runStops:   make(map[uint]context.CancelFunc),
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

type extractedEvidenceNote struct {
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

type extractedTruthFiles struct {
	Characters    []model.Character
	DurableFacts  []model.Fact
	Hooks         []model.Hook
	EvidenceNotes []extractedEvidenceNote
	Summary       *model.ChapterSummary
}

var allowedCurrentStatePatchKeys = map[string]struct{}{
	"currentLocation":   {},
	"protagonistState":  {},
	"currentGoal":       {},
	"currentConstraint": {},
	"currentAlliances":  {},
	"currentConflict":   {},
}

func emitProgress(w ProgressWriter, stage, msg string) {
	if w == nil {
		return
	}
	data, _ := json.Marshal(map[string]string{"stage": stage, "message": msg})
	w.Write([]byte("data: " + string(data) + "\n\n"))
	w.Flush()
}

func (p *Pipeline) withTruthRepo(truth repository.TruthFileRepository) *Pipeline {
	cloned := *p
	cloned.truth = truth
	return &cloned
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
	if err := p.truth.RefreshWikiGraph(book.ID); err != nil {
		return fmt.Errorf("build wiki graph: %w", err)
	}
	if err := p.truth.RefreshKnowledgeIndex(book.ID); err != nil {
		return fmt.Errorf("build knowledge index: %w", err)
	}
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

	contextPkg, err := p.buildContext(ctx, in.BookID, chapterNumber, in.UserInput)
	if err != nil {
		return nil, fmt.Errorf("build context: %w", err)
	}

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

	composed, err := p.composeChapterContext(book, chapterNumber, memo, in.UserInput, model.WriteRunTypeNormal, nil)
	if err != nil {
		return nil, fmt.Errorf("compose context: %w", err)
	}
	contextPkg = composed.ContextText

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
		return nil, fmt.Errorf("writer 输出缺少 CHAPTER_CONTENT，未写入章节正文")
	}

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
		"composer": map[string]interface{}{
			"context_package": composed.ContextPackage,
			"rule_stack":      composed.RuleStack,
			"trace":           composed.Trace,
		},
	}

	if err := emit("auditing", "Auditor 正在审查章节结构与连续性..."); err != nil {
		return nil, err
	}

	auditResult, auditRaw, err := p.runAuditor(ctx, book, chapterNumber, memo, contextPkg, content, in.ModelID)
	if err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}
	tracePayload["audit"] = map[string]interface{}{
		"raw_output": auditRaw,
		"result":     auditResult,
	}

	if !auditResult.Passed {
		if err := emit("revising", "Reviser 正在根据审稿意见修订章节..."); err != nil {
			return nil, err
		}
		revisedContent, revisedSections, reviserRaw, reviseErr := p.runReviser(ctx, book, chapterNumber, memo, contextPkg, content, auditRaw, in.ModelID)
		if reviseErr != nil {
			return nil, fmt.Errorf("revise: %w", reviseErr)
		}
		if strings.TrimSpace(revisedContent) == "" {
			return nil, fmt.Errorf("revise: reviser 未产出可用修订结果")
		}
		revisedContent = strings.TrimSpace(revisedContent)
		revisedAudit, revisedAuditRaw, auditErr := p.runAuditor(ctx, book, chapterNumber, memo, contextPkg, revisedContent, in.ModelID)
		var decision agent.ScoreDecision
		if auditErr != nil {
			decision = p.failedRevisionGateDecision(auditResult, "修订稿候选审计失败，保留原稿："+auditErr.Error())
		} else {
			var scoreErr error
			decision, scoreErr = p.decideRevisionGate(auditResult, revisedAudit)
			if scoreErr != nil {
				return nil, scoreErr
			}
		}
		tracePayload["reviser"] = map[string]interface{}{
			"raw_output":          reviserRaw,
			"sections":            sortedSectionNames(revisedSections),
			"fixed_issues":        revisedSections["FIXED_ISSUES"],
			"updated_state":       revisedSections["UPDATED_STATE"],
			"candidate_audit_raw": revisedAuditRaw,
			"evaluation":          decision,
		}
		if decision.Applied {
			content = revisedContent
			auditResult = revisedAudit
			auditRaw = revisedAuditRaw
			tracePayload["audit"] = map[string]interface{}{
				"raw_output": auditRaw,
				"result":     auditResult,
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
		}
	}

	if err := emit("polishing", "Polisher 正在润色正文..."); err != nil {
		return nil, err
	}

	polishedContent, polishErr := p.runPolisher(ctx, book, chapterNumber, content, in.ModelID)
	if polishErr != nil {
		return nil, fmt.Errorf("polish: %w", polishErr)
	}
	if strings.TrimSpace(polishedContent) == "" {
		return nil, fmt.Errorf("polish: polisher 未产出可用正文")
	}
	content = strings.TrimSpace(polishedContent)
	tracePayload["polisher"] = map[string]interface{}{
		"applied": true,
	}
	tracePayload["writer"].(map[string]interface{})["finalized_source"] = "polisher"

	ch := &model.Chapter{
		BookID:        in.BookID,
		ChapterNumber: chapterNumber,
		Title:         title,
		Content:       content,
		WordCount:     uint(len([]rune(content))),
		Status:        model.ChapterDraft,
	}
	if err := emit("extracting", "Settler 正在结算真相文件增量..."); err != nil {
		return nil, err
	}

	sections["CHAPTER_TITLE"] = title
	sections["CHAPTER_CONTENT"] = content

	settleSections, settleDelta, settlerRaw, settleErr := p.settleTruthFiles(ctx, book, chapterNumber, title, content, writerModelID)
	if settleErr != nil {
		return nil, fmt.Errorf("settle truth files: %w", settleErr)
	}
	for key, value := range settleSections {
		if strings.TrimSpace(value) != "" {
			sections[key] = value
		}
	}
	tracePayload["settler"] = map[string]any{
		"raw_output": settlerRaw,
		"delta":      settleDelta,
	}
	extractedTruth, extractErr := p.extractTruthFiles(
		ctx,
		in.BookID,
		chapterNumber,
		fmt.Sprintf("章节标题：%s\n\n章节正文：\n%s", title, content),
		writerModelID,
	)
	if extractErr != nil {
		return nil, fmt.Errorf("extract truth files: %w", extractErr)
	}
	if extractedTruth == nil {
		return nil, fmt.Errorf("extract truth files: empty result")
	}
	tracePayload["extract"] = map[string]any{
		"characters": len(extractedTruth.Characters),
		"facts":      len(extractedTruth.DurableFacts),
		"hooks":      len(extractedTruth.Hooks),
		"evidence":   len(extractedTruth.EvidenceNotes),
	}
	if err := emit("snapshot", "正在保存章节快照和运行时产物..."); err != nil {
		return nil, err
	}

	txErr := p.truth.WithinTx(func(txTruth repository.TruthFileRepository) error {
		txPipeline := p.withTruthRepo(txTruth)

		for _, artifact := range []*model.RuntimeArtifact{
			{
				BookID:        in.BookID,
				ChapterNumber: chapterNumber,
				ArtifactType:  model.ArtifactContext,
				Content:       contextPkg,
			},
			{
				BookID:        in.BookID,
				ChapterNumber: chapterNumber,
				ArtifactType:  model.ArtifactIntent,
				Content:       memo,
			},
			{
				BookID:        in.BookID,
				ChapterNumber: chapterNumber,
				ArtifactType:  model.ArtifactPlan,
				Content:       sections["PRE_WRITE_CHECK"],
			},
			{
				BookID:        in.BookID,
				ChapterNumber: chapterNumber,
				ArtifactType:  model.ArtifactRuleStack,
				Content:       marshalArtifactPayload(composed.RuleStack),
			},
		} {
			if err := txTruth.SaveRuntimeArtifact(artifact); err != nil {
				return err
			}
		}

		if err := txTruth.SaveChapter(ch); err != nil {
			return fmt.Errorf("save chapter: %w", err)
		}
		if err := txPipeline.applySettlerDelta(in.BookID, chapterNumber, title, sections["POST_SETTLEMENT"], settleDelta); err != nil {
			return fmt.Errorf("apply settler delta: %w", err)
		}
		txPipeline.saveDebugTrace(in.BookID, chapterNumber, "settler_done", map[string]any{
			"sections":  sortedSectionNames(settleSections),
			"has_delta": true,
		})

		txPipeline.saveDebugTrace(in.BookID, chapterNumber, "extract_start", map[string]any{
			"save_hooks":   false,
			"save_summary": true,
		})
		if err := txPipeline.persistExtractedTruthFiles(in.BookID, chapterNumber, extractedTruth, extractionOptions{
			SaveHooks:   false,
			SaveSummary: true,
		}); err != nil {
			return fmt.Errorf("persist extracted truth files: %w", err)
		}
		if err := txTruth.RefreshWikiGraph(in.BookID); err != nil {
			return fmt.Errorf("refresh wiki graph: %w", err)
		}
		if err := txTruth.RefreshKnowledgeIndex(in.BookID); err != nil {
			return fmt.Errorf("refresh knowledge index: %w", err)
		}
		txPipeline.saveDebugTrace(in.BookID, chapterNumber, "extract_done", map[string]any{
			"characters": len(extractedTruth.Characters),
			"facts":      len(extractedTruth.DurableFacts),
			"hooks":      len(extractedTruth.Hooks),
			"evidence":   len(extractedTruth.EvidenceNotes),
		})

		txPipeline.saveDebugTrace(in.BookID, chapterNumber, "snapshot_start", nil)
		txPipeline.saveChapterSnapshot(in.BookID, chapterNumber, sections)
		if payload, marshalErr := json.Marshal(tracePayload); marshalErr == nil {
			if err := txTruth.SaveRuntimeArtifact(&model.RuntimeArtifact{
				BookID:        in.BookID,
				ChapterNumber: chapterNumber,
				ArtifactType:  model.ArtifactTrace,
				Content:       string(payload),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, fmt.Errorf("persist chapter state transaction: %w", txErr)
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

## 硬性输出要求
必须输出完整的 === REVISED_CONTENT ===，内容为修订后的完整章节正文。不要输出 PATCH、diff、局部替换或解释文本。

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

	raw, err := p.llm.ChatForAgent(ctx, "polisher", modelID, polisherAny.SystemPrompt(), []llm.AgentMessage{
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

	return p.llm.ChatForAgent(ctx, "architect", modelID, systemPrompt, []llm.AgentMessage{
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
	if err := p.validateSettlerDelta(book.ID, chapterNumber, title, content, &delta); err != nil {
		return sections, delta, raw, fmt.Errorf("validate settler delta: %w", err)
	}

	if len(delta.CurrentStatePatch) > 0 {
		if payload, err := json.Marshal(delta.CurrentStatePatch); err == nil {
			sections["UPDATED_STATE"] = string(payload)
		}
	}

	return sections, delta, raw, nil
}

func (p *Pipeline) validateSettlerDelta(bookID uint, chapterNumber uint, title, content string, delta *settlerDelta) error {
	if delta == nil {
		return fmt.Errorf("delta is nil")
	}
	if delta.Chapter > 0 && delta.Chapter != chapterNumber {
		delta.Notes = appendSettlerNote(delta.Notes, fmt.Sprintf("sanitize: chapter corrected from %d to %d", delta.Chapter, chapterNumber))
	}
	delta.Chapter = chapterNumber
	if delta.ChapterSummary.Chapter > 0 && delta.ChapterSummary.Chapter != chapterNumber {
		delta.Notes = appendSettlerNote(delta.Notes, fmt.Sprintf("sanitize: chapterSummary.chapter corrected from %d to %d", delta.ChapterSummary.Chapter, chapterNumber))
	}
	delta.ChapterSummary.Chapter = chapterNumber

	patch, err := p.normalizeSettlerStatePatch(bookID, delta.CurrentStatePatch)
	if err != nil {
		return err
	}
	delta.CurrentStatePatch = patch

	delta.HookOps = sanitizeSettlerHookOps(delta.HookOps)
	delta.NewHookCandidates = sanitizeSettlerHookCandidates(delta.NewHookCandidates)
	delta.Notes = sanitizeSettlerNotes(delta.Notes)
	delta.ChapterSummary = sanitizeSettlerChapterSummary(title, chapterNumber, delta.ChapterSummary)
	p.sanitizeSettlerGrounding(content, delta)
	return nil
}

func (p *Pipeline) normalizeSettlerStatePatch(bookID uint, rawPatch map[string]string) (map[string]string, error) {
	if len(rawPatch) == 0 {
		return map[string]string{}, nil
	}

	var current *model.BookState
	state, err := p.truth.GetBookState(bookID)
	if err != nil {
		return nil, fmt.Errorf("load current state for validation: %w", err)
	}
	current = state

	normalized := make(map[string]string, len(rawPatch))
	for key, value := range rawPatch {
		key = strings.TrimSpace(key)
		if _, ok := allowedCurrentStatePatchKeys[key]; !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if isMetaPlaceholder(value) {
			continue
		}
		if current != nil && sameStateValue(current, key, value) {
			continue
		}
		normalized[key] = value
	}
	return normalized, nil
}

func validateSettlerHookOps(ops settlerHookOps) error {
	seenUpsert := make(map[string]struct{}, len(ops.Upsert))
	for _, upsert := range ops.Upsert {
		upsert.HookID = strings.TrimSpace(upsert.HookID)
		if upsert.HookID == "" {
			return fmt.Errorf("hook upsert missing hookId")
		}
		if _, ok := seenUpsert[upsert.HookID]; ok {
			return fmt.Errorf("duplicate hook upsert: %s", upsert.HookID)
		}
		seenUpsert[upsert.HookID] = struct{}{}
		if !isRecognizedHookType(upsert.Type) {
			return fmt.Errorf("hook %s has invalid type: %s", upsert.HookID, upsert.Type)
		}
		if !isRecognizedHookStatus(upsert.Status) {
			return fmt.Errorf("hook %s has invalid status: %s", upsert.HookID, upsert.Status)
		}
		if strings.TrimSpace(upsert.PayoffTiming) != "" && !isRecognizedPayoffTiming(upsert.PayoffTiming) {
			return fmt.Errorf("hook %s has invalid payoff timing: %s", upsert.HookID, upsert.PayoffTiming)
		}
		if strings.TrimSpace(upsert.Notes) != "" && isMetaPlaceholder(strings.TrimSpace(upsert.Notes)) {
			return fmt.Errorf("hook %s notes are not grounded", upsert.HookID)
		}
	}

	resolveSet := make(map[string]struct{}, len(ops.Resolve))
	for _, hookID := range ops.Resolve {
		hookID = strings.TrimSpace(hookID)
		if hookID == "" {
			return fmt.Errorf("resolve hook contains empty hookId")
		}
		resolveSet[hookID] = struct{}{}
	}
	for _, hookID := range ops.Defer {
		hookID = strings.TrimSpace(hookID)
		if hookID == "" {
			return fmt.Errorf("defer hook contains empty hookId")
		}
		if _, ok := resolveSet[hookID]; ok {
			return fmt.Errorf("hook %s cannot be both resolved and deferred", hookID)
		}
	}
	return nil
}

func sanitizeSettlerHookOps(ops settlerHookOps) settlerHookOps {
	seenUpsert := make(map[string]struct{}, len(ops.Upsert))
	clean := settlerHookOps{
		Upsert:  make([]settlerHookOp, 0, len(ops.Upsert)),
		Mention: uniqueNonEmptyStrings(ops.Mention),
		Resolve: uniqueNonEmptyStrings(ops.Resolve),
		Defer:   uniqueNonEmptyStrings(ops.Defer),
	}
	resolveSet := make(map[string]struct{}, len(clean.Resolve))
	for _, hookID := range clean.Resolve {
		resolveSet[hookID] = struct{}{}
	}
	filteredDefer := clean.Defer[:0]
	for _, hookID := range clean.Defer {
		if _, resolved := resolveSet[hookID]; resolved {
			continue
		}
		filteredDefer = append(filteredDefer, hookID)
	}
	clean.Defer = filteredDefer

	for _, upsert := range ops.Upsert {
		upsert.HookID = strings.TrimSpace(upsert.HookID)
		if upsert.HookID == "" {
			continue
		}
		if _, ok := seenUpsert[upsert.HookID]; ok {
			continue
		}
		seenUpsert[upsert.HookID] = struct{}{}
		upsert.Type = string(normalizeHookType(upsert.Type))
		upsert.Status = string(normalizeHookStatus(upsert.Status))
		upsert.PayoffTiming = string(normalizePayoffTiming(upsert.PayoffTiming))
		upsert.Notes = sanitizeSettlerText(upsert.Notes)
		upsert.ExpectedPayoff = sanitizeSettlerText(upsert.ExpectedPayoff)
		clean.Upsert = append(clean.Upsert, upsert)
	}
	return clean
}

func validateSettlerHookCandidates(candidates []settlerHookCandidate) error {
	for i, candidate := range candidates {
		if !isRecognizedHookType(candidate.Type) {
			return fmt.Errorf("newHookCandidates[%d] has invalid type: %s", i, candidate.Type)
		}
		if strings.TrimSpace(candidate.PayoffTiming) != "" && !isRecognizedPayoffTiming(candidate.PayoffTiming) {
			return fmt.Errorf("newHookCandidates[%d] has invalid payoff timing: %s", i, candidate.PayoffTiming)
		}
		if strings.TrimSpace(candidate.ExpectedPayoff) == "" && strings.TrimSpace(candidate.Notes) == "" {
			return fmt.Errorf("newHookCandidates[%d] missing description", i)
		}
		if isMetaPlaceholder(strings.TrimSpace(candidate.ExpectedPayoff)) || isMetaPlaceholder(strings.TrimSpace(candidate.Notes)) {
			return fmt.Errorf("newHookCandidates[%d] is not grounded", i)
		}
	}
	return nil
}

func sanitizeSettlerHookCandidates(candidates []settlerHookCandidate) []settlerHookCandidate {
	clean := make([]settlerHookCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.Type = string(normalizeHookType(candidate.Type))
		candidate.PayoffTiming = string(normalizePayoffTiming(candidate.PayoffTiming))
		candidate.ExpectedPayoff = sanitizeSettlerText(candidate.ExpectedPayoff)
		candidate.Notes = sanitizeSettlerText(candidate.Notes)
		if strings.TrimSpace(candidate.ExpectedPayoff) == "" && strings.TrimSpace(candidate.Notes) == "" {
			continue
		}
		clean = append(clean, candidate)
	}
	return clean
}

func validateSettlerNotes(notes []string) error {
	for i, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		if isMetaPlaceholder(note) {
			return fmt.Errorf("notes[%d] is not grounded", i)
		}
	}
	return nil
}

func sanitizeSettlerNotes(notes []string) []string {
	clean := make([]string, 0, len(notes))
	for _, note := range notes {
		note = sanitizeSettlerText(note)
		if note == "" {
			continue
		}
		clean = append(clean, note)
	}
	return uniqueNonEmptyStrings(clean)
}

func validateSettlerChapterSummary(title string, summary settlerChapterSummary) error {
	fields := []string{
		strings.TrimSpace(summary.Characters),
		strings.TrimSpace(summary.Events),
		strings.TrimSpace(summary.State),
		strings.TrimSpace(summary.Hook),
		strings.TrimSpace(summary.Mood),
		strings.TrimSpace(summary.Type),
	}
	for _, field := range fields {
		if field != "" && isMetaPlaceholder(field) {
			return fmt.Errorf("chapter summary contains placeholder-like value")
		}
	}
	if strings.TrimSpace(summary.Title) != "" && strings.TrimSpace(title) != "" {
		summaryTitle := strings.TrimSpace(summary.Title)
		if !roughlySameTitle(summaryTitle, strings.TrimSpace(title)) {
			return fmt.Errorf("chapter summary title mismatch: got %q want %q", summaryTitle, strings.TrimSpace(title))
		}
	}
	return nil
}

func sanitizeSettlerChapterSummary(title string, chapterNumber uint, summary settlerChapterSummary) settlerChapterSummary {
	summary.Chapter = chapterNumber
	if strings.TrimSpace(title) != "" {
		summary.Title = strings.TrimSpace(title)
	} else {
		summary.Title = sanitizeSettlerText(summary.Title)
	}
	summary.Characters = sanitizeSettlerText(summary.Characters)
	summary.Events = sanitizeSettlerText(summary.Events)
	summary.State = sanitizeSettlerText(summary.State)
	summary.Hook = sanitizeSettlerText(summary.Hook)
	summary.Mood = sanitizeSettlerText(summary.Mood)
	summary.Type = sanitizeSettlerText(summary.Type)
	return summary
}

func validateSettlerGrounding(content string, delta *settlerDelta) error {
	if delta == nil {
		return nil
	}
	body := normalizeGroundingText(content)
	if body == "" {
		return nil
	}
	for key, value := range delta.CurrentStatePatch {
		if value == "" {
			continue
		}
		if !hasStateGroundingSignal(key, body, value) {
			return fmt.Errorf("state patch %s looks ungrounded: %s", key, value)
		}
	}
	summaryChecks := []struct {
		label string
		value string
	}{
		{"events", delta.ChapterSummary.Events},
		{"state", delta.ChapterSummary.State},
		{"hook", delta.ChapterSummary.Hook},
	}
	for _, item := range summaryChecks {
		value := strings.TrimSpace(item.value)
		if value == "" {
			continue
		}
		if !hasGroundingSignal(body, value) {
			return fmt.Errorf("chapter summary %s looks ungrounded: %s", item.label, value)
		}
	}
	return nil
}

func (p *Pipeline) sanitizeSettlerGrounding(content string, delta *settlerDelta) {
	if delta == nil {
		return
	}
	body := normalizeGroundingText(content)
	if body == "" {
		return
	}
	for key, value := range delta.CurrentStatePatch {
		if value == "" {
			continue
		}
		if !hasStateGroundingSignal(key, body, value) && !softStatePatchAllowed(key) {
			delete(delta.CurrentStatePatch, key)
			delta.Notes = appendSettlerNote(delta.Notes, fmt.Sprintf("sanitize: dropped weakly grounded state patch %s", key))
		}
	}
}

func softStatePatchAllowed(key string) bool {
	switch key {
	case "currentGoal", "protagonistState", "currentConflict", "currentAlliances":
		return true
	default:
		return false
	}
}

func sanitizeSettlerText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || isMetaPlaceholder(raw) {
		return ""
	}
	return raw
}

func appendSettlerNote(notes []string, note string) []string {
	note = strings.TrimSpace(note)
	if note == "" {
		return notes
	}
	for _, existing := range notes {
		if strings.TrimSpace(existing) == note {
			return notes
		}
	}
	return append(notes, note)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		clean = append(clean, value)
	}
	return clean
}

func settlerDeltaHasSignal(delta settlerDelta) bool {
	if len(delta.CurrentStatePatch) > 0 || len(delta.HookOps.Upsert) > 0 || len(delta.HookOps.Resolve) > 0 ||
		len(delta.HookOps.Defer) > 0 || len(delta.NewHookCandidates) > 0 || len(delta.Notes) > 0 {
		return true
	}
	summary := delta.ChapterSummary
	return strings.TrimSpace(summary.Title) != "" ||
		strings.TrimSpace(summary.Characters) != "" ||
		strings.TrimSpace(summary.Events) != "" ||
		strings.TrimSpace(summary.State) != "" ||
		strings.TrimSpace(summary.Hook) != "" ||
		strings.TrimSpace(summary.Mood) != "" ||
		strings.TrimSpace(summary.Type) != ""
}

func sameStateValue(state *model.BookState, key, value string) bool {
	if state == nil {
		return false
	}
	switch key {
	case "currentLocation":
		return strings.TrimSpace(state.CurrentLocation) == value
	case "protagonistState":
		return strings.TrimSpace(state.ProtagonistState) == value
	case "currentGoal":
		return strings.TrimSpace(state.CurrentGoal) == value
	case "currentConstraint":
		return strings.TrimSpace(state.CurrentConstraint) == value
	case "currentAlliances":
		return strings.TrimSpace(state.CurrentAlliances) == value
	case "currentConflict":
		return strings.TrimSpace(state.CurrentConflict) == value
	default:
		return false
	}
}

func roughlySameTitle(a, b string) bool {
	a = normalizeGroundingText(a)
	b = normalizeGroundingText(b)
	return a == b || strings.Contains(a, b) || strings.Contains(b, a)
}

func hasStateGroundingSignal(key, body, candidate string) bool {
	if hasGroundingSignal(body, candidate) {
		return true
	}
	if key == "currentLocation" {
		return hasLocationGroundingSignal(body, candidate)
	}
	return false
}

func hasGroundingSignal(body, candidate string) bool {
	candidate = normalizeGroundingText(candidate)
	if candidate == "" {
		return false
	}
	if strings.Contains(body, candidate) {
		return true
	}
	for _, token := range groundingTokens(candidate) {
		if len([]rune(token)) < 2 {
			continue
		}
		if strings.Contains(body, token) {
			return true
		}
	}
	return false
}

func hasLocationGroundingSignal(body, candidate string) bool {
	candidate = normalizeGroundingText(candidate)
	if candidate == "" {
		return false
	}
	if strings.Contains(body, candidate) {
		return true
	}

	segments := locationGroundingSegments(candidate)
	if len(segments) == 0 {
		return false
	}

	matched := 0
	longMatched := 0
	for _, segment := range segments {
		if strings.Contains(body, segment) {
			matched++
			if len([]rune(segment)) >= 3 {
				longMatched++
			}
		}
	}
	if longMatched >= 1 && matched >= 2 {
		return true
	}
	if matched >= 3 {
		return true
	}

	return sharedGroundingNGrams(body, candidate, 2) >= 2 || sharedGroundingNGrams(body, candidate, 3) >= 1
}

func groundingTokens(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '，' || r == '。' || r == '；' || r == '：' || r == ',' || r == '.' || r == ';' || r == ':' || r == '\n'
	})
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tokens = append(tokens, part)
		}
	}
	return tokens
}

func normalizeGroundingText(raw string) string {
	raw = strings.TrimSpace(raw)
	replacer := strings.NewReplacer(
		"\n", "",
		"\r", "",
		"\t", "",
		" ", "",
		"“", "",
		"”", "",
		"\"", "",
	)
	return replacer.Replace(raw)
}

func locationGroundingSegments(raw string) []string {
	raw = normalizeGroundingText(raw)
	if raw == "" {
		return nil
	}
	separators := []string{
		"·", "/", "-", "到", "内", "外", "里", "中", "前", "后", "旁", "边",
	}
	seen := make(map[string]struct{})
	var segments []string
	appendSegment := func(part string) {
		part = strings.TrimSpace(part)
		part = normalizeGroundingText(part)
		if len([]rune(part)) < 2 {
			return
		}
		if _, ok := seen[part]; ok {
			return
		}
		seen[part] = struct{}{}
		segments = append(segments, part)
	}
	appendSegment(raw)
	for _, sep := range separators {
		if !strings.Contains(raw, sep) {
			continue
		}
		for _, part := range strings.Split(raw, sep) {
			appendSegment(part)
		}
	}
	suffixes := []string{"偏房", "厢房", "东厢", "西厢", "院", "阁", "楼", "堂", "房", "府", "殿", "宫", "门", "街", "巷"}
	for _, suffix := range suffixes {
		idx := strings.Index(raw, suffix)
		if idx <= 0 {
			continue
		}
		prefix := []rune(raw[:idx])
		if len(prefix) > 4 {
			prefix = prefix[len(prefix)-4:]
		}
		appendSegment(string(prefix) + suffix)
		appendSegment(suffix)
	}
	return segments
}

func sharedGroundingNGrams(body, candidate string, n int) int {
	bodyRunes := []rune(body)
	candidateRunes := []rune(candidate)
	if n <= 0 || len(bodyRunes) < n || len(candidateRunes) < n {
		return 0
	}
	bodySet := make(map[string]struct{})
	for i := 0; i <= len(bodyRunes)-n; i++ {
		bodySet[string(bodyRunes[i:i+n])] = struct{}{}
	}
	count := 0
	seen := make(map[string]struct{})
	for i := 0; i <= len(candidateRunes)-n; i++ {
		token := string(candidateRunes[i : i+n])
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if _, ok := bodySet[token]; ok {
			count++
		}
	}
	return count
}

func isMetaPlaceholder(raw string) bool {
	raw = normalizeGroundingText(raw)
	if raw == "" {
		return false
	}

	exactBadPhrases := []string{
		"同上", "略", "暂无", "无变化", "未变化", "保持不变", "延续上章", "沿用上章",
		"未提及", "未知", "待定", "n/a", "na", "无",
	}
	for _, phrase := range exactBadPhrases {
		if raw == normalizeGroundingText(phrase) {
			return true
		}
	}
	return false
}

func isRecognizedHookType(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "plot", "剧情", "conflict", "冲突", "item", "道具", "物品", "mystery", "悬疑", "谜题", "character", "人物", "relationship", "关系":
		return true
	default:
		return false
	}
}

func isRecognizedHookStatus(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "seed", "种子", "open", "开放", "progressing", "推进中", "advanced", "resolved", "已回收", "回收", "deferred", "延后", "stale", "过期":
		return true
	default:
		return false
	}
}

func isRecognizedPayoffTiming(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "immediate", "立即", "near-term", "近期", "mid-arc", "mid-term", "中程", "中期", "slow-burn", "慢热", "长线":
		return true
	default:
		return false
	}
}

func (p *Pipeline) buildSettlerContext(bookID uint) (string, error) {
	foundations, err := p.truth.ListFoundations(bookID)
	if err != nil {
		return "", err
	}
	bookState, err := p.truth.GetBookState(bookID)
	if err != nil {
		return "", err
	}
	characters, err := p.truth.GetCharacters(bookID)
	if err != nil {
		return "", err
	}
	facts, err := p.truth.GetFacts(bookID)
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

	b.WriteString("### Book State\n")
	if payload, err := json.Marshal(bookState); err == nil {
		b.WriteString(string(payload))
		b.WriteString("\n\n")
	}

	b.WriteString("### Characters\n")
	if payload, err := json.Marshal(characters); err == nil {
		b.WriteString(string(payload))
		b.WriteString("\n\n")
	}

	b.WriteString("### Active Facts\n")
	if payload, err := json.Marshal(facts); err == nil {
		b.WriteString(string(payload))
		b.WriteString("\n\n")
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

func (p *Pipeline) extractTruthFiles(ctx context.Context, bookID uint, chapterNumber uint, rawOutput string, modelID uint) (*extractedTruthFiles, error) {
	extractorAny, ok := p.registry.Get("truth_extractor")
	if !ok {
		return nil, fmt.Errorf("truth_extractor agent not found")
	}
	extractor, ok := extractorAny.(*agent.TruthExtractorAgent)
	if !ok {
		return nil, fmt.Errorf("invalid truth_extractor agent")
	}
	bookTitle := ""
	if book, err := p.truth.GetBook(bookID); err == nil && book != nil {
		bookTitle = book.Title
	}
	extractPrompt := extractor.BuildUserPrompt(bookTitle, chapterNumber, rawOutput)

	result, err := p.llm.ChatForAgent(ctx, extractor.Name(), modelID, extractor.SystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: extractPrompt},
	}, 0.3)
	if err != nil {
		return nil, fmt.Errorf("extract truth llm: %w", err)
	}

	jsonStr := extractJSON(result)
	if jsonStr == "" {
		return nil, fmt.Errorf("extract truth json missing")
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
		EvidenceNotes []extractedEvidenceNote `json:"evidence_notes"`
		Summary       struct {
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
		return nil, fmt.Errorf("extract truth json parse: %w", err)
	}

	resultData := &extractedTruthFiles{
		Characters:    make([]model.Character, 0, len(extracted.Characters)),
		DurableFacts:  make([]model.Fact, 0, len(extracted.DurableFacts)),
		Hooks:         make([]model.Hook, 0, len(extracted.Hooks)),
		EvidenceNotes: extracted.EvidenceNotes,
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
		resultData.Characters = append(resultData.Characters, model.Character{
			BookID:          bookID,
			Name:            c.Name,
			RoleType:        rt,
			Profile:         c.Profile,
			SourceChapter:   chapterNumber,
			LastSeenChapter: chapterNumber,
		})
	}

	for _, f := range extracted.DurableFacts {
		if strings.TrimSpace(f.Subject) == "" || strings.TrimSpace(f.Predicate) == "" || strings.TrimSpace(f.Object) == "" {
			continue
		}
		category := normalizeFactCategory(f.Category)
		resultData.DurableFacts = append(resultData.DurableFacts, model.Fact{
			BookID:           bookID,
			Subject:          strings.TrimSpace(f.Subject),
			Predicate:        strings.TrimSpace(f.Predicate),
			Object:           strings.TrimSpace(f.Object),
			Category:         category,
			ValidFromChapter: chapterNumber,
			SourceChapter:    chapterNumber,
		})
	}

	for _, h := range extracted.Hooks {
		if h.HookID == "" {
			continue
		}
		resultData.Hooks = append(resultData.Hooks, model.Hook{
			BookID:       bookID,
			HookID:       h.HookID,
			StartChapter: chapterNumber,
			Type:         normalizeHookType(h.Type),
			Status:       model.HookSeed,
			Notes:        h.Description,
		})
	}

	if strings.TrimSpace(extracted.Summary.KeyEvents) == "" {
		return nil, fmt.Errorf("extract truth summary missing key_events")
	}
	resultData.Summary = &model.ChapterSummary{
		BookID:             bookID,
		ChapterNumber:      chapterNumber,
		Title:              strings.TrimSpace(extracted.Summary.Title),
		CharactersAppeared: strings.TrimSpace(extracted.Summary.CharactersAppeared),
		KeyEvents:          strings.TrimSpace(extracted.Summary.KeyEvents),
		StateChanges:       strings.TrimSpace(extracted.Summary.StateChanges),
		HookActivity:       strings.TrimSpace(extracted.Summary.HookActivity),
		Mood:               strings.TrimSpace(extracted.Summary.Mood),
		ChapterType:        strings.TrimSpace(extracted.Summary.ChapterType),
	}

	return resultData, nil
}

func (p *Pipeline) persistExtractedTruthFiles(bookID uint, chapterNumber uint, extracted *extractedTruthFiles, opts extractionOptions) error {
	if extracted == nil {
		return nil
	}

	for i := range extracted.Characters {
		if err := p.truth.SaveCharacter(&extracted.Characters[i]); err != nil {
			return err
		}
	}
	for i := range extracted.DurableFacts {
		if err := p.truth.SaveFact(&extracted.DurableFacts[i]); err != nil {
			return err
		}
	}
	if len(extracted.EvidenceNotes) > 0 {
		evidenceJSON, _ := json.Marshal(extracted.EvidenceNotes)
		if err := p.truth.SaveRuntimeArtifact(&model.RuntimeArtifact{
			BookID:        bookID,
			ChapterNumber: chapterNumber,
			ArtifactType:  model.ArtifactEvidence,
			Content:       string(evidenceJSON),
		}); err != nil {
			return err
		}
	}
	if opts.SaveHooks {
		for i := range extracted.Hooks {
			if err := p.truth.SaveHook(&extracted.Hooks[i]); err != nil {
				return err
			}
		}
	}
	if opts.SaveSummary && extracted.Summary != nil {
		if existing, ok := p.findChapterSummary(bookID, chapterNumber); ok {
			existing.Title = defaultString(strings.TrimSpace(extracted.Summary.Title), existing.Title)
			existing.CharactersAppeared = strings.TrimSpace(extracted.Summary.CharactersAppeared)
			existing.KeyEvents = strings.TrimSpace(extracted.Summary.KeyEvents)
			existing.StateChanges = strings.TrimSpace(extracted.Summary.StateChanges)
			existing.HookActivity = strings.TrimSpace(extracted.Summary.HookActivity)
			existing.Mood = strings.TrimSpace(extracted.Summary.Mood)
			existing.ChapterType = strings.TrimSpace(extracted.Summary.ChapterType)
			if err := p.truth.SaveChapterSummary(existing); err != nil {
				return err
			}
		} else if err := p.truth.SaveChapterSummary(extracted.Summary); err != nil {
			return err
		}
	}
	return nil
}

func normalizeFactCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "identity", "身份":
		return "identity"
	case "resource", "资源":
		return "resource"
	case "item", "物品", "道具":
		return "item"
	case "rule", "规则", "设定":
		return "rule"
	case "relationship", "关系":
		return "relationship"
	default:
		return "relationship"
	}
}

func (p *Pipeline) saveChapterSnapshot(bookID uint, chapterNumber uint, sections map[string]string) {
	foundations, _ := p.truth.ListFoundations(bookID)
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
	foundationsJSON, _ := json.Marshal(foundations)
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
		FoundationsJSON:      string(foundationsJSON),
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
	foundations, _ := p.truth.ListFoundations(book.ID)
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
	foundationsJSON, _ := json.Marshal(foundations)
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
		FoundationsJSON:      string(foundationsJSON),
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

func (p *Pipeline) buildContext(_ context.Context, bookID uint, chapterNumber uint, userInput string) (string, error) {
	var b strings.Builder

	book, err := p.truth.GetBook(bookID)
	if err != nil {
		return "", err
	}
	state, err := p.truth.GetBookState(bookID)
	if err != nil {
		return "", err
	}

	for _, fileType := range []model.FoundationFileType{
		model.FoundationBookRules,
		model.FoundationAuthorIntent,
		model.FoundationCurrentFocus,
		model.FoundationAuditDrift,
	} {
		foundation, err := p.truth.GetFoundation(bookID, fileType)
		if err == nil && strings.TrimSpace(foundation.Content) != "" {
			b.WriteString(fmt.Sprintf("## %s\n%s\n\n", fileType, clipText(foundation.Content, 1800)))
		}
	}

	if state != nil {
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

	hooks, _ := p.truth.GetHooks(bookID)
	if len(hooks) > 0 {
		b.WriteString("## 必须跟踪的伏笔\n")
		for _, h := range hooks {
			if h.Status != model.HookResolved && h.Status != model.HookStale &&
				(h.IsCritical || h.Status == model.HookProgressing || h.Status == model.HookDeferred) {
				b.WriteString(fmt.Sprintf("- %s [%s] 状态:%s 始于第%d章\n", h.HookID, h.Type, h.Status, h.StartChapter))
			}
		}
		b.WriteString("\n")
	}

	summaries, _ := p.truth.GetChapterSummaries(bookID)
	if len(summaries) > 0 {
		b.WriteString("## 最近章节摘要\n")
		start := 0
		if len(summaries) > 3 {
			start = len(summaries) - 3
		}
		for _, s := range summaries[start:] {
			b.WriteString(fmt.Sprintf("- 第%d章 %s: %s\n", s.ChapterNumber, s.Title, s.KeyEvents))
		}
		b.WriteString("\n")
	}

	retrieved, err := p.truth.SearchKnowledge(model.KnowledgeSearchQuery{
		BookID:        bookID,
		Query:         buildKnowledgeRetrievalQuery(book, "", userInput, state),
		ChapterNumber: chapterNumber,
		Limit:         8,
	})
	if err != nil {
		return "", err
	}
	if len(retrieved) > 0 {
		b.WriteString("## 相关设定检索\n")
		remaining := 2600
		for _, result := range retrieved {
			if remaining <= 0 {
				break
			}
			content := clipText(result.Content, minInt(600, remaining))
			if strings.TrimSpace(content) == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("- [来源：%s/%s] %s\n%s\n", result.SourceType, result.SourceID, result.Title, content))
			remaining -= len([]rune(content))
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

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
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

	raw, err := p.llm.ChatForAgent(ctx, "role_namer", modelID, "你是小说角色命名编辑，只返回 JSON。", []llm.AgentMessage{
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
	case "character", "人物", "relationship", "关系":
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

func (p *Pipeline) applyFallbackBookState(bookID uint, chapterNumber uint, updatedState string) error {
	state, err := p.truth.GetBookState(bookID)
	if err != nil {
		return fmt.Errorf("get book state: %w", err)
	}
	if state == nil {
		state = &model.BookState{BookID: bookID}
	}

	if state.ProtagonistName == "" {
		characters, _ := p.truth.GetCharacters(bookID)
		for _, c := range characters {
			if c.RoleType == model.CharacterProtagonist {
				state.ProtagonistName = c.Name
				break
			}
		}
	}

	patch := parseFallbackStatePatch(updatedState)
	if summary, ok := p.findChapterSummary(bookID, chapterNumber); ok {
		mergeSummaryFallbackIntoPatch(patch, *summary)
	}

	applyFallbackStatePatch(state, patch)
	state.CurrentChapter = chapterNumber
	state.SourceChapter = chapterNumber
	if strings.TrimSpace(state.SituationSummary) == "" {
		state.SituationSummary = buildSituationSummary(state)
	} else {
		state.SituationSummary = clipText(firstNonEmpty(buildSituationSummary(state), state.SituationSummary), 500)
	}

	if err := p.truth.SaveBookState(state); err != nil {
		return fmt.Errorf("save fallback book state: %w", err)
	}
	if len(patch) > 0 {
		if err := p.upsertFoundation(bookID, model.FoundationCurrentFocus, buildCurrentFocusDelta(chapterNumber, patch)); err != nil {
			return fmt.Errorf("update current_focus fallback: %w", err)
		}
	}
	return nil
}

func parseFallbackStatePatch(raw string) map[string]string {
	patch := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return patch
	}

	if jsonStr := extractJSON(raw); jsonStr != "" {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
			for key, value := range parsed {
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)
				if _, ok := allowedCurrentStatePatchKeys[key]; ok && value != "" {
					patch[key] = clipText(value, 240)
				}
			}
			if len(patch) > 0 {
				return patch
			}
		}
	}

	for _, row := range parseMarkdownTableRows(raw) {
		if len(row) < 2 {
			continue
		}
		label := normalizeFallbackStateLabel(cellAt(row, 0))
		value := strings.TrimSpace(cellAt(row, 1))
		if label == "" || value == "" {
			continue
		}
		switch label {
		case "currentLocation":
			patch[label] = mergeStateText(patch[label], value, 240)
		case "protagonistState":
			patch[label] = mergeStateText(patch[label], value, 240)
		case "currentGoal":
			patch[label] = mergeStateText(patch[label], value, 240)
		case "currentConstraint":
			patch[label] = mergeStateText(patch[label], value, 240)
		case "currentAlliances":
			patch[label] = mergeStateText(patch[label], value, 240)
		case "currentConflict":
			patch[label] = mergeStateText(patch[label], value, 240)
		}
	}

	return patch
}

func normalizeFallbackStateLabel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "状态项", "item", "字段":
		return ""
	case "当前锚点", "当前位置", "当前地点", "current anchor", "current location":
		return "currentLocation"
	case "主角状态", "当前状态", "人物状态", "protagonist state":
		return "protagonistState"
	case "当前目标", "目标", "current goal":
		return "currentGoal"
	case "环境状态", "当前限制", "限制", "constraint", "environment":
		return "currentConstraint"
	case "当前敌我", "敌我态势", "当前盟友", "alliances":
		return "currentAlliances"
	case "当前冲突", "核心冲突", "威胁评级", "threat", "current conflict":
		return "currentConflict"
	default:
		return ""
	}
}

func mergeSummaryFallbackIntoPatch(patch map[string]string, summary model.ChapterSummary) {
	if patch == nil {
		return
	}
	if strings.TrimSpace(patch["protagonistState"]) == "" {
		patch["protagonistState"] = clipText(strings.TrimSpace(summary.StateChanges), 240)
	}
	if strings.TrimSpace(patch["currentConflict"]) == "" {
		patch["currentConflict"] = clipText(firstNonEmpty(summary.HookActivity, summary.KeyEvents), 240)
	}
}

func applyFallbackStatePatch(state *model.BookState, patch map[string]string) {
	if state == nil || len(patch) == 0 {
		return
	}
	if v := strings.TrimSpace(patch["currentLocation"]); v != "" {
		state.CurrentLocation = v
	}
	if v := strings.TrimSpace(patch["protagonistState"]); v != "" {
		state.ProtagonistState = v
	}
	if v := strings.TrimSpace(patch["currentGoal"]); v != "" {
		state.CurrentGoal = v
	}
	if v := strings.TrimSpace(patch["currentConstraint"]); v != "" {
		state.CurrentConstraint = v
	}
	if v := strings.TrimSpace(patch["currentAlliances"]); v != "" {
		state.CurrentAlliances = v
	}
	if v := strings.TrimSpace(patch["currentConflict"]); v != "" {
		state.CurrentConflict = v
	}
}

func mergeStateText(current, incoming string, max int) string {
	current = strings.TrimSpace(current)
	incoming = strings.TrimSpace(incoming)
	if incoming == "" {
		return current
	}
	if current == "" {
		return clipText(incoming, max)
	}
	if current == incoming || strings.Contains(current, incoming) {
		return clipText(current, max)
	}
	if strings.Contains(incoming, current) {
		return clipText(incoming, max)
	}
	return clipText(current+"；"+incoming, max)
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

请根据“当前上下文”和“用户指令”一起生成本章 chapter_memo。
用户指令是本章局部意图，但不能覆盖已发生剧情、真相文件、角色状态和伏笔账本；如果用户指令与既有剧情冲突，优先保持连续性，并把用户意图改写成可落地的相邻推进。`

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
