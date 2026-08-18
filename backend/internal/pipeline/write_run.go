package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"whwriter/backend/internal/agent"
	"whwriter/backend/internal/llm"
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"
)

var errWriteRunCancelled = errors.New("write run cancelled")

const maxWriteStageAttempts = 3

type StartWriteRunInput struct {
	BookID      uint
	ModelID     uint
	UserInput   string
	RunType     model.ChapterWriteRunType
	RetryMode   model.ChapterWriteRetryMode
	ParentRunID *uint
}

type writeRunState struct {
	Book            *model.Book
	OriginalChapter *model.Chapter
	ChapterNumber   uint
	ContextPkg      string
	Composed        *agent.ComposeOutput
	Memo            string
	Title           string
	Content         string
	Sections        map[string]string
	AuditRaw        string
	AuditResult     auditResult
	SettleSections  map[string]string
	SettleDelta     settlerDelta
	ExtractedTruth  *extractedTruthFiles
	FinalizedSource string
}

type stageTokenSummary struct {
	AgentName        string `json:"agent_name"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	CachedTokens     int64  `json:"cached_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
}

type stageRunMeta struct {
	Attempt      uint                `json:"attempt,omitempty"`
	MaxAttempts  uint                `json:"max_attempts,omitempty"`
	TokenSummary []stageTokenSummary `json:"token_summary,omitempty"`
}

type stageContextPayload struct {
	Context string       `json:"context"`
	Meta    stageRunMeta `json:"meta,omitempty"`
}

type stagePlanningPayload struct {
	Memo     string               `json:"memo"`
	Context  string               `json:"context,omitempty"`
	Composed *agent.ComposeOutput `json:"composed,omitempty"`
	Meta     stageRunMeta         `json:"meta,omitempty"`
}

type stageWritingPayload struct {
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Sections map[string]string `json:"sections"`
	Raw      string            `json:"raw"`
	Source   string            `json:"source"`
	Meta     stageRunMeta      `json:"meta,omitempty"`
}

type stageAuditingPayload struct {
	Raw    string       `json:"raw"`
	Result auditResult  `json:"result"`
	Meta   stageRunMeta `json:"meta,omitempty"`
}

type stageRevisingPayload struct {
	Applied           bool                 `json:"applied"`
	Content           string               `json:"content"`
	Sections          map[string]string    `json:"sections"`
	Raw               string               `json:"raw"`
	Reason            string               `json:"reason,omitempty"`
	Evaluation        *agent.ScoreDecision `json:"evaluation,omitempty"`
	CandidateAuditRaw string               `json:"candidate_audit_raw,omitempty"`
	Meta              stageRunMeta         `json:"meta,omitempty"`
}

type stagePolishingPayload struct {
	Content string       `json:"content"`
	Meta    stageRunMeta `json:"meta,omitempty"`
}

type stageExtractingPayload struct {
	SettlerSections map[string]string    `json:"settler_sections"`
	SettlerDelta    settlerDelta         `json:"settler_delta"`
	SettlerRaw      string               `json:"settler_raw"`
	ExtractedTruth  *extractedTruthFiles `json:"extracted_truth"`
	Meta            stageRunMeta         `json:"meta,omitempty"`
}

type stageSnapshotPayload struct {
	ChapterNumber uint         `json:"chapter_number"`
	Title         string       `json:"title"`
	Content       string       `json:"content"`
	Meta          stageRunMeta `json:"meta,omitempty"`
}

