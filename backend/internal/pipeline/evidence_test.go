package pipeline

import "testing"

func TestLocateEvidenceQuoteUsesRuneOffsetsAndIgnoresWhitespace(t *testing.T) {
	content := "林秋抬手。\n\n阿秋 推开七号门，冷风扑面。"

	start, end := locateEvidenceQuote(content, "阿秋推开七号门")
	if start < 0 || end <= start {
		t.Fatalf("quote was not located: start=%d end=%d", start, end)
	}
	runes := []rune(content)
	located := string(runes[start:end])
	if located != "阿秋 推开七号门" {
		t.Fatalf("located quote = %q", located)
	}

	start, end = locateEvidenceQuote(content, "正文中不存在的证据")
	if start != -1 || end != -1 {
		t.Fatalf("missing quote offsets = %d,%d, want -1,-1", start, end)
	}
}
