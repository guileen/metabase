package cli

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/guileen/metabase/pkg/infra/llm"
	"github.com/guileen/metabase/pkg/infra/embedding"
	"github.com/guileen/metabase/pkg/infra/skills"
	"github.com/guileen/metabase/pkg/infra/vocab"
)

// Performance statistics
type PerformanceStats struct {
    TotalTime         time.Duration `json:"total_time"`
    QueryTime         time.Duration `json:"query_time"`
    EmbeddingTime     time.Duration `json:"embedding_time"`
    RerankTime       time.Duration `json:"rerank_time"`
    FilesScanned      int           `json:"files_scanned"`
    PathHits         int           `json:"path_hits"`
    ContentHits      int           `json:"content_hits"`
    CandidatesFound   int           `json:"candidates_found"`
    EmbeddingsProcessed int         `json:"embeddings_processed"`
    TokensUsed       int           `json:"tokens_used"`
    ChunksProcessed  int           `json:"chunks_processed"`
    Model            string        `json:"model"`
    AvgCharsPerEmbed  float64       `json:"avg_chars_per_embed"`
    LocalMode        bool          `json:"local_mode"`
    ExpansionUsed    bool          `json:"expansion_used"`
    SkillsUsed       bool          `json:"skills_used"`
    FileTypes        map[string]int `json:"file_types"`
    TopFileTypes     []FileTypeInfo `json:"top_file_types"`
}

type FileTypeInfo struct {
    Type  string `json:"type"`
    Count int    `json:"count"`
    Pct   float64 `json:"pct"`
}

