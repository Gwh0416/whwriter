package sqlite

import (
	"path/filepath"
	"testing"

	"whwriter/backend/internal/model"
)

func TestBuildWikiGraphContextExpandsOnlyOneHopFromMentionedSeeds(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "wiki-context.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	book := createWikiTestBook(t, db)
	repo := NewTruthFileRepo(db)
	for _, character := range []*model.Character{
		{BookID: book.ID, Name: "林秋（阿秋）", RoleType: model.CharacterProtagonist, Profile: "追查旧账"},
		{BookID: book.ID, Name: "沈知夏", RoleType: model.CharacterMajor, Profile: "林秋的盟友"},
		{BookID: book.ID, Name: "王五", RoleType: model.CharacterMinor, Profile: "无关人物"},
	} {
		if err := repo.SaveCharacter(character); err != nil {
			t.Fatalf("save character %s: %v", character.Name, err)
		}
	}
	for _, fact := range []*model.Fact{
		{BookID: book.ID, Subject: "林秋", Predicate: "持有", Object: "青铜戒指", Category: "item", ValidFromChapter: 1, SourceChapter: 1},
		{BookID: book.ID, Subject: "林秋", Predicate: "盟友", Object: "沈知夏", Category: "relationship", ValidFromChapter: 2, SourceChapter: 2},
		{BookID: book.ID, Subject: "王五", Predicate: "持有", Object: "黑刀", Category: "item", ValidFromChapter: 1, SourceChapter: 1},
	} {
		if err := repo.SaveFact(fact); err != nil {
			t.Fatalf("save fact: %v", err)
		}
	}
	if err := repo.RefreshWikiGraph(book.ID); err != nil {
		t.Fatalf("refresh graph: %v", err)
	}

	graph, err := repo.BuildWikiGraphContext(model.WikiGraphQuery{
		BookID:        book.ID,
		Text:          "阿秋准备带着青铜戒指去找沈知夏。",
		ChapterNumber: 3,
		SeedLimit:     1,
		RelationLimit: 20,
		EventLimit:    10,
	})
	if err != nil {
		t.Fatalf("build graph context: %v", err)
	}
	if len(graph.Seeds) != 1 || graph.Seeds[0].CanonicalName != "林秋" {
		t.Fatalf("unexpected seeds: %#v", graph.Seeds)
	}
	if !hasWikiEntity(graph.Entities, model.WikiEntityItem, "青铜戒指") ||
		!hasWikiEntity(graph.Entities, model.WikiEntityCharacter, "沈知夏") {
		t.Fatalf("one-hop neighbors missing: %#v", graph.Entities)
	}
	if hasWikiEntity(graph.Entities, model.WikiEntityCharacter, "王五") ||
		hasWikiEntity(graph.Entities, model.WikiEntityItem, "黑刀") {
		t.Fatalf("unrelated branch leaked into one-hop graph: %#v", graph.Entities)
	}
	if len(graph.Relations) != 2 {
		t.Fatalf("relations = %d, want 2: %#v", len(graph.Relations), graph.Relations)
	}
}
