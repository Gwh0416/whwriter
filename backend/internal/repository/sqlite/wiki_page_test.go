package sqlite

import (
	"path/filepath"
	"testing"

	"whwriter/backend/internal/model"
)

func TestWikiEntityPageIncludesAliasesRelationsEvidenceAndEvents(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "wiki-page.db"))
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
		Profile:         "追查旧账",
		LastSeenChapter: 5,
	}); err != nil {
		t.Fatalf("save character: %v", err)
	}
	if err := repo.SaveFact(&model.Fact{
		BookID:           book.ID,
		Subject:          "林秋",
		Predicate:        "持有",
		Object:           "青铜戒指",
		Category:         "item",
		ValidFromChapter: 2,
		SourceChapter:    2,
		EvidenceQuote:    "林秋收起青铜戒指",
		EvidenceStart:    10,
		EvidenceEnd:      19,
	}); err != nil {
		t.Fatalf("save fact: %v", err)
	}
	if err := repo.SaveRuntimeArtifact(&model.RuntimeArtifact{
		BookID:        book.ID,
		ChapterNumber: 2,
		ArtifactType:  model.ArtifactEvidence,
		Content:       `{"facts":[{"evidence_quote":"林秋收起青铜戒指"}]}`,
	}); err != nil {
		t.Fatalf("save evidence: %v", err)
	}
	if err := repo.RefreshWikiGraph(book.ID); err != nil {
		t.Fatalf("refresh graph: %v", err)
	}
	if err := repo.ReplaceChapterWikiEvents(book.ID, 5, []model.WikiEventDraft{{
		EventKey:      "CH0005-E01",
		Title:         "戒指示警",
		EventType:     "discovery",
		Summary:       "林秋发现戒指在七号门附近发热。",
		Participants:  []string{"阿秋"},
		Location:      "七号门",
		Consequence:   "林秋决定调查暗室。",
		EvidenceQuote: "戒指忽然发热",
		EvidenceStart: 40,
		EvidenceEnd:   46,
	}}); err != nil {
		t.Fatalf("save event: %v", err)
	}
	if err := repo.RefreshWikiGraph(book.ID); err != nil {
		t.Fatalf("refresh graph after event: %v", err)
	}

	entities, total, err := repo.SearchWikiEntities(
		book.ID,
		"阿秋",
		[]model.WikiEntityType{model.WikiEntityCharacter},
		20,
		0,
	)
	if err != nil {
		t.Fatalf("search wiki entities: %v", err)
	}
	if total != 1 || len(entities) != 1 || entities[0].CanonicalName != "林秋" {
		t.Fatalf("unexpected wiki search result: total=%d items=%#v", total, entities)
	}

	page, err := repo.GetWikiEntityPage(book.ID, entities[0].ID)
	if err != nil {
		t.Fatalf("get wiki entity page: %v", err)
	}
	if len(page.Aliases) < 2 {
		t.Fatalf("aliases missing: %#v", page.Aliases)
	}
	if !hasWikiRelation(page.Relations, "林秋", "持有", "青铜戒指", "") ||
		!hasWikiRelation(page.Relations, "林秋", "参与", "第5章·戒指示警", "") {
		t.Fatalf("relations missing from page: %#v", page.Relations)
	}
	if len(page.RelationEvidence) < 2 {
		t.Fatalf("relation evidence missing: %#v", page.RelationEvidence)
	}
	if len(page.Events) != 1 || page.Events[0].Title != "戒指示警" ||
		page.Events[0].LocationName != "七号门" ||
		len(page.Events[0].Participants) != 1 {
		t.Fatalf("event timeline missing: %#v", page.Events)
	}
}

func TestWikiGraphPrefersKnownEntityTypeForAnnotatedFactNames(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "wiki-entity-type.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	book := createWikiTestBook(t, db)
	repo := NewTruthFileRepo(db)
	for _, name := range []string{"许舟", "沈知夏"} {
		if err := repo.SaveCharacter(&model.Character{
			BookID:   book.ID,
			Name:     name,
			RoleType: model.CharacterMajor,
			Profile:  "测试角色",
		}); err != nil {
			t.Fatalf("save character %s: %v", name, err)
		}
	}
	if err := repo.SaveFact(&model.Fact{
		BookID:           book.ID,
		Subject:          "许舟",
		Predicate:        "初识",
		Object:           "沈知夏（因坐错教室而坐在一起）",
		Category:         "relationship",
		ValidFromChapter: 1,
		SourceChapter:    1,
	}); err != nil {
		t.Fatalf("save annotated fact: %v", err)
	}
	if err := repo.SaveBookState(&model.BookState{
		BookID:          book.ID,
		CurrentChapter:  1,
		CurrentLocation: "江城大学；白天：201教室；夜晚：宿舍",
		SourceChapter:   1,
	}); err != nil {
		t.Fatalf("save composite location: %v", err)
	}
	if err := repo.RefreshWikiGraph(book.ID); err != nil {
		t.Fatalf("refresh graph: %v", err)
	}

	xuzhou, err := repo.ResolveWikiEntity(book.ID, "许舟", model.WikiEntityCharacter)
	if err != nil || xuzhou == nil {
		t.Fatalf("resolve subject: %#v, %v", xuzhou, err)
	}
	relations, err := repo.ListWikiRelations(book.ID, 1, []uint{xuzhou.ID}, 20)
	if err != nil {
		t.Fatalf("list relations: %v", err)
	}
	if !hasWikiRelation(relations, "许舟", "初识", "沈知夏", "") {
		t.Fatalf("annotated object did not resolve to canonical character: %#v", relations)
	}
	for _, relation := range relations {
		if relation.ObjectName == "沈知夏" && relation.ObjectType != model.WikiEntityCharacter {
			t.Fatalf("annotated character resolved as %s", relation.ObjectType)
		}
	}
	places, total, err := repo.SearchWikiEntities(
		book.ID, "江城大学", []model.WikiEntityType{model.WikiEntityPlace}, 20, 0,
	)
	if err != nil {
		t.Fatalf("search place: %v", err)
	}
	if total != 1 || len(places) != 1 || places[0].CanonicalName != "江城大学" {
		t.Fatalf("composite current location was not normalized: %#v", places)
	}
}