var searchCmd = &cobra.Command{
    Use:   "search",
    Short: "代码语义搜索",
    Run: func(cmd *cobra.Command, args []string) {
        q := strings.TrimSpace(strings.Join(args, " "))
        if q == "" { cmd.PrintErrln("请输入查询文本"); return }
        k, _ := cmd.Flags().GetInt("top")
        win, _ := cmd.Flags().GetInt("win")
        localGo, _ := cmd.Flags().GetBool("local-go")
        doExpand, _ := cmd.Flags().GetBool("expand")
        useSkills, _ := cmd.Flags().GetBool("use-skills")
        includeGlobs, _ := cmd.Flags().GetStringSlice("include")
        excludeGlobs, _ := cmd.Flags().GetStringSlice("exclude")
        vocabUpdate, _ := cmd.Flags().GetBool("vocab-update")
        vocabBuild, _ := cmd.Flags().GetBool("vocab-build")
        vocabMaxAge, _ := cmd.Flags().GetInt("vocab-max-age")
        start := time.Now()

        // 自动管理词表
        vocabManager := NewVocabularyManager()
        if err := vocabManager.EnsureVocabulary(vocabBuild, vocabUpdate, vocabMaxAge); err != nil {
            cmd.Printf("[Warning] 词表管理失败: %v\n", err)
        }
    embeddingStart := start
    var rerankStart time.Time
    stats := &PerformanceStats{
        Model:      os.Getenv("LLM_EMBEDDING_MODEL"),
        FileTypes:  make(map[string]int),
    }
    files := rgFiles()

    // Apply enhanced file filtering for noise reduction and user-specified patterns
    var fileFilter *FileFilter
    if len(includeGlobs) > 0 || len(excludeGlobs) > 0 {
        fileFilter = createFileFilterWithGlobs(includeGlobs, excludeGlobs)
    } else {
        fileFilter = createDefaultFileFilter()
    }
    filteredFiles := filterFiles(files, fileFilter)

    stats.FilesScanned = len(files)

    fmt.Printf("[CLI] File filtering: %d -> %d files (reduced by %.1f%%)\n",
        len(files), len(filteredFiles),
        float64(len(files)-len(filteredFiles))/float64(len(files))*100)

    // Use filtered files for processing
    files = filteredFiles

    // 使用词表扩展查询
    var expandedTerms []string
    if vocabManager.GetVocabularyBuilder() != nil {
        vocabExpandedTerms, err := vocabManager.ExpandQuery(q, 10)
        if err != nil {
            fmt.Printf("[Warning] 词表扩展失败: %v\n", err)
        } else {
            expandedTerms = vocabExpandedTerms
            fmt.Printf("[CLI] Vocabulary expansion: %d terms\n", len(expandedTerms))
        }
    }

    // 合并原始查询词和扩展词
    ts := toks(q)
    for _, term := range expandedTerms {
        ts = append(ts, term)
    }

    // 去重
    uniqueTerms := make(map[string]bool)
    var finalTs []string
    for _, term := range ts {
        if !uniqueTerms[term] {
            uniqueTerms[term] = true
            finalTs = append(finalTs, term)
        }
    }
    ts = finalTs

    stats.ExpansionUsed = doExpand || len(expandedTerms) > 0
    stats.SkillsUsed = useSkills
        if doExpand || useSkills {
            var ex []string

            if useSkills {
                // Use the new skills system
                tm := skills.NewTemplateManager()
                skillInput := &skills.SkillInput{
                    Query: q,
                    Parameters: map[string]interface{}{
                        "max_expansions": 8,
                    },
                }

                output, err := tm.ExecuteSkill("expandQuery", skillInput, nil)
                if err == nil && output.Success {
                    if result, ok := output.Result.(map[string]interface{}); ok {
                        if expanded, ok := result["expanded_terms"].([]string); ok {
                            ex = expanded
                        }
                    }
                }
            } else {
                // Use legacy LLM expansion
                ex = llm.ExpandKeywords(q)
            }

            if len(ex)>0 {
                m := make(map[string]bool);
                for _,t:=range ts{m[t]=true};
                for _,t:=range ex{ m[t]=true };
                ts = make([]string,0,len(m));
                for t:=range m{ ts = append(ts,t) }
            }
        }
        scored := make([]string, 0)
        type ps struct{ f string; s int }
        var arr []ps
        for _, f := range files { s := pathScore(f, ts, fileFilter); if s>0 { arr = append(arr, ps{f, s}) } }
        stats.PathHits = len(arr)
        sort.Slice(arr, func(i,j int)bool{ return arr[i].s>arr[j].s })
        for i:=0;i<len(arr)&&i<1000;i++{ scored=append(scored, arr[i].f) }
        var hits []hit
        if doExpand { hits = rgContentMulti(ts) } else { hits = rgContent(q) }
        stats.ContentHits = len(hits)
        set := make(map[string]bool)
        for _, f := range scored { set[f]=true }
        pool := make([]hit,0)
        for _, h := range hits { if set[h.file] { pool = append(pool, h) } }
        if len(pool)==0 { pool = hits }
        uniq := make(map[string]item)
        for _, h := range pool {
            key := h.file+":"+strconv.Itoa(h.line/20)
            if _,ok:=uniq[key]; ok{continue}
            text := readSnippet(h.file, h.line, win)
            if text!="" {
                uniq[key] = item{
                    file: h.file,
                    line: h.line,
                    text: text,
                    snippet: text,
                    context: win,
                }
            }
        }
        // Configurable max candidates from environment
        maxCandidates := 300
        if v := os.Getenv("SEARCH_MAX_CANDIDATES"); v!="" {
            if n,err := strconv.Atoi(v); err==nil && n>0 {
                maxCandidates = n
            }
        }

        items := make([]item,0,maxCandidates)
        for _, it := range uniq { items = append(items, it); if len(items)>=maxCandidates { break } }

        fmt.Printf("[CLI] Using max candidates limit: %d\n", maxCandidates)

        // Log how many items will be processed for embedding
        totalCandidateChars := 0
        for _, it := range items {
            totalCandidateChars += len(it.text)
            // Track file types
            ext := strings.ToLower(filepath.Ext(it.file))
            if ext != "" {
                stats.FileTypes[ext]++
            } else {
                stats.FileTypes["no_ext"]++
            }
        }
        fmt.Printf("[CLI] Will embed %d candidate texts (%d characters)\n", len(items), totalCandidateChars)
        stats.CandidatesFound = len(items)

        var qv []float64
        var evs [][]float64
        var err error
        embeddingStart = time.Now()
        if localGo {
            stats.LocalMode = true
            qv = embedding.EmbedLocalMiniLM([]string{q})[0]
            texts := make([]string,len(items)); for i:=range items{ texts[i]=items[i].text }
            evs = embedding.EmbedLocalMiniLM(texts)
            stats.EmbeddingTime = time.Since(embeddingStart)
        } else {
            qEmb, e := llm.Embeddings([]string{q}); if e!=nil { cmd.PrintErrln(e.Error()); return }
            qv = qEmb[0]
            texts := make([]string,len(items)); for i:=range items{ texts[i]=items[i].text }
            evs, err = embedRemoteBatchTexts(texts); if err!=nil { cmd.PrintErrln(err.Error()); return }
            stats.EmbeddingTime = time.Since(embeddingStart)
        }
        stats.EmbeddingsProcessed = len(evs)
        if totalCandidateChars > 0 {
            stats.AvgCharsPerEmbed = float64(totalCandidateChars) / float64(len(evs))
        }
        type sc struct{ item item; score float64 }
        rs := make([]sc,len(items))
        for i:=range items {
            var score float64
            if i < len(evs) { score = cos(qv, evs[i]) }
            rs[i] = sc{items[i], score}
        }
        rerankStart = time.Now()
        if os.Getenv("LLM_RERANK_MODEL")!="" && !localGo {
            texts := make([]string,len(items)); for i:=range items{ texts[i]=items[i].text }
            if rr, e := llm.Rerank(q, texts); e==nil && len(rr)>0 {
                for i:=range rs { if i < len(rr) { rs[i].score = rr[i] } }
            }
        }
        stats.RerankTime = time.Since(rerankStart)
        sort.Slice(rs, func(i,j int)bool{ return rs[i].score>rs[j].score })
        if len(rs)>k { rs=rs[:k] }
        // Calculate top file types
        var topTypes []FileTypeInfo
        totalFiles := len(stats.FileTypes)
        for ext, count := range stats.FileTypes {
            pct := float64(count) / float64(totalFiles) * 100
            topTypes = append(topTypes, FileTypeInfo{Type: ext, Count: count, Pct: pct})
        }
        sort.Slice(topTypes, func(i, j int) bool { return topTypes[i].Count > topTypes[j].Count })
        if len(topTypes) > 5 { topTypes = topTypes[:5] }
        stats.TopFileTypes = topTypes

        // Final timing calculation
        stats.TotalTime = time.Since(start)
        stats.QueryTime = stats.TotalTime - stats.EmbeddingTime - stats.RerankTime

        // Print enhanced results
        fmt.Printf("\n=== SEARCH RESULTS ===\n")
        fmt.Printf("Query: %s\n", q)
        fmt.Printf("Performance:\n")
        fmt.Printf("  Total time: %v\n", stats.TotalTime)
        fmt.Printf("  Query processing: %v\n", stats.QueryTime)
        fmt.Printf("  Embedding: %v\n", stats.EmbeddingTime)
        fmt.Printf("  Reranking: %v\n", stats.RerankTime)
        fmt.Printf("\nStatistics:\n")
        fmt.Printf("  Files scanned: %d\n", stats.FilesScanned)
        fmt.Printf("  Path matches: %d\n", stats.PathHits)
        fmt.Printf("  Content matches: %d\n", stats.ContentHits)
        fmt.Printf("  Candidates: %d\n", stats.CandidatesFound)
        fmt.Printf("  Embeddings: %d\n", stats.EmbeddingsProcessed)
        if stats.AvgCharsPerEmbed > 0 {
            fmt.Printf("  Avg chars/embed: %.1f\n", stats.AvgCharsPerEmbed)
        }
        fmt.Printf("  Chunks processed: %d\n", stats.ChunksProcessed)
        if stats.TokensUsed > 0 {
            fmt.Printf("  Tokens used: %d\n", stats.TokensUsed)
        }
        fmt.Printf("  File types: %v\n", stats.TopFileTypes)
        fmt.Printf("\nConfiguration:\n")
        if stats.LocalMode {
            fmt.Printf("  Mode: Local embeddings\n")
            fmt.Printf("  Dimension: %d\n", len(qv))
        } else {
            fmt.Printf("  Mode: Remote embeddings\n")
            fmt.Printf("  Model: %s\n", stats.Model)
            if stats.RerankTime > 0 {
                fmt.Printf("  Reranker: %s\n", os.Getenv("LLM_RERANK_MODEL"))
            }
        }
        fmt.Printf("  Query expansion: %t\n", stats.ExpansionUsed)
        fmt.Printf("  Skills system: %t\n", stats.SkillsUsed)
        fmt.Printf("  Max candidates: %d\n", maxCandidates)
        fmt.Printf("  Window size: %d lines\n", win)

        // 显示词表统计信息
        if vocabManager.GetVocabularyBuilder() != nil {
            vocabStats := vocabManager.GetVocabularyStats()
            if globalStats, ok := vocabStats["global_stats"]; ok {
                if gs, ok := globalStats.(*vocab.GlobalStats); ok {
                    fmt.Printf("Vocabulary: %d terms, %d docs\n", gs.UniqueTerms, gs.TotalDocuments)
                }
            }
        }

        fmt.Printf("\nTop %d results:\n", k)
        for i, s := range rs {
            fmt.Printf("%d. %s:%d score=%.3f\n", i+1, s.item.file, s.item.line, s.score)
            fmt.Printf("   %s\n", s.item.snippet)
            fmt.Printf("---\n")
        }
    },
}

