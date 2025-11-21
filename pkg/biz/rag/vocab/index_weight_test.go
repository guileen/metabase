package vocab

import ("math"
    "testing"
    "time")

func TestWeightsAreFinite(t *testing.T) {
    vi := NewVocabularyIndex(nil)

    vi.DocToTerms["doc1"] = &DocumentInfo{
        Path:        "doc1",
        TotalTerms:  3,
        UniqueTerms: 2,
        TermFreqs:   map[string]int{"a": 2, "b": 1},
        TermPositions: map[string][]int{"a": {0, 2}, "b": {1}},
        LastModified: time.Now(),
    }

    vi.TermToDocs["a"] = &TermInfo{
        Term:         "a",
        DocumentFreq: 1,
        TotalFreq:    2,
        Documents:    map[string]int{"doc1": 2},
        Positions:    map[string][]int{"doc1": {0, 2}},
        LastSeen:     time.Now(),
        Category:     "general",
    }
    vi.TermToDocs["b"] = &TermInfo{
        Term:         "b",
        DocumentFreq: 1,
        TotalFreq:    1,
        Documents:    map[string]int{"doc1": 1},
        Positions:    map[string][]int{"doc1": {1}},
        LastSeen:     time.Now(),
        Category:     "general",
    }

    vi.updateGlobalStats()
    vi.calculateWeights()

    for term, ti := range vi.TermToDocs {
        if math.IsNaN(ti.Weight) || math.IsInf(ti.Weight, 0) {
            t.Fatalf("weight not finite for term %s: %v", term, ti.Weight)
        }
    }
}