package mysql

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"errors"

	"gorm.io/gorm"
)

type truthFileRepo struct {
	db *gorm.DB
}

func NewTruthFileRepo(db *gorm.DB) repository.TruthFileRepository {
	return &truthFileRepo{db: db}
}

func (r *truthFileRepo) WithinTx(fn func(repository.TruthFileRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&truthFileRepo{db: tx})
	})
}

func (r *truthFileRepo) GetCharacters(bookID uint) ([]model.Character, error) {
	var chars []model.Character
	err := r.db.Where("book_id = ?", bookID).Order("updated_at desc, id desc").Find(&chars).Error
	if err != nil {
		return nil, err
	}
	return mergeCharacters(chars), nil
}

func (r *truthFileRepo) SaveCharacter(c *model.Character) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return nil
	}
	c.Profile = strings.TrimSpace(c.Profile)
	c.CoreTags = strings.TrimSpace(c.CoreTags)
	c.ContrastDetails = strings.TrimSpace(c.ContrastDetails)
	c.Backstory = strings.TrimSpace(c.Backstory)
	c.CharacterArc = strings.TrimSpace(c.CharacterArc)
	c.CurrentStatus = strings.TrimSpace(c.CurrentStatus)
	c.RelationshipNetwork = strings.TrimSpace(c.RelationshipNetwork)
	c.InnerDrive = strings.TrimSpace(c.InnerDrive)
	c.GrowthArc = strings.TrimSpace(c.GrowthArc)

	var existing model.Character
	err := r.db.Where("book_id = ? AND name = ?", c.BookID, c.Name).Order("updated_at desc, id desc").First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.Create(c).Error
		}
		return err
	}

	merged := mergeCharacter(existing, *c)
	return r.db.Save(&merged).Error
}

func (r *truthFileRepo) GetFacts(bookID uint) ([]model.Fact, error) {
	var facts []model.Fact
	err := r.db.Where("book_id = ?", bookID).Order("valid_from_chapter desc, id desc").Find(&facts).Error
	if err != nil {
		return nil, err
	}
	return activeFactsView(facts), nil
}

func (r *truthFileRepo) SaveFact(f *model.Fact) error {
	f.Subject = strings.TrimSpace(f.Subject)
	f.Predicate = strings.TrimSpace(f.Predicate)
	f.Object = strings.TrimSpace(f.Object)
	if f.Subject == "" || f.Predicate == "" || f.Object == "" {
		return nil
	}
	f.Category = normalizeFactCategoryValue(f.Category, f.Predicate, f.Object)
	if !isDurableFact(*f) {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var active []model.Fact
		if err := tx.Where("book_id = ? AND subject = ? AND predicate = ? AND valid_until_chapter IS NULL", f.BookID, f.Subject, f.Predicate).
			Order("valid_from_chapter desc, id desc").
			Find(&active).Error; err != nil {
			return err
		}

		for _, current := range active {
			if strings.EqualFold(strings.TrimSpace(current.Object), f.Object) {
				if current.SourceChapter == f.SourceChapter || current.ValidFromChapter == f.ValidFromChapter {
					return tx.Model(&current).Updates(map[string]any{
						"object":         f.Object,
						"category":       f.Category,
						"source_chapter": f.SourceChapter,
					}).Error
				}
				return nil
			}

			until := f.ValidFromChapter
			if until > 0 {
				until--
			}
			if until < current.ValidFromChapter {
				until = current.ValidFromChapter
			}
			if err := tx.Model(&current).Update("valid_until_chapter", until).Error; err != nil {
				return err
			}
		}

		return tx.Create(f).Error
	})
}

func (r *truthFileRepo) GetHooks(bookID uint) ([]model.Hook, error) {
	var hooks []model.Hook
	err := r.db.Where("book_id = ?", bookID).Order("id").Find(&hooks).Error
	return hooks, err
}

func (r *truthFileRepo) SaveHook(h *model.Hook) error {
	return r.db.Save(h).Error
}

