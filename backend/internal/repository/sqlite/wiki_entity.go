package sqlite

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"whwriter/backend/internal/model"

	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	wikiSourceCharacter = "character"
	wikiSourceHook      = "hook"
	wikiSourceBookState = "book_state"
	wikiSourceFact      = "fact"
	wikiSourceEvent     = "event"
	wikiSourceEventPart = "event_participant"
	wikiSourceEventLoc  = "event_location"
)

type wikiEntitySpec struct {
	EntityType    model.WikiEntityType
	Canonical     string
	Aliases       []string
	Summary       string
	Status        model.WikiEntityStatus
	FirstChapter  uint
	LastChapter   uint
	SourceType    string
	SourceID      string
	SourceChapter uint
}

type wikiAliasHit struct {
	EntityID        uint
	Alias           string
	NormalizedAlias string
	IsCanonical     bool
	Position        int
}

func rebuildWikiEntities(db *gorm.DB) error {
	var bookIDs []uint
	if err := db.Model(&model.Book{}).Order("id").Pluck("id", &bookIDs).Error; err != nil {
		return err
	}
	repo := &truthFileRepo{db: db}
	for _, bookID := range bookIDs {
		if err := repo.RefreshWikiEntities(bookID); err != nil {
			return fmt.Errorf("refresh book %d wiki entities: %w", bookID, err)
		}
	}
	return nil
}

func (r *truthFileRepo) RefreshWikiEntities(bookID uint) error {
	if bookID == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		specs, err := collectWikiEntitySpecs(tx, bookID)
		if err != nil {
			return err
		}
		return syncWikiEntitySpecs(tx, bookID, specs)
	})
}

func (r *truthFileRepo) UpsertWikiEntity(entity *model.WikiEntity, aliases []string) error {
	if entity == nil || entity.BookID == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		entity.Managed = false
		saved, err := upsertWikiEntity(tx, entity)
		if err != nil {
			return err
		}
		for _, alias := range append([]string{saved.CanonicalName}, aliases...) {
			if err := upsertWikiAlias(tx, saved, alias, alias == saved.CanonicalName, false); err != nil {
				return err
			}
		}
		*entity = *saved
		return nil
	})
}

