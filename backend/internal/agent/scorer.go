package agent

import "strings"

type ScorerAgent struct{}

func NewScorerAgent() *ScorerAgent {
	return &ScorerAgent{}
}

func (a *ScorerAgent) Name() string { return "scorer" }

func (a *ScorerAgent) SystemPrompt() string {
	return scorerPrompt
}

type ScoreIssue struct {
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type ScoreAuditResult struct {
	Passed       bool         `json:"passed"`
	OverallScore int          `json:"overall_score"`
	Issues       []ScoreIssue `json:"issues"`
}

type CandidateMetrics struct {
	Passed        bool `json:"passed"`
	OverallScore  int  `json:"overall_score"`
	CriticalCount int  `json:"critical_count"`
	WarningCount  int  `json:"warning_count"`
	AITellCount   int  `json:"ai_tell_count"`
}

type ScoreDecision struct {
	Applied          bool             `json:"applied"`
	Reason           string           `json:"reason"`
	Base             CandidateMetrics `json:"base"`
	Revised          CandidateMetrics `json:"revised"`
	WorsenedMetrics  []string         `json:"worsened_metrics,omitempty"`
	ImprovedMetrics  []string         `json:"improved_metrics,omitempty"`
	EvaluationFailed bool             `json:"evaluation_failed,omitempty"`
}

func (a *ScorerAgent) DecideRevision(baseResult, revisedResult ScoreAuditResult) ScoreDecision {
	base := a.EvaluateAuditCandidate(baseResult)
	revised := a.EvaluateAuditCandidate(revisedResult)
	decision := ScoreDecision{
		Base:    base,
		Revised: revised,
	}

	if revised.CriticalCount > base.CriticalCount {
		decision.WorsenedMetrics = append(decision.WorsenedMetrics, "critical_count")
	}
	if revised.WarningCount > base.WarningCount {
		decision.WorsenedMetrics = append(decision.WorsenedMetrics, "warning_count")
	}
	if revised.AITellCount > base.AITellCount {
		decision.WorsenedMetrics = append(decision.WorsenedMetrics, "ai_tell_count")
	}
	if len(decision.WorsenedMetrics) > 0 {
		decision.Reason = "修订稿引入了更差的审稿指标，保留原稿"
		return decision
	}

	if revised.CriticalCount < base.CriticalCount {
		decision.ImprovedMetrics = append(decision.ImprovedMetrics, "critical_count")
	}
	if revised.WarningCount < base.WarningCount {
		decision.ImprovedMetrics = append(decision.ImprovedMetrics, "warning_count")
	}
	if revised.AITellCount < base.AITellCount {
		decision.ImprovedMetrics = append(decision.ImprovedMetrics, "ai_tell_count")
	}
	if revised.OverallScore > base.OverallScore {
		decision.ImprovedMetrics = append(decision.ImprovedMetrics, "overall_score")
	}
	if revised.Passed && !base.Passed {
		decision.ImprovedMetrics = append(decision.ImprovedMetrics, "passed")
	}
	if len(decision.ImprovedMetrics) == 0 {
		decision.Reason = "修订稿没有带来可验证的净提升，保留原稿"
		return decision
	}

	decision.Applied = true
	decision.Reason = "修订稿通过候选评估，应用修订"
	return decision
}

func (a *ScorerAgent) FailedDecision(baseResult ScoreAuditResult, reason string) ScoreDecision {
	return ScoreDecision{
		Applied:          false,
		Reason:           reason,
		Base:             a.EvaluateAuditCandidate(baseResult),
		Revised:          CandidateMetrics{},
		EvaluationFailed: true,
	}
}

func (a *ScorerAgent) EvaluateAuditCandidate(result ScoreAuditResult) CandidateMetrics {
	metrics := CandidateMetrics{
		Passed:       result.Passed,
		OverallScore: result.OverallScore,
	}
	for _, issue := range result.Issues {
		switch strings.ToLower(strings.TrimSpace(issue.Severity)) {
		case "critical":
			metrics.CriticalCount++
		case "warning":
			metrics.WarningCount++
		}
		if a.isAITellIssue(issue) {
			metrics.AITellCount++
		}
	}
	return metrics
}

func (a *ScorerAgent) isAITellIssue(issue ScoreIssue) bool {
	text := strings.ToLower(strings.TrimSpace(issue.Category + " " + issue.Description + " " + issue.Suggestion))
	markers := []string{
		"ai味",
		"ai 味",
		"ai-tell",
		"ai tell",
		"llm",
		"段落等长",
		"套话密度",
		"公式化转折",
		"列表式结构",
		"词汇疲劳",
		"高疲劳词",
		"报告术语",
		"元叙事",
		"连续短段",
		"段落过碎",
		"paragraph uniformity",
		"hedge density",
		"formulaic",
		"list-like",
	}
	for _, marker := range markers {
		if strings.Contains(text, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

const scorerPrompt = `你是候选章节评分器。职责是比较原稿与修订稿的审稿结果，只决定是否允许修订稿替换原稿。

默认规则：
1. critical_count、warning_count、ai_tell_count 任一项变差，拒绝修订稿
2. 无可验证净提升，拒绝修订稿
3. 只有修订稿不变差且至少一项关键指标提升，才允许应用

当前实现由代码层执行确定性评分；本 prompt 保留给后续 LLM 评分扩展。`
