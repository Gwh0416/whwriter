package agent

import (
	"strings"
	"testing"

	"whwriter/backend/internal/model"
)

func TestComposerComposeBuildsContextRuleStackAndTrace(t *testing.T) {
	book := &model.Book{ID: 1, Title: "测试书", ChapterWordCount: 3000}
	composer := NewComposerAgent()

	out := composer.Compose(ComposeInput{
		Book:          book,
		ChapterNumber: 3,
		Memo:          "## 当前任务\n推进 H002，并不要偏离主线。",
		UserInput:     "本章重点写师徒矛盾。",
		Foundations: []model.BookFoundation{
			{FileType: model.FoundationStoryFrame, Content: "故事框架"},
			{FileType: model.FoundationVolumeMap, Content: "卷纲"},
		},
		BookState: &model.BookState{
			ProtagonistName: "林秋",
			CurrentGoal:     "查明借条来源",
		},
		Characters: []model.Character{
			{Name: "林秋", RoleType: model.CharacterProtagonist, Profile: "行动驱动"},
		},
		Hooks: []model.Hook{
			{HookID: "H001", Type: model.HookPlot, Status: model.HookOpen, LastAdvancedChapter: 1, ExpectedPayoff: "旧账"},
			{HookID: "H002", Type: model.HookMystery, Status: model.HookProgressing, LastAdvancedChapter: 2, ExpectedPayoff: "借条真相"},
		},
	})

	if out.ContextPackage.ChapterNumber != 3 {
		t.Fatalf("chapter number = %d, want 3", out.ContextPackage.ChapterNumber)
	}
	if len(out.ContextPackage.SelectedContext) == 0 {
		t.Fatalf("selected context is empty")
	}
	if !containsString(out.Trace.SelectedSources, "runtime/chapter_memo") {
		t.Fatalf("trace missing chapter memo source: %#v", out.Trace.SelectedSources)
	}
	if !strings.Contains(out.ContextText, "truth/hooks") || !strings.Contains(out.ContextText, "H002") {
		t.Fatalf("context text should include referenced hook H002:\n%s", out.ContextText)
	}
	if len(out.RuleStack.ActiveOverrides) == 0 {
		t.Fatalf("rule stack should record active overrides")
	}
}

func TestComposerComposeAddsRetrievedKnowledgeWithoutReplacingHardContext(t *testing.T) {
	composer := NewComposerAgent()

	out := composer.Compose(ComposeInput{
		ChapterNumber: 8,
		Memo:          "推进 H007 的账本线索。",
		BookState: &model.BookState{
			ProtagonistName: "林秋",
			CurrentGoal:     "查明账本去向",
		},
		RetrievedKnowledge: []model.KnowledgeSearchResult{
			{
				SourceType: model.KnowledgeSourceFoundation,
				SourceID:   "story_frame",
				Title:      "基础设定：story_frame",
				Content:    "七号门下藏着与旧账相关的暗室。",
			},
		},
	})

	if !containsString(out.Trace.ProtectedSources, "truth/book_state") {
		t.Fatalf("book state must remain protected: %#v", out.Trace.ProtectedSources)
	}
	if !containsString(out.Trace.SupportingSources, "retrieval/foundation/story_frame#0") {
		t.Fatalf("retrieval source missing: %#v", out.Trace.SupportingSources)
	}
	if !strings.Contains(out.ContextText, "七号门下藏着与旧账相关的暗室") {
		t.Fatalf("context should include retrieved knowledge:\n%s", out.ContextText)
	}
}

func TestComposerPrefersOneHopWikiContextOverGlobalFacts(t *testing.T) {
	composer := NewComposerAgent()
	out := composer.Compose(ComposeInput{
		ChapterNumber: 9,
		Memo:          "林秋检查青铜戒指。",
		Facts: []model.Fact{
			{Subject: "王五", Predicate: "持有", Object: "无关黑刀", Category: "item"},
		},
		WikiContext: &model.WikiGraphContext{
			Seeds: []model.WikiEntity{
				{ID: 1, EntityType: model.WikiEntityCharacter, CanonicalName: "林秋"},
			},
			Entities: []model.WikiEntity{
				{ID: 1, EntityType: model.WikiEntityCharacter, CanonicalName: "林秋", Summary: "追查旧账"},
				{ID: 2, EntityType: model.WikiEntityItem, CanonicalName: "青铜戒指", Summary: "旧账线索"},
			},
			Relations: []model.WikiRelationView{{
				WikiRelation: model.WikiRelation{
					SubjectEntityID:  1,
					Predicate:        "持有",
					ObjectEntityID:   uintTestPointer(2),
					ValidFromChapter: 3,
				},
				SubjectName: "林秋",
				SubjectType: model.WikiEntityCharacter,
				ObjectName:  "青铜戒指",
				ObjectType:  model.WikiEntityItem,
			}},
		},
	})

	if !strings.Contains(out.ContextText, "wiki/graph_context") ||
		!strings.Contains(out.ContextText, "林秋 --持有--> 青铜戒指") {
		t.Fatalf("wiki graph context missing:\n%s", out.ContextText)
	}
	if strings.Contains(out.ContextText, "无关黑刀") {
		t.Fatalf("global unrelated fact leaked into graph-first context:\n%s", out.ContextText)
	}
}

func uintTestPointer(value uint) *uint {
	return &value
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
