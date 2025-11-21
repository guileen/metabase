package vocab

import "testing"

func TestMatchPattern(t *testing.T) {
	vi := NewVocabularyIndex(CreateDefaultConfig())
	if !vi.matchPattern("/a/b/c.go", "*.go") {
		t.Fatalf("basename *.go should match")
	}
	if !vi.matchPattern("/a/b/c.go", "**/*.go") {
		t.Fatalf("recursive **/*.go should match")
	}
	if vi.matchPattern("/a/node_modules/x.js", "node_modules/*") == false {
		t.Fatalf("dir wildcard should match")
	}
}

func TestShouldProcessFile(t *testing.T) {
	cfg := CreateDefaultConfig()
	cfg.IncludePatterns = []string{"*.go"}
	cfg.ExcludePatterns = []string{"**/vendor/**"}
	vi := NewVocabularyIndex(cfg)
	if !vi.shouldProcessFile("/p/q/main.go") {
		t.Fatalf("should include .go files")
	}
	if vi.shouldProcessFile("/p/q/main.js") {
		t.Fatalf("should not include non-go files")
	}
	if vi.shouldProcessFile("/p/vendor/x.go") {
		t.Fatalf("should exclude vendor path")
	}
}
