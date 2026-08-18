package agent

import "testing"

func TestScorerRejectsWorseWarningCount(t *testing.T) {
	scorer := NewScorerAgent()
	base := ScoreAuditResult{
		Passed:       false,
		OverallScore: 70,
		Issues: []ScoreIssue{
			{Severity: "critical", Category: "continuity", Description: "主线断裂"},
		},
	}
	revised := ScoreAuditResult{
		Passed:       false,
		OverallScore: 80,
		Issues: []ScoreIssue{
			{Severity: "critical", Category: "continuity", Description: "主线断裂"},
			{Severity: "warning", Category: "pacing", Description: "节奏拖沓"},
		},
	}

	decision := scorer.DecideRevision(base, revised)

	if decision.Applied {
		t.Fatalf("revision should be rejected when warning count worsens")
	}
	if len(decision.WorsenedMetrics) != 1 || decision.WorsenedMetrics[0] != "warning_count" {
		t.Fatalf("unexpected worsened metrics: %#v", decision.WorsenedMetrics)
	}
}

func TestScorerAppliesNetImprovement(t *testing.T) {
	scorer := NewScorerAgent()
	base := ScoreAuditResult{
		Passed:       false,
		OverallScore: 70,
		Issues: []ScoreIssue{
			{Severity: "critical", Category: "continuity", Description: "主线断裂"},
			{Severity: "warning", Category: "段落等长", Description: "AI痕迹"},
		},
	}
	revised := ScoreAuditResult{
		Passed:       true,
		OverallScore: 88,
		Issues:       []ScoreIssue{},
	}

	decision := scorer.DecideRevision(base, revised)

	if !decision.Applied {
		t.Fatalf("revision should be applied: %#v", decision)
	}
	if decision.Base.AITellCount != 1 {
		t.Fatalf("base ai tell count = %d, want 1", decision.Base.AITellCount)
	}
	if len(decision.ImprovedMetrics) == 0 {
		t.Fatalf("expected improved metrics")
	}
}

func TestScorerRejectsNoVerifiableImprovement(t *testing.T) {
	scorer := NewScorerAgent()
	base := ScoreAuditResult{Passed: false, OverallScore: 80}
	revised := ScoreAuditResult{Passed: false, OverallScore: 80}

	decision := scorer.DecideRevision(base, revised)

	if decision.Applied {
		t.Fatalf("revision without verifiable improvement should be rejected")
	}
}
