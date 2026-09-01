package hanzi

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// KangxiStrokesMap 汉字 → 康熙字典笔画
// 来自 data/raw/kangxi-strokecount.csv（Kawai Lo 维护的 MIT 许可数据）。
// 加载后供 fate.Character.KangxiStroke 字段填充，
// 满足传统易学（河图数理、易经卦象）的笔画口径。
var (
	kangxiMu            sync.RWMutex
	kangxiStrokesMap    = map[string]int{}
	kangxiStrokesLoaded bool
)

// LoadKangxiStrokesFromCSV 从 data/raw/kangxi-strokecount.csv 加载康熙笔画
//
// 文件来源：https://github.com/breezyreeds/kangxi-strokecount
// 格式：UTF-8 (with BOM) + 4 行 MIT License 注释 + header (CodePoint,Value,Character,Strokes)
// 包含 ~63700 个汉字（含繁体、日韩、异体字等）。
//
// 路径约定：CSV 属于生成原料，按项目 .gitignore 约定置于 backend/data/raw/。
// 兼容旧路径 dataDir/kangxi-strokecount.csv 以方便测试与历史兼容。
//
// 数据缺失时降级为空映射（仅警告、不阻断启动），与 LoadNamingQualityFromJSON 一致；
// 文件存在但解析失败时返回错误以暴露数据损坏。
//
// 加载是幂等的：重复调用覆盖既有集合，可用于热更新。
func LoadKangxiStrokesFromCSV(dataDir string) error {
	// 优先 raw 路径（生产约定），其次 dataDir 根（旧兼容）
	candidates := []string{
		filepath.Join(dataDir, "raw", "kangxi-strokecount.csv"),
		filepath.Join(dataDir, "kangxi-strokecount.csv"),
	}
	var path string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		kangxiMu.Lock()
		kangxiStrokesMap = make(map[string]int)
		kangxiStrokesLoaded = true
		kangxiMu.Unlock()
		fmt.Fprintf(os.Stderr, "警告: 康熙笔画表 %s 不存在，已降级为简体笔画兜底\n", candidates[0])
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开康熙笔画表失败: %w", err)
	}
	defer f.Close()

	// 跳过 BOM（如果存在）+ 前 4 行 MIT License 注释
	// 注释行形如 "MIT License" / "Copyright..." / URL / 空行
	br := bufio.NewReader(f)
	// 简化策略：交给 csv.Reader 直接处理，它会读到第一个合法 header 即开始解析。
	// 我们的实现：手动吃掉前 4 行注释，再交给 csv.Reader。
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek 康熙笔画表失败: %w", err)
	}
	br = bufio.NewReader(f)

	// 跳过 BOM
	if b, _ := br.Peek(3); len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		br.Discard(3)
	}

	// 跳过 4 行注释（MIT License 头）
	for i := 0; i < 4; i++ {
		if _, err := readLine(br); err != nil && err != io.EOF {
			return fmt.Errorf("跳过康熙笔画表注释行 %d 失败: %w", i+1, err)
		}
	}

	// 从第 5 行开始是 CSV header + 数据
	reader := csv.NewReader(br)
	reader.FieldsPerRecord = 4 // CodePoint,Value,Character,Strokes
	reader.TrimLeadingSpace = true

	newMap := make(map[string]int, 8192)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("解析康熙笔画表失败: %w", err)
		}
		if len(record) < 4 {
			continue
		}
		char := record[2]
		if char == "" {
			continue
		}
		strokes, err := strconv.Atoi(record[3])
		if err != nil {
			// 单行解析失败不阻断（数据可能含特殊行）
			continue
		}
		// 多音字/异体字按首次出现为准（行序由 CodePoint 单调递增决定）
		if _, exists := newMap[char]; !exists {
			newMap[char] = strokes
		}
	}

	kangxiMu.Lock()
	kangxiStrokesMap = newMap
	kangxiStrokesLoaded = true
	kangxiMu.Unlock()
	return nil
}

// readLine 读取一行（含 \r\n 或 \n），返回字节切片
func readLine(r *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return line, err
		}
		if b == '\n' {
			return line, nil
		}
		line = append(line, b)
	}
}

// GetKangxiStrokes 查询单字康熙笔画，未收录返回 0
//
// 注意：0 必须由调用方判定为"未收录"，与"0 笔画字"（无）混淆。
// 当前《康熙字典》中所有汉字笔画 ≥ 1，0 即视为未收录。
func GetKangxiStrokes(char string) int {
	kangxiMu.RLock()
	defer kangxiMu.RUnlock()
	return kangxiStrokesMap[char]
}

// IsKangxiStrokesLoaded 康熙笔画表是否已加载（用于诊断/测试跳过逻辑）
func IsKangxiStrokesLoaded() bool {
	kangxiMu.RLock()
	defer kangxiMu.RUnlock()
	return kangxiStrokesLoaded
}

// KangxiStrokesCount 返回康熙笔画表已收录汉字数（诊断/测试用）
func KangxiStrokesCount() int {
	kangxiMu.RLock()
	defer kangxiMu.RUnlock()
	return len(kangxiStrokesMap)
}