package sqlite

import (
	"path/filepath"
	"testing"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

func TestWikiEntityAliasResolutionAndRefresh(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "wiki-entity.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	book := createWikiTestBook(t, db)
	repo := NewTruthFileRepo(db)

	character := &model.Character{
		BookID:          book.ID,
		Name:            "林秋（阿秋/小秋）",
		RoleType:        model.CharacterProtagonist,
		Profile:         "谨慎的杂役弟子。化名：顾远",
		SourceChapter:   0,
		LastSeenChapter: 3,
	}
	if err := repo.SaveCharacter(character); err != nil {
		t.Fatalf("save character: %v", err)
	}
	if err := repo.SaveBookState(&model.BookState{
		BookID:          book.ID,
		CurrentChapter:  3,
		CurrentLocation: "七号门",
		SourceChapter:   3,
	}); err != nil {
		t.Fatalf("save book state: %v", err)
	}
	if err := repo.RefreshWikiEntities(book.ID); err != nil {
		t.Fatalf("refresh wiki entities: %v", err)
	}

	for _, alias := range []string{"林秋", "阿秋", "小秋", "顾远"} {
		entity, err := repo.ResolveWikiEntity(book.ID, alias, model.WikiEntityCharacter)
		if err != nil {
			t.Fatalf("resolve %s: %v", alias, err)
		}
		if entity == nil || entity.CanonicalName != "林秋" {
			t.Fatalf("resolve %s = %#v, want 林秋", alias, entity)
		}
	}

	mentions, err := repo.ResolveWikiEntityMentions(book.ID, "顾远准备返回七号门。", 8)
	if err != nil {
		t.Fatalf("resolve mentions: %v", err)
	}
	if !hasWikiEntity(mentions, model.WikiEntityCharacter, "林秋") ||
		!hasWikiEntity(mentions, model.WikiEntityPlace, "七号门") {
		t.Fatalf("unexpected mention result: %#v", mentions)
	}

	entity, err := repo.ResolveWikiEntity(book.ID, "林秋", model.WikiEntityCharacter)
	if err != nil || entity == nil {
		t.Fatalf("resolve canonical entity: %#v, %v", entity, err)
	}
	if err := repo.UpsertWikiEntity(entity, []string{"秋哥"}); err != nil {
		t.Fatalf("save manual alias: %v", err)
	}
	if err := repo.RefreshWikiEntities(book.ID); err != nil {
		t.Fatalf("refresh after manual alias: %v", err)
	}
	manualAlias, err := repo.ResolveWikiEntity(book.ID, "秋哥", model.WikiEntityCharacter)
	if err != nil {
		t.Fatalf("resolve manual alias: %v", err)
	}
	if manualAlias == nil || manualAlias.ID != entity.ID {
		t.Fatalf("manual alias was not preserved: %#v", manualAlias)
	}
}

func createWikiTestBook(t *testing.T, db *gorm.DB) *model.Book {
	t.Helper()
	genre := &model.Genre{Name: "Wiki测试题材"}
	if err := db.Create(genre).Error; err != nil {
		t.Fatalf("create genre: %v", err)
	}
	platform := &model.Platform{Name: "Wiki测试平台"}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	book := &model.Book{GenreID: genre.ID, PlatformID: platform.ID, Title: "Wiki测试书"}
	if err := db.Create(book).Error; err != nil {
		t.Fatalf("create book: %v", err)
	}
	return book
}

func hasWikiEntity(entities []model.WikiEntity, entityType model.WikiEntityType, name string) bool {
	for _, entity := range entities {
		if entity.EntityType == entityType && entity.CanonicalName == name {
			return true
		}
	}
	return false
}
