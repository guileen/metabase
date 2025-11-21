package rag

import ("fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/vocab")

// VocabularyManager 词表管理器
type VocabularyManager struct {
	builder *vocab.VocabularyBuilder
	config  *vocab.Config
}

// NewVocabularyManager 创建词表管理器
func NewVocabularyManager() *VocabularyManager {
	return &VocabularyManager{
		config: vocab.CreateDefaultConfig(),
	}
}

// EnsureVocabulary 确保词表可用
func (vm *VocabularyManager) EnsureVocabulary(autoBuild, autoUpdate bool, maxAgeHours int) error {
	// 检查词表是否存在和有效
	shouldBuild := false
	shouldUpdate := false

	// 尝试加载现有词表
	builder, err := vocab.LoadVocabularyBuilder(vm.config)
	if err != nil {
		fmt.Printf("[Vocab] 加载词表失败: %v\n", err)
		shouldBuild = true
	} else {
		vm.builder = builder
		stats := builder.GetIndex().GetStats()

		// 检查词表是否太旧
		if maxAgeHours > 0 {
			maxAge := time.Duration(maxAgeHours) * time.Hour
			if time.Since(stats.LastUpdated) > maxAge {
				fmt.Printf("[Vocab] 词表过期 (上次更新: %s), 需要更新\n",
					stats.LastUpdated.Format("2006-01-02 15:04"))
				shouldUpdate = true
			}
		}

		// 检查词表是否为空
		if stats.TotalDocuments == 0 {
			fmt.Printf("[Vocab] 词表为空，需要构建\n")
			shouldBuild = true
		}
	}

	// 构建词表
	if shouldBuild && autoBuild {
		fmt.Printf("[Vocab] 开始自动构建词表...\n")
		if err := vm.buildVocabulary(); err != nil {
			return fmt.Errorf("构建词表失败: %w", err)
		}
	} else if shouldUpdate && autoUpdate {
		fmt.Printf("[Vocab] 开始自动更新词表...\n")
		if err := vm.updateVocabulary(); err != nil {
			return fmt.Errorf("更新词表失败: %w", err)
		}
	} else if vm.builder == nil {
		// 如果没有构建器，创建一个空的
		vm.builder = vocab.NewVocabularyBuilder(vm.config)
	}

	return nil
}

// buildVocabulary 构建词表
func (vm *VocabularyManager) buildVocabulary() error {
	builder := vocab.NewVocabularyBuilder(vm.config)

	// 获取当前目录下的所有相关文件
	filePaths, err := vm.discoverFiles(".")
	if err != nil {
		return fmt.Errorf("发现文件失败: %w", err)
	}

	if len(filePaths) == 0 {
		fmt.Printf("[Vocab] 未发现可构建的文件\n")
		return nil
	}

	fmt.Printf("[Vocab] 发现 %d 个文件用于构建词表\n", len(filePaths))

	// 构建词表
	result, err := builder.BuildFromFiles(filePaths)
	if err != nil {
		return err
	}

	printVocabResult(result)
	vm.builder = builder
	return nil
}

// updateVocabulary 更新词表
func (vm *VocabularyManager) updateVocabulary() error {
	if vm.builder == nil {
		return fmt.Errorf("词表构建器未初始化")
	}

	// 获取当前目录下的所有相关文件
	filePaths, err := vm.discoverFiles(".")
	if err != nil {
		return fmt.Errorf("发现文件失败: %w", err)
	}

	if len(filePaths) == 0 {
		fmt.Printf("[Vocab] 未发现可更新的文件\n")
		return nil
	}

	fmt.Printf("[Vocab] 发现 %d 个文件用于更新词表\n", len(filePaths))

	// 更新词表
	result, err := vm.builder.UpdateFromDirectory(".", true)
	if err != nil {
		return err
	}

	printVocabResult(result)
	return nil
}

// discoverFiles 发现相关文件
func (vm *VocabularyManager) discoverFiles(rootDir string) ([]string, error) {
	var filePaths []string

	// 使用文件遍历发现相关文件
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// 跳过隐藏目录和常见的忽略目录
			dirName := filepath.Base(path)
			if strings.HasPrefix(dirName, ".") ||
				dirName == "node_modules" ||
				dirName == "vendor" ||
				dirName == "target" ||
				dirName == "dist" ||
				dirName == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		if vm.shouldIncludeFile(ext) {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	return filePaths, err
}

// shouldIncludeFile 检查文件是否应该被包含
func (vm *VocabularyManager) shouldIncludeFile(ext string) bool {
	// 代码文件
	codeExts := map[string]bool{
		".go":  true,
		".rs":  true,
		".js":  true,
		".ts":  true,
		".py":  true,
		".java": true,
		".cpp": true,
		".c":   true,
		".h":   true,
		".hpp": true,
		".cs":  true,
		".php": true,
		".rb":  true,
		".swift": true,
		".kt":  true,
		".scala": true,
		".jsx": true,
		".tsx": true,
	}

	// 配置和文档文件
	configExts := map[string]bool{
		".md":  true,
		".txt": true,
		".json": true,
		".yaml": true,
		".yml": true,
		".toml": true,
		".xml": true,
		".env": true,
		".ini": true,
		".cfg": true,
		".conf": true,
		".dockerfile": true,
	}

	return codeExts[ext] || configExts[ext]
}

// GetVocabularyBuilder 获取词表构建器
func (vm *VocabularyManager) GetVocabularyBuilder() *vocab.VocabularyBuilder {
	return vm.builder
}

// ExpandQuery 使用词表扩展查询
func (vm *VocabularyManager) ExpandQuery(query string, maxExpansions int) ([]string, error) {
	if vm.builder == nil {
		return nil, fmt.Errorf("词表未初始化")
	}

	result := vm.builder.GetIndex().ExpandQuery(query, maxExpansions)

	// 合并原始词汇和扩展词汇
	allTerms := make(map[string]bool)
	for _, term := range result.OriginalTerms {
		allTerms[term] = true
	}
	for _, term := range result.ExpandedTerms {
		allTerms[term] = true
	}

	// 转换为切片
	var terms []string
	for term := range allTerms {
		terms = append(terms, term)
	}

	return terms, nil
}

// GetVocabularyStats 获取词表统计信息
func (vm *VocabularyManager) GetVocabularyStats() map[string]interface{} {
	if vm.builder == nil {
		return map[string]interface{}{
			"error": "词表未初始化",
		}
	}

	return vm.builder.GetVocabularyStats()
}

// printVocabResult 打印词表结果
func printVocabResult(result *vocab.UpdateResult) {
	fmt.Printf("[Vocab] 构建完成，用时: %v\n", result.Duration)
	fmt.Printf("[Vocab] 添加文件: %d, 更新文件: %d, 删除文件: %d\n",
		result.AddedFiles, result.UpdatedFiles, result.DeletedFiles)
	fmt.Printf("[Vocab] 新增词汇: %d, 删除词汇: %d\n",
		result.NewTerms, result.RemovedTerms)

	if len(result.Errors) > 0 {
		fmt.Printf("[Vocab] 错误数量: %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("[Vocab]   - %s\n", err)
		}
	}
}