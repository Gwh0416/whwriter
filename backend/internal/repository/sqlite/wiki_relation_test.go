package sqlite

import (
	"path/filepath"
	"testing"

	"whwriter/backend/internal/model"
)

func TestWikiRelationsMirrorFactEntitiesAndValidity(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "wiki-relation.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	book := createWikiTestBook(t, db)
	repo := NewTruthFileRepo(db)
	if err := repo.SaveCharacter(&model.Character{
		BookID:          book.ID,
		Name:            "林秋",
		RoleType:        model.CharacterProtagonist,
		Profile:         "追查旧账的杂役弟子",
		LastSeenChapter: 3,
	}); err != nil {
		t.Fatalf("save character: %v", err)
	}

	first := &model.Fact{
		BookID:           book.ID,
		Subject:          "林秋",
		Predicate:        "持有",
		Object:           "青铜戒指",
		Category:         "item",
		ValidFromChapter: 1,
		SourceChapter:    1,
	}
	if err := repo.SaveFact(first); err != nil {
		t.Fatalf("save first fact: %v", err)
	}
	second := &model.Fact{
		BookID:           book.ID,
		Subject:          "林秋",
		Predicate:        "持有",
		Object:           "黑铁钥匙",
		Category:         "item",
		ValidFromChapter: 3,
		SourceChapter:    3,
	}
	if err := repo.SaveFact(second); err != nil {
		t.Fatalf("save replacement fact: %v", err)
	}
	resource := &model.Fact{
		BookID:           book.ID,
		Subject:          "林秋",
		Predicate:        "灵石",
		Object:           "120",
		Category:         "resource",
		ValidFromChapter: 3,
		SourceChapter:    3,
	}
	if err := repo.SaveFact(resource); err != nil {
		t.Fatalf("save literal fact: %v", err)
	}

	if err := repo.RefreshWikiRelations(book.ID); err != nil {
		t.Fatalf("refresh wiki relations: %v", err)
	}

	var facts []model.Fact
	if err := db.Where("book_id = ?", book.ID).Order("id").Find(&facts).Error; err != nil {
		t.Fatalf("load facts: %v", err)
	}
	if len(facts) != 3 {
		t.Fatalf("facts = %d, want 3", len(facts))
	}
	for _, fact := range facts {
		if fact.SubjectEntityID == nil {
			t.Fatalf("fact %d missing subject entity: %#v", fact.ID, fact)
		}
	}
	if facts[0].ObjectEntityID == nil || facts[1].ObjectEntityID == nil {
		t.Fatalf("item facts should reference object entities: %#v", facts)
	}
	if facts[2].ObjectEntityID != nil {
		t.Fatalf("numeric resource must remain a literal: %#v", facts[2])
	}

	atChapterTwo, err := repo.ListWikiRelations(book.ID, 2, nil, 20)
	if err != nil {
		t.Fatalf("list chapter 2 relations: %v", err)
	}
	if !hasWikiRelation(atChapterTwo, "林秋", "持有", "青铜戒指", "") ||
		hasWikiRelation(atChapterTwo, "林秋", "持有", "黑铁钥匙", "") {
		t.Fatalf("unexpected chapter 2 relations: %#v", atChapterTwo)
	}

	atChapterThree, err := repo.ListWikiRelations(book.ID, 3, nil, 20)
	if err != nil {
		t.Fatalf("list chapter 3 relations: %v", err)
	}
	if !hasWikiRelation(atChapterThree, "林秋", "持有", "黑铁钥匙", "") ||
		!hasWikiRelation(atChapterThree, "林秋", "灵石", "", "120") ||
		hasWikiRelation(atChapterThree, "林秋", "持有", "青铜戒指", "") {
		t.Fatalf("unexpected chapter 3 relations: %#v", atChapterThree)
	}
}

func hasWikiRelation(relations []model.WikiRelationView, subject, predicate, object, literal string) bool {
	for _, relation := range relations {
		if relation.SubjectName == subject &&
			relation.Predicate == predicate &&
			relation.ObjectName == object &&
			relation.ObjectLiteral == literal {
			return true
		}
	}
	return false
}