func init() {
    searchCmd.Flags().Int("top", 15, "TopK");
    searchCmd.Flags().Int("win", 8, "上下文窗口");
    searchCmd.Flags().Bool("local-go", false, "使用Go本地嵌入");
    searchCmd.Flags().Bool("expand", true, "启用LLM关键词扩展");
    searchCmd.Flags().Bool("use-skills", false, "使用新的技能系统");
    searchCmd.Flags().StringSlice("include", []string{}, "Glob patterns for file inclusion (e.g., '*.go','src/**/*.rs')");
    searchCmd.Flags().StringSlice("exclude", []string{}, "Glob patterns for file exclusion (e.g., '*.log','test/*')");
    searchCmd.Flags().Bool("vocab-update", true, "自动更新词表索引");
    searchCmd.Flags().Bool("vocab-build", true, "自动构建词表索引");
    searchCmd.Flags().Int("vocab-max-age", 24, "词表最大有效时间（小时）");
    AddCommand(searchCmd)
}

type item struct{
    file string
    line int
    text string
    snippet string   // The actual code snippet around the match
    context int      // Context window size used
}

type hit struct{
    file string
    line int
    match string    // The matching line content
}

// File filtering configuration
type FileFilter struct {
    ExcludePatterns []string
    IncludePatterns []string
    PriorityTypes   map[string]int
    NoiseExtensions []string
}

