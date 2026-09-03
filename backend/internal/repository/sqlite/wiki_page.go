package sqlite

import (
	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

func (r *truthFileRepo) SearchWikiEntities(
	bookID uint,
	search string,
	entityTypes []model.WikiEntityType,
	limit int,
	offset int,
) ([]model.WikiEntity, int64, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.Table("wiki_entities AS e").Where("e.book_id = ?", bookID)
	if len(entityTypes) > 0 {
		base = base.Where("e.entity_type IN ?", entityTypes)
	}
	normalized := normalizeWikiEntityName(search)
	if normalized != "" {
		pattern := "%" + normalized + "%"
		base = base.Where(`e.normalized_name LIKE ? OR EXISTS (
			SELECT 1 FROM wiki_entity_aliases a
			WHERE a.entity_id = e.id AND a.normalized_alias LIKE ?
		)`, pattern, pattern)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Distinct("e.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var entities []model.WikiEntity
	if err := base.Select("e.*").
		Order("CASE WHEN e.status = 'active' THEN 0 ELSE 1 END, e.entity_type, e.canonical_name, e.id").
		Limit(limit).
		Offset(offset).
		Find(&entities).Error; err != nil {
		return nil, 0, err
	}
	return entities, total, nil
}

func (r *truthFileRepo) GetWikiEntityPage(bookID uint, entityID uint) (*model.WikiEntityPage, error) {
	var entity model.WikiEntity
	if err := r.db.Where("book_id = ? AND id = ?", bookID, entityID).First(&entity).Error; err != nil {
		return nil, err
	}
	aliases, err := r.GetWikiEntityAliases(entityID)
	if err != nil {
		return nil, err
	}
	relations, err := listWikiRelationHistory(r.db, bookID, entityID, 500)
	if err != nil {
		return nil, err
	}
	relationIDs := make([]uint, 0, len(relations))
	for _, relation := range relations {
		relationIDs = append(relationIDs, relation.ID)
	}
	evidence, err := r.GetWikiRelationEvidence(relationIDs)
	if err != nil {
		return nil, err
	}
	events, err := listWikiEventViews(r.db, bookID, entityID, 200)
	if err != nil {
		return nil, err
	}
	return &model.WikiEntityPage{
		Entity:           entity,
		Aliases:          aliases,
		Relations:        relations,
		RelationEvidence: evidence,
		Events:           events,
	}, nil
}

func listWikiRelationHistory(db *gorm.DB, bookID uint, entityID uint, limit int) ([]model.WikiRelationView, error) {
	var relations []model.WikiRelationView
	err := db.Table("wiki_relations AS r").
		Select(`r.*,
			subject.canonical_name AS subject_name,
			subject.entity_type AS subject_type,
			COALESCE(object.canonical_name, '') AS object_name,
			COALESCE(object.entity_type, '') AS object_type`).
		Joins("JOIN wiki_entities AS subject ON subject.id = r.subject_entity_id").
		Joins("LEFT JOIN wiki_entities AS object ON object.id = r.object_entity_id").
		Where("r.book_id = ? AND (r.subject_entity_id = ? OR r.object_entity_id = ?)", bookID, entityID, entityID).
		Order("r.valid_from_chapter DESC, r.id DESC").
		Limit(limit).
		Scan(&relations).Error
	return relations, err
}

func listWikiEventViews(db *gorm.DB, bookID uint, entityID uint, limit int) ([]model.WikiEventView, error) {
	var events []model.WikiEvent
	if err := db.Model(&model.WikiEvent{}).
		Joins("LEFT JOIN wiki_event_participants p ON p.event_id = wiki_events.id").
		Where(`wiki_events.book_id = ? AND (
			wiki_events.entity_id = ? OR wiki_events.location_entity_id = ? OR p.entity_id = ?
		)`, bookID, entityID, entityID, entityID).
		Distinct("wiki_events.*").
		Order("wiki_events.chapter_number DESC, wiki_events.id DESC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, err
	}

	views := make([]model.WikiEventView, 0, len(events))
	for _, event := range events {
		view := model.WikiEventView{WikiEvent: event}
		if event.LocationEntityID != nil {
			var location model.WikiEntity
			if err := db.First(&location, *event.LocationEntityID).Error; err == nil {
				view.LocationName = location.CanonicalName
			}
		}
		if err := db.Table("wiki_entities AS e").
			Select("e.*").
			Joins("JOIN wiki_event_participants p ON p.entity_id = e.id").
			Where("p.event_id = ?", event.ID).
			Order("e.canonical_name, e.id").
			Find(&view.Participants).Error; err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}
