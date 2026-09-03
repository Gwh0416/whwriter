package sqlite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const wikiRelationSourceFact = "fact"

func rebuildWikiRelations(db *gorm.DB) error {
	var bookIDs []uint
	if err := db.Model(&model.Book{}).Order("id").Pluck("id", &bookIDs).Error; err != nil {
		return err
	}
	repo := &truthFileRepo{db: db}
	for _, bookID := range bookIDs {
		if err := repo.RefreshWikiRelations(bookID); err != nil {
			return fmt.Errorf("refresh book %d wiki relations: %w", bookID, err)
		}
	}
	return nil
}

func (r *truthFileRepo) RefreshWikiRelations(bookID uint) error {
	return r.RefreshWikiGraph(bookID)
}

func (r *truthFileRepo) RefreshWikiGraph(bookID uint) error {
	if bookID == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		specs, err := collectWikiEntitySpecs(tx, bookID)
		if err != nil {
			return err
		}
		if err := syncWikiEntitySpecs(tx, bookID, specs); err != nil {
			return err
		}
		return syncWikiFactRelations(tx, bookID)
	})
}

func (r *truthFileRepo) ListWikiRelations(bookID uint, chapterNumber uint, entityIDs []uint, limit int) ([]model.WikiRelationView, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	query := r.db.Table("wiki_relations AS r").
		Select(`r.*,
			subject.canonical_name AS subject_name,
			subject.entity_type AS subject_type,
			COALESCE(object.canonical_name, '') AS object_name,
			COALESCE(object.entity_type, '') AS object_type`).
		Joins("JOIN wiki_entities AS subject ON subject.id = r.subject_entity_id").
		Joins("LEFT JOIN wiki_entities AS object ON object.id = r.object_entity_id").
		Where("r.book_id = ?", bookID)
	if chapterNumber > 0 {
		query = query.Where(
			"r.valid_from_chapter <= ? AND (r.valid_until_chapter IS NULL OR r.valid_until_chapter >= ?)",
			chapterNumber, chapterNumber,
		)
	} else {
		query = query.Where("r.valid_until_chapter IS NULL")
	}
	if len(entityIDs) > 0 {
		query = query.Where("r.subject_entity_id IN ? OR r.object_entity_id IN ?", entityIDs, entityIDs)
	}

	var relations []model.WikiRelationView
	err := query.Order("r.valid_from_chapter DESC, r.id DESC").Limit(limit).Scan(&relations).Error
	return relations, err
}

func syncWikiFactRelations(db *gorm.DB, bookID uint) error {
	var facts []model.Fact
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&facts).Error; err != nil {
		return err
	}

	sourceIDs := make([]string, 0, len(facts))
	for i := range facts {
		fact := &facts[i]
		subject, err := resolveWikiEntityInTx(db, bookID, fact.Subject, "")
		if err != nil {
			return err
		}
		if subject == nil {
			return fmt.Errorf("fact %d subject entity not found: %s", fact.ID, fact.Subject)
		}

		var object *model.WikiEntity
		if looksLikeWikiEntityName(fact.Object) {
			object, err = resolveWikiEntityInTx(db, bookID, fact.Object, "")
			if err != nil {
				return err
			}
		}

		fact.SubjectEntityID = uintPointer(subject.ID)
		fact.ObjectEntityID = nil
		objectLiteral := strings.TrimSpace(fact.Object)
		if object != nil {
			fact.ObjectEntityID = uintPointer(object.ID)
			objectLiteral = ""
		}
		if err := db.Model(&model.Fact{}).Where("id = ?", fact.ID).Updates(map[string]any{
			"subject_entity_id": fact.SubjectEntityID,
			"object_entity_id":  fact.ObjectEntityID,
		}).Error; err != nil {
			return err
		}

		sourceID := strconv.FormatUint(uint64(fact.ID), 10)
		sourceIDs = append(sourceIDs, sourceID)
		relation := model.WikiRelation{
			BookID:            bookID,
			SubjectEntityID:   subject.ID,
			Predicate:         strings.TrimSpace(fact.Predicate),
			ObjectEntityID:    fact.ObjectEntityID,
			ObjectLiteral:     objectLiteral,
			ValidFromChapter:  fact.ValidFromChapter,
			ValidUntilChapter: fact.ValidUntilChapter,
			SourceChapter:     fact.SourceChapter,
			SourceType:        wikiRelationSourceFact,
			SourceID:          sourceID,
			Confidence:        1,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "book_id"},
				{Name: "source_type"},
				{Name: "source_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"subject_entity_id", "predicate", "object_entity_id", "object_literal",
				"qualifier_json", "valid_from_chapter", "valid_until_chapter",
				"source_chapter", "confidence", "updated_at",
			}),
		}).Create(&relation).Error; err != nil {
			return err
		}
	}

	stale := db.Where("book_id = ? AND source_type = ?", bookID, wikiRelationSourceFact)
	if len(sourceIDs) > 0 {
		stale = stale.Where("source_id NOT IN ?", sourceIDs)
	}
	return stale.Delete(&model.WikiRelation{}).Error
}

func resolveWikiEntityInTx(db *gorm.DB, bookID uint, name string, entityType model.WikiEntityType) (*model.WikiEntity, error) {
	normalized := normalizeWikiEntityName(name)
	if normalized == "" {
		return nil, nil
	}
	query := db.Model(&model.WikiEntity{}).
		Joins("JOIN wiki_entity_aliases a ON a.entity_id = wiki_entities.id").
		Where("wiki_entities.book_id = ? AND a.normalized_alias = ?", bookID, normalized)
	if entityType != "" {
		query = query.Where("wiki_entities.entity_type = ?", entityType)
	}
	var entity model.WikiEntity
	err := query.Order("a.is_canonical DESC, wiki_entities.status ASC, wiki_entities.id").First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func deleteBookWikiRelations(db *gorm.DB, bookID uint) error {
	return db.Where("book_id = ?", bookID).Delete(&model.WikiRelation{}).Error
}

func uintPointer(value uint) *uint {
	return &value
}