// Create file filter with user-specified glob patterns
func createFileFilterWithGlobs(includeGlobs, excludeGlobs []string) *FileFilter {
    filter := createDefaultFileFilter()

    // Add custom include patterns
    if len(includeGlobs) > 0 {
        filter.IncludePatterns = append(filter.IncludePatterns, includeGlobs...)
        fmt.Printf("[CLI] Added %d include patterns: %v\n", len(includeGlobs), includeGlobs)
    }

    // Add custom exclude patterns
    if len(excludeGlobs) > 0 {
        filter.ExcludePatterns = append(filter.ExcludePatterns, excludeGlobs...)
        fmt.Printf("[CLI] Added %d exclude patterns: %v\n", len(excludeGlobs), excludeGlobs)
    }

    return filter
}

// Default file filter with sensible defaults
func createDefaultFileFilter() *FileFilter {
    return &FileFilter{
        ExcludePatterns: []string{
            ".git/*", "node_modules/*", "vendor/*", "dist/*", "build/*",
            "out/*", ".cache/*", "*.log", "*.tmp", "*.lock", "*.bak",
            "*.swp", "*.swo", ".DS_Store", "Thumbs.db", "*.pyc", "__pycache__/*",
            "*.class", "*.jar", "*.war", "*.ear", "*.exe", "*.dll", "*.so",
            "*.dylib", "*.a", "*.lib", "*.obj", "*.o", "*.bin",
            ".vscode/*", ".idea/*", "*.sublime-*", ".svn/*", ".hg/*",
            "target/*", "cargo-lock", "Cargo.lock", "poetry.lock", "yarn.lock",
            "package-lock.json", "go.sum", "*.min.js", "*.min.css",
            "*.map", "*.tsbuildinfo", "coverage/*", ".coverage*", "*.prof",
            "*.orig", "*.rej", ".#*", "*~", "#*#",
        },
        IncludePatterns: []string{
            "*.go", "*.rs", "*.js", "*.ts", "*.py", "*.java", "*.cpp", "*.c",
            "*.h", "*.hpp", "*.cs", "*.php", "*.rb", "*.swift", "*.kt",
            "*.scala", "*.clj", "*.hs", "*.ml", "*.sh", "*.sql", "*.html",
            "*.css", "*.scss", "*.less", "*.vue", "*.jsx", "*.tsx",
            "*.md", "*.txt", "*.json", "*.yaml", "*.yml", "*.toml", "*.xml",
            "*.dockerfile", "Dockerfile*", "*.env", "*.ini", "*.cfg", "*.conf",
        },
        PriorityTypes: map[string]int{
            // High priority - source code files
            ".go": 10, ".rs": 10, ".py": 10, ".js": 9, ".ts": 9, ".java": 10,
            ".cpp": 10, ".c": 10, ".h": 9, ".hpp": 9, ".cs": 10, ".php": 9,
            ".rb": 9, ".swift": 10, ".kt": 10, ".scala": 9, ".jsx": 9, ".tsx": 9,
            // Medium priority - config and markup
            ".html": 7, ".css": 7, ".scss": 7, ".less": 7, ".vue": 8,
            ".sql": 8, ".sh": 7, ".json": 6, ".yaml": 6, ".yml": 6, ".toml": 6,
            ".xml": 6, ".dockerfile": 6, ".env": 6, ".ini": 5, ".cfg": 5,
            // Lower priority - docs and other
            ".md": 5, ".txt": 4, ".pdf": 3, ".doc": 3, ".docx": 3,
        },
        NoiseExtensions: []string{
            ".log", ".tmp", ".lock", ".bak", ".swp", ".swo", ".orig", ".rej",
            ".pyc", ".class", ".jar", ".war", ".ear", ".exe", ".dll", ".so",
            ".dylib", ".a", ".lib", ".obj", ".o", ".bin", ".min.js", ".min.css",
            ".map", ".tsbuildinfo", ".prof",
        },
    }
}

