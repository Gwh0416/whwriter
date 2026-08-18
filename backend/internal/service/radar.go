package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"whwriter/backend/internal/agent"
	"whwriter/backend/internal/llm"
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"
)

type RadarService struct {
	repo                       repository.RadarRepository
	llm                        *llm.Client
	browserCDPURL              string
	browserChapterFetchTimeout time.Duration
	browserLaunch              browserLaunchOptions
	fanqieContentAPIURL        string
	fanqieContentAPITimeout    time.Duration
	fanqieCookieMu             sync.RWMutex
	fanqieCookieHeader         string
	activeScanMu               sync.RWMutex
	activeScanJobs             map[uint]struct{}
}

const (
	radarAllTagsCategory    = "__all__"
	radarChapterSampleLimit = 10
)

func NewRadarService(repo repository.RadarRepository, llmClient *llm.Client, browserCDPURL string, browserChapterFetchTimeoutSeconds int, browserAutoLaunch bool, browserChromeAppName string, browserUserDataDir string, fanqieContentAPIURL string, fanqieContentAPITimeoutSeconds int) *RadarService {
	if browserChapterFetchTimeoutSeconds <= 0 {
		browserChapterFetchTimeoutSeconds = 120
	}
	if fanqieContentAPITimeoutSeconds <= 0 {
		fanqieContentAPITimeoutSeconds = 8
	}
	return &RadarService{
		repo:                       repo,
		llm:                        llmClient,
		browserCDPURL:              strings.TrimSpace(browserCDPURL),
		browserChapterFetchTimeout: time.Duration(browserChapterFetchTimeoutSeconds) * time.Second,
		browserLaunch: browserLaunchOptions{
			AutoLaunch:    browserAutoLaunch,
			ChromeAppName: strings.TrimSpace(browserChromeAppName),
			UserDataDir:   strings.TrimSpace(browserUserDataDir),
		},
		fanqieContentAPIURL:     strings.TrimSpace(fanqieContentAPIURL),
		fanqieContentAPITimeout: time.Duration(fanqieContentAPITimeoutSeconds) * time.Second,
		activeScanJobs:          make(map[uint]struct{}),
	}
}

type RadarOverview struct {
	Taxonomies   []model.RadarTaxonomy        `json:"taxonomies"`
	Tags         []model.RadarTag             `json:"tags"`
	Jobs         []model.RadarScanJob         `json:"jobs"`
	Sources      []model.RadarSource          `json:"sources"`
	BookProfiles []model.RadarBookProfile     `json:"book_profiles"`
	IntroSamples []model.RadarIntroSample     `json:"intro_samples"`
	Profiles     []model.RadarTaxonomyProfile `json:"profiles"`
	Rules        []model.RadarRule            `json:"rules"`
}

func (s *RadarService) Overview() (*RadarOverview, error) {
	taxonomies, err := s.repo.ListTaxonomies(model.RadarPlatformFanqie)
	if err != nil {
		return nil, err
	}
	tags, err := s.repo.ListTags(model.RadarPlatformFanqie, "")
	if err != nil {
		return nil, err
	}
	jobs, err := s.repo.ListScanJobs(20)
	if err != nil {
		return nil, err
	}
	sources, err := s.repo.ListSources(50)
	if err != nil {
		return nil, err
	}
	bookProfiles, err := s.repo.ListBookProfiles(model.RadarPlatformFanqie, "", 100)
	if err != nil {
		return nil, err
	}
	introSamples, err := s.repo.ListIntroSamples(model.RadarPlatformFanqie, "", 100)
	if err != nil {
		return nil, err
	}
	introSamples = cleanRadarIntroSamples(introSamples)
	profiles, err := s.repo.ListTaxonomyProfiles(model.RadarPlatformFanqie, "")
	if err != nil {
		return nil, err
	}
	rules, err := s.repo.ListRules(model.RadarPlatformFanqie, "", 100)
	if err != nil {
		return nil, err
	}
	return &RadarOverview{
		Taxonomies:   taxonomies,
		Tags:         tags,
		Jobs:         jobs,
		Sources:      sources,
		BookProfiles: bookProfiles,
		IntroSamples: introSamples,
		Profiles:     profiles,
		Rules:        rules,
	}, nil
}

func (s *RadarService) WritableTags() ([]model.RadarTag, error) {
	tags, err := s.repo.ListTags(model.RadarPlatformFanqie, "")
	if err != nil {
		return nil, err
	}
	profiles, err := s.repo.ListTaxonomyProfiles(model.RadarPlatformFanqie, "")
	if err != nil {
		return nil, err
	}
	rules, err := s.repo.ListRules(model.RadarPlatformFanqie, "", 0)
	if err != nil {
		return nil, err
	}

	profileReady := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		key := strings.TrimSpace(profile.Category)
		if key != "" && profile.IsActive {
			profileReady[key] = struct{}{}
		}
	}
	ruleReady := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		key := strings.TrimSpace(rule.Category)
		if key != "" && rule.IsActive {
			ruleReady[key] = struct{}{}
		}
	}

	ready := make([]model.RadarTag, 0, len(tags))
	for _, tag := range tags {
		if _, ok := profileReady[tag.TagKey]; !ok {
			continue
		}
		if _, ok := ruleReady[tag.TagKey]; !ok {
			continue
		}
		ready = append(ready, tag)
	}
	return ready, nil
}

func (s *RadarService) CheckBrowserSession(ctx context.Context) *BrowserSessionStatus {
	status := checkFanqieBrowserSession(ctx, s.browserCDPURL, 30*time.Second, s.browserLaunch)
	s.cacheFanqieCookieHeader(status.CookieHeader)
	return status
}

func (s *RadarService) OpenBrowserLoginPage(ctx context.Context) *BrowserSessionStatus {
	return openFanqieBrowserPage(ctx, s.browserCDPURL, s.browserLaunch)
}

func (s *RadarService) cacheFanqieCookieHeader(cookieHeader string) {
	cookieHeader = strings.TrimSpace(cookieHeader)
	if cookieHeader == "" {
		return
	}
	s.fanqieCookieMu.Lock()
	defer s.fanqieCookieMu.Unlock()
	s.fanqieCookieHeader = cookieHeader
}

func (s *RadarService) cachedFanqieCookieHeader() string {
	s.fanqieCookieMu.RLock()
	defer s.fanqieCookieMu.RUnlock()
	return s.fanqieCookieHeader
}

func (s *RadarService) refreshFanqieCookieHeader(ctx context.Context) (string, error) {
	status := checkFanqieBrowserSession(ctx, s.browserCDPURL, 30*time.Second, s.browserLaunch)
	if !status.Ready {
		return "", fmt.Errorf("%s", status.Message)
	}
	if strings.TrimSpace(status.CookieHeader) == "" {
		return "", fmt.Errorf("未从本地 Chrome 获取到番茄 Cookie")
	}
	s.cacheFanqieCookieHeader(status.CookieHeader)
	return status.CookieHeader, nil
}

