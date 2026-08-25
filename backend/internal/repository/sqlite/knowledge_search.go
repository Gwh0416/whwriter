package sqlite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

const knowledgeChunkSize = 480

type knowledgeDocumentSpec struct {
	SourceType        model.KnowledgeSourceType
	SourceID          string
	Title             string
	Content           string
	Importance        int
	ValidFromChapter  uint
	ValidUntilChapter *uint
	IsActive          bool
}

type knowledgeEvidenceNote struct {
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

func migrateKnowledgeSearchSchema(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_chunks_fts
		USING fts5(search_text, tokenize = 'unicode61 remove_diacritics 2')
	`).Error; err != nil {
		return fmt.Errorf("create FTS5 table (build with -tags sqlite_fts5): %w", err)
	}
	return nil
}

func rebuildKnowledgeSearchIndex(db *gorm.DB) error {
	var bookIDs []uint
	if err := db.Model(&model.Book{}).Order("id").Pluck("id", &bookIDs).Error; err != nil {
		return err
	}

	repo := &truthFileRepo{db: db}
	for _, bookID := range bookIDs {
		if err := repo.RefreshKnowledgeIndex(bookID); err != nil {
			return fmt.Errorf("refresh book %d: %w", bookID, err)
		}
	}

	return rebuildKnowledgeFTSRows(db)
}

func rebuildKnowledgeFTSRows(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM knowledge_chunks_fts").Error; err != nil {
			return err
		}

		var chunks []model.KnowledgeChunk
		if err := tx.Order("id").Find(&chunks).Error; err != nil {
			return err
		}
		for _, chunk := range chunks {
			if err := insertKnowledgeFTSRow(tx, chunk.ID, chunk.SearchText); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *truthFileRepo) RefreshKnowledgeIndex(bookID uint) error {
	if bookID == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		specs, err := collectKnowledgeDocumentSpecs(tx, bookID)
		if err != nil {
			return err
		}
		return syncKnowledgeDocumentSpecs(tx, bookID, specs)
	})
}

func (r *truthFileRepo) SearchKnowledge(query model.KnowledgeSearchQuery) ([]model.KnowledgeSearchResult, error) {
	if query.BookID == 0 {
		return nil, nil
	}
	matchQuery := buildKnowledgeMatchQuery(query.Query)
	if matchQuery == "" {
		return nil, nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 12
	}
	if limit > 40 {
		limit = 40
	}

	var b strings.Builder
	b.WriteString(`
		SELECT
			c.id AS chunk_id,
			d.id AS document_id,
			d.source_type AS source_type,
			d.source_id AS source_id,
			d.title AS title,
			c.content AS content,
			c.chunk_index AS chunk_index,
			d.importance AS importance,
			d.valid_from_chapter AS valid_from_chapter,
			-bm25(knowledge_chunks_fts) AS score
		FROM knowledge_chunks_fts
		JOIN knowledge_chunks c ON c.id = knowledge_chunks_fts.rowid
		JOIN knowledge_documents d ON d.id = c.document_id
		WHERE knowledge_chunks_fts MATCH ?
			AND c.book_id = ?
			AND d.is_active = 1
			AND (d.valid_from_chapter = 0 OR d.valid_from_chapter <= ?)
			AND (d.valid_until_chapter IS NULL OR d.valid_until_chapter >= ?)
	`)
	args := []any{matchQuery, query.BookID, query.ChapterNumber, query.ChapterNumber}
	if len(query.SourceTypes) > 0 {
		b.WriteString(" AND d.source_type IN (")
		for i, sourceType := range query.SourceTypes {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("?")
			args = append(args, sourceType)
		}
		b.WriteString(")")
	}
	b.WriteString(" ORDER BY bm25(knowledge_chunks_fts), d.importance DESC, c.id DESC LIMIT ?")
	args = append(args, limit)

	var results []model.KnowledgeSearchResult
	if err := r.db.Raw(b.String(), args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}
	return results, nil
}

func collectKnowledgeDocumentSpecs(db *gorm.DB, bookID uint) ([]knowledgeDocumentSpec, error) {
	specs := make([]knowledgeDocumentSpec, 0, 32)

	var foundations []model.BookFoundation
	if err := db.Where("book_id = ?", bookID).Order("file_type").Find(&foundations).Error; err != nil {
		return nil, err
	}
	for _, foundation := range foundations {
		content := strings.TrimSpace(foundation.Content)
		if content == "" {
			continue
		}
		specs = append(specs, knowledgeDocumentSpec{
			SourceType: model.KnowledgeSourceFoundation,
			SourceID:   string(foundation.FileType),
			Title:      "基础设定：" + string(foundation.FileType),
			Content:    content,
			Importance: foundationImportance(foundation.FileType),
			IsActive:   true,
		})
	}

	var characters []model.Character
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&characters).Error; err != nil {
		return nil, err
	}
	for _, character := range characters {
		content := renderKnowledgeCharacter(character)
		if content == "" {
			continue
		}
		specs = append(specs, knowledgeDocumentSpec{
			SourceType:       model.KnowledgeSourceCharacter,
			SourceID:         strconv.FormatUint(uint64(character.ID), 10),
			Title:            "角色：" + character.Name,
			Content:          content,
			Importance:       characterImportance(character.RoleType),
			ValidFromChapter: character.SourceChapter,
			IsActive:         true,
		})
	}

	var facts []model.Fact
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&facts).Error; err != nil {
		return nil, err
	}
	for _, fact := range facts {
		content := strings.TrimSpace(fmt.Sprintf("%s %s %s\n类别：%s\n来源章节：%d", fact.Subject, fact.Predicate, fact.Object, fact.Category, fact.SourceChapter))
		specs = append(specs, knowledgeDocumentSpec{
			SourceType:        model.KnowledgeSourceFact,
			SourceID:          strconv.FormatUint(uint64(fact.ID), 10),
			Title:             "长期事实：" + fact.Subject,
			Content:           content,
			Importance:        factImportance(fact.Category),
			ValidFromChapter:  fact.ValidFromChapter,
			ValidUntilChapter: fact.ValidUntilChapter,
			IsActive:          fact.ValidUntilChapter == nil,
		})
	}

	var hooks []model.Hook
	if err := db.Where("book_id = ?", bookID).Order("id").Find(&hooks).Error; err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		content := strings.TrimSpace(fmt.Sprintf(
			"伏笔：%s\n类型：%s\n状态：%s\n起始章节：%d\n最近推进：%d\n预期回收：%s\n回收节奏：%s\n备注：%s",
			hook.HookID, hook.Type, hook.Status, hook.StartChapter, hook.LastAdvancedChapter,
			hook.ExpectedPayoff, hook.PayoffTiming, hook.Notes,
		))
		specs = append(specs, knowledgeDocumentSpec{
			SourceType:       model.KnowledgeSourceHook,
			SourceID:         hook.HookID,
			Title:            "伏笔：" + hook.HookID,
			Content:          content,
			Importance:       hookImportance(hook),
			ValidFromChapter: hook.StartChapter,
			IsActive:         hook.Status != model.HookResolved && hook.Status != model.HookStale,
		})
	}

	var summaries []model.ChapterSummary
	if err := db.Where("book_id = ?", bookID).Order("chapter_number").Find(&summaries).Error; err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		content := strings.TrimSpace(fmt.Sprintf(
			"第%d章：%s\n出场人物：%s\n关键事件：%s\n状态变化：%s\n伏笔动态：%s\n情绪：%s\n章节类型：%s",
			summary.ChapterNumber, summary.Title, summary.CharactersAppeared, summary.KeyEvents,
			summary.StateChanges, summary.HookActivity, summary.Mood, summary.ChapterType,
		))
		specs = append(specs, knowledgeDocumentSpec{
			SourceType:       model.KnowledgeSourceSummary,
			SourceID:         strconv.FormatUint(uint64(summary.ChapterNumber), 10),
			Title:            fmt.Sprintf("第%d章摘要：%s", summary.ChapterNumber, summary.Title),
			Content:          content,
			Importance:       3,
			ValidFromChapter: summary.ChapterNumber,
			IsActive:         true,
		})
	}

	var evidenceArtifacts []model.RuntimeArtifact
	if err := db.Where("book_id = ? AND artifact_type = ?", bookID, model.ArtifactEvidence).
		Order("chapter_number").Find(&evidenceArtifacts).Error; err != nil {
		return nil, err
	}
	for _, artifact := range evidenceArtifacts {
		content := renderKnowledgeEvidence(artifact.Content)
		if content == "" {
			continue
		}
		specs = append(specs, knowledgeDocumentSpec{
			SourceType:       model.KnowledgeSourceEvidence,
			SourceID:         strconv.FormatUint(uint64(artifact.ChapterNumber), 10),
			Title:            fmt.Sprintf("第%d章证据", artifact.ChapterNumber),
			Content:          content,
			Importance:       3,
			ValidFromChapter: artifact.ChapterNumber,
			IsActive:         true,
		})
	}

	return specs, nil
}

func syncKnowledgeDocumentSpecs(db *gorm.DB, bookID uint, specs []knowledgeDocumentSpec) error {
	var existing []model.KnowledgeDocument
	if err := db.Where("book_id = ?", bookID).Find(&existing).Error; err != nil {
		return err
	}
	existingByKey := make(map[string]model.KnowledgeDocument, len(existing))
	for _, document := range existing {
		existingByKey[knowledgeDocumentKey(document.SourceType, document.SourceID)] = document
	}

	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		spec.Content = strings.TrimSpace(spec.Content)
		if spec.Content == "" {
			continue
		}
		key := knowledgeDocumentKey(spec.SourceType, spec.SourceID)
		seen[key] = struct{}{}
		document := model.KnowledgeDocument{
			BookID:            bookID,
			SourceType:        spec.SourceType,
			SourceID:          spec.SourceID,
			Title:             strings.TrimSpace(spec.Title),
			Content:           spec.Content,
			ContentHash:       hashKnowledgeDocument(spec),
			Importance:        spec.Importance,
			ValidFromChapter:  spec.ValidFromChapter,
			ValidUntilChapter: spec.ValidUntilChapter,
			IsActive:          spec.IsActive,
		}

		current, found := existingByKey[key]
		if found {
			document.ID = current.ID
			if knowledgeDocumentEqual(current, document) {
				continue
			}
			if err := deleteKnowledgeDocumentChunks(db, current.ID); err != nil {
				return err
			}
			if err := db.Save(&document).Error; err != nil {
				return err
			}
		} else if err := db.Create(&document).Error; err != nil {
			return err
		}

		if err := createKnowledgeDocumentChunks(db, document); err != nil {
			return err
		}
	}

	for _, document := range existing {
		if _, ok := seen[knowledgeDocumentKey(document.SourceType, document.SourceID)]; ok {
			continue
		}
		if err := deleteKnowledgeDocumentChunks(db, document.ID); err != nil {
			return err
		}
		if err := db.Delete(&document).Error; err != nil {
			return err
		}
	}

	return nil
}

func createKnowledgeDocumentChunks(db *gorm.DB, document model.KnowledgeDocument) error {
	chunks := splitKnowledgeChunks(document.Content, knowledgeChunkSize)
	for index, content := range chunks {
		chunk := model.KnowledgeChunk{
			DocumentID: document.ID,
			BookID:     document.BookID,
			SourceType: document.SourceType,
			ChunkIndex: uint(index),
			Content:    content,
			SearchText: buildKnowledgeSearchText(document.Title + "\n" + content),
			TokenCount: len([]rune(content)),
		}
		if err := db.Create(&chunk).Error; err != nil {
			return err
		}
		if err := insertKnowledgeFTSRow(db, chunk.ID, chunk.SearchText); err != nil {
			return err
		}
	}
	return nil
}

func deleteKnowledgeDocumentChunks(db *gorm.DB, documentID uint) error {
	var chunks []model.KnowledgeChunk
	if err := db.Where("document_id = ?", documentID).Find(&chunks).Error; err != nil {
		return err
	}
	for _, chunk := range chunks {
		if err := db.Exec("DELETE FROM knowledge_chunks_fts WHERE rowid = ?", chunk.ID).Error; err != nil {
			return err
		}
	}
	return db.Where("document_id = ?", documentID).Delete(&model.KnowledgeChunk{}).Error
}

func deleteBookKnowledgeIndex(db *gorm.DB, bookID uint) error {
	var documents []model.KnowledgeDocument
	if err := db.Where("book_id = ?", bookID).Find(&documents).Error; err != nil {
		return err
	}
	for _, document := range documents {
		if err := deleteKnowledgeDocumentChunks(db, document.ID); err != nil {
			return err
		}
	}
	return db.Where("book_id = ?", bookID).Delete(&model.KnowledgeDocument{}).Error
}

func insertKnowledgeFTSRow(db *gorm.DB, chunkID uint, searchText string) error {
	if strings.TrimSpace(searchText) == "" {
		return nil
	}
	return db.Exec("INSERT INTO knowledge_chunks_fts(rowid, search_text) VALUES (?, ?)", chunkID, searchText).Error
}

func splitKnowledgeChunks(raw string, maxRunes int) []string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	if raw == "" {
		return nil
	}
	if maxRunes <= 0 {
		maxRunes = knowledgeChunkSize
	}

	paragraphs := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	chunks := make([]string, 0, len(paragraphs))
	var current strings.Builder
	currentLen := 0
	flush := func() {
		if text := strings.TrimSpace(current.String()); text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
		currentLen = 0
	}

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		for _, piece := range splitKnowledgeLongText(paragraph, maxRunes) {
			pieceLen := len([]rune(piece))
			if currentLen > 0 && currentLen+1+pieceLen > maxRunes {
				flush()
			}
			if currentLen > 0 {
				current.WriteByte('\n')
				currentLen++
			}
			current.WriteString(piece)
			currentLen += pieceLen
		}
	}
	flush()
	return chunks
}

func splitKnowledgeLongText(raw string, maxRunes int) []string {
	runes := []rune(strings.TrimSpace(raw))
	if len(runes) <= maxRunes {
		return []string{string(runes)}
	}
	out := make([]string, 0, len(runes)/maxRunes+1)
	for len(runes) > maxRunes {
		out = append(out, strings.TrimSpace(string(runes[:maxRunes])))
		runes = runes[maxRunes:]
	}
	if tail := strings.TrimSpace(string(runes)); tail != "" {
		out = append(out, tail)
	}
	return out
}

func buildKnowledgeSearchText(raw string) string {
	return strings.Join(knowledgeSearchTokens(raw), " ")
}

func buildKnowledgeMatchQuery(raw string) string {
	tokens := knowledgeSearchTokens(raw)
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) > 32 {
		tokens = tokens[:32]
	}
	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		quoted = append(quoted, `"`+token+`"`)
	}
	return strings.Join(quoted, " OR ")
}

func knowledgeSearchTokens(raw string) []string {
	const maxTokens = 96
	seen := make(map[string]struct{}, maxTokens)
	tokens := make([]string, 0, maxTokens)
	add := func(token string) {
		token = strings.ToLower(strings.TrimSpace(token))
		if len([]rune(token)) < 2 || isCommonKnowledgeToken(token) {
			return
		}
		if _, ok := seen[token]; ok {
			return
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}

	var cjkRun []rune
	var latinRun []rune
	flushCJK := func() {
		for i := 0; i+1 < len(cjkRun); i++ {
			add(string(cjkRun[i : i+2]))
		}
		cjkRun = cjkRun[:0]
	}
	flushLatin := func() {
		add(string(latinRun))
		latinRun = latinRun[:0]
	}

	for _, r := range []rune(raw) {
		switch {
		case isCJK(r):
			flushLatin()
			cjkRun = append(cjkRun, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			flushCJK()
			latinRun = append(latinRun, r)
		default:
			flushCJK()
			flushLatin()
		}
	}
	flushCJK()
	flushLatin()
	return tokens
}

func isCJK(r rune) bool {
	return (r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0xF900 && r <= 0xFAFF)
}

func isCommonKnowledgeToken(token string) bool {
	switch token {
	case "当前", "本章", "主角", "故事", "小说", "需要", "必须", "进行", "什么", "一个",
		"通过", "以及", "不要", "这一", "之后", "已经", "可以", "如果", "然后", "为了",
		"根据", "内容", "章节", "角色", "设定", "关系", "状态":
		return true
	default:
		return false
	}
}

func hashKnowledgeDocument(spec knowledgeDocumentSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n%s\n%s\n%d\n%d\n%t\n", spec.SourceType, spec.SourceID, spec.Title, spec.Content, spec.Importance, spec.ValidFromChapter, spec.IsActive)
	if spec.ValidUntilChapter != nil {
		fmt.Fprintf(&b, "%d", *spec.ValidUntilChapter)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func knowledgeDocumentEqual(left, right model.KnowledgeDocument) bool {
	if left.ContentHash != right.ContentHash ||
		left.Title != right.Title ||
		left.Importance != right.Importance ||
		left.ValidFromChapter != right.ValidFromChapter ||
		left.IsActive != right.IsActive {
		return false
	}
	if left.ValidUntilChapter == nil || right.ValidUntilChapter == nil {
		return left.ValidUntilChapter == nil && right.ValidUntilChapter == nil
	}
	return *left.ValidUntilChapter == *right.ValidUntilChapter
}

func knowledgeDocumentKey(sourceType model.KnowledgeSourceType, sourceID string) string {
	return string(sourceType) + ":" + sourceID
}

func renderKnowledgeCharacter(character model.Character) string {
	lines := []string{
		"姓名：" + character.Name,
		"角色类型：" + string(character.RoleType),
		"核心标签：" + character.CoreTags,
		"反差细节：" + character.ContrastDetails,
		"过往经历：" + character.Backstory,
		"人物弧线：" + character.CharacterArc,
		"当前状态：" + character.CurrentStatus,
		"关系网络：" + character.RelationshipNetwork,
		"内在驱动：" + character.InnerDrive,
		"成长弧光：" + character.GrowthArc,
		"简介：" + character.Profile,
	}
	return strings.Join(nonEmptyKnowledgeLines(lines), "\n")
}

func renderKnowledgeEvidence(raw string) string {
	var notes []knowledgeEvidenceNote
	if err := json.Unmarshal([]byte(raw), &notes); err != nil {
		return strings.TrimSpace(raw)
	}
	lines := make([]string, 0, len(notes))
	for _, note := range notes {
		line := strings.TrimSpace(fmt.Sprintf("%s（%s）：%s", note.Title, note.Kind, note.Content))
		if line != "（）：" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func nonEmptyKnowledgeLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "：") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func foundationImportance(fileType model.FoundationFileType) int {
	switch fileType {
	case model.FoundationBookRules, model.FoundationStoryFrame:
		return 5
	case model.FoundationCurrentFocus, model.FoundationAuthorIntent:
		return 4
	case model.FoundationVolumeMap:
		return 3
	default:
		return 2
	}
}

func characterImportance(roleType model.CharacterRoleType) int {
	switch roleType {
	case model.CharacterProtagonist:
		return 5
	case model.CharacterMajor:
		return 4
	default:
		return 2
	}
}

func factImportance(category string) int {
	if category == "rule" {
		return 5
	}
	return 4
}

func hookImportance(hook model.Hook) int {
	if hook.IsCritical {
		return 5
	}
	if hook.Status == model.HookProgressing || hook.Status == model.HookStale {
		return 4
	}
	return 3
}