func (r *truthFileRepo) GetChapterSummaries(bookID uint) ([]model.ChapterSummary, error) {
	var summaries []model.ChapterSummary
	err := r.db.Where("book_id = ?", bookID).Order("chapter_number").Find(&summaries).Error
	return summaries, err
}

func (r *truthFileRepo) SaveChapterSummary(s *model.ChapterSummary) error {
	return r.db.Save(s).Error
}

func (r *truthFileRepo) GetFoundation(bookID uint, fileType model.FoundationFileType) (*model.BookFoundation, error) {
	var f model.BookFoundation
	err := r.db.Where("book_id = ? AND file_type = ?", bookID, fileType).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *truthFileRepo) ListFoundations(bookID uint) ([]model.BookFoundation, error) {
	var foundations []model.BookFoundation
	err := r.db.Where("book_id = ?", bookID).Order("file_type").Find(&foundations).Error
	return foundations, err
}

func (r *truthFileRepo) SaveFoundation(f *model.BookFoundation) error {
	return r.db.Save(f).Error
}

func (r *truthFileRepo) GetChapter(bookID uint, chapterNumber uint) (*model.Chapter, error) {
	var ch model.Chapter
	err := r.db.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).First(&ch).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *truthFileRepo) SaveChapter(ch *model.Chapter) error {
	return r.db.Save(ch).Error
}

func (r *truthFileRepo) ListChapters(bookID uint) ([]model.Chapter, error) {
	var chapters []model.Chapter
	err := r.db.Where("book_id = ?", bookID).Order("chapter_number").Find(&chapters).Error
	return chapters, err
}

func (r *truthFileRepo) GetNextChapterNumber(bookID uint) (uint, error) {
	var maxNum uint
	err := r.db.Model(&model.Chapter{}).Where("book_id = ?", bookID).Select("COALESCE(MAX(chapter_number), 0)").Scan(&maxNum).Error
	return maxNum + 1, err
}

func (r *truthFileRepo) GetBook(bookID uint) (*model.Book, error) {
	var book model.Book
	err := r.db.Preload("Genre").Preload("Platform").First(&book, bookID).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *truthFileRepo) GetBookState(bookID uint) (*model.BookState, error) {
	var state model.BookState
	err := r.db.Where("book_id = ?", bookID).First(&state).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (r *truthFileRepo) SaveBookState(state *model.BookState) error {
	return r.db.Save(state).Error
}

func (r *truthFileRepo) UpdateBookStatus(bookID uint, status model.BookStatus) error {
	return r.db.Model(&model.Book{}).Where("id = ?", bookID).Update("status", status).Error
}

func (r *truthFileRepo) TransitionBookStatus(bookID uint, from []model.BookStatus, to model.BookStatus) (bool, error) {
	tx := r.db.Model(&model.Book{}).Where("id = ? AND status IN ?", bookID, from).Updates(map[string]interface{}{
		"status":     to,
		"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
	})
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *truthFileRepo) SaveChapterSnapshot(s *model.ChapterSnapshot) error {
	return r.db.Save(s).Error
}

func (r *truthFileRepo) GetChapterSnapshots(bookID uint) ([]model.ChapterSnapshot, error) {
	var snapshots []model.ChapterSnapshot
	err := r.db.Where("book_id = ?", bookID).Order("chapter_number").Find(&snapshots).Error
	return snapshots, err
}

func (r *truthFileRepo) SaveRuntimeArtifact(a *model.RuntimeArtifact) error {
	return r.db.Save(a).Error
}

func (r *truthFileRepo) GetRuntimeArtifacts(bookID uint, chapterNumber uint) ([]model.RuntimeArtifact, error) {
	var artifacts []model.RuntimeArtifact
	err := r.db.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).Order("artifact_type").Find(&artifacts).Error
	return artifacts, err
}

func (r *truthFileRepo) GetAgentModelRoute(bookID uint, agentName string) (*model.AgentModelRoute, error) {
	var route model.AgentModelRoute
	err := r.db.Where("book_id = ? AND agent_name = ?", bookID, agentName).First(&route).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &route, nil
}

