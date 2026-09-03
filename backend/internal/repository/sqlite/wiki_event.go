package sqlite

import (
	"fmt"
	"strconv"
	"strings"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const wikiRelationSourceEvent = "event"

func (r *truthFileRepo) ReplaceChapterWikiEvents(bookID uint, chapterNumber uint, drafts []model.WikiEventDraft) error {
	if bookID == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteChapterWikiEvents(tx, bookID, chapterNumber); err != nil {
			return err
		}

		artifactID, err := findEvidenceArtifactID(tx, bookID, chapterNumber)
		if err != nil {
			return err
		}
		for index, draft := range drafts {
			if err := saveWikiEventDraft(tx, bookID, chapterNumber, index, draft, artifactID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *truthFileRepo) ListWikiEvents(bookID uint, entityIDs []uint, limit int) ([]model.WikiEvent, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	query := r.db.Model(&model.WikiEvent{}).Where("wiki_events.book_id = ?", bookID)
	if len(entityIDs) > 0 {
		query = query.
			Joins("LEFT JOIN wiki_event_participants p ON p.event_id = wiki_events.id").
			Where("wiki_events.entity_id IN ? OR wiki_events.location_entity_id IN ? OR p.entity_id IN ?",
				entityIDs, entityIDs, entityIDs).
			Distinct("wiki_events.*")
	}
	var events []model.WikiEvent
	err := query.Order("wiki_events.chapter_number DESC, wiki_events.id DESC").Limit(limit).Find(&events).Error
	return events, err
}

func saveWikiEventDraft(
	db *gorm.DB,
	bookID uint,
	chapterNumber uint,
	index int,
	draft model.WikiEventDraft,
	artifactID *uint,
) error {
	eventKey := strings.TrimSpace(draft.EventKey)
	if eventKey == "" {
		eventKey = fmt.Sprintf("CH%04d-E%02d", chapterNumber, index+1)
	}
	title := strings.TrimSpace(draft.Title)
	if title == "" {
		title = fmt.Sprintf("第%d章事件%d", chapterNumber, index+1)
	}
	eventEntity, err := upsertWikiEntity(db, &model.WikiEntity{
		BookID:           bookID,
		EntityType:       model.WikiEntityEvent,
		CanonicalName:    fmt.Sprintf("第%d章·%s", chapterNumber, title),
		Summary:          strings.TrimSpace(draft.Summary),
		Status:           model.WikiEntityActive,
		FirstSeenChapter: chapterNumber,
		LastSeenChapter:  chapterNumber,
		Managed:          true,
	})
	if err != nil {
		return err
	}
	for _, alias := range []string{eventEntity.CanonicalName, title, eventKey} {
		if err := upsertWikiAlias(db, eventEntity, alias, alias == eventEntity.CanonicalName, true); err != nil {
			return err
		}
	}
	if err := saveWikiEntitySource(db, model.WikiEntitySource{
		BookID:        bookID,
		EntityID:      eventEntity.ID,
		SourceType:    wikiSourceEvent,
		SourceID:      eventKey,
		SourceChapter: chapterNumber,
	}); err != nil {
		return err
	}

	location, err := resolveOrCreateEventEntity(
		db, bookID, draft.Location, model.WikiEntityPlace, chapterNumber,
		wikiSourceEventLoc, eventKey,
	)
	if err != nil {
		return err
	}

	event := model.WikiEvent{
		BookID:        bookID,
		EntityID:      eventEntity.ID,
		EventKey:      eventKey,
		ChapterNumber: chapterNumber,
		Title:         title,
		EventType:     strings.TrimSpace(draft.EventType),
		Summary:       strings.TrimSpace(draft.Summary),
		Consequence:   strings.TrimSpace(draft.Consequence),
		EvidenceQuote: strings.TrimSpace(draft.EvidenceQuote),
		EvidenceStart: draft.EvidenceStart,
		EvidenceEnd:   draft.EvidenceEnd,
	}
	if location != nil {
		event.LocationEntityID = uintPointer(location.ID)
	}
	if err := db.Create(&event).Error; err != nil {
		return err
	}

	participantIDs := make(map[uint]struct{}, len(draft.Participants))
	for _, name := range draft.Participants {
		participant, err := resolveOrCreateEventEntity(
			db, bookID, name, model.WikiEntityCharacter, chapterNumber,
			wikiSourceEventPart, eventKey+":"+normalizeWikiEntityName(name),
		)
		if err != nil {
			return err
		}
		if participant == nil {
			continue
		}
		if _, exists := participantIDs[participant.ID]; exists {
			continue
		}
		participantIDs[participant.ID] = struct{}{}
		row := model.WikiEventParticipant{
			BookID:   bookID,
			EventID:  event.ID,
			EntityID: participant.ID,
			Role:     "participant",
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}

		relation, err := upsertWikiRelation(db, &model.WikiRelation{
			BookID:           bookID,
			SubjectEntityID:  participant.ID,
			Predicate:        "参与",
			ObjectEntityID:   uintPointer(eventEntity.ID),
			ValidFromChapter: chapterNumber,
			SourceChapter:    chapterNumber,
			SourceType:       wikiRelationSourceEvent,
			SourceID:         eventKey + ":participant:" + strconv.FormatUint(uint64(participant.ID), 10),
			Confidence:       1,
		})
		if err != nil {
			return err
		}
		if err := replaceWikiRelationEvidence(db, relation.ID, bookID, chapterNumber, artifactID,
			draft.EvidenceQuote, draft.EvidenceStart, draft.EvidenceEnd); err != nil {
			return err
		}
	}

	if location != nil {
		relation, err := upsertWikiRelation(db, &model.WikiRelation{
			BookID:           bookID,
			SubjectEntityID:  eventEntity.ID,
			Predicate:        "发生于",
			ObjectEntityID:   uintPointer(location.ID),
			ValidFromChapter: chapterNumber,
			SourceChapter:    chapterNumber,
			SourceType:       wikiRelationSourceEvent,
			SourceID:         eventKey + ":location",
			Confidence:       1,
		})
		if err != nil {
			return err
		}
		if err := replaceWikiRelationEvidence(db, relation.ID, bookID, chapterNumber, artifactID,
			draft.EvidenceQuote, draft.EvidenceStart, draft.EvidenceEnd); err != nil {
			return err
		}
	}
	return nil
}

func resolveOrCreateEventEntity(
	db *gorm.DB,
	bookID uint,
	rawName string,
	entityType model.WikiEntityType,
	chapterNumber uint,
	sourceType string,
	sourceID string,
) (*model.WikiEntity, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return nil, nil
	}
	entity, err := resolveWikiEntityInTx(db, bookID, name, entityType)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		canonical, aliases := parseWikiCanonicalAndAliases(name)
		entity, err = upsertWikiEntity(db, &model.WikiEntity{
			BookID:           bookID,
			EntityType:       entityType,
			CanonicalName:    canonical,
			Status:           model.WikiEntityActive,
			FirstSeenChapter: chapterNumber,
			LastSeenChapter:  chapterNumber,
			Managed:          true,
		})
		if err != nil {
			return nil, err
		}
		for _, alias := range append([]string{canonical}, aliases...) {
			if err := upsertWikiAlias(db, entity, alias, alias == canonical, true); err != nil {
				return nil, err
			}
		}
	}
	if chapterNumber > entity.LastSeenChapter {
		entity.LastSeenChapter = chapterNumber
		if err := db.Save(entity).Error; err != nil {
			return nil, err
		}
	}
	if err := saveWikiEntitySource(db, model.WikiEntitySource{
		BookID:        bookID,
		EntityID:      entity.ID,
		SourceType:    sourceType,
		SourceID:      sourceID,
		SourceChapter: chapterNumber,
	}); err != nil {
		return nil, err
	}
	return entity, nil
}

func saveWikiEntitySource(db *gorm.DB, source model.WikiEntitySource) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "entity_id"},
			{Name: "source_type"},
			{Name: "source_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"book_id", "source_chapter", "updated_at"}),
	}).Create(&source).Error
}