func (p *Pipeline) StartWriteRun(ctx context.Context, in StartWriteRunInput) (*model.ChapterWriteRun, error) {
	book, err := p.truth.GetBook(in.BookID)
	if err != nil {
		return nil, fmt.Errorf("get book: %w", err)
	}
	if book.Status == model.BookStatusInitializing || book.Status == model.BookStatusPaused || book.Status == model.BookStatusCompleted {
		return nil, fmt.Errorf("当前状态不允许继续写作")
	}
	active, err := p.truth.GetActiveChapterWriteRun(in.BookID)
	if err != nil {
		return nil, fmt.Errorf("get active write run: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("该书已有写作任务在进行中")
	}

	recoverStatus := model.BookStatusActive
	if book.Status == model.BookStatusOutlining {
		recoverStatus = model.BookStatusOutlining
	}

	runType := normalizeWriteRunType(in.RunType)
	if runType == model.WriteRunTypeRewriteLatest && strings.TrimSpace(in.UserInput) == "" {
		return nil, fmt.Errorf("重写最后一章需要填写重写要求")
	}

	targetChapter, resumeStage, err := p.resolveTargetChapterAndResumeStage(in.BookID, runType, in.RetryMode, in.ParentRunID)
	if err != nil {
		return nil, err
	}

	locked, err := p.truth.TransitionBookStatus(in.BookID, []model.BookStatus{
		model.BookStatusOutlining,
		model.BookStatusActive,
	}, model.BookStatusWriting)
	if err != nil {
		return nil, fmt.Errorf("transition book status: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("该书已有写作任务在进行中")
	}

	run := &model.ChapterWriteRun{
		BookID:           in.BookID,
		TargetChapter:    targetChapter,
		RequestedModelID: in.ModelID,
		UserInput:        in.UserInput,
		RunType:          runType,
		Status:           model.WriteRunQueued,
		RetryMode:        normalizeWriteRetryMode(in.RetryMode),
		ParentRunID:      in.ParentRunID,
		ResumeFromStage:  resumeStage,
	}
	if err := p.truth.CreateChapterWriteRun(run); err != nil {
		_ = p.truth.UpdateBookStatus(in.BookID, recoverStatus)
		return nil, fmt.Errorf("create write run: %w", err)
	}
	if err := p.truth.CreateChapterWriteBaseline(&model.ChapterWriteBaseline{
		RunID:             run.ID,
		BookID:            in.BookID,
		BaseChapterNumber: targetChapter - 1,
		RecoverStatus:     recoverStatus,
	}); err != nil {
		_ = p.truth.UpdateBookStatus(in.BookID, recoverStatus)
		return nil, fmt.Errorf("create write baseline: %w", err)
	}

	go p.executeWriteRun(run.ID)
	return run, nil
}

func (p *Pipeline) CancelWriteRun(runID uint) error {
	run, err := p.truth.GetChapterWriteRun(runID)
	if err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("write run not found")
	}
	if run.Status != model.WriteRunQueued && run.Status != model.WriteRunRunning {
		return fmt.Errorf("当前 run 不可取消")
	}
	run.CancelRequested = true
	if err := p.truth.SaveChapterWriteRun(run); err != nil {
		return err
	}

	p.runMu.Lock()
	cancel := p.runStops[runID]
	p.runMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (p *Pipeline) RetryWriteRun(ctx context.Context, runID uint, mode model.ChapterWriteRetryMode) (*model.ChapterWriteRun, error) {
	run, err := p.truth.GetChapterWriteRun(runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("write run not found")
	}
	if run.Status != model.WriteRunFailed && run.Status != model.WriteRunCancelled {
		return nil, fmt.Errorf("仅失败或取消的 run 支持重试")
	}
	parentID := run.ID
	return p.StartWriteRun(ctx, StartWriteRunInput{
		BookID:      run.BookID,
		ModelID:     run.RequestedModelID,
		UserInput:   run.UserInput,
		RunType:     run.RunType,
		RetryMode:   mode,
		ParentRunID: &parentID,
	})
}

func (p *Pipeline) executeWriteRun(runID uint) {
	run, err := p.truth.GetChapterWriteRun(runID)
	if err != nil || run == nil {
		return
	}
	baseline, _ := p.truth.GetChapterWriteBaseline(runID)
	if baseline == nil {
		return
	}
	book, err := p.truth.GetBook(run.BookID)
	if err != nil {
		_ = p.finishWriteRun(run, baseline, model.WriteRunFailed, "", fmt.Sprintf("get book: %v", err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.runMu.Lock()
	p.runStops[runID] = cancel
	p.runMu.Unlock()
	defer func() {
		cancel()
		p.runMu.Lock()
		delete(p.runStops, runID)
		p.runMu.Unlock()
	}()

	now := time.Now()
	run.Status = model.WriteRunRunning
	run.StartedAt = &now
	if err := p.truth.SaveChapterWriteRun(run); err != nil {
		_ = p.finishWriteRun(run, baseline, model.WriteRunFailed, "", err.Error())
		return
	}

	state := &writeRunState{
		Book:            book,
		ChapterNumber:   run.TargetChapter,
		Sections:        map[string]string{},
		FinalizedSource: "writer",
	}
	if normalizeWriteRunType(run.RunType) == model.WriteRunTypeRewriteLatest {
		original, err := p.truth.GetChapter(run.BookID, run.TargetChapter)
		if err != nil {
			_ = p.finishWriteRun(run, baseline, model.WriteRunFailed, "", fmt.Sprintf("load original chapter: %v", err))
			return
		}
		state.OriginalChapter = original
	}

	if run.ParentRunID != nil && normalizeWriteRetryMode(run.RetryMode) == model.WriteRetryResumeFailedStage {
		if err := p.loadResumeState(*run.ParentRunID, run.ID, run.ResumeFromStage, state); err != nil {
			_ = p.finishWriteRun(run, baseline, model.WriteRunFailed, "", fmt.Sprintf("resume state: %v", err))
			return
		}
	}

	stageOrder := []model.ChapterWriteStage{
		model.WriteStageContext,
		model.WriteStagePlanning,
		model.WriteStageWriting,
		model.WriteStageAuditing,
		model.WriteStageRevising,
		model.WriteStagePolishing,
		model.WriteStageExtracting,
		model.WriteStageSnapshot,
	}
	started := false
	for _, stage := range stageOrder {
		if !started {
			started = run.ResumeFromStage == "" || run.ResumeFromStage == stage
			if !started {
				continue
			}
		}
		if err := p.assertRunNotCancelled(ctx, run.ID); err != nil {
			_ = p.finishWriteRun(run, baseline, model.WriteRunCancelled, stage, "写作已取消")
			return
		}
		if err := p.executeStageWithRetries(ctx, run, state, stage); err != nil {
			if errors.Is(err, errWriteRunCancelled) || errors.Is(err, context.Canceled) {
				_ = p.finishWriteRun(run, baseline, model.WriteRunCancelled, stage, "写作已取消")
				return
			}
			_ = p.finishWriteRun(run, baseline, model.WriteRunFailed, stage, err.Error())
			return
		}
	}

	_ = p.finishWriteRun(run, baseline, model.WriteRunSucceeded, model.WriteStageSnapshot, "")
}

func (p *Pipeline) ReconcileInterruptedRuns() error {
	runs, err := p.truth.ListInterruptedChapterWriteRuns()
	if err != nil {
		return fmt.Errorf("list interrupted write runs: %w", err)
	}
	var lastErr error
	for i := range runs {
		run := runs[i]
		baseline, err := p.truth.GetChapterWriteBaseline(run.ID)
		if err != nil {
			lastErr = err
			continue
		}
		if baseline == nil {
			now := time.Now()
			run.Status = model.WriteRunFailed
			run.ErrorMessage = "服务重启，写作任务中断"
			run.FinishedAt = &now
			if err := p.truth.SaveChapterWriteRun(&run); err != nil {
				lastErr = err
			}
			continue
		}
		if err := p.finishWriteRun(&run, baseline, model.WriteRunFailed, run.CurrentStage, "服务重启，写作任务中断"); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (p *Pipeline) resolveTargetChapterAndResumeStage(bookID uint, runType model.ChapterWriteRunType, mode model.ChapterWriteRetryMode, parentRunID *uint) (uint, model.ChapterWriteStage, error) {
	mode = normalizeWriteRetryMode(mode)
	if parentRunID == nil {
		if normalizeWriteRunType(runType) == model.WriteRunTypeRewriteLatest {
			next, err := p.truth.GetNextChapterNumber(bookID)
			if err != nil {
				return 0, "", err
			}
			if next <= 1 {
				return 0, "", fmt.Errorf("当前没有可重写的章节")
			}
			return next - 1, "", nil
		}
		chapterNumber, err := p.truth.GetNextChapterNumber(bookID)
		return chapterNumber, "", err
	}
	parentRun, err := p.truth.GetChapterWriteRun(*parentRunID)
	if err != nil {
		return 0, "", err
	}
	if parentRun == nil || parentRun.BookID != bookID {
		return 0, "", fmt.Errorf("parent run not found")
	}
	if mode == model.WriteRetryRestart {
		return parentRun.TargetChapter, "", nil
	}
	stages, err := p.truth.GetChapterWriteStages(parentRun.ID)
	if err != nil {
		return 0, "", err
	}
	for _, stage := range stages {
		if stage.Status == model.WriteStageFailed || stage.Status == model.WriteStageCancelled {
			return parentRun.TargetChapter, stage.Stage, nil
		}
	}
	return 0, "", fmt.Errorf("parent run has no failed stage to resume")
}

func normalizeWriteRetryMode(mode model.ChapterWriteRetryMode) model.ChapterWriteRetryMode {
	if mode == model.WriteRetryResumeFailedStage {
		return mode
	}
	return model.WriteRetryRestart
}

func normalizeWriteRunType(runType model.ChapterWriteRunType) model.ChapterWriteRunType {
	if runType == model.WriteRunTypeRewriteLatest {
		return runType
	}
	return model.WriteRunTypeNormal
}

func (p *Pipeline) assertRunNotCancelled(ctx context.Context, runID uint) error {
	select {
	case <-ctx.Done():
		return errWriteRunCancelled
	default:
	}
	run, err := p.truth.GetChapterWriteRun(runID)
	if err != nil {
		return err
	}
	if run != nil && run.CancelRequested {
		return errWriteRunCancelled
	}
	return nil
}

func (p *Pipeline) executeStageWithRetries(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, stage model.ChapterWriteStage) error {
	var lastErr error
	for attempt := uint(1); attempt <= maxWriteStageAttempts; attempt++ {
		if err := p.assertRunNotCancelled(ctx, run.ID); err != nil {
			return err
		}
		tokenCursor, trackTokens := p.tokenUsageCursor()
		err := p.executeStage(ctx, run, state, stage, attempt)
		stageRun, _ := p.truth.GetChapterWriteStage(run.ID, stage)
		if stageRun != nil && stageRun.Attempt == attempt {
			_ = p.attachStageMeta(stageRun, stageRunMeta{
				Attempt:      attempt,
				MaxAttempts:  maxWriteStageAttempts,
				TokenSummary: p.tokenSummaryAfter(tokenCursor, trackTokens),
			})
		}
		if err == nil {
			return nil
		}
		if errors.Is(err, errWriteRunCancelled) || errors.Is(err, context.Canceled) {
			return err
		}
		lastErr = err
		if attempt < maxWriteStageAttempts {
			continue
		}
	}
	return lastErr
}

func (p *Pipeline) executeStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, stage model.ChapterWriteStage, attempt uint) error {
	switch stage {
	case model.WriteStageContext:
		return p.executeContextStage(ctx, run, state, attempt)
	case model.WriteStagePlanning:
		return p.executePlanningStage(ctx, run, state, attempt)
	case model.WriteStageWriting:
		return p.executeWritingStage(ctx, run, state, attempt)
	case model.WriteStageAuditing:
		return p.executeAuditingStage(ctx, run, state, attempt)
	case model.WriteStageRevising:
		return p.executeRevisingStage(ctx, run, state, attempt)
	case model.WriteStagePolishing:
		return p.executePolishingStage(ctx, run, state, attempt)
	case model.WriteStageExtracting:
		return p.executeExtractingStage(ctx, run, state, attempt)
	case model.WriteStageSnapshot:
		return p.executeSnapshotStage(ctx, run, state, attempt)
	default:
		return fmt.Errorf("unsupported stage: %s", stage)
	}
}

func (p *Pipeline) startStage(run *model.ChapterWriteRun, stage model.ChapterWriteStage, attempt uint, inputPayload any, inputSummary string) (*model.ChapterWriteStageRun, error) {
	now := time.Now()
	if attempt == 0 {
		attempt = 1
	}
	run.CurrentStage = stage
	if err := p.truth.SaveChapterWriteRun(run); err != nil {
		return nil, err
	}
	payload := marshalStagePayload(inputPayload)
	stageRun := &model.ChapterWriteStageRun{
		RunID:        run.ID,
		Stage:        stage,
		Attempt:      attempt,
		Status:       model.WriteStageRunning,
		InputSummary: clipText(inputSummary, 400),
		InputPayload: payload,
		StartedAt:    &now,
	}
	if err := p.truth.CreateChapterWriteStageRun(stageRun); err != nil {
		return nil, err
	}
	return stageRun, nil
}

func (p *Pipeline) finishStage(stageRun *model.ChapterWriteStageRun, status model.ChapterWriteStageStatus, outputPayload any, outputSummary, errorMessage string) error {
	now := time.Now()
	stageRun.Status = status
	stageRun.OutputSummary = clipText(outputSummary, 400)
	stageRun.OutputPayload = marshalStagePayload(outputPayload)
	stageRun.ErrorMessage = errorMessage
	stageRun.FinishedAt = &now
	return p.truth.SaveChapterWriteStageRun(stageRun)
}

func (p *Pipeline) tokenUsageCursor() (uint, bool) {
	if p.tokenUsage == nil {
		return 0, false
	}
	id, err := p.tokenUsage.LatestID()
	if err != nil {
		return 0, false
	}
	return id, true
}

func (p *Pipeline) tokenSummaryAfter(cursor uint, ok bool) []stageTokenSummary {
	if !ok || p.tokenUsage == nil {
		return nil
	}
	rows, err := p.tokenUsage.SummaryAfterID(cursor)
	if err != nil {
		return nil
	}
	result := make([]stageTokenSummary, 0, len(rows))
	for _, row := range rows {
		if row.TotalTokens <= 0 && row.PromptTokens <= 0 && row.CompletionTokens <= 0 && row.CachedTokens <= 0 {
			continue
		}
		result = append(result, stageTokenSummary{
			AgentName:        row.AgentName,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
			CachedTokens:     row.CachedTokens,
			TotalTokens:      row.TotalTokens,
		})
	}
	return result
}

func (p *Pipeline) attachStageMeta(stageRun *model.ChapterWriteStageRun, meta stageRunMeta) error {
	if stageRun == nil {
		return nil
	}
	stageRun.OutputPayload = attachStageMetaToPayload(stageRun.OutputPayload, meta)
	return p.truth.SaveChapterWriteStageRun(stageRun)
}

func attachStageMetaToPayload(payload string, meta stageRunMeta) string {
	raw := map[string]any{}
	if strings.TrimSpace(payload) != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(payload), &parsed); err == nil && parsed != nil {
			raw = parsed
		}
	}
	raw["meta"] = meta
	b, err := json.Marshal(raw)
	if err != nil {
		return payload
	}
	return string(b)
}

func (p *Pipeline) executeContextStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	stageRun, err := p.startStage(run, model.WriteStageContext, attempt, map[string]any{
		"chapter_number": state.ChapterNumber,
		"run_type":       normalizeWriteRunType(run.RunType),
	}, fmt.Sprintf("构建第 %d 章上下文", state.ChapterNumber))
	if err != nil {
		return err
	}
	contextPkg, err := p.buildContext(ctx, run.BookID, state.ChapterNumber)
	if err != nil {
		_ = p.finishStage(stageRun, model.WriteStageFailed, nil, "", err.Error())
		return err
	}
	if normalizeWriteRunType(run.RunType) == model.WriteRunTypeRewriteLatest && state.OriginalChapter != nil {
		contextPkg = appendRewriteLatestContext(contextPkg, state.OriginalChapter, run.UserInput)
	}
	state.ContextPkg = contextPkg
	return p.finishStage(stageRun, model.WriteStageSucceeded, stageContextPayload{Context: contextPkg}, contextPkg, "")
}

func (p *Pipeline) executePlanningStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	plannerModelID := p.resolveModelID(run.BookID, "planner", run.RequestedModelID)
	plannerAny, ok := p.registry.Get("planner")
	if !ok {
		return fmt.Errorf("planner agent not found")
	}
	plannerInput := fmt.Sprintf(plannerBuildInput,
		state.Book.Title,
		state.Book.Genre.Name,
		state.Book.Platform.Name,
		state.ChapterNumber,
		state.Book.ChapterWordCount,
		state.ContextPkg,
		run.UserInput,
	)
	stageRun, err := p.startStage(run, model.WriteStagePlanning, attempt, map[string]any{
		"model_id": plannerModelID,
		"prompt":   plannerInput,
	}, plannerInput)
	if err != nil {
		return err
	}
	memo, err := p.llm.ChatForAgent(ctx, "planner", plannerModelID, plannerAny.SystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: plannerInput},
	}, 0.7)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
	}
	state.Memo = memo
	composed, err := p.composeChapterContext(state.Book, state.ChapterNumber, memo, run.UserInput, normalizeWriteRunType(run.RunType), state.OriginalChapter)
	if err != nil {
		_ = p.finishStage(stageRun, model.WriteStageFailed, stagePlanningPayload{Memo: memo}, memo, err.Error())
		return err
	}
	state.Composed = composed
	state.ContextPkg = composed.ContextText
	return p.finishStage(stageRun, model.WriteStageSucceeded, stagePlanningPayload{
		Memo:     memo,
		Context:  composed.ContextText,
		Composed: composed,
	}, memo, "")
}