// Check if file should be excluded based on patterns
func (f *FileFilter) shouldExcludeFile(filePath string) bool {
    for _, pattern := range f.ExcludePatterns {
        if f.matchesPattern(filePath, pattern) {
            return true
        }
    }
    return false
}

// Check if file should be included based on patterns
func (f *FileFilter) shouldIncludeFile(filePath string) bool {
    // If no include patterns, include everything except excluded
    if len(f.IncludePatterns) == 0 {
        return true
    }

    for _, pattern := range f.IncludePatterns {
        if f.matchesPattern(filePath, pattern) {
            return true
        }
    }
    return false
}

// Enhanced pattern matching that supports ** recursive patterns
func (f *FileFilter) matchesPattern(filePath, pattern string) bool {
    // Handle recursive patterns with **
    if strings.Contains(pattern, "**") {
        return f.matchesRecursivePattern(filePath, pattern)
    }

    // Standard filepath matching for the basename
    if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
        return true
    }

    // Check directory patterns
    if strings.HasSuffix(pattern, "/*") {
        dirPattern := strings.TrimSuffix(pattern, "/*")
        if strings.Contains(filePath, dirPattern+string(filepath.Separator)) {
            return true
        }
    }

    // Check if the pattern contains a path separator and try path matching
    if strings.Contains(pattern, string(filepath.Separator)) {
        if matched, _ := filepath.Match(pattern, filePath); matched {
            return true
        }
    }

    return false
}