func (r *truthFileRepo) SaveAgentModelRoute(route *model.AgentModelRoute) error {
	return r.db.Save(route).Error
}

func (r *truthFileRepo) DeleteBookCascade(bookID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		deleteOps := []struct {
			query string
			value interface{}
		}{
			{"book_id = ?", &model.AgentModelRoute{}},
			{"book_id = ?", &model.RuntimeArtifact{}},
			{"book_id = ?", &model.ChapterSnapshot{}},
			{"book_id = ?", &model.ChapterSummary{}},
			{"book_id = ?", &model.Fact{}},
			{"book_id = ?", &model.Hook{}},
			{"book_id = ?", &model.BookFoundation{}},
			{"book_id = ?", &model.BookState{}},
			{"book_id = ?", &model.Character{}},
			{"book_id = ?", &model.Chapter{}},
		}

		for _, op := range deleteOps {
			if err := tx.Where(op.query, bookID).Delete(op.value).Error; err != nil {
				return err
			}
		}

		if err := tx.Delete(&model.Book{}, bookID).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *truthFileRepo) DeleteLatestChapterCascade(bookID uint, chapterNumber uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var maxChapter uint
		if err := tx.Model(&model.Chapter{}).
			Where("book_id = ?", bookID).
			Select("COALESCE(MAX(chapter_number), 0)").
			Scan(&maxChapter).Error; err != nil {
			return err
		}
		if maxChapter == 0 {
			return fmt.Errorf("book has no chapters")
		}
		if maxChapter != chapterNumber {
			return fmt.Errorf("only the latest chapter can be deleted")
		}

		prevChapter := chapterNumber - 1
		var prevSnapshot model.ChapterSnapshot
		if err := tx.Where("book_id = ? AND chapter_number = ?", bookID, prevChapter).First(&prevSnapshot).Error; err != nil {
			return fmt.Errorf("load previous snapshot: %w", err)
		}

		if err := tx.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).Delete(&model.RuntimeArtifact{}).Error; err != nil {
			return err
		}
		if err := tx.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).Delete(&model.ChapterSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).Delete(&model.ChapterSummary{}).Error; err != nil {
			return err
		}
		if err := tx.Where("book_id = ? AND source_chapter = ?", bookID, chapterNumber).Delete(&model.Fact{}).Error; err != nil {
			return err
		}
		if err := tx.Where("book_id = ? AND start_chapter = ?", bookID, chapterNumber).Delete(&model.Hook{}).Error; err != nil {
			return err
		}
		if err := tx.Where("book_id = ? AND chapter_number = ?", bookID, chapterNumber).Delete(&model.Chapter{}).Error; err != nil {
			return err
		}

		status := model.BookStatusActive
		if prevChapter == 0 {
			status = model.BookStatusOutlining
		}
		if err := tx.Model(&model.Book{}).Where("id = ?", bookID).Updates(map[string]any{
			"status": status,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.BookState{}).
			Where("book_id = ?", bookID).
			Updates(map[string]any{
				"current_chapter": prevChapter,
				"source_chapter":  prevChapter,
			}).Error; err != nil {
			return err
		}

		if err := restoreTruthStateFromSnapshot(tx, bookID, chapterNumber, prevSnapshot); err != nil {
			return err
		}

		return nil
	})
}