func (s *RadarService) CreateManualSource(ctx context.Context, req model.CreateRadarSourceRequest) (*model.RadarSource, error) {
	platform := defaultStringRadar(req.Platform, model.RadarPlatformFanqie)
	bookID := strings.TrimSpace(req.SourceBookID)
	if bookID == "" {
		bookID = extractFanqieBookID(req.BookURL)
	}
	if bookID == "" {
		return nil, fmt.Errorf("请填写番茄 book_id 或书籍 URL")
	}

	source := &model.RadarSource{
		Platform:     platform,
		SourceBookID: bookID,
		BookURL:      defaultStringRadar(req.BookURL, fanqieBookURL(bookID)),
		Title:        strings.TrimSpace(req.Title),
		Author:       strings.TrimSpace(req.Author),
		Category:     strings.TrimSpace(req.Category),
		Intro:        cleanFanqieIntro(req.Intro, req.Title),
		Status:       "active",
		ScanJobID:    req.ScanJobID,
		Confidence:   1,
	}

	var fetchedChapters []fanqieChapter
	var fetchedTagNames []string
	fetched, err := s.fetchFanqieBook(ctx, bookID, radarChapterSampleLimit)
	if err == nil {
		if source.Title == "" {
			source.Title = fetched.Title
		}
		if source.Author == "" {
			source.Author = fetched.Author
		}
		if source.Intro == "" {
			source.Intro = fetched.Intro
		}
		source.ChapterCount = len(fetched.Chapters)
		source.WordCount = int64(totalSampleWords(fetched.Chapters))
		source.ContentHash = hashText(fetched.Title + fetched.Intro)
		fetchedChapters = fetched.Chapters
		fetchedTagNames = fetched.Tags
	}
	if strings.TrimSpace(req.SampleText) == "" {
		if err != nil {
			return nil, err
		}
		if len(fetchedChapters) == 0 {
			if fetched != nil && fetched.ChapterFetchError != nil {
				return nil, fetched.ChapterFetchError
			}
			return nil, fmt.Errorf("未抓取到真实章节正文样本")
		}
	}

	tags := append([]string(nil), req.Tags...)
	if len(tags) == 0 && len(fetchedTagNames) > 0 {
		tags = s.tagKeysByNames(platform, fetchedTagNames)
	}
	if source.Category != "" {
		tags = appendUniqueStrings([]string{source.Category}, tags...)
	}
	tags = uniqueNonEmptyStrings(tags)
	if source.Category == "" && len(tags) > 0 {
		source.Category = tags[0]
	}
	if source.Category == "" {
		source.Category = "other_pending"
	}
	source.TagsJSON = marshalStringSlice(tags)

	if source.Title == "" {
		source.Title = "番茄书籍 " + bookID
	}
	if source.ContentHash == "" {
		source.ContentHash = hashText(source.Title + source.Intro + req.SampleText)
	}
	if err := s.repo.SaveSource(source); err != nil {
		return nil, err
	}
	saved, err := s.repo.FindSourceByBookID(platform, bookID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return source, nil
	}
	source = saved
	if len(fetchedChapters) > 0 {
		if err := s.repo.SaveChapterSamples(toChapterSamples(source.ID, fetchedChapters)); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(req.SampleText) != "" {
		sample := model.RadarChapterSample{
			SourceID:       source.ID,
			ChapterNo:      1,
			Title:          "人工样本",
			Content:        strings.TrimSpace(req.SampleText),
			WordCount:      len([]rune(strings.TrimSpace(req.SampleText))),
			ParagraphCount: countParagraphs(req.SampleText),
			DialogueRatio:  estimateDialogueRatio(req.SampleText),
			ContentHash:    hashText(req.SampleText),
		}
		if err := s.repo.SaveChapterSamples([]model.RadarChapterSample{sample}); err != nil {
			return nil, err
		}
	}
	return source, nil
}

func (s *RadarService) RefreshSourceChapters(ctx context.Context, sourceID uint) ([]model.RadarChapterSample, error) {
	source, err := s.repo.GetSource(sourceID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("雷达书籍不存在")
	}
	bookID := strings.TrimSpace(source.SourceBookID)
	if bookID == "" {
		bookID = extractFanqieBookID(source.BookURL)
	}
	if bookID == "" {
		return nil, fmt.Errorf("雷达书籍缺少番茄 book_id")
	}
	fetched, err := s.fetchFanqieBook(ctx, bookID, radarChapterSampleLimit)
	if err != nil {
		return nil, err
	}
	if len(fetched.Chapters) == 0 {
		if fetched.ChapterFetchError != nil {
			return nil, fetched.ChapterFetchError
		}
		return nil, fmt.Errorf("未抓取到章节正文样本")
	}
	if strings.TrimSpace(fetched.Title) != "" {
		source.Title = fetched.Title
	}
	if strings.TrimSpace(fetched.Author) != "" {
		source.Author = fetched.Author
	}
	if strings.TrimSpace(fetched.Intro) != "" {
		source.Intro = fetched.Intro
	}
	source.ChapterCount = len(fetched.Chapters)
	source.WordCount = int64(totalSampleWords(fetched.Chapters))
	source.ContentHash = hashText(source.Title + source.Intro)
	if err := s.repo.SaveSource(source); err != nil {
		return nil, err
	}
	if err := s.repo.SaveChapterSamples(toChapterSamples(source.ID, fetched.Chapters)); err != nil {
		return nil, err
	}
	return s.repo.GetChapterSamples(source.ID, radarChapterSampleLimit)
}

func (s *RadarService) AnalyzeSource(ctx context.Context, sourceID uint, requestedModelID uint) (*model.RadarBookProfile, error) {
	source, err := s.repo.GetSource(sourceID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("雷达书籍不存在")
	}
	samples, err := s.repo.GetChapterSamples(source.ID, radarChapterSampleLimit)
	if err != nil {
		return nil, err
	}
	if len(samples) == 0 {
		samples, err = s.RefreshSourceChapters(ctx, source.ID)
		if err != nil {
			return nil, err
		}
		if len(samples) == 0 {
			return nil, fmt.Errorf("章节样本为空：请先确认本地 Chrome 已通过番茄验证码，并开放 DevTools 端口")
		}
		source, _ = s.repo.GetSource(source.ID)
	}
	modelID, err := s.modelIDOrDefault(requestedModelID)
	if err != nil {
		return nil, err
	}

	analysis, err := s.analyze(ctx, modelID, source, samples)
	if err != nil {
		return nil, err
	}
	version := source.ProfileVersion + 1
	if version <= 0 {
		version = 1
	}
	profile := &model.RadarBookProfile{
		SourceID:        source.ID,
		Platform:        source.Platform,
		Category:        source.Category,
		TagsJSON:        source.TagsJSON,
		ProfileJSON:     analysis.ProfileJSON,
		ProfileMarkdown: analysis.ProfileMarkdown,
		SampleChapters:  len(samples),
		Confidence:      analysis.Confidence,
		Version:         version,
	}
	if err := s.repo.SaveBookProfile(profile); err != nil {
		return nil, err
	}
	source.ProfileVersion = version
	_ = s.repo.SaveSource(source)
	return profile, nil
}

func (s *RadarService) ListSourceChapterSamples(sourceID uint) ([]model.RadarChapterSample, error) {
	source, err := s.repo.GetSource(sourceID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("雷达书籍不存在")
	}
	return s.repo.GetChapterSamples(sourceID, radarChapterSampleLimit)
}

func (s *RadarService) fetchFanqieBook(ctx context.Context, bookID string, maxChapters int) (*fanqieBook, error) {
	cookieHeader := s.cachedFanqieCookieHeader()
	if cookieHeader == "" && s.browserCDPURL != "" {
		cookieHeader, _ = s.refreshFanqieCookieHeader(ctx)
	}
	book, err := fetchFanqieBookWithCookie(ctx, bookID, maxChapters, cookieHeader)
	if err != nil {
		return nil, err
	}
	if len(book.Chapters) > 0 || len(book.ChapterIDs) == 0 {
		return book, nil
	}
	if s.fanqieContentAPIURL != "" {
		chapterIDs := book.ChapterIDs
		if maxChapters > 0 && len(chapterIDs) > maxChapters {
			chapterIDs = chapterIDs[:maxChapters]
		}
		if chapters, apiErr := fetchFanqieChaptersWithContentAPI(ctx, s.fanqieContentAPIURL, chapterIDs, s.fanqieContentAPITimeout); apiErr == nil && len(chapters) > 0 {
			book.Chapters = chapters
			return book, nil
		} else if apiErr != nil {
			book.ChapterFetchError = apiErr
		}
	}
	if s.browserCDPURL != "" {
		if refreshedCookie, refreshErr := s.refreshFanqieCookieHeader(ctx); refreshErr == nil && refreshedCookie != "" && refreshedCookie != cookieHeader {
			if retried, retryErr := fetchFanqieBookWithCookie(ctx, bookID, maxChapters, refreshedCookie); retryErr == nil {
				book = retried
			}
		}
		if len(book.Chapters) == 0 && len(book.ChapterIDs) > 0 {
			chapterIDs := book.ChapterIDs
			if maxChapters > 0 && len(chapterIDs) > maxChapters {
				chapterIDs = chapterIDs[:maxChapters]
			}
			if chapters, browserErr := fetchFanqieChaptersWithBrowser(ctx, s.browserCDPURL, chapterIDs, s.browserChapterFetchTimeout, s.browserLaunch); browserErr == nil {
				book.Chapters = chapters
			} else {
				book.ChapterFetchError = browserErr
			}
		}
	}
	if len(book.Chapters) == 0 && len(book.ChapterIDs) > 0 {
		if book.ChapterFetchError == nil {
			book.ChapterFetchError = fmt.Errorf("未能用番茄登录 Cookie 或后台 Chrome 抓取章节正文，请确认专用 Chrome 中 reader 页面可阅读全文")
		}
	}
	return book, nil
}

func (s *RadarService) CreateCategoryScanJob(ctx context.Context, req model.CreateRadarScanJobRequest) (*model.RadarScanJob, error) {
	tagKey := strings.TrimSpace(req.TagKey)
	if tagKey == "" {
		tagKey = strings.TrimSpace(req.Category)
	}
	if tagKey == "" {
		return nil, fmt.Errorf("请选择番茄标签")
	}
	job := &model.RadarScanJob{
		Platform:    defaultStringRadar(req.Platform, model.RadarPlatformFanqie),
		Category:    tagKey,
		LLMModelID:  req.LLMModelID,
		Mode:        model.RadarJobCategoryAuto,
		Status:      model.RadarJobQueued,
		TargetCount: defaultInt(req.TargetCount, 5),
	}
	if err := s.repo.CreateScanJob(job); err != nil {
		return nil, err
	}
	s.registerActiveScanJob(job.ID)
	go s.runCategoryScan(context.Background(), job.ID)
	return job, nil
}

func (s *RadarService) runCategoryScan(ctx context.Context, jobID uint) {
	defer s.unregisterActiveScanJob(jobID)
	job, err := s.repo.GetScanJob(jobID)
	if err != nil || job == nil {
		return
	}
	now := time.Now()
	job.Status = model.RadarJobRunning
	job.StartedAt = &now
	_ = s.repo.SaveScanJob(job)

	fail := func(err error) {
		done := time.Now()
		job.Status = model.RadarJobFailed
		job.ErrorMessage = err.Error()
		job.FinishedAt = &done
		_ = s.repo.SaveScanJob(job)
	}

	existingIDs := map[string]struct{}{}
	existingSources, _ := s.repo.ListSources(0)
	for _, source := range existingSources {
		if source.Platform == job.Platform && strings.TrimSpace(source.SourceBookID) != "" {
			existingIDs[source.SourceBookID] = struct{}{}
		}
	}

	bookIDs, err := discoverFanqieBooks(ctx, job.Category, job.TargetCount, existingIDs)
	if err != nil {
		fail(err)
		return
	}
	failures := make([]string, 0)
	for _, bookID := range bookIDs {
		req := model.CreateRadarSourceRequest{
			Platform:     job.Platform,
			SourceBookID: bookID,
			Category:     job.Category,
			ScanJobID:    job.ID,
		}
		if _, err := s.CreateManualSource(ctx, req); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", bookID, err.Error()))
			if len(failures) > 3 {
				failures = failures[:3]
			}
			job.ErrorMessage = strings.Join(failures, "\n")
			_ = s.repo.SaveScanJob(job)
			continue
		}
		job.ScannedCount++
		_ = s.repo.SaveScanJob(job)
	}
	done := time.Now()
	if job.ScannedCount == 0 && len(failures) > 0 {
		job.Status = model.RadarJobFailed
		job.ErrorMessage = strings.Join(failures, "\n")
		job.FinishedAt = &done
		_ = s.repo.SaveScanJob(job)
		return
	}
	job.Status = model.RadarJobSucceeded
	job.FinishedAt = &done
	_ = s.repo.SaveScanJob(job)
}