func (p *Pipeline) executeWritingStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	writerModelID := p.resolveModelID(run.BookID, "writer", run.RequestedModelID)
	writerAny, ok := p.registry.Get("writer")
	if !ok {
		return fmt.Errorf("writer agent not found")
	}
	writerAgent, ok := writerAny.(*agent.WriterAgent)
	if !ok {
		return fmt.Errorf("invalid writer agent")
	}
	systemPrompt := writerAgent.BuildSystemPrompt(agent.WriterInput{
		Platform:         state.Book.Platform.Name,
		GenreName:        state.Book.Genre.Name,
		ChapterWordCount: state.Book.ChapterWordCount,
		ChapterNumber:    int(state.ChapterNumber),
		IsGoverned:       true,
	})
	writerInput := fmt.Sprintf(writerBuildInput, state.Book.Title, state.ChapterNumber, state.Memo, state.ContextPkg)
	stageRun, err := p.startStage(run, model.WriteStageWriting, attempt, map[string]any{
		"model_id":      writerModelID,
		"system_prompt": systemPrompt,
		"prompt":        writerInput,
	}, writerInput)
	if err != nil {
		return err
	}
	rawOutput, err := p.llm.ChatForAgent(ctx, "writer", writerModelID, systemPrompt, []llm.AgentMessage{
		{Role: "user", Content: writerInput},
	}, 0.8)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
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
		title = fmt.Sprintf("第%d章", state.ChapterNumber)
	}
	if content == "" {
		_ = p.finishStage(stageRun, model.WriteStageFailed, stageWritingPayload{Title: title, Sections: sections, Raw: rawOutput}, "", "writer 输出缺少 CHAPTER_CONTENT")
		return fmt.Errorf("writer 输出缺少 CHAPTER_CONTENT，未写入章节正文")
	}
	state.Title = title
	state.Content = strings.TrimSpace(content)
	state.Sections = sections
	state.FinalizedSource = "writer"
	return p.finishStage(stageRun, model.WriteStageSucceeded, stageWritingPayload{
		Title:    title,
		Content:  state.Content,
		Sections: sections,
		Raw:      rawOutput,
		Source:   "writer",
	}, firstNonEmpty(title, state.Content), "")
}