// Handle recursive ** patterns
func (f *FileFilter) matchesRecursivePattern(filePath, pattern string) bool {
    // Replace ** with a recursive pattern match
    parts := strings.Split(pattern, "**")
    if len(parts) != 2 {
        return false
    }

    prefix := parts[0]
    suffix := parts[1]

    // Check if filePath starts with the prefix
    if prefix != "" && !strings.HasPrefix(filePath, prefix) {
        return false
    }

    // Check if filePath ends with the suffix
    if suffix != "" && !strings.HasSuffix(filePath, suffix) {
        return false
    }

    return true
}

// Get file priority score
func (f *FileFilter) getFilePriority(filePath string) int {
    ext := strings.ToLower(filepath.Ext(filePath))
    if priority, exists := f.PriorityTypes[ext]; exists {
        return priority
    }
    // Default priority for unknown file types
    return 3
}

// Is noise file based on extension
func (f *FileFilter) isNoiseFile(filePath string) bool {
    ext := strings.ToLower(filepath.Ext(filePath))
    for _, noiseExt := range f.NoiseExtensions {
        if ext == noiseExt {
            return true
        }
    }
    return false
}

func sh(args ...string) string { c := exec.Command("rg", args...); c.Env = os.Environ(); c.Dir = "."; b, _ := c.Output(); return string(b) }

func rgFiles() []string {
    out := sh("--files", "--hidden", "--no-ignore", "-g", "!{.git,node_modules,vendor,dist,build,out,.cache}")
    lines := strings.Split(out, "\n")
    r := make([]string,0,len(lines))
    for _, l := range lines { if strings.TrimSpace(l)!="" { r = append(r, l) } }
    return r
}

// Enhanced file filtering function
func filterFiles(files []string, filter *FileFilter) []string {
    if filter == nil {
        filter = createDefaultFileFilter()
    }

    var filtered []string
    for _, file := range files {
        // Skip empty files
        if strings.TrimSpace(file) == "" {
            continue
        }

        // Apply exclusion filters
        if filter.shouldExcludeFile(file) {
            continue
        }

        // Apply inclusion filters
        if !filter.shouldIncludeFile(file) {
            continue
        }

        // Skip noise files
        if filter.isNoiseFile(file) {
            continue
        }

        filtered = append(filtered, file)
    }

    return filtered
}

func rgContent(q string) []hit {
    out := sh("-n", "--hidden", "--no-ignore", "-S", "-g", "!{.git,node_modules,vendor,dist,build,out,.cache}", "-e", q)
    sc := bufio.NewScanner(strings.NewReader(out))
    r := make([]hit,0)
    for sc.Scan() {
        l := sc.Text()
        m := strings.SplitN(l, ":", 3)
        if len(m)>=3 {
            if ln,err := strconv.Atoi(m[1]); err==nil {
                r = append(r, hit{file:m[0], line:ln, match: m[2]})
            }
        }
    }
    return r
}

func rgContentMulti(ts []string) []hit {
    seen := make(map[string]bool)
    r := make([]hit,0)
    for i, t := range ts {
        if i>=8 { break }
        hs := rgContent(t)
        for _, h := range hs {
            key := h.file+":"+strconv.Itoa(h.line/20)
            if seen[key] { continue }
            seen[key] = true
            r = append(r, h)
        }
    }
    return r
}