func (s *RadarService) AnalyzeCategory(ctx context.Context, platform, category string, requestedModelID uint) (int, error) {
	platform = defaultStringRadar(platform, model.RadarPlatformFanqie)
	category = strings.TrimSpace(category)
	if category == "" {
		return 0, fmt.Errorf("请选择番茄标签")
	}
	var sources []model.RadarSource
	var err error
	if category == radarAllTagsCategory {
		sources, err = s.repo.ListSources(0)
	} else {
		sources, err = s.repo.ListSourcesByCategory(platform, category, 0)
	}
	if err != nil {
		return 0, err
	}
	if len(sources) == 0 {
		return 0, fmt.Errorf("当前标签暂无书籍样本")
	}
	count := 0
	for _, source := range sources {
		if source.Platform != platform {
			continue
		}
		if source.ProfileVersion > 0 {
			continue
		}
		if _, err := s.AnalyzeSource(ctx, source.ID, requestedModelID); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *RadarService) Synthesize(ctx context.Context, platform, category, tagKey string, requestedModelID uint) error {
	category = strings.TrimSpace(category)
	if category == "" {
		return fmt.Errorf("请选择番茄标签")
	}
	modelID, err := s.modelIDOrDefault(requestedModelID)
	if err != nil {
		return err
	}
	profiles, err := s.repo.ListBookProfiles(platform, category, 30)
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		return fmt.Errorf("当前标签暂无单书画像")
	}
	inputs := make([]string, 0, len(profiles))
	sampleChapters := 0
	for _, profile := range profiles {
		inputs = append(inputs, profile.ProfileMarkdown)
		sampleChapters += profile.SampleChapters
	}
	if len(inputs) == 0 {
		return fmt.Errorf("当前标签暂无单书画像")
	}
	raw, err := s.llm.ChatForAgent(ctx, "radar_synthesizer", modelID, radarSynthesizerSystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: buildSynthesisPrompt(platform, category, inputs)},
	}, 0.2)
	if err != nil {
		return err
	}
	parsed, err := parseSynthesisOutput(raw)
	if err != nil {
		return err
	}
	version := 1
	existing, _ := s.repo.ListTaxonomyProfiles(platform, category)
	for _, item := range existing {
		if item.TagKey == "" && item.Version >= version {
			version = item.Version + 1
		}
	}
	profile := &model.RadarTaxonomyProfile{
		Platform:           platform,
		Category:           category,
		TagKey:             "",
		ProfileJSON:        parsed.ProfileJSON,
		ProfileMarkdown:    parsed.ProfileMarkdown,
		ProfileSummary:     parsed.ProfileSummary,
		WriterBrief:        parsed.WriterBrief,
		PlannerBrief:       parsed.PlannerBrief,
		AuditorBrief:       parsed.AuditorBrief,
		SourceCount:        len(inputs),
		SampleChapterCount: sampleChapters,
		Confidence:         parsed.Confidence,
		Version:            version,
		IsActive:           true,
	}
	if err := s.repo.SaveTaxonomyProfile(profile); err != nil {
		return err
	}
	rules := make([]model.RadarRule, 0, len(parsed.Rules))
	for _, rule := range parsed.Rules {
		rules = append(rules, model.RadarRule{
			Platform:        platform,
			Category:        category,
			TagKey:          "",
			RuleType:        rule.RuleType,
			Title:           rule.Title,
			Content:         rule.Content,
			EvidenceSummary: rule.EvidenceSummary,
			Confidence:      rule.Confidence,
			Weight:          rule.Weight,
			IsActive:        true,
		})
	}
	return s.repo.ReplaceRules(platform, category, "", rules)
}