func (p *Pipeline) executeAuditingStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	stageRun, err := p.startStage(run, model.WriteStageAuditing, attempt, map[string]any{
		"memo":    state.Memo,
		"context": state.ContextPkg,
		"content": state.Content,
	}, state.Content)
	if err != nil {
		return err
	}
	result, raw, err := p.runAuditor(ctx, state.Book, state.ChapterNumber, state.Memo, state.ContextPkg, state.Content, run.RequestedModelID)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
	}
	state.AuditRaw = raw
	state.AuditResult = result
	return p.finishStage(stageRun, model.WriteStageSucceeded, stageAuditingPayload{Raw: raw, Result: result}, raw, "")
}

func (p *Pipeline) executeRevisingStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	stageRun, err := p.startStage(run, model.WriteStageRevising, attempt, map[string]any{
		"audit_result": state.AuditResult,
		"audit_raw":    state.AuditRaw,
		"content":      state.Content,
	}, state.AuditRaw)
	if err != nil {
		return err
	}
	if state.AuditResult.Passed {
		return p.finishStage(stageRun, model.WriteStageSkipped, stageRevisingPayload{
			Applied: false,
			Content: state.Content,
			Reason:  "auditor passed",
		}, "审稿通过，无需修订", "")
	}
	revisedContent, revisedSections, reviserRaw, err := p.runReviser(ctx, state.Book, state.ChapterNumber, state.Memo, state.ContextPkg, state.Content, state.AuditRaw, run.RequestedModelID)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
	}
	revisedContent = strings.TrimSpace(revisedContent)
	revisedAudit, revisedAuditRaw, auditErr := p.runAuditor(ctx, state.Book, state.ChapterNumber, state.Memo, state.ContextPkg, revisedContent, run.RequestedModelID)
	if auditErr != nil {
		decision := p.failedRevisionGateDecision(state.AuditResult, "修订稿候选审计失败，保留原稿："+auditErr.Error())
		return p.finishStage(stageRun, model.WriteStageSkipped, stageRevisingPayload{
			Applied:    false,
			Content:    state.Content,
			Sections:   revisedSections,
			Raw:        reviserRaw,
			Reason:     decision.Reason,
			Evaluation: &decision,
		}, decision.Reason, "")
	}
	decision, scoreErr := p.decideRevisionGate(state.AuditResult, revisedAudit)
	if scoreErr != nil {
		_ = p.finishStage(stageRun, model.WriteStageFailed, nil, "", scoreErr.Error())
		return scoreErr
	}
	if !decision.Applied {
		return p.finishStage(stageRun, model.WriteStageSkipped, stageRevisingPayload{
			Applied:           false,
			Content:           state.Content,
			Sections:          revisedSections,
			Raw:               reviserRaw,
			Reason:            decision.Reason,
			Evaluation:        &decision,
			CandidateAuditRaw: revisedAuditRaw,
		}, decision.Reason, "")
	}

	state.Content = revisedContent
	if strings.TrimSpace(revisedSections["UPDATED_STATE"]) != "" {
		state.Sections["UPDATED_STATE"] = revisedSections["UPDATED_STATE"]
	}
	if strings.TrimSpace(revisedSections["UPDATED_HOOKS"]) != "" {
		state.Sections["UPDATED_HOOKS"] = revisedSections["UPDATED_HOOKS"]
	}
	if strings.TrimSpace(revisedSections["POST_SETTLEMENT"]) != "" {
		state.Sections["POST_SETTLEMENT"] = revisedSections["POST_SETTLEMENT"]
	}
	state.AuditResult = revisedAudit
	state.AuditRaw = revisedAuditRaw
	state.FinalizedSource = "reviser"
	return p.finishStage(stageRun, model.WriteStageSucceeded, stageRevisingPayload{
		Applied:           true,
		Content:           state.Content,
		Sections:          revisedSections,
		Raw:               reviserRaw,
		Evaluation:        &decision,
		CandidateAuditRaw: revisedAuditRaw,
	}, state.Content, "")
}