func restoreTruthStateFromSnapshot(tx *gorm.DB, bookID uint, deletedChapter uint, snapshot model.ChapterSnapshot) error {
	var (
		foundations []model.BookFoundation
		characters  []model.Character
		facts       []model.Fact
		hooks       []model.Hook
		summaries   []model.ChapterSummary
		bookState   *model.BookState
	)

	if err := unmarshalSnapshotJSON(snapshot.FoundationsJSON, &foundations); err != nil {
		return fmt.Errorf("parse snapshot foundations: %w", err)
	}
	if err := unmarshalSnapshotJSON(snapshot.CharactersJSON, &characters); err != nil {
		return fmt.Errorf("parse snapshot characters: %w", err)
	}
	if err := unmarshalSnapshotJSON(snapshot.FactsJSON, &facts); err != nil {
		return fmt.Errorf("parse snapshot facts: %w", err)
	}
	if err := unmarshalSnapshotJSON(snapshot.HooksJSON, &hooks); err != nil {
		return fmt.Errorf("parse snapshot hooks: %w", err)
	}
	if err := unmarshalSnapshotJSON(snapshot.ChapterSummariesJSON, &summaries); err != nil {
		return fmt.Errorf("parse snapshot summaries: %w", err)
	}
	if strings.TrimSpace(snapshot.BookStateJSON) != "" && strings.TrimSpace(snapshot.BookStateJSON) != "null" {
		var state model.BookState
		if err := json.Unmarshal([]byte(snapshot.BookStateJSON), &state); err != nil {
			return fmt.Errorf("parse snapshot book state: %w", err)
		}
		bookState = &state
	} else {
		rebuilt, err := rebuildBookStateFromSnapshots(tx, bookID, snapshot.ChapterNumber)
		if err != nil {
			return fmt.Errorf("rebuild snapshot book state: %w", err)
		}
		bookState = rebuilt
	}

	if len(characters) > 0 {
		if err := tx.Where("book_id = ?", bookID).Delete(&model.Character{}).Error; err != nil {
			return err
		}
		for i := range characters {
			characters[i].ID = 0
			characters[i].BookID = bookID
			if err := tx.Create(&characters[i]).Error; err != nil {
				return err
			}
		}
	} else {
		if err := tx.Where("book_id = ? AND source_chapter = ?", bookID, deletedChapter).Delete(&model.Character{}).Error; err != nil {
			return err
		}
	}

	if err := tx.Where("book_id = ?", bookID).Delete(&model.Fact{}).Error; err != nil {
		return err
	}
	for i := range facts {
		facts[i].ID = 0
		facts[i].BookID = bookID
		if err := tx.Create(&facts[i]).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("book_id = ?", bookID).Delete(&model.Hook{}).Error; err != nil {
		return err
	}
	for i := range hooks {
		hooks[i].ID = 0
		hooks[i].BookID = bookID
		if err := tx.Create(&hooks[i]).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("book_id = ?", bookID).Delete(&model.ChapterSummary{}).Error; err != nil {
		return err
	}
	for i := range summaries {
		summaries[i].ID = 0
		summaries[i].BookID = bookID
		if err := tx.Create(&summaries[i]).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("book_id = ?", bookID).Delete(&model.BookState{}).Error; err != nil {
		return err
	}
	if bookState != nil {
		bookState.ID = 0
		bookState.BookID = bookID
		if err := tx.Create(bookState).Error; err != nil {
			return err
		}
	}
	if err := restoreFoundationsFromSnapshot(tx, bookID, snapshot.ChapterNumber, foundations, bookState); err != nil {
		return err
	}

	return nil
}

func unmarshalSnapshotJSON[T any](raw string, out *[]T) error {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" {
		*out = []T{}
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

func rebuildBookStateFromSnapshots(tx *gorm.DB, bookID uint, targetChapter uint) (*model.BookState, error) {
	var snapshots []model.ChapterSnapshot
	if err := tx.Where("book_id = ? AND chapter_number <= ?", bookID, targetChapter).
		Order("chapter_number").
		Find(&snapshots).Error; err != nil {
		return nil, err
	}

	state := &model.BookState{
		BookID:         bookID,
		CurrentChapter: targetChapter,
		SourceChapter:  targetChapter,
	}

	for _, snapshot := range snapshots {
		if strings.TrimSpace(snapshot.BookStateJSON) != "" && strings.TrimSpace(snapshot.BookStateJSON) != "null" {
			var full model.BookState
			if err := json.Unmarshal([]byte(snapshot.BookStateJSON), &full); err == nil {
				state = &full
				continue
			}
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(snapshot.CurrentStateJSON), &raw); err != nil {
			continue
		}

		if v, ok := raw["protagonist"].(string); ok && strings.TrimSpace(v) != "" {
			state.ProtagonistName = strings.TrimSpace(v)
		}
		if v, ok := raw["situation_summary"].(string); ok && strings.TrimSpace(v) != "" {
			state.SituationSummary = strings.TrimSpace(v)
		}
		if v, ok := raw["current_location"].(string); ok && strings.TrimSpace(v) != "" {
			state.CurrentLocation = strings.TrimSpace(v)
		}
		if v, ok := raw["protagonist_state"].(string); ok && strings.TrimSpace(v) != "" {
			state.ProtagonistState = strings.TrimSpace(v)
		}
		if v, ok := raw["current_goal"].(string); ok && strings.TrimSpace(v) != "" {
			state.CurrentGoal = strings.TrimSpace(v)
		}
		if v, ok := raw["current_constraint"].(string); ok && strings.TrimSpace(v) != "" {
			state.CurrentConstraint = strings.TrimSpace(v)
		}
		if v, ok := raw["current_alliances"].(string); ok && strings.TrimSpace(v) != "" {
			state.CurrentAlliances = strings.TrimSpace(v)
		}
		if v, ok := raw["current_conflict"].(string); ok && strings.TrimSpace(v) != "" {
			state.CurrentConflict = strings.TrimSpace(v)
		}

		if patchRaw, ok := raw["writer_state"].(string); ok && strings.TrimSpace(patchRaw) != "" {
			var patch map[string]string
			if err := json.Unmarshal([]byte(patchRaw), &patch); err == nil {
				applyStatePatch(state, patch)
			}
		}
	}

	state.BookID = bookID
	state.CurrentChapter = targetChapter
	state.SourceChapter = targetChapter
	if strings.TrimSpace(state.SituationSummary) == "" {
		state.SituationSummary = buildBookStateSummary(state)
	}
	return state, nil
}

func applyStatePatch(state *model.BookState, patch map[string]string) {
	if state == nil {
		return
	}
	if v := strings.TrimSpace(patch["currentLocation"]); v != "" {
		state.CurrentLocation = v
	}
	if v := strings.TrimSpace(patch["protagonistState"]); v != "" {
		state.ProtagonistState = v
	}
	if v := strings.TrimSpace(patch["currentGoal"]); v != "" {
		state.CurrentGoal = v
	}
	if v := strings.TrimSpace(patch["currentConstraint"]); v != "" {
		state.CurrentConstraint = v
	}
	if v := strings.TrimSpace(patch["currentAlliances"]); v != "" {
		state.CurrentAlliances = v
	}
	if v := strings.TrimSpace(patch["currentConflict"]); v != "" {
		state.CurrentConflict = v
	}
	state.SituationSummary = buildBookStateSummary(state)
}

func buildBookStateSummary(state *model.BookState) string {
	if state == nil {
		return ""
	}
	parts := []string{
		strings.TrimSpace(state.ProtagonistState),
		strings.TrimSpace(state.CurrentLocation),
		strings.TrimSpace(state.CurrentGoal),
		strings.TrimSpace(state.CurrentConstraint),
		strings.TrimSpace(state.CurrentConflict),
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return strings.TrimSpace(state.SituationSummary)
	}
	return clipRunes(strings.Join(filtered, "；"), 500)
}

func restoreFoundationsFromSnapshot(tx *gorm.DB, bookID uint, chapterNumber uint, foundations []model.BookFoundation, bookState *model.BookState) error {
	if len(foundations) > 0 {
		if err := tx.Where("book_id = ?", bookID).Delete(&model.BookFoundation{}).Error; err != nil {
			return err
		}
		for i := range foundations {
			foundations[i].ID = 0
			foundations[i].BookID = bookID
			if err := tx.Create(&foundations[i]).Error; err != nil {
				return err
			}
		}
		return nil
	}

	dynamicTypes := []model.FoundationFileType{
		model.FoundationCurrentFocus,
		model.FoundationAuditDrift,
	}
	if err := tx.Where("book_id = ? AND file_type IN ?", bookID, dynamicTypes).Delete(&model.BookFoundation{}).Error; err != nil {
		return err
	}
	if bookState != nil {
		if content := buildCurrentFocusFoundation(chapterNumber, bookState); strings.TrimSpace(content) != "" {
			if err := tx.Create(&model.BookFoundation{
				BookID:   bookID,
				FileType: model.FoundationCurrentFocus,
				Content:  content,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func buildCurrentFocusFoundation(chapterNumber uint, state *model.BookState) string {
	if state == nil {
		return ""
	}

	patch := map[string]string{
		"currentLocation":   strings.TrimSpace(state.CurrentLocation),
		"protagonistState":  strings.TrimSpace(state.ProtagonistState),
		"currentGoal":       strings.TrimSpace(state.CurrentGoal),
		"currentConstraint": strings.TrimSpace(state.CurrentConstraint),
		"currentAlliances":  strings.TrimSpace(state.CurrentAlliances),
		"currentConflict":   strings.TrimSpace(state.CurrentConflict),
	}
	keys := []string{
		"currentLocation",
		"protagonistState",
		"currentGoal",
		"currentConstraint",
		"currentAlliances",
		"currentConflict",
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## 第 %d 章后当前焦点\n", chapterNumber))
	for _, key := range keys {
		value := strings.TrimSpace(patch[key])
		if value == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s：%s\n", key, value))
	}
	if b.Len() == 0 {
		return ""
	}
	return strings.TrimSpace(b.String())
}

func clipRunes(raw string, max int) string {
	runes := []rune(raw)
	if len(runes) <= max {
		return raw
	}
	return string(runes[:max])
}

func mergeCharacters(chars []model.Character) []model.Character {
	merged := make(map[string]model.Character, len(chars))
	order := make([]string, 0, len(chars))
	for _, c := range chars {
		key := normalizeEntityKey(c.Name)
		if key == "" {
			key = fmt.Sprintf("id:%d", c.ID)
		}
		if existing, ok := merged[key]; ok {
			merged[key] = mergeCharacter(existing, c)
			continue
		}
		merged[key] = c
		order = append(order, key)
	}

	result := make([]model.Character, 0, len(order))
	for _, key := range order {
		result = append(result, merged[key])
	}
	sort.SliceStable(result, func(i, j int) bool {
		pi := characterRolePriority(result[i].RoleType)
		pj := characterRolePriority(result[j].RoleType)
		if pi != pj {
			return pi < pj
		}
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func mergeCharacter(base model.Character, incoming model.Character) model.Character {
	if characterRolePriority(incoming.RoleType) < characterRolePriority(base.RoleType) {
		base.RoleType = incoming.RoleType
	}
	base.IsPlaceholder = base.IsPlaceholder && incoming.IsPlaceholder
	base.Profile = preferNonEmpty(incoming.Profile, base.Profile)
	base.CoreTags = preferNonEmpty(incoming.CoreTags, base.CoreTags)
	base.ContrastDetails = preferNonEmpty(incoming.ContrastDetails, base.ContrastDetails)
	base.Backstory = preferNonEmpty(incoming.Backstory, base.Backstory)
	base.CharacterArc = preferNonEmpty(incoming.CharacterArc, base.CharacterArc)
	base.CurrentStatus = preferNonEmpty(incoming.CurrentStatus, base.CurrentStatus)
	base.RelationshipNetwork = preferNonEmpty(incoming.RelationshipNetwork, base.RelationshipNetwork)
	base.InnerDrive = preferNonEmpty(incoming.InnerDrive, base.InnerDrive)
	base.GrowthArc = preferNonEmpty(incoming.GrowthArc, base.GrowthArc)
	if base.SourceChapter == 0 || (incoming.SourceChapter != 0 && incoming.SourceChapter < base.SourceChapter) {
		base.SourceChapter = incoming.SourceChapter
	}
	if incoming.LastSeenChapter > base.LastSeenChapter {
		base.LastSeenChapter = incoming.LastSeenChapter
	}
	if strings.TrimSpace(incoming.Name) != "" {
		base.Name = strings.TrimSpace(incoming.Name)
	}
	return base
}

func activeFactsView(facts []model.Fact) []model.Fact {
	current := make(map[string]model.Fact, len(facts))
	order := make([]string, 0, len(facts))
	for _, f := range facts {
		if f.ValidUntilChapter != nil {
			continue
		}
		f.Category = normalizeFactCategoryValue(f.Category, f.Predicate, f.Object)
		if !isDurableFact(f) {
			continue
		}
		key := normalizeEntityKey(f.Subject) + "|" + normalizeEntityKey(f.Predicate)
		if key == "|" {
			continue
		}
		if _, ok := current[key]; ok {
			continue
		}
		current[key] = f
		order = append(order, key)
	}

	result := make([]model.Fact, 0, len(order))
	for _, key := range order {
		result = append(result, current[key])
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Subject != result[j].Subject {
			return result[i].Subject < result[j].Subject
		}
		if result[i].Predicate != result[j].Predicate {
			return result[i].Predicate < result[j].Predicate
		}
		if result[i].ValidFromChapter != result[j].ValidFromChapter {
			return result[i].ValidFromChapter < result[j].ValidFromChapter
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func normalizeFactCategoryValue(explicit string, predicate string, object string) string {
	if v := normalizeEntityKey(explicit); v != "" {
		switch v {
		case "identity", "resource", "item", "rule", "relationship":
			return v
		}
	}

	p := normalizeEntityKey(predicate)
	o := normalizeEntityKey(object)

	switch {
	case containsAny(p, "身份", "身份变更", "身世", "灵根", "修为", "境界", "血脉", "职业", "职位", "少主", "身份标签"),
		containsAny(o, "灵根", "杂灵根", "双灵根", "三灵根", "炼气", "筑基", "金丹", "少主", "弟子", "家主"):
		return "identity"
	case containsAny(p, "获得", "持有", "拿到", "得到", "失去", "持有物", "归属", "线索载体"),
		containsAny(o, "古书", "戒指", "玉牌", "令牌", "丹药", "法器", "灵石", "钥匙", "卷轴", "地图"):
		return "item"
	case containsAny(p, "资源", "供给", "配额", "月例", "俸禄", "修炼资源", "代价", "消耗", "灵石", "配额"):
		return "resource"
	case containsAny(p, "规则", "核心规则", "修行规则", "禁忌", "条件", "门槛", "法则", "必须", "不可"),
		containsAny(o, "不可", "必须", "否则", "方可", "才能"):
		return "rule"
	case containsAny(p, "关系", "敌对", "盟友", "婚约", "师徒", "父子", "母子", "提及", "归属势力", "跟踪目标", "目的"):
		return "relationship"
	default:
		return ""
	}
}

func isDurableFact(f model.Fact) bool {
	category := normalizeFactCategoryValue(f.Category, f.Predicate, f.Object)
	if category == "" {
		return false
	}
	p := normalizeEntityKey(f.Predicate)
	o := normalizeEntityKey(f.Object)
	if containsAny(p, "内容", "第一页", "第二页", "第三页", "第八页", "批注", "痕迹", "脚步声", "气味", "光线", "屋内", "窗外", "页", "观察") {
		return false
	}
	if containsAny(o, "潮气", "霉味", "漏光", "脚步声", "影子", "第一页", "第二页", "第八页", "批注", "内容", "字迹") {
		return false
	}
	return true
}

func normalizeEntityKey(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func characterRolePriority(role model.CharacterRoleType) int {
	switch role {
	case model.CharacterProtagonist:
		return 0
	case model.CharacterMajor:
		return 1
	default:
		return 2
	}
}

func preferNonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(text, normalizeEntityKey(keyword)) {
			return true
		}
	}
	return false
}