type IntroGenerationOutput struct {
	Title         string   `json:"title"`
	Intro         string   `json:"intro"`
	SellingPoints []string `json:"selling_points"`
}

func (s *RadarService) ScanIntroSamples(ctx context.Context, req model.CreateRadarIntroScanRequest) (int, error) {
	platform := defaultStringRadar(req.Platform, model.RadarPlatformFanqie)
	tagKey := strings.TrimSpace(req.TagKey)
	if tagKey == "" {
		tagKey = strings.TrimSpace(req.Category)
	}
	if tagKey == "" {
		return 0, fmt.Errorf("请选择番茄标签")
	}
	target := defaultInt(req.TargetCount, 10)

	exclude := map[string]struct{}{}
	existing, _ := s.repo.ListIntroSamples(platform, "", 0)
	for _, sample := range existing {
		if strings.TrimSpace(sample.SourceBookID) != "" {
			exclude[sample.SourceBookID] = struct{}{}
		}
	}
	bookIDs, err := discoverFanqieBooks(ctx, tagKey, target, exclude)
	if err != nil {
		return 0, err
	}

	samples := make([]model.RadarIntroSample, 0, len(bookIDs))
	for _, bookID := range bookIDs {
		fetched, err := fetchFanqieBookIntro(ctx, bookID)
		if err != nil || fetched == nil || strings.TrimSpace(fetched.Intro) == "" {
			continue
		}
		intro := cleanFanqieIntro(fetched.Intro, fetched.Title)
		if intro == "" {
			continue
		}
		tags := s.tagKeysByNames(platform, fetched.Tags)
		tags = appendUniqueStrings([]string{tagKey}, tags...)
		samples = append(samples, model.RadarIntroSample{
			Platform:     platform,
			SourceBookID: bookID,
			BookURL:      fanqieBookURL(bookID),
			Title:        defaultStringRadar(fetched.Title, "番茄书籍 "+bookID),
			Author:       fetched.Author,
			Category:     tagKey,
			TagsJSON:     marshalStringSlice(tags),
			Intro:        intro,
			WordCount:    len([]rune(intro)),
			ContentHash:  hashText(fetched.Title + intro),
		})
	}
	if len(samples) == 0 {
		return 0, fmt.Errorf("未获取到可用简介样本")
	}
	if err := s.repo.SaveIntroSamples(samples); err != nil {
		return 0, err
	}
	return len(samples), nil
}

func (s *RadarService) GenerateIntro(ctx context.Context, req model.GenerateRadarIntroRequest) (*IntroGenerationOutput, error) {
	platform := defaultStringRadar(req.Platform, model.RadarPlatformFanqie)
	tagKeys := generateIntroTagKeys(req)
	if len(tagKeys) == 0 {
		return nil, fmt.Errorf("请选择番茄标签")
	}
	samples, err := s.listIntroSamplesByTags(platform, tagKeys, 60)
	if err != nil {
		return nil, err
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("当前标签暂无简介样本")
	}
	samples = cleanRadarIntroSamples(samples)
	modelID, err := s.modelIDOrDefault(req.ModelID)
	if err != nil {
		return nil, err
	}
	tagLabels := s.tagLabelsByKeys(platform, tagKeys)
	raw, err := s.llm.ChatForAgent(ctx, "radar_intro_generator", modelID, agent.NewRadarIntroGeneratorAgent().SystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: buildIntroGenerationPrompt(tagLabels, strings.TrimSpace(req.Requirement), samples, 16000)},
	}, 0.7)
	if err != nil {
		return nil, err
	}
	var parsed IntroGenerationOutput
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &parsed); err != nil {
		return nil, err
	}
	parsed.Title = strings.TrimSpace(parsed.Title)
	parsed.Intro = strings.TrimSpace(parsed.Intro)
	if parsed.Title == "" || parsed.Intro == "" {
		return nil, fmt.Errorf("简介生成结果缺少书名或简介")
	}
	return &parsed, nil
}

