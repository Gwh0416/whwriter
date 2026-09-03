package sqlite

import (
	"path/filepath"
	"strings"
	"testing"

	"whwriter/backend/internal/model"
)

func TestKnowledgeSearchRefreshesBM25Projection(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "knowledge.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	genre := &model.Genre{Name: "测试题材"}
	if err := db.Create(genre).Error; err != nil {
		t.Fatalf("create genre: %v", err)
	}
	platform := &model.Platform{Name: "测试平台"}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	book := &model.Book{
		GenreID:    genre.ID,
		PlatformID: platform.ID,
		Title:      "青铜门",
	}
	if err := db.Create(book).Error; err != nil {
		t.Fatalf("create book: %v", err)
	}

	repo := NewTruthFileRepo(db)
	if err := repo.SaveFoundation(&model.BookFoundation{
		BookID:   book.ID,
		FileType: model.FoundationStoryFrame,
		Content:  "林秋在七号门发现青铜戒指，戒指内藏有旧账线索。",
	}); err != nil {
		t.Fatalf("save foundation: %v", err)
	}
	if err := repo.SaveCharacter(&model.Character{
		BookID:        book.ID,
		Name:          "林秋",
		RoleType:      model.CharacterProtagonist,
		Profile:       "谨慎的杂役弟子，正在追查旧账。",
		SourceChapter: 0,
	}); err != nil {
		t.Fatalf("save character: %v", err)
	}
	if err := repo.RefreshKnowledgeIndex(book.ID); err != nil {
		t.Fatalf("refresh index: %v", err)
	}

	results, err := repo.SearchKnowledge(model.KnowledgeSearchQuery{
		BookID:        book.ID,
		Query:         "林秋要查青铜戒指里的旧账",
		ChapterNumber: 1,
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("search knowledge: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected BM25 search results")
	}
	if !containsKnowledgeText(results, "青铜戒指") {
		t.Fatalf("expected story-frame result, got %#v", results)
	}
	filtered, err := repo.SearchKnowledge(model.KnowledgeSearchQuery{
		BookID:        book.ID,
		Query:         "林秋 青铜戒指",
		ChapterNumber: 1,
		SourceTypes:   []model.KnowledgeSourceType{model.KnowledgeSourceFoundation},
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("search filtered knowledge: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatal("expected filtered foundation result")
	}
	for _, result := range filtered {
		if result.SourceType != model.KnowledgeSourceFoundation {
			t.Fatalf("unexpected supplemental source: %#v", result)
		}
	}

	foundation, err := repo.GetFoundation(book.ID, model.FoundationStoryFrame)
	if err != nil {
		t.Fatalf("get foundation: %v", err)
	}
	foundation.Content = "林秋已将青铜戒指交给师父，当前线索转向黑铁钥匙。"
	if err := repo.SaveFoundation(foundation); err != nil {
		t.Fatalf("update foundation: %v", err)
	}
	if err := repo.RefreshKnowledgeIndex(book.ID); err != nil {
		t.Fatalf("refresh updated index: %v", err)
	}

	results, err = repo.SearchKnowledge(model.KnowledgeSearchQuery{
		BookID:        book.ID,
		Query:         "黑铁钥匙",
		ChapterNumber: 1,
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("search updated knowledge: %v", err)
	}
	if !containsKnowledgeText(results, "黑铁钥匙") {
		t.Fatalf("expected updated result, got %#v", results)
	}
	oldResults, err := repo.SearchKnowledge(model.KnowledgeSearchQuery{
		BookID:        book.ID,
		Query:         "七号门暗室",
		ChapterNumber: 1,
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("search stale knowledge: %v", err)
	}
	if containsKnowledgeText(oldResults, "七号门发现青铜戒指") {
		t.Fatalf("stale foundation chunk remained in FTS: %#v", oldResults)
	}

	var ftsRows int64
	if err := db.Raw("SELECT COUNT(*) FROM knowledge_chunks_fts").Scan(&ftsRows).Error; err != nil {
		t.Fatalf("count FTS rows: %v", err)
	}
	var chunks int64
	if err := db.Model(&model.KnowledgeChunk{}).Count(&chunks).Error; err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if ftsRows != chunks {
		t.Fatalf("FTS rows = %d, chunks = %d", ftsRows, chunks)
	}
	if err := repo.DeleteBookCascade(book.ID); err != nil {
		t.Fatalf("delete book cascade: %v", err)
	}
	if err := db.Raw("SELECT COUNT(*) FROM knowledge_chunks_fts").Scan(&ftsRows).Error; err != nil {
		t.Fatalf("count FTS rows after deletion: %v", err)
	}
	if ftsRows != 0 {
		t.Fatalf("FTS rows after book deletion = %d, want 0", ftsRows)
	}
}

func containsKnowledgeText(results []model.KnowledgeSearchResult, target string) bool {
	for _, result := range results {
		if strings.Contains(result.Content, target) {
			return true
		}
	}
	return false
}