func (r *truthFileRepo) ResolveWikiEntity(bookID uint, name string, entityType model.WikiEntityType) (*model.WikiEntity, error) {
	normalized := normalizeWikiEntityName(name)
	if bookID == 0 || normalized == "" {
		return nil, nil
	}

	query := r.db.Model(&model.WikiEntity{}).
		Joins("JOIN wiki_entity_aliases a ON a.entity_id = wiki_entities.id").
		Where("wiki_entities.book_id = ? AND wiki_entities.status = ? AND a.normalized_alias = ?",
			bookID, model.WikiEntityActive, normalized)
	if entityType != "" {
		query = query.Where("wiki_entities.entity_type = ?", entityType)
	}

	var entity model.WikiEntity
	err := query.Order("a.is_canonical DESC, wiki_entities.id").First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *truthFileRepo) ResolveWikiEntityMentions(bookID uint, text string, limit int) ([]model.WikiEntity, error) {
	normalizedText := normalizeWikiEntityName(text)
	if bookID == 0 || normalizedText == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 8
	}
	if limit > 32 {
		limit = 32
	}

	var aliases []model.WikiEntityAlias
	if err := r.db.Table("wiki_entity_aliases AS a").
		Select("a.*").
		Joins("JOIN wiki_entities e ON e.id = a.entity_id").
		Where("a.book_id = ? AND e.status = ?", bookID, model.WikiEntityActive).
		Find(&aliases).Error; err != nil {
		return nil, err
	}

	hits := make([]wikiAliasHit, 0, len(aliases))
	for _, alias := range aliases {
		if alias.NormalizedAlias == "" {
			continue
		}
		position := strings.Index(normalizedText, alias.NormalizedAlias)
		if position < 0 {
			continue
		}
		if len([]rune(alias.NormalizedAlias)) < 2 && normalizedText != alias.NormalizedAlias {
			continue
		}
		hits = append(hits, wikiAliasHit{
			EntityID:        alias.EntityID,
			Alias:           alias.Alias,
			NormalizedAlias: alias.NormalizedAlias,
			IsCanonical:     alias.IsCanonical,
			Position:        position,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Position != hits[j].Position {
			return hits[i].Position < hits[j].Position
		}
		if len([]rune(hits[i].NormalizedAlias)) != len([]rune(hits[j].NormalizedAlias)) {
			return len([]rune(hits[i].NormalizedAlias)) > len([]rune(hits[j].NormalizedAlias))
		}
		return hits[i].IsCanonical && !hits[j].IsCanonical
	})

	ids := make([]uint, 0, limit)
	seen := make(map[uint]struct{}, limit)
	for _, hit := range hits {
		if _, ok := seen[hit.EntityID]; ok {
			continue
		}
		seen[hit.EntityID] = struct{}{}
		ids = append(ids, hit.EntityID)
		if len(ids) == limit {
			break
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	var entities []model.WikiEntity
	if err := r.db.Where("id IN ?", ids).Find(&entities).Error; err != nil {
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

func (r *truthFileRepo) ListWikiEntities(bookID uint, entityTypes []model.WikiEntityType, limit int) ([]model.WikiEntity, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	query := r.db.Where("book_id = ? AND status = ?", bookID, model.WikiEntityActive)
	if len(entityTypes) > 0 {
		query = query.Where("entity_type IN ?", entityTypes)
	}
	var entities []model.WikiEntity
	err := query.Order("entity_type, canonical_name, id").Limit(limit).Find(&entities).Error
	return entities, err
}

func (r *truthFileRepo) GetWikiEntityAliases(entityID uint) ([]model.WikiEntityAlias, error) {
	var aliases []model.WikiEntityAlias
	err := r.db.Where("entity_id = ?", entityID).
		Order("is_canonical DESC, alias, id").
		Find(&aliases).Error
	return aliases, err
}

func collectWikiEntitySpecs(db *gorm.DB, bookID uint) ([]wikiEntitySpec, error) {
	specs := make([]wikiEntitySpec, 0, 32)
	knownTypes := make(map[string]model.WikiEntityType)
	add := func(spec wikiEntitySpec) {
		spec.Canonical, spec.Aliases = parseWikiCanonicalAndAliases(spec.Canonical, spec.Aliases...)
		if spec.Canonical == "" {
			return
		}
		if spec.Status == "" {
			spec.Status = model.WikiEntityActive
		}
		specs = append(specs, spec)
		knownTypes[normalizeWikiEntityName(spec.Canonical)] = spec.EntityType
		for _, alias := range spec.Aliases {
			knownTypes[normalizeWikiEntityName(alias)] = spec.EntityType
		}
	}

	var characters []model.Character
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&characters).Error; err != nil {
		return nil, err
	}
	for _, character := range characters {
		aliases := extractWikiAliasesFromText(character.Profile + "\n" + character.Backstory)
		add(wikiEntitySpec{
			EntityType:    model.WikiEntityCharacter,
			Canonical:     character.Name,
			Aliases:       aliases,
			Summary:       firstNonEmptyWiki(character.Profile, character.CurrentStatus, character.CoreTags),
			FirstChapter:  character.SourceChapter,
			LastChapter:   character.LastSeenChapter,
			SourceType:    wikiSourceCharacter,
			SourceID:      strconv.FormatUint(uint64(character.ID), 10),
			SourceChapter: character.SourceChapter,
		})
	}

	var hooks []model.Hook
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&hooks).Error; err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		status := model.WikiEntityActive
		if hook.Status == model.HookResolved || hook.Status == model.HookStale {
			status = model.WikiEntityInactive
		}
		add(wikiEntitySpec{
			EntityType:    model.WikiEntityHook,
			Canonical:     hook.HookID,
			Summary:       firstNonEmptyWiki(hook.ExpectedPayoff, hook.Notes),
			Status:        status,
			FirstChapter:  hook.StartChapter,
			LastChapter:   hook.LastAdvancedChapter,
			SourceType:    wikiSourceHook,
			SourceID:      strconv.FormatUint(uint64(hook.ID), 10),
			SourceChapter: hook.StartChapter,
		})
	}

	var state model.BookState
	if err := db.Where("book_id = ?", bookID).First(&state).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if strings.TrimSpace(state.CurrentLocation) != "" {
		add(wikiEntitySpec{
			EntityType:    model.WikiEntityPlace,
			Canonical:     state.CurrentLocation,
			Summary:       "当前故事发生地点",
			FirstChapter:  state.SourceChapter,
			LastChapter:   state.CurrentChapter,
			SourceType:    wikiSourceBookState,
			SourceID:      "current_location",
			SourceChapter: state.SourceChapter,
		})
	}

	var events []model.WikiEvent
	if err := db.Where("book_id = ?", bookID).Order("chapter_number, id").Find(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		var eventEntity model.WikiEntity
		if err := db.First(&eventEntity, event.EntityID).Error; err != nil {
			return nil, err
		}
		add(wikiEntitySpec{
			EntityType:    model.WikiEntityEvent,
			Canonical:     eventEntity.CanonicalName,
			Aliases:       []string{event.Title, event.EventKey},
			Summary:       event.Summary,
			FirstChapter:  event.ChapterNumber,
			LastChapter:   event.ChapterNumber,
			SourceType:    wikiSourceEvent,
			SourceID:      event.EventKey,
			SourceChapter: event.ChapterNumber,
		})

		var participants []model.WikiEventParticipant
		if err := db.Where("event_id = ?", event.ID).Find(&participants).Error; err != nil {
			return nil, err
		}
		for _, participant := range participants {
			var entity model.WikiEntity
			if err := db.First(&entity, participant.EntityID).Error; err != nil {
				return nil, err
			}
			add(wikiEntitySpec{
				EntityType:    entity.EntityType,
				Canonical:     entity.CanonicalName,
				Summary:       entity.Summary,
				FirstChapter:  entity.FirstSeenChapter,
				LastChapter:   maxWikiChapter(entity.LastSeenChapter, event.ChapterNumber),
				SourceType:    wikiSourceEventPart,
				SourceID:      event.EventKey + ":" + strconv.FormatUint(uint64(entity.ID), 10),
				SourceChapter: event.ChapterNumber,
			})
		}
		if event.LocationEntityID != nil {
			var location model.WikiEntity
			if err := db.First(&location, *event.LocationEntityID).Error; err != nil {
				return nil, err
			}
			add(wikiEntitySpec{
				EntityType:    model.WikiEntityPlace,
				Canonical:     location.CanonicalName,
				Summary:       location.Summary,
				FirstChapter:  location.FirstSeenChapter,
				LastChapter:   maxWikiChapter(location.LastSeenChapter, event.ChapterNumber),
				SourceType:    wikiSourceEventLoc,
				SourceID:      event.EventKey,
				SourceChapter: event.ChapterNumber,
			})
		}
	}

	var facts []model.Fact
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&facts).Error; err != nil {
		return nil, err
	}
	for _, fact := range facts {
		if strings.TrimSpace(fact.Subject) != "" {
			add(wikiEntitySpec{
				EntityType:    inferWikiEntityType(fact.Subject, fact.Predicate, fact.Category, true, knownTypes),
				Canonical:     fact.Subject,
				Summary:       fmt.Sprintf("%s %s", fact.Predicate, fact.Object),
				FirstChapter:  fact.ValidFromChapter,
				LastChapter:   fact.SourceChapter,
				SourceType:    wikiSourceFact,
				SourceID:      strconv.FormatUint(uint64(fact.ID), 10) + ":subject",
				SourceChapter: fact.SourceChapter,
			})
		}
		if looksLikeWikiEntityName(fact.Object) {
			add(wikiEntitySpec{
				EntityType:    inferWikiEntityType(fact.Object, fact.Predicate, fact.Category, false, knownTypes),
				Canonical:     fact.Object,
				Summary:       fmt.Sprintf("%s %s", fact.Subject, fact.Predicate),
				FirstChapter:  fact.ValidFromChapter,
				LastChapter:   fact.SourceChapter,
				SourceType:    wikiSourceFact,
				SourceID:      strconv.FormatUint(uint64(fact.ID), 10) + ":object",
				SourceChapter: fact.SourceChapter,
			})
		}
	}

	return specs, nil
}

