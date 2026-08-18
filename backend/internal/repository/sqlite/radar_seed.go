package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

func SeedRadarTaxonomies(db *gorm.DB) {
	_ = db.Model(&model.RadarTaxonomy{}).
		Where("platform = ?", model.RadarPlatformFanqie).
		Update("is_active", false).Error
	_ = db.Model(&model.RadarTag{}).
		Where("platform = ?", model.RadarPlatformFanqie).
		Update("is_active", false).Error

	tags, err := FetchFanqieOfficialTags(context.Background())
	if err != nil {
		log.Printf("seed radar tags: fetch fanqie tags failed: %v", err)
		tags = fallbackFanqieOfficialTags()
	}
	if err := (&radarRepo{db: db}).SaveTags(tags); err != nil {
		log.Printf("seed radar tags: save tags failed: %v", err)
		return
	}
	log.Printf("seed radar tags: ensured %d fanqie official tags", len(tags))
}

func FetchFanqieOfficialTags(ctx context.Context) ([]model.RadarTag, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://fanqienovel.com/api/author/book/category_list/v0/?gender=-1", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fanqie tag api failed: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Code int `json:"code"`
		Data []struct {
			CategoryID  int64  `json:"category_id"`
			Label       string `json:"label"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("fanqie tag api: %s", parsed.Message)
	}

	tags := make([]model.RadarTag, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if item.CategoryID <= 0 || item.Name == "" {
			continue
		}
		tagType := normalizeFanqieTagType(item.Label)
		tags = append(tags, model.RadarTag{
			Platform:      model.RadarPlatformFanqie,
			PlatformTagID: item.CategoryID,
			Category:      tagType,
			TagType:       tagType,
			TagKey:        strconv.FormatInt(item.CategoryID, 10),
			TagName:       item.Name,
			Description:   item.Description,
			IsActive:      true,
		})
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("fanqie tag api returned no tags")
	}
	return tags, nil
}

func normalizeFanqieTagType(label string) string {
	switch label {
	case "主题":
		return "theme"
	case "角色":
		return "role"
	default:
		return "plot"
	}
}

func fallbackFanqieOfficialTags() []model.RadarTag {
	items := []struct {
		id      int64
		tagType string
		name    string
	}{
		{262, "plot", "都市脑洞"},
		{1, "theme", "都市"},
		{778, "theme", "搞笑轻松"},
		{91, "role", "多女主"},
		{856, "role", "全能"},
		{515, "theme", "末日求生"},
		{851, "theme", "规则怪谈"},
		{514, "theme", "灵气复苏"},
		{257, "plot", "玄幻脑洞"},
		{539, "plot", "悬疑脑洞"},
	}
	tags := make([]model.RadarTag, 0, len(items))
	for _, item := range items {
		tags = append(tags, model.RadarTag{
			Platform:      model.RadarPlatformFanqie,
			PlatformTagID: item.id,
			Category:      item.tagType,
			TagType:       item.tagType,
			TagKey:        strconv.FormatInt(item.id, 10),
			TagName:       item.name,
			IsActive:      true,
		})
	}
	return tags
}