func (s *RadarService) listIntroSamplesByTags(platform string, tagKeys []string, limit int) ([]model.RadarIntroSample, error) {
	seen := make(map[uint]struct{})
	out := make([]model.RadarIntroSample, 0, limit)
	for _, tagKey := range tagKeys {
		rows, err := s.repo.ListIntroSamples(platform, tagKey, 40)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := seen[row.ID]; ok {
				continue
			}
			seen[row.ID] = struct{}{}
			out = append(out, row)
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (s *RadarService) tagLabelsByKeys(platform string, keys []string) []string {
	tags, err := s.repo.ListTags(platform, "")
	if err != nil {
		return keys
	}
	byKey := make(map[string]string, len(tags))
	for _, tag := range tags {
		byKey[tag.TagKey] = tag.TagName
	}
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		if label := byKey[key]; label != "" {
			labels = append(labels, label)
			continue
		}
		labels = append(labels, key)
	}
	return labels
}

func (s *RadarService) registerActiveScanJob(jobID uint) {
	s.activeScanMu.Lock()
	defer s.activeScanMu.Unlock()
	s.activeScanJobs[jobID] = struct{}{}
}

func (s *RadarService) unregisterActiveScanJob(jobID uint) {
	s.activeScanMu.Lock()
	defer s.activeScanMu.Unlock()
	delete(s.activeScanJobs, jobID)
}

func (s *RadarService) isActiveScanJob(jobID uint) bool {
	s.activeScanMu.RLock()
	defer s.activeScanMu.RUnlock()
	_, ok := s.activeScanJobs[jobID]
	return ok
}

func (s *RadarService) DeleteScanJob(jobID uint) error {
	job, err := s.repo.GetScanJob(jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("扫描任务不存在")
	}
	if (job.Status == model.RadarJobQueued || job.Status == model.RadarJobRunning) && s.isActiveScanJob(jobID) {
		return fmt.Errorf("扫描任务仍在运行，暂不能删除")
	}
	return s.repo.DeleteScanJob(jobID)
}

func (s *RadarService) DeleteSource(sourceID uint) error {
	source, err := s.repo.GetSource(sourceID)
	if err != nil {
		return err
	}
	if source == nil {
		return fmt.Errorf("书籍样本不存在")
	}
	return s.repo.DeleteSourceCascade(sourceID)
}

func (s *RadarService) DeleteSources(sourceIDs []uint) (int, error) {
	sourceIDs = uniqueUintIDs(sourceIDs)
	if len(sourceIDs) == 0 {
		return 0, fmt.Errorf("请选择要删除的书籍样本")
	}
	if err := s.repo.DeleteSourcesCascade(sourceIDs); err != nil {
		return 0, err
	}
	return len(sourceIDs), nil
}

func (s *RadarService) DeleteTaxonomyProfile(profileID uint) error {
	return s.repo.DeleteTaxonomyProfile(profileID)
}

func (s *RadarService) DeleteRule(ruleID uint) error {
	return s.repo.DeleteRule(ruleID)
}

func (s *RadarService) DeleteTaxonomyProfilesByCategories(platform string, categories []string) (int, error) {
	platform = defaultStringRadar(platform, model.RadarPlatformFanqie)
	categories = uniqueNonEmptyStrings(categories)
	if len(categories) == 0 {
		return 0, fmt.Errorf("请选择要删除的画像标签")
	}
	if err := s.repo.DeleteTaxonomyProfilesByCategories(platform, categories); err != nil {
		return 0, err
	}
	return len(categories), nil
}

func (s *RadarService) DeleteRulesByCategories(platform string, categories []string) (int, error) {
	platform = defaultStringRadar(platform, model.RadarPlatformFanqie)
	categories = uniqueNonEmptyStrings(categories)
	if len(categories) == 0 {
		return 0, fmt.Errorf("请选择要删除的规则标签")
	}
	if err := s.repo.DeleteRulesByCategories(platform, categories); err != nil {
		return 0, err
	}
	return len(categories), nil
}

func (s *RadarService) DeleteIntroSamples(ids []uint) (int, error) {
	ids = uniqueUintIDs(ids)
	if len(ids) == 0 {
		return 0, fmt.Errorf("请选择要删除的简介样本")
	}
	if err := s.repo.DeleteIntroSamples(ids); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (s *RadarService) modelIDOrDefault(modelID uint) (uint, error) {
	if modelID > 0 {
		return modelID, nil
	}
	return s.defaultModelID()
}

func (s *RadarService) defaultModelID() (uint, error) {
	m, err := s.llm.GetDefaultModel()
	if err != nil {
		return 0, err
	}
	return m.ID, nil
}

type analysisOutput struct {
	ProfileJSON     string
	ProfileMarkdown string
	Confidence      float64
}

func (s *RadarService) analyze(ctx context.Context, modelID uint, source *model.RadarSource, samples []model.RadarChapterSample) (*analysisOutput, error) {
	raw, err := s.llm.ChatForAgent(ctx, "radar_analyzer", modelID, radarAnalyzerSystemPrompt(), []llm.AgentMessage{
		{Role: "user", Content: buildBookSamplePrompt(source, samples, 16000)},
	}, 0.2)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Profile         map[string]any `json:"profile"`
		ProfileMarkdown string         `json:"profile_markdown"`
		Confidence      float64        `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &parsed); err != nil {
		return nil, err
	}
	profileJSON, _ := json.Marshal(parsed.Profile)
	return &analysisOutput{
		ProfileJSON:     string(profileJSON),
		ProfileMarkdown: parsed.ProfileMarkdown,
		Confidence:      parsed.Confidence,
	}, nil
}

type synthesisOutput struct {
	ProfileJSON     string
	ProfileMarkdown string
	ProfileSummary  string
	WriterBrief     string
	PlannerBrief    string
	AuditorBrief    string
	Confidence      float64
	Rules           []struct {
		RuleType        string  `json:"rule_type"`
		Title           string  `json:"title"`
		Content         string  `json:"content"`
		EvidenceSummary string  `json:"evidence_summary"`
		Confidence      float64 `json:"confidence"`
		Weight          int     `json:"weight"`
	}
}

func parseSynthesisOutput(raw string) (*synthesisOutput, error) {
	var parsed struct {
		TaxonomyProfile struct {
			Profile         map[string]any `json:"profile"`
			ProfileMarkdown string         `json:"profile_markdown"`
			ProfileSummary  string         `json:"profile_summary"`
			WriterBrief     string         `json:"writer_brief"`
			PlannerBrief    string         `json:"planner_brief"`
			AuditorBrief    string         `json:"auditor_brief"`
			Confidence      float64        `json:"confidence"`
		} `json:"taxonomy_profile"`
		Rules []struct {
			RuleType        string  `json:"rule_type"`
			Title           string  `json:"title"`
			Content         string  `json:"content"`
			EvidenceSummary string  `json:"evidence_summary"`
			Confidence      float64 `json:"confidence"`
			Weight          int     `json:"weight"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &parsed); err != nil {
		return nil, err
	}
	profileJSON, _ := json.Marshal(parsed.TaxonomyProfile.Profile)
	return &synthesisOutput{
		ProfileJSON:     string(profileJSON),
		ProfileMarkdown: parsed.TaxonomyProfile.ProfileMarkdown,
		ProfileSummary:  parsed.TaxonomyProfile.ProfileSummary,
		WriterBrief:     parsed.TaxonomyProfile.WriterBrief,
		PlannerBrief:    parsed.TaxonomyProfile.PlannerBrief,
		AuditorBrief:    parsed.TaxonomyProfile.AuditorBrief,
		Confidence:      parsed.TaxonomyProfile.Confidence,
		Rules:           parsed.Rules,
	}, nil
}

func radarAnalyzerSystemPrompt() string {
	return agent.NewRadarAnalyzerAgent().SystemPrompt()
}
func radarSynthesizerSystemPrompt() string {
	return agent.NewRadarSynthesizerAgent().SystemPrompt()
}

func buildBookSamplePrompt(source *model.RadarSource, samples []model.RadarChapterSample, limit int) string {
	var b strings.Builder
	b.WriteString("## 书籍信息\n")
	b.WriteString("标题：" + source.Title + "\n")
	b.WriteString("作者：" + source.Author + "\n")
	b.WriteString("简介：" + source.Intro + "\n")
	b.WriteString("分类：" + source.Category + "\n")
	b.WriteString("标签：" + source.TagsJSON + "\n\n")
	b.WriteString("## 章节样本\n")
	total := 0
	for _, sample := range samples {
		chunk := fmt.Sprintf("\n### 第%d章 %s\n%s\n", sample.ChapterNo, sample.Title, sample.Content)
		if total+len([]rune(chunk)) > limit {
			break
		}
		b.WriteString(chunk)
		total += len([]rune(chunk))
	}
	return b.String()
}

func buildSynthesisPrompt(platform, tagKey string, profiles []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("平台：%s\n番茄标签：%s\n\n", platform, tagKey))
	b.WriteString("## 单书画像\n")
	for i, profile := range profiles {
		b.WriteString(fmt.Sprintf("\n### 样本 %d\n%s\n", i+1, clipRunesRadar(profile, 5000)))
	}
	return b.String()
}