func deleteChapterWikiEvents(db *gorm.DB, bookID uint, chapterNumber uint) error {
	var events []model.WikiEvent
	if err := db.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).Find(&events).Error; err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	eventIDs := make([]uint, 0, len(events))
	candidateEntityIDs := make([]uint, 0, len(events)*3)
	eventKeys := make([]string, 0, len(events))
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
		candidateEntityIDs = append(candidateEntityIDs, event.EntityID)
		if event.LocationEntityID != nil {
			candidateEntityIDs = append(candidateEntityIDs, *event.LocationEntityID)
		}
		eventKeys = append(eventKeys, event.EventKey)
	}
	var participantEntityIDs []uint
	if err := db.Model(&model.WikiEventParticipant{}).
		Where("event_id IN ?", eventIDs).
		Pluck("entity_id", &participantEntityIDs).Error; err != nil {
		return err
	}
	candidateEntityIDs = append(candidateEntityIDs, participantEntityIDs...)

	var relationIDs []uint
	var eventRelations []model.WikiRelation
	if err := db.Where("book_id = ? AND source_type = ?", bookID, wikiRelationSourceEvent).
		Find(&eventRelations).Error; err != nil {
		return err
	}
	for _, relation := range eventRelations {
		for _, eventKey := range eventKeys {
			if strings.HasPrefix(relation.SourceID, eventKey+":") {
				relationIDs = append(relationIDs, relation.ID)
				break
			}
		}
	}
	if len(relationIDs) > 0 {
		if err := db.Where("relation_id IN ?", relationIDs).Delete(&model.WikiRelationEvidence{}).Error; err != nil {
			return err
		}
		if err := db.Where("id IN ?", relationIDs).Delete(&model.WikiRelation{}).Error; err != nil {
			return err
		}
	}
	if err := db.Where("event_id IN ?", eventIDs).Delete(&model.WikiEventParticipant{}).Error; err != nil {
		return err
	}
	if err := db.Where("id IN ?", eventIDs).Delete(&model.WikiEvent{}).Error; err != nil {
		return err
	}

	for _, eventKey := range eventKeys {
		if err := db.Where(
			"book_id = ? AND ((source_type = ? AND source_id = ?) OR (source_type IN ? AND source_id LIKE ?))",
			bookID, wikiSourceEvent, eventKey,
			[]string{wikiSourceEventPart, wikiSourceEventLoc}, eventKey+"%",
		).Delete(&model.WikiEntitySource{}).Error; err != nil {
			return err
		}
	}
	return deleteOrphanWikiEntities(db, bookID, candidateEntityIDs)
}

func deleteBookWikiEvents(db *gorm.DB, bookID uint) error {
	if err := db.Where("book_id = ?", bookID).Delete(&model.WikiEventParticipant{}).Error; err != nil {
		return err
	}
	return db.Where("book_id = ?", bookID).Delete(&model.WikiEvent{}).Error
}

func deleteOrphanWikiEntities(db *gorm.DB, bookID uint, candidates []uint) error {
	if len(candidates) == 0 {
		return nil
	}
	var orphanIDs []uint
	if err := db.Table("wiki_entities AS e").
		Select("e.id").
		Joins("LEFT JOIN wiki_entity_sources s ON s.entity_id = e.id").
		Where("e.book_id = ? AND e.managed = ? AND e.id IN ?", bookID, true, candidates).
		Group("e.id").
		Having("COUNT(s.id) = 0").
		Pluck("e.id", &orphanIDs).Error; err != nil {
		return err
	}
	if len(orphanIDs) == 0 {
		return nil
	}
	if err := db.Where("entity_id IN ?", orphanIDs).Delete(&model.WikiEntityAlias{}).Error; err != nil {
		return err
	}
	return db.Where("id IN ?", orphanIDs).Delete(&model.WikiEntity{}).Error
}
