package service

import (
	"strings"
	"testing"
)

func TestCleanThirdPartyChapterContent(t *testing.T) {
	raw := `<?xml version="1.0"?><html><body><h1 class="chapterTitle1">第1章 标题</h1><p idx="0">第一段正文。</p><p idx="1">第二段正文。</p><footer>广告</footer></body></html>`
	got := cleanThirdPartyChapterContent(raw)
	if strings.Contains(got, "第1章 标题") {
		t.Fatalf("title should be stripped from content: %q", got)
	}
	if !strings.Contains(got, "第一段正文。") || !strings.Contains(got, "第二段正文。") {
		t.Fatalf("content paragraphs missing: %q", got)
	}
	if strings.Contains(got, "广告") {
		t.Fatalf("footer should be stripped: %q", got)
	}
}