func buildIntroGenerationPrompt(tagLabels []string, requirement string, samples []model.RadarIntroSample, limit int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("番茄标签：%s\n\n", strings.Join(tagLabels, "、")))
	if strings.TrimSpace(requirement) != "" {
		b.WriteString("## 用户要求\n")
		b.WriteString(strings.TrimSpace(requirement))
		b.WriteString("\n\n")
	}
	b.WriteString("## 同标签真实简介样本\n")
	total := 0
	for i, sample := range samples {
		chunk := fmt.Sprintf("\n### 样本 %d\n书名：%s\n作者：%s\n标签：%s\n简介：%s\n", i+1, sample.Title, sample.Author, sample.TagsJSON, sample.Intro)
		if total+len([]rune(chunk)) > limit {
			break
		}
		b.WriteString(chunk)
		total += len([]rune(chunk))
	}
	return b.String()
}

type fanqieChapter struct {
	No      int
	Title   string
	Content string
}

type fanqieBook struct {
	Title             string
	Author            string
	Intro             string
	Tags              []string
	ChapterIDs        []string
	Chapters          []fanqieChapter
	ChapterFetchError error
}

func fetchFanqieBook(ctx context.Context, bookID string, maxChapters int) (*fanqieBook, error) {
	return fetchFanqieBookWithCookie(ctx, bookID, maxChapters, "")
}

func fetchFanqieBookWithCookie(ctx context.Context, bookID string, maxChapters int, cookieHeader string) (*fanqieBook, error) {
	pageURL := fanqieBookURL(bookID)
	body, err := fetchTextWithCookie(ctx, pageURL, cookieHeader)
	if err != nil {
		return nil, err
	}
	book := &fanqieBook{
		Title:  firstRegex(body, `<h1[^>]*>([^<]+)</h1>`, `"bookName"\s*:\s*"([^"]+)"`, `"book_name"\s*:\s*"([^"]+)"`),
		Author: firstRegex(body, `"author"\s*:\s*"([^"]+)"`, `"authorName"\s*:\s*"([^"]+)"`),
		Intro:  extractFanqieIntro(body),
		Tags:   extractFanqieBookTags(body),
	}
	book.Intro = cleanFanqieIntro(book.Intro, book.Title)
	if book.Title == "" {
		book.Title = "番茄书籍 " + bookID
	}
	chapterIDs := extractFanqieDirectoryChapterIDs(body)
	if len(chapterIDs) == 0 {
		chapterIDs = uniqueRegex(body, `/reader/(\d+)`)
	}
	book.ChapterIDs = chapterIDs
	if len(chapterIDs) == 0 {
		return book, nil
	}
	if maxChapters <= 0 || maxChapters > len(chapterIDs) {
		maxChapters = len(chapterIDs)
	}
	for i := 0; i < maxChapters; i++ {
		chapter, err := fetchFanqieChapterWithCookie(ctx, chapterIDs[i], i+1, cookieHeader)
		if err == nil && strings.TrimSpace(chapter.Content) != "" {
			book.Chapters = append(book.Chapters, *chapter)
		}
	}
	return book, nil
}

func fetchFanqieBookIntro(ctx context.Context, bookID string) (*fanqieBook, error) {
	pageURL := fanqieBookURL(bookID)
	body, err := fetchText(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	book := &fanqieBook{
		Title:  firstRegex(body, `<h1[^>]*>([^<]+)</h1>`, `"bookName"\s*:\s*"([^"]+)"`, `"book_name"\s*:\s*"([^"]+)"`),
		Author: firstRegex(body, `"author"\s*:\s*"([^"]+)"`, `"authorName"\s*:\s*"([^"]+)"`),
		Intro:  extractFanqieIntro(body),
		Tags:   extractFanqieBookTags(body),
	}
	book.Intro = cleanFanqieIntro(book.Intro, book.Title)
	if book.Title == "" {
		book.Title = "番茄书籍 " + bookID
	}
	return book, nil
}

func fetchFanqieChapter(ctx context.Context, chapterID string, no int) (*fanqieChapter, error) {
	return fetchFanqieChapterWithCookie(ctx, chapterID, no, "")
}

func fetchFanqieChapterWithCookie(ctx context.Context, chapterID string, no int, cookieHeader string) (*fanqieChapter, error) {
	body, err := fetchTextWithCookie(ctx, "https://fanqienovel.com/reader/"+chapterID, cookieHeader)
	if err != nil {
		return nil, err
	}
	title := firstRegex(body, `<h1[^>]*>([^<]+)</h1>`, `"title"\s*:\s*"([^"]+)"`)
	content := firstRegex(body, `<div[^>]+class="[^"]*muye-reader-content[^"]*"[^>]*>([\s\S]*?)</div>`)
	if content == "" {
		content = firstRegexJSON(body, `"content"\s*:\s*"((?:\\.|[^"\\])*)"`)
	}
	content = normalizeSampleText(stripHTML(content))
	if !isUsableChapterContent(content) {
		return nil, fmt.Errorf("章节正文不可用，疑似只抓到元信息或登录提示")
	}
	return &fanqieChapter{No: no, Title: html.UnescapeString(title), Content: content}, nil
}

func fetchFanqieChaptersWithContentAPI(ctx context.Context, apiURL string, chapterIDs []string, timeout time.Duration) ([]fanqieChapter, error) {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return nil, fmt.Errorf("未配置番茄第三方正文接口")
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	chapters := make([]fanqieChapter, 0, len(chapterIDs))
	var failures []string
	for i, chapterID := range chapterIDs {
		chapter, err := fetchFanqieChapterWithContentAPI(ctx, apiURL, chapterID, i+1, timeout)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", chapterID, err.Error()))
			continue
		}
		chapters = append(chapters, *chapter)
	}
	if len(chapters) == 0 {
		if len(failures) > 0 {
			if len(failures) > 3 {
				failures = failures[:3]
			}
			return nil, fmt.Errorf("第三方正文接口未返回可用章节：%s", strings.Join(failures, "; "))
		}
		return nil, fmt.Errorf("第三方正文接口未返回可用章节")
	}
	return chapters, nil
}

