package sqlite

import (
	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

func (r *truthFileRepo) BuildWikiGraphContext(query model.WikiGraphQuery) (*model.WikiGraphContext, error) {
	if query.BookID == 0 {
		return &model.WikiGraphContext{}, nil
	}
	if query.SeedLimit <= 0 {
		query.SeedLimit = 8
	}
	if query.RelationLimit <= 0 {
		query.RelationLimit = 32
	}
	if query.EventLimit <= 0 {
		query.EventLimit = 12
	}

	seedIDs := make([]uint, 0, query.SeedLimit)
	seenSeedIDs := make(map[uint]struct{}, query.SeedLimit)
	for _, id := range query.SeedEntityIDs {
		if id == 0 {
			continue
		}
		if _, ok := seenSeedIDs[id]; ok {
			continue
		}
		seenSeedIDs[id] = struct{}{}
		seedIDs = append(seedIDs, id)
		if len(seedIDs) == query.SeedLimit {
			break
		}
	}
	if len(seedIDs) < query.SeedLimit {
		mentions, err := r.ResolveWikiEntityMentions(query.BookID, query.Text, query.SeedLimit-len(seedIDs))
		if err != nil {
			return nil, err
		}
		for _, entity := range mentions {
			if _, ok := seenSeedIDs[entity.ID]; ok {
				continue
			}
			seenSeedIDs[entity.ID] = struct{}{}
			seedIDs = append(seedIDs, entity.ID)
		}
	}

	seeds, err := loadWikiEntitiesByIDs(r.db, query.BookID, seedIDs)
	if err != nil {
		return nil, err
	}
	if len(seedIDs) == 0 {
		return &model.WikiGraphContext{Seeds: seeds, Entities: seeds}, nil
	}

	relations, err := r.ListWikiRelations(query.BookID, query.ChapterNumber, seedIDs, query.RelationLimit)
	if err != nil {
		return nil, err
	}
	entityIDs := append([]uint(nil), seedIDs...)
	seenEntityIDs := make(map[uint]struct{}, len(seedIDs)+len(relations))
	for _, id := range seedIDs {
		seenEntityIDs[id] = struct{}{}
	}
	for _, relation := range relations {
		for _, id := range []uint{relation.SubjectEntityID, valueOrZero(relation.ObjectEntityID)} {
			if id == 0 {
				continue
			}
			if _, ok := seenEntityIDs[id]; ok {
				continue
			}
			seenEntityIDs[id] = struct{}{}
			entityIDs = append(entityIDs, id)
		}
	}

	entities, err := loadWikiEntitiesByIDs(r.db, query.BookID, entityIDs)
	if err != nil {
		return nil, err
	}
	events, err := r.ListWikiEvents(query.BookID, entityIDs, query.EventLimit)
	if err != nil {
		return nil, err
	}
	return &model.WikiGraphContext{
		Seeds:     seeds,
		Entities:  entities,
		Relations: relations,
		Events:    events,
	}, nil
}

func loadWikiEntitiesByIDs(db *gorm.DB, bookID uint, ids []uint) ([]model.WikiEntity, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var entities []model.WikiEntity
	if err := db.Where("book_id = ? AND id IN ?", bookID, ids).Find(&entities).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint]model.WikiEntity, len(entities))
	for _, entity := range entities {
		byID[entity.ID] = entity
	}
	ordered := make([]model.WikiEntity, 0, len(ids))
	for _, id := range ids {
		if entity, ok := byID[id]; ok {
			ordered = append(ordered, entity)
		}
	}
	return ordered, nil
}

func valueOrZero(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}
