package sqlite

import (
	"path/filepath"
	"testing"

	"whwriter/backend/internal/model"
)

func TestReplaceChapterWikiEventsBuildsGraphAndEvidence(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "wiki-event.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	book := createWikiTestBook(t, db)
	repo := NewTruthFileRepo(db)
	if err := repo.SaveCharacter(&model.Character{
		BookID:          book.ID,
		Name:            "林秋（阿秋）",
		RoleType:        model.CharacterProtagonist,
		Profile:         "谨慎的杂役弟子",
		LastSeenChapter: 4,
	}); err != nil {
		t.Fatalf("save character: %v", err)
	}
	if err := repo.RefreshWikiEntities(book.ID); err != nil {
		t.Fatalf("refresh entities: %v", err)
	}
	if err := repo.SaveRuntimeArtifact(&model.RuntimeArtifact{
		BookID:        book.ID,
		ChapterNumber: 4,
		ArtifactType:  model.ArtifactEvidence,
		Content:       `{"events":[{"evidence_quote":"阿秋推开七号门"}]}`,
	}); err != nil {
		t.Fatalf("save evidence artifact: %v", err)
	}

	drafts := []model.WikiEventDraft{{
		EventKey:      "CH0004-E01",
		Title:         "推开七号门",
		EventType:     "discovery",
		Summary:       "林秋进入七号门后的暗室。",
		Participants:  []string{"阿秋"},
		Location:      "七号门",
		Consequence:   "旧账线索进入调查阶段。",
		EvidenceQuote: "阿秋推开七号门",
		EvidenceStart: 12,
		EvidenceEnd:   20,
	}}
	if err := repo.ReplaceChapterWikiEvents(book.ID, 4, drafts); err != nil {
		t.Fatalf("replace chapter events: %v", err)
	}
	if err := repo.RefreshWikiGraph(book.ID); err != nil {
		t.Fatalf("refresh graph after event persistence: %v", err)
	}

	events, err := repo.ListWikiEvents(book.ID, nil, 20)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].EventKey != "CH0004-E01" {
		t.Fatalf("unexpected events: %#v", events)
	}

	relations, err := repo.ListWikiRelations(book.ID, 4, nil, 20)
	if err != nil {
		t.Fatalf("list event relations: %v", err)
	}
	var eventRelationIDs []uint
	if !hasWikiRelation(relations, "林秋", "参与", "第4章·推开七号门", "") ||
		!hasWikiRelation(relations, "第4章·推开七号门", "发生于", "七号门", "") {
		t.Fatalf("event graph is incomplete: %#v", relations)
	}
	for _, relation := range relations {
		if relation.SourceType == wikiRelationSourceEvent {
			eventRelationIDs = append(eventRelationIDs, relation.ID)
		}
	}
	evidence, err := repo.GetWikiRelationEvidence(eventRelationIDs)
	if err != nil {
		t.Fatalf("get relation evidence: %v", err)
	}
	if len(evidence) != 2 {
		t.Fatalf("event evidence rows = %d, want 2: %#v", len(evidence), evidence)
	}
	for _, item := range evidence {
		if item.Quote != "阿秋推开七号门" || item.StartOffset != 12 || item.EndOffset != 20 || item.ArtifactID == nil {
			t.Fatalf("invalid event evidence: %#v", item)
		}
	}

	drafts[0].Title = "进入暗室"
	drafts[0].Summary = "林秋已经进入暗室。"
	if err := repo.ReplaceChapterWikiEvents(book.ID, 4, drafts); err != nil {
		t.Fatalf("replace chapter events again: %v", err)
	}
	events, err = repo.ListWikiEvents(book.ID, nil, 20)
	if err != nil {
		t.Fatalf("list replaced events: %v", err)
	}
	if len(events) != 1 || events[0].Title != "进入暗室" {
		t.Fatalf("event replacement is not idempotent: %#v", events)
	}
}