func fetchFanqieChapterWithContentAPI(ctx context.Context, apiURL, chapterID string, no int, timeout time.Duration) (*fanqieChapter, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	body, err := fetchText(reqCtx, fanqieContentAPIEndpoint(apiURL, chapterID))
	if err != nil {
		return nil, err
	}
	var payload any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, fmt.Errorf("第三方正文接口返回非 JSON：%w", err)
	}
	content := jsonFindString(payload, "content")
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("第三方正文接口缺少 content")
	}
	title := html.UnescapeString(stripHTML(firstRegex(content, `<h1[^>]*>([\s\S]*?)</h1>`)))
	content = cleanThirdPartyChapterContent(content)
	if !isUsableChapterContent(content) {
		return nil, fmt.Errorf("第三方正文接口返回内容不可用")
	}
	if strings.TrimSpace(title) == "" {
		title = jsonFindString(payload, "title", "chapter_title", "name")
	}
	return &fanqieChapter{No: no, Title: html.UnescapeString(title), Content: content}, nil
}

func fanqieContentAPIEndpoint(apiURL, chapterID string) string {
	if strings.Contains(apiURL, "{item_id}") {
		return strings.ReplaceAll(apiURL, "{item_id}", chapterID)
	}
	if strings.Contains(apiURL, "${item_id}") {
		return strings.ReplaceAll(apiURL, "${item_id}", chapterID)
	}
	sep := "?"
	if strings.Contains(apiURL, "?") {
		sep = "&"
	}
	return apiURL + sep + "item_id=" + chapterID
}

func cleanThirdPartyChapterContent(raw string) string {
	raw = regexp.MustCompile(`(?is)<header>.*?</header>`).ReplaceAllString(raw, "")
	raw = regexp.MustCompile(`(?is)<footer>.*?</footer>`).ReplaceAllString(raw, "")
	raw = regexp.MustCompile(`(?is)<h1[^>]*>.*?</h1>`).ReplaceAllString(raw, "")
	raw = regexp.MustCompile(`(?is)</?article[^>]*>`).ReplaceAllString(raw, "")
	raw = regexp.MustCompile(`(?is)<p[^>]*>`).ReplaceAllString(raw, "\n")
	raw = regexp.MustCompile(`(?is)</p>`).ReplaceAllString(raw, "\n")
	return normalizeSampleText(stripHTML(raw))
}

func jsonFindString(v any, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	var walk func(any) string
	walk = func(value any) string {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if _, ok := keySet[key]; ok {
					if s, ok := child.(string); ok && strings.TrimSpace(s) != "" {
						return s
					}
				}
			}
			for _, child := range typed {
				if s := walk(child); s != "" {
					return s
				}
			}
		case []any:
			for _, child := range typed {
				if s := walk(child); s != "" {
					return s
				}
			}
		}
		return ""
	}
	return walk(v)
}

func extractFanqieDirectoryChapterIDs(body string) []string {
	re := regexp.MustCompile(`/reader/(\d+)"[^>]*>\s*第\s*\d+\s*章`)
	matches := re.FindAllStringSubmatch(body, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			ids = appendUniqueStrings(ids, match[1])
		}
	}
	return ids
}

func discoverFanqieBooks(ctx context.Context, category string, target int, exclude map[string]struct{}) ([]string, error) {
	if target <= 0 {
		target = 5
	}
	tagID, err := strconv.ParseInt(strings.TrimSpace(category), 10, 64)
	if err != nil || tagID <= 0 {
		return nil, fmt.Errorf("番茄标签无效：%s", category)
	}
	ids, err := fetchFanqieLibraryBookIDs(ctx, tagID, target, exclude)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("未发现番茄标签书籍，请稍后重试或人工输入 book_id")
	}
	if len(ids) > target {
		return ids[:target], nil
	}
	return ids, nil
}

func fetchFanqieLibraryBookIDs(ctx context.Context, tagID int64, target int, exclude map[string]struct{}) ([]string, error) {
	var ids []string
	for page := 0; page < 20 && len(ids) < target; page++ {
		apiURL := fmt.Sprintf("https://fanqienovel.com/api/library/book_list?page_count=18&page_index=%d&gender=-1&category_id=%d&creation_status=-1&word_count=-1&book_type=-1&sort=0", page, tagID)
		body, err := fetchText(ctx, apiURL)
		if err != nil {
			continue
		}
		var parsed struct {
			Code int `json:"code"`
			Data struct {
				BookList []struct {
					BookID string `json:"book_id"`
				} `json:"book_list"`
				HasMore bool `json:"has_more"`
			} `json:"data"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(body), &parsed); err != nil {
			continue
		}
		if parsed.Code != 0 {
			return nil, fmt.Errorf("番茄书库接口失败：%s", parsed.Message)
		}
		for _, book := range parsed.Data.BookList {
			bookID := strings.TrimSpace(book.BookID)
			if bookID == "" {
				continue
			}
			if _, exists := exclude[bookID]; exists {
				continue
			}
			ids = appendUniqueStrings(ids, bookID)
			if len(ids) >= target {
				return ids[:target], nil
			}
		}
		if !parsed.Data.HasMore {
			break
		}
	}
	return ids, nil
}

func appendUniqueStrings(values []string, next ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(next))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func fetchText(ctx context.Context, rawURL string) (string, error) {
	return fetchTextWithCookie(ctx, rawURL, "")
}

func fetchTextWithCookie(ctx context.Context, rawURL string, cookieHeader string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://fanqienovel.com/")
	if strings.TrimSpace(cookieHeader) != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s failed: %d", rawURL, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func toChapterSamples(sourceID uint, chapters []fanqieChapter) []model.RadarChapterSample {
	samples := make([]model.RadarChapterSample, 0, len(chapters))
	for _, chapter := range chapters {
		samples = append(samples, model.RadarChapterSample{
			SourceID:       sourceID,
			ChapterNo:      chapter.No,
			Title:          chapter.Title,
			Content:        chapter.Content,
			WordCount:      len([]rune(chapter.Content)),
			ParagraphCount: countParagraphs(chapter.Content),
			DialogueRatio:  estimateDialogueRatio(chapter.Content),
			ContentHash:    hashText(chapter.Content),
		})
	}
	return samples
}

func extractFanqieBookID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if _, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return raw
	}
	return firstRegex(raw, `/page/(\d+)`, `book_id=(\d+)`, `/reader/(\d+)`)
}

func fanqieBookURL(bookID string) string {
	return "https://fanqienovel.com/page/" + strings.TrimSpace(bookID)
}

func firstRegex(raw string, patterns ...string) string {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(raw)
		if len(match) > 1 {
			return html.UnescapeString(strings.TrimSpace(match[1]))
		}
	}
	return ""
}

func extractFanqieIntro(body string) string {
	if intro := firstRegexJSON(body, `"abstract"\s*:\s*"((?:\\.|[^"\\])*)"`); intro != "" {
		return stripHTML(intro)
	}
	if intro := firstRegexJSON(body, `"intro"\s*:\s*"((?:\\.|[^"\\])*)"`); intro != "" {
		return stripHTML(intro)
	}
	return stripHTML(firstRegex(
		body,
		`<meta\s+property="og:description"\s+content="([^"]+)"`,
		`<meta\s+name="description"\s+content="([^"]+)"`,
	))
}