func syncWikiEntitySpecs(db *gorm.DB, bookID uint, specs []wikiEntitySpec) error {
	if err := db.Where("book_id = ? AND is_derived = ?", bookID, true).Delete(&model.WikiEntityAlias{}).Error; err != nil {
		return err
	}
	if err := db.Where("book_id = ? AND source_type IN ?", bookID, []string{
		wikiSourceCharacter, wikiSourceHook, wikiSourceBookState, wikiSourceFact,
		wikiSourceEvent, wikiSourceEventPart, wikiSourceEventLoc,
	}).Delete(&model.WikiEntitySource{}).Error; err != nil {
		return err
	}

	for _, spec := range specs {
		entity := &model.WikiEntity{
			BookID:           bookID,
			EntityType:       spec.EntityType,
			CanonicalName:    spec.Canonical,
			Summary:          strings.TrimSpace(spec.Summary),
			Status:           spec.Status,
			FirstSeenChapter: spec.FirstChapter,
			LastSeenChapter:  spec.LastChapter,
			Managed:          true,
		}
		saved, err := upsertWikiEntity(db, entity)
		if err != nil {
			return err
		}
		for _, alias := range append([]string{saved.CanonicalName}, spec.Aliases...) {
			if err := upsertWikiAlias(db, saved, alias, normalizeWikiEntityName(alias) == saved.NormalizedName, true); err != nil {
				return err
			}
		}
		source := model.WikiEntitySource{
			BookID:        bookID,
			EntityID:      saved.ID,
			SourceType:    spec.SourceType,
			SourceID:      spec.SourceID,
			SourceChapter: spec.SourceChapter,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "entity_id"},
				{Name: "source_type"},
				{Name: "source_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"book_id", "source_chapter", "updated_at"}),
		}).Create(&source).Error; err != nil {
			return err
		}
	}

	var orphanIDs []uint
	if err := db.Table("wiki_entities AS e").
		Select("e.id").
		Joins("LEFT JOIN wiki_entity_sources s ON s.entity_id = e.id").
		Where("e.book_id = ? AND e.managed = ?", bookID, true).
		Group("e.id").
		Having("COUNT(s.id) = 0").
		Pluck("e.id", &orphanIDs).Error; err != nil {
		return err
	}
	if len(orphanIDs) > 0 {
		if err := db.Where("entity_id IN ?", orphanIDs).Delete(&model.WikiEntityAlias{}).Error; err != nil {
			return err
		}
		if err := db.Where("id IN ?", orphanIDs).Delete(&model.WikiEntity{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteBookWikiEntities(db *gorm.DB, bookID uint) error {
	if err := db.Where("book_id = ?", bookID).Delete(&model.WikiEntitySource{}).Error; err != nil {
		return err
	}
	if err := db.Where("book_id = ?", bookID).Delete(&model.WikiEntityAlias{}).Error; err != nil {
		return err
	}
	return db.Where("book_id = ?", bookID).Delete(&model.WikiEntity{}).Error
}

func upsertWikiEntity(db *gorm.DB, entity *model.WikiEntity) (*model.WikiEntity, error) {
	entity.CanonicalName = strings.TrimSpace(entity.CanonicalName)
	entity.NormalizedName = normalizeWikiEntityName(entity.CanonicalName)
	if entity.CanonicalName == "" || entity.NormalizedName == "" {
		return nil, fmt.Errorf("wiki entity name is empty")
	}
	if entity.EntityType == "" {
		entity.EntityType = model.WikiEntityConcept
	}
	if entity.Status == "" {
		entity.Status = model.WikiEntityActive
	}

	var existing model.WikiEntity
	err := db.Where("book_id = ? AND entity_type = ? AND normalized_name = ?",
		entity.BookID, entity.EntityType, entity.NormalizedName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(entity).Error; err != nil {
			return nil, err
		}
		return entity, nil
	}
	if err != nil {
		return nil, err
	}

	existing.CanonicalName = entity.CanonicalName
	if strings.TrimSpace(entity.Summary) != "" {
		existing.Summary = entity.Summary
	}
	existing.Status = entity.Status
	existing.Managed = existing.Managed || entity.Managed
	if existing.FirstSeenChapter == 0 || (entity.FirstSeenChapter > 0 && entity.FirstSeenChapter < existing.FirstSeenChapter) {
		existing.FirstSeenChapter = entity.FirstSeenChapter
	}
	if entity.LastSeenChapter > existing.LastSeenChapter {
		existing.LastSeenChapter = entity.LastSeenChapter
	}
	if strings.TrimSpace(entity.MetadataJSON) != "" {
		existing.MetadataJSON = entity.MetadataJSON
	}
	if err := db.Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func upsertWikiAlias(db *gorm.DB, entity *model.WikiEntity, alias string, canonical, derived bool) error {
	alias = strings.TrimSpace(alias)
	normalized := normalizeWikiEntityName(alias)
	if alias == "" || normalized == "" {
		return nil
	}
	row := model.WikiEntityAlias{
		BookID:          entity.BookID,
		EntityID:        entity.ID,
		EntityType:      entity.EntityType,
		Alias:           alias,
		NormalizedAlias: normalized,
		IsCanonical:     canonical,
		IsDerived:       derived,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "entity_id"},
			{Name: "normalized_alias"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"book_id", "entity_type", "alias", "is_canonical", "is_derived", "updated_at",
		}),
	}).Create(&row).Error
}

func parseWikiCanonicalAndAliases(raw string, extraAliases ...string) (string, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	canonical := raw
	aliases := append([]string(nil), extraAliases...)

	re := regexp.MustCompile(`^(.+?)[（(]([^）)]+)[）)]$`)
	if match := re.FindStringSubmatch(raw); len(match) == 3 {
		canonical = strings.TrimSpace(match[1])
		aliases = append(aliases, raw)
		aliases = append(aliases, splitWikiAliases(match[2])...)
	}

	seen := map[string]struct{}{normalizeWikiEntityName(canonical): {}}
	out := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		for _, item := range splitWikiAliases(alias) {
			normalized := normalizeWikiEntityName(item)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, strings.TrimSpace(item))
		}
	}
	return canonical, out
}

