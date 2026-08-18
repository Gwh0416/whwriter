package pipeline

import (
	"fmt"

	"whwriter/backend/internal/agent"
)

func (p *Pipeline) decideRevisionGate(baseResult, revisedResult auditResult) (agent.ScoreDecision, error) {
	scorer, err := p.scorer()
	if err != nil {
		return agent.ScoreDecision{}, err
	}
	return scorer.DecideRevision(toScoreAuditResult(baseResult), toScoreAuditResult(revisedResult)), nil
}

func (p *Pipeline) failedRevisionGateDecision(baseResult auditResult, reason string) agent.ScoreDecision {
	scorer, err := p.scorer()
	if err != nil {
		return agent.ScoreDecision{
			Applied:          false,
			Reason:           reason,
			EvaluationFailed: true,
		}
	}
	return scorer.FailedDecision(toScoreAuditResult(baseResult), reason)
}

func (p *Pipeline) scorer() (*agent.ScorerAgent, error) {
	scorerAny, ok := p.registry.Get("scorer")
	if !ok {
		return nil, fmt.Errorf("scorer agent not found")
	}
	scorer, ok := scorerAny.(*agent.ScorerAgent)
	if !ok {
		return nil, fmt.Errorf("invalid scorer agent")
	}
	return scorer, nil
}

func toScoreAuditResult(result auditResult) agent.ScoreAuditResult {
	issues := make([]agent.ScoreIssue, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issues = append(issues, agent.ScoreIssue{
			Severity:    issue.Severity,
			Category:    issue.Category,
			Description: issue.Description,
			Suggestion:  issue.Suggestion,
		})
	}
	return agent.ScoreAuditResult{
		Passed:       result.Passed,
		OverallScore: result.OverallScore,
		Issues:       issues,
	}
}