func firstRegexJSON(raw string, patterns ...string) string {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(raw)
		if len(match) <= 1 {
			continue
		}
		value := strings.TrimSpace(match[1])
		if decoded, err := strconv.Unquote(`"` + value + `"`); err == nil {
			return html.UnescapeString(strings.TrimSpace(decoded))
		}
		return html.UnescapeString(value)
	}
	return ""
}

func uniqueRegex(raw string, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(raw, -1)
	seen := map[string]struct{}{}
	var out []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		id := strings.TrimSpace(match[1])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func extractFanqieBookTags(body string) []string {
	re := regexp.MustCompile(`<span[^>]*class="[^"]*info-label-[^"]*"[^>]*>([^<]+)</span>`)
	matches := re.FindAllStringSubmatch(body, -1)
	labels := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		label := strings.TrimSpace(html.UnescapeString(match[1]))
		switch label {
		case "", "连载中", "已完结", "完结", "断更":
			continue
		default:
			labels = append(labels, label)
		}
	}
	return uniqueNonEmptyStrings(labels)
}

func (s *RadarService) tagKeysByNames(platform string, names []string) []string {
	tags, err := s.repo.ListTags(platform, "")
	if err != nil {
		return nil
	}
	byName := make(map[string]string, len(tags))
	for _, tag := range tags {
		byName[tag.TagName] = tag.TagKey
	}
	keys := make([]string, 0, len(names))
	for _, name := range names {
		if key := byName[strings.TrimSpace(name)]; key != "" {
			keys = append(keys, key)
		}
	}
	return uniqueNonEmptyStrings(keys)
}

func stripHTML(raw string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	raw = re.ReplaceAllString(raw, "\n")
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	return html.UnescapeString(raw)
}

func normalizeSampleText(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n\n")
}

func isUsableChapterContent(content string) bool {
	content = strings.TrimSpace(content)
	if len([]rune(content)) < 200 {
		return false
	}
	blockMarkers := []string{
		"本章字数",
		"更新时间",
		"扫码下载APP",
		"登录后",
		"验证码",
		"TTGCaptcha",
		"SVIP",
	}
	for _, marker := range blockMarkers {
		if strings.Contains(content, marker) {
			return false
		}
	}
	return true
}

func cleanRadarIntroSamples(samples []model.RadarIntroSample) []model.RadarIntroSample {
	for i := range samples {
		samples[i].Intro = cleanFanqieIntro(samples[i].Intro, samples[i].Title)
		samples[i].WordCount = len([]rune(samples[i].Intro))
	}
	return samples
}

func cleanFanqieIntro(raw, title string) string {
	intro := strings.TrimSpace(stripHTML(raw))
	if intro == "" {
		return ""
	}
	intro = regexp.MustCompile(`\s+`).ReplaceAllString(intro, " ")
	intro = strings.TrimSpace(intro)

	if title = strings.TrimSpace(title); title != "" {
		titlePattern := regexp.QuoteMeta(title)
		intro = regexp.MustCompile(`^番茄小说提供`+titlePattern+`(?:小说)?完整版在线免费阅读[，,。！!\s]*`).ReplaceAllString(intro, "")
	}
	intro = regexp.MustCompile(`^番茄小说提供[\s\S]*?精彩小说尽在番茄小说网[。！!，,\s]*`).ReplaceAllString(intro, "")
	intro = regexp.MustCompile(`^番茄小说提供[^。！？!?]{0,160}(?:完整版在线免费阅读|在线免费阅读)[，,。！!\s]*`).ReplaceAllString(intro, "")
	intro = regexp.MustCompile(`^(?:精彩小说尽在番茄小说网[。！!，,\s]*)+`).ReplaceAllString(intro, "")
	return strings.TrimSpace(intro)
}

func countParagraphs(raw string) int {
	count := 0
	for _, part := range strings.Split(raw, "\n") {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

func estimateDialogueRatio(raw string) float64 {
	total := len([]rune(raw))
	if total == 0 {
		return 0
	}
	dialogue := 0
	for _, r := range raw {
		if r == '“' || r == '”' || r == '"' || r == '「' || r == '」' {
			dialogue++
		}
	}
	return float64(dialogue) / float64(total)
}

func totalSampleWords(chapters []fanqieChapter) int {
	total := 0
	for _, chapter := range chapters {
		total += len([]rune(chapter.Content))
	}
	return total
}

func hashText(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func marshalStringSlice(values []string) string {
	if values == nil {
		values = []string{}
	}
	payload, _ := json.Marshal(values)
	return string(payload)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func generateIntroTagKeys(req model.GenerateRadarIntroRequest) []string {
	values := make([]string, 0, len(req.Tags)+2)
	values = append(values, req.Tags...)
	values = append(values, req.TagKey, req.Category)
	return uniqueNonEmptyStrings(values)
}

func uniqueUintIDs(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	out := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func jsonArrayContains(raw string, value string) bool {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return false
	}
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return raw
	}
	return raw[start : end+1]
}

func defaultStringRadar(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func defaultInt(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func clipRunesRadar(raw string, max int) string {
	runes := []rune(raw)
	if len(runes) <= max {
		return raw
	}
	return string(runes[:max])
}

func SortRulesForPrompt(rules []model.RadarRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Weight != rules[j].Weight {
			return rules[i].Weight > rules[j].Weight
		}
		return rules[i].Confidence > rules[j].Confidence
	})
}