func (p *Pipeline) executePolishingStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	stageRun, err := p.startStage(run, model.WriteStagePolishing, attempt, map[string]any{
		"content": state.Content,
	}, state.Content)
	if err != nil {
		return err
	}
	polishedContent, err := p.runPolisher(ctx, state.Book, state.ChapterNumber, state.Content, run.RequestedModelID)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
	}
	if strings.TrimSpace(polishedContent) == "" {
		_ = p.finishStage(stageRun, model.WriteStageFailed, nil, "", "polisher 未产出可用正文")
		return fmt.Errorf("polisher 未产出可用正文")
	}
	state.Content = strings.TrimSpace(polishedContent)
	state.FinalizedSource = "polisher"
	return p.finishStage(stageRun, model.WriteStageSucceeded, stagePolishingPayload{Content: state.Content}, state.Content, "")
}

func (p *Pipeline) executeExtractingStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	stageRun, err := p.startStage(run, model.WriteStageExtracting, attempt, map[string]any{
		"title":   state.Title,
		"content": state.Content,
	}, state.Content)
	if err != nil {
		return err
	}
	state.Sections["CHAPTER_TITLE"] = state.Title
	state.Sections["CHAPTER_CONTENT"] = state.Content
	writerModelID := p.resolveModelID(run.BookID, "writer", run.RequestedModelID)
	settleSections, settleDelta, settlerRaw, err := p.settleTruthFiles(ctx, state.Book, state.ChapterNumber, state.Title, state.Content, writerModelID)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
	}
	for key, value := range settleSections {
		if strings.TrimSpace(value) != "" {
			state.Sections[key] = value
		}
	}
	extractedTruth, err := p.extractTruthFiles(
		ctx,
		run.BookID,
		state.ChapterNumber,
		fmt.Sprintf("章节标题：%s\n\n章节正文：\n%s", state.Title, state.Content),
		writerModelID,
	)
	if err != nil {
		_ = p.finishStage(stageRun, cancelledStageStatus(err), nil, "", err.Error())
		return err
	}
	if extractedTruth == nil {
		_ = p.finishStage(stageRun, model.WriteStageFailed, nil, "", "extract truth files returned empty result")
		return fmt.Errorf("extract truth files returned empty result")
	}
	state.SettleSections = settleSections
	state.SettleDelta = settleDelta
	state.ExtractedTruth = extractedTruth
	return p.finishStage(stageRun, model.WriteStageSucceeded, stageExtractingPayload{
		SettlerSections: settleSections,
		SettlerDelta:    settleDelta,
		SettlerRaw:      settlerRaw,
		ExtractedTruth:  extractedTruth,
	}, firstNonEmpty(state.Title, state.Content), "")
}