func embedRemoteBatchTexts(texts []string) ([][]float64, error) {
    fmt.Printf("[CLI] Starting batch embedding for %d texts\n", len(texts))

    // Calculate total characters for logging
    totalChars := 0
    for _, text := range texts {
        totalChars += len(text)
    }
    fmt.Printf("[CLI] Total characters to embed: %d\n", totalChars)

    // Use the enhanced embeddings with automatic token management
    config := &llm.Config{
        BaseURL:        os.Getenv("LLM_BASE_URL"),
        APIKey:         os.Getenv("LLM_API_KEY"),
        EmbeddingModel: os.Getenv("LLM_EMBEDDING_MODEL"),
        Timeout:        60 * time.Second,
        RetryAttempts:  3,
        RetryDelay:     time.Second,
    }

    // Set a reasonable batch limit based on environment or default
    limit := 32 // Reduced from 64 to avoid token issues
    if v := os.Getenv("LLM_EMBED_BATCH_LIMIT"); v!="" {
        if n,err := strconv.Atoi(v); err==nil && n>0 {
            limit = n
            fmt.Printf("[CLI] Using batch limit from environment: %d\n", limit)
        }
    } else {
        fmt.Printf("[CLI] Using default batch limit: %d\n", limit)
    }

    res := make([][]float64, 0, len(texts))
    processedChunks := 0

    for i:=0; i<len(texts); i+=limit {
        end := i + limit; if end>len(texts) { end = len(texts) }
        chunk := texts[i:end]

        chunkChars := 0
        for _, text := range chunk {
            chunkChars += len(text)
        }

        fmt.Printf("[CLI] Processing chunk %d: %d texts, %d characters\n",
            processedChunks + 1, len(chunk), chunkChars)

        // Use enhanced embeddings which handles token limits automatically
        emb, err := llm.EnhancedEmbeddings(chunk, config)
        if err != nil {
            fmt.Printf("[CLI] Error in chunk %d: %v\n", processedChunks + 1, err)
            return nil, err
        }

        res = append(res, emb...)
        processedChunks++
        fmt.Printf("[CLI] Completed chunk %d, total embeddings so far: %d\n",
            processedChunks, len(res))
    }

    fmt.Printf("[CLI] Successfully generated %d embeddings from %d texts in %d chunks\n",
        len(res), len(texts), processedChunks)
    return res, nil
}

func readSnippet(p string, ln, win int) string {
    b, err := os.ReadFile(p); if err!=nil { return "" }
    lines := strings.Split(string(b), "\n")
    a := ln - win - 1; if a<0 { a=0 }
    z := ln + win; if z>len(lines) { z=len(lines) }
    return strings.Join(lines[a:z], "\n")
}

func toks(s string) []string {
    s = strings.ToLower(s)
    f := func(r rune) bool { return !(r>='a'&&r<='z'||r>='0'&&r<='9'||r=='_') }
    return strings.FieldsFunc(s, f)
}

func pathScore(p string, ts []string, filter *FileFilter) int {
    if filter == nil {
        filter = createDefaultFileFilter()
    }

    b := strings.ToLower(filepath.Base(p))
    d := strings.ToLower(filepath.Dir(p))
    s := 0

    // Token-based scoring (existing logic)
    for _, t := range ts {
        if strings.Contains(b, t) { s += 2 }
        if strings.Contains(d, t) { s += 1 }
    }

    // File priority bonus
    priority := filter.getFilePriority(p)
    s += priority / 2 // Add half of priority as bonus to not overwhelm token scores

    return s
}

func cos(a, b []float64) float64 { var d, na, nb float64; n := len(a); if len(b)<n { n=len(b) }; for i:=0;i<n;i++{ x:=a[i]; y:=b[i]; d+=x*y; na+=x*x; nb+=y*y }; t := (math.Sqrt(na)*math.Sqrt(nb)); if t==0 { return 0 }; return d/t }