func extractWikiAliasesFromText(raw string) []string {
	re := regexp.MustCompile(`(?:别名|又名|化名|昵称|称呼)[：:]\s*([^\n；;。]+)`)
	aliases := make([]string, 0, 4)
	for _, match := range re.FindAllStringSubmatch(raw, -1) {
		if len(match) == 2 {
			aliases = append(aliases, splitWikiAliases(match[1])...)
		}
	}
	return aliases
}

func splitWikiAliases(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '、', ',', '，', '/', '|', ';', '；':
			return true
		default:
			return false
		}
	})
}

func normalizeWikiEntityName(raw string) string {
	raw = strings.ToLower(norm.NFKC.String(strings.TrimSpace(raw)))
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '"', '\'', '`', '“', '”', '‘', '’', '《', '》', '「', '」', '『', '』':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func looksLikeWikiEntityName(raw string) bool {
	raw = strings.TrimSpace(raw)
	runes := []rune(raw)
	if len(runes) == 0 || len(runes) > 32 {
		return false
	}
	if strings.ContainsAny(raw, "\n\r。！？!?；;：:") {
		return false
	}
	allDigits := true
	for _, r := range runes {
		if !unicode.IsDigit(r) && r != '.' && r != '%' {
			allDigits = false
			break
		}
	}
	return !allDigits
}

func inferWikiEntityType(name, predicate, category string, subject bool, known map[string]model.WikiEntityType) model.WikiEntityType {
	if entityType, ok := known[normalizeWikiEntityName(name)]; ok {
		return entityType
	}
	if strings.Contains(predicate, "位于") || strings.Contains(predicate, "前往") ||
		strings.Contains(predicate, "抵达") || strings.Contains(predicate, "来自") {
		if !subject {
			return model.WikiEntityPlace
		}
	}
	if category == "item" && !subject {
		return model.WikiEntityItem
	}
	if category == "rule" {
		return model.WikiEntityRule
	}
	if strings.Contains(predicate, "加入") || strings.Contains(predicate, "隶属") {
		if !subject {
			return model.WikiEntityOrganization
		}
	}
	return model.WikiEntityConcept
}

func firstNonEmptyWiki(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func maxWikiChapter(left, right uint) uint {
	if left > right {
		return left
	}
	return right
}