func (p *Pipeline) executeSnapshotStage(ctx context.Context, run *model.ChapterWriteRun, state *writeRunState, attempt uint) error {
	stageRun, err := p.startStage(run, model.WriteStageSnapshot, attempt, map[string]any{
		"title":   state.Title,
		"content": state.Content,
	}, "提交章节与真相状态")
	if err != nil {
		return err
	}
	tracePayload := map[string]any{
		"run_id":           run.ID,
		"chapter_number":   state.ChapterNumber,
		"title":            state.Title,
		"finalized_source": state.FinalizedSource,
		"writer_sections":  sortedSectionNames(state.Sections),
		"settler_sections": sortedSectionNames(state.SettleSections),
		"extract_counts": map[string]any{
			"characters": len(state.ExtractedTruth.Characters),
			"facts":      len(state.ExtractedTruth.DurableFacts),
			"hooks":      len(state.ExtractedTruth.Hooks),
			"evidence":   len(state.ExtractedTruth.EvidenceNotes),
		},
	}
	if state.Composed != nil {
		tracePayload["composer"] = map[string]any{
			"context_package": state.Composed.ContextPackage,
			"rule_stack":      state.Composed.RuleStack,
			"trace":           state.Composed.Trace,
		}
	}

	err = p.truth.WithinTx(func(txTruth repository.TruthFileRepository) error {
		txPipeline := p.withTruthRepo(txTruth)
		if normalizeWriteRunType(run.RunType) == model.WriteRunTypeRewriteLatest {
			if err := txTruth.DeleteLatestChapterCascade(run.BookID, state.ChapterNumber); err != nil {
				return fmt.Errorf("rollback original latest chapter: %w", err)
			}
		}
		ch := &model.Chapter{
			BookID:        run.BookID,
			ChapterNumber: state.ChapterNumber,
			Title:         state.Title,
			Content:       state.Content,
			WordCount:     uint(len([]rune(state.Content))),
			Status:        model.ChapterDraft,
		}
		ruleStackContent := ""
		if state.Composed != nil {
			ruleStackContent = marshalArtifactPayload(state.Composed.RuleStack)
		}
		for _, artifact := range []*model.RuntimeArtifact{
			{BookID: run.BookID, ChapterNumber: state.ChapterNumber, ArtifactType: model.ArtifactContext, Content: state.ContextPkg},
			{BookID: run.BookID, ChapterNumber: state.ChapterNumber, ArtifactType: model.ArtifactIntent, Content: state.Memo},
			{BookID: run.BookID, ChapterNumber: state.ChapterNumber, ArtifactType: model.ArtifactPlan, Content: state.Sections["PRE_WRITE_CHECK"]},
			{BookID: run.BookID, ChapterNumber: state.ChapterNumber, ArtifactType: model.ArtifactRuleStack, Content: ruleStackContent},
		} {
			if artifact.ArtifactType == model.ArtifactRuleStack && ruleStackContent == "" {
				continue
			}
			if err := txTruth.SaveRuntimeArtifact(artifact); err != nil {
				return err
			}
		}
		if err := txTruth.SaveChapter(ch); err != nil {
			return err
		}
		if err := txPipeline.applySettlerDelta(run.BookID, state.ChapterNumber, state.Title, state.Sections["POST_SETTLEMENT"], state.SettleDelta); err != nil {
			return err
		}
		if err := txPipeline.persistExtractedTruthFiles(run.BookID, state.ChapterNumber, state.ExtractedTruth, extractionOptions{
			SaveHooks:   false,
			SaveSummary: true,
		}); err != nil {
			return err
		}
		txPipeline.saveChapterSnapshot(run.BookID, state.ChapterNumber, state.Sections)
		if payload, marshalErr := json.Marshal(tracePayload); marshalErr == nil {
			if err := txTruth.SaveRuntimeArtifact(&model.RuntimeArtifact{
				BookID:        run.BookID,
				ChapterNumber: state.ChapterNumber,
				ArtifactType:  model.ArtifactTrace,
				Content:       string(payload),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_ = p.finishStage(stageRun, model.WriteStageFailed, nil, "", err.Error())
		return err
	}
	return p.finishStage(stageRun, model.WriteStageSucceeded, stageSnapshotPayload{
		ChapterNumber: state.ChapterNumber,
		Title:         state.Title,
		Content:       state.Content,
	}, fmt.Sprintf("第%d章已提交", state.ChapterNumber), "")
}

func appendRewriteLatestContext(contextPkg string, original *model.Chapter, userInput string) string {
	if original == nil {
		return contextPkg
	}
	var b strings.Builder
	b.WriteString(contextPkg)
	if !strings.HasSuffix(contextPkg, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n## 重写最后一章任务\n")
	b.WriteString(fmt.Sprintf("- 本次不是写下一章，而是重写第 %d 章《%s》。\n", original.ChapterNumber, strings.TrimSpace(original.Title)))
	b.WriteString("- 必须保持章节编号不变，基于上一章承接重新写这一章。\n")
	b.WriteString("- 当前数据库里的状态可能包含旧版本章产生的结果；重写时以用户要求和旧章正文为对照，不要把旧版本章的结算状态当成不可更改事实。\n")
	if strings.TrimSpace(userInput) != "" {
		b.WriteString("\n### 用户重写要求\n")
		b.WriteString(strings.TrimSpace(userInput))
		b.WriteString("\n")
	}
	b.WriteString("\n### 旧版章节正文\n")
	b.WriteString(fmt.Sprintf("标题：%s\n\n", strings.TrimSpace(original.Title)))
	content := strings.TrimSpace(original.Content)
	if len([]rune(content)) > 6000 {
		runes := []rune(content)
		content = string(runes[:6000]) + "\n\n（旧版正文过长，已截断展示。）"
	}
	b.WriteString(content)
	b.WriteString("\n")
	return b.String()
}

func cancelledStageStatus(err error) model.ChapterWriteStageStatus {
	if errors.Is(err, errWriteRunCancelled) || errors.Is(err, context.Canceled) {
		return model.WriteStageCancelled
	}
	return model.WriteStageFailed
}

func marshalStagePayload(v any) string {
	if v == nil {
		return ""
	}
	payload, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(payload)
}

func (p *Pipeline) finishWriteRun(run *model.ChapterWriteRun, baseline *model.ChapterWriteBaseline, status model.ChapterWriteRunStatus, stage model.ChapterWriteStage, errorMessage string) error {
	now := time.Now()
	run.Status = status
	run.CurrentStage = stage
	run.ErrorMessage = errorMessage
	run.FinishedAt = &now
	if err := p.truth.SaveChapterWriteRun(run); err != nil {
		return err
	}

	targetStatus := baseline.RecoverStatus
	if status == model.WriteRunSucceeded {
		targetStatus = model.BookStatusActive
	}
	return p.truth.UpdateBookStatus(run.BookID, targetStatus)
}

func (p *Pipeline) loadResumeState(parentRunID, newRunID uint, resumeStage model.ChapterWriteStage, state *writeRunState) error {
	if resumeStage == "" {
		return nil
	}
	stages, err := p.truth.GetChapterWriteStages(parentRunID)
	if err != nil {
		return err
	}
	latestBeforeResume := map[model.ChapterWriteStage]model.ChapterWriteStageRun{}
	for _, stageRun := range stages {
		if stageRun.Stage == resumeStage {
			break
		}
		latestBeforeResume[stageRun.Stage] = stageRun
	}
	stageOrder := []model.ChapterWriteStage{
		model.WriteStageContext,
		model.WriteStagePlanning,
		model.WriteStageWriting,
		model.WriteStageAuditing,
		model.WriteStageRevising,
		model.WriteStagePolishing,
		model.WriteStageExtracting,
		model.WriteStageSnapshot,
	}
	for _, stage := range stageOrder {
		if stage == resumeStage {
			break
		}
		stageRun, ok := latestBeforeResume[stage]
		if !ok {
			continue
		}
		if stageRun.Status != model.WriteStageSucceeded && stageRun.Status != model.WriteStageSkipped {
			return fmt.Errorf("stage %s is not reusable", stageRun.Stage)
		}
		cloned := stageRun
		cloned.ID = 0
		cloned.RunID = newRunID
		cloned.Attempt = 1
		if err := p.truth.CreateChapterWriteStageRun(&cloned); err != nil {
			return err
		}
		if err := hydrateRunState(stageRun.Stage, stageRun.OutputPayload, state); err != nil {
			return err
		}
	}
	return nil
}

func hydrateRunState(stage model.ChapterWriteStage, payload string, state *writeRunState) error {
	if strings.TrimSpace(payload) == "" {
		return nil
	}
	switch stage {
	case model.WriteStageContext:
		var out stageContextPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		state.ContextPkg = out.Context
	case model.WriteStagePlanning:
		var out stagePlanningPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		state.Memo = out.Memo
		if out.Composed != nil {
			state.Composed = out.Composed
			state.ContextPkg = out.Composed.ContextText
		}
		if strings.TrimSpace(out.Context) != "" {
			state.ContextPkg = out.Context
		}
	case model.WriteStageWriting:
		var out stageWritingPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		state.Title = out.Title
		state.Content = out.Content
		state.Sections = out.Sections
		state.FinalizedSource = out.Source
	case model.WriteStageAuditing:
		var out stageAuditingPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		state.AuditRaw = out.Raw
		state.AuditResult = out.Result
	case model.WriteStageRevising:
		var out stageRevisingPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		if out.Applied {
			state.Content = out.Content
			for key, value := range out.Sections {
				if strings.TrimSpace(value) != "" {
					state.Sections[key] = value
				}
			}
			state.FinalizedSource = "reviser"
		}
	case model.WriteStagePolishing:
		var out stagePolishingPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		state.Content = out.Content
		state.FinalizedSource = "polisher"
	case model.WriteStageExtracting:
		var out stageExtractingPayload
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			return err
		}
		state.SettleSections = out.SettlerSections
		state.SettleDelta = out.SettlerDelta
		state.ExtractedTruth = out.ExtractedTruth
		for key, value := range out.SettlerSections {
			if strings.TrimSpace(value) != "" {
				state.Sections[key] = value
			}
		}
	}
	return nil
}
