// namer-cli 命令行起名工具
//
// 与 Web UI 复用同一套领域链路（NameService.GenerateWithAnalysis），
// 用于在无浏览器环境下快速测试验证起名功能。
//
// 参数优先级：命令行参数 > TOML 配置文件 > 默认值
//
// 用法示例：
//
//	# 纯命令行参数
//	go run ./cmd/namer-cli -surname 王 -gender male -birth_year 2024 -birth_month 1 -birth_day 15 -birth_hour 12
//
//	# TOML 配置文件
//	go run ./cmd/namer-cli -config cmd/namer-cli/example.toml
//
//	# 配置文件 + 命令行覆盖（surname 以命令行为准）
//	go run ./cmd/namer-cli -config cmd/namer-cli/example.toml -surname 李
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/logger"
)

var (
	configFile = flag.String("config", "", "TOML 配置文件路径（可选，命令行参数优先级更高）")
	dataFlag   = flag.String("data", "./data", "文化数据 JSON 文件目录")
	help       = flag.Bool("help", false, "显示帮助")

	// 核心输入（与 Web UI 表单一致）
	fSurname     = flag.String("surname", "", "姓氏（必填）")
	fGeneration  = flag.String("generation", "", "字辈")
	fGenPosition = flag.String("generation_position", "", "字辈位置：middle（名中）/ tail（名尾）")
	fGender      = flag.String("gender", "", "性别：male / female（必填）")
	fBirthYear   = flag.Int("birth_year", 0, "出生年份（必填，1900-2100）")
	fBirthMonth  = flag.Int("birth_month", 0, "出生月份（必填，1-12）")
	fBirthDay    = flag.Int("birth_day", 0, "出生日期（必填）")
	fBirthHour   = flag.Int("birth_hour", 0, "出生小时（必填，0-23）")
	fBirthMin    = flag.Int("birth_minute", 0, "出生分钟（0-59）")
	fBirthLoc    = flag.String("birth_location", "", "出生地")

	// 筛选条件
	fNameType    = flag.String("name_type", "", "名字类型：single / double")
	fNameLength  = flag.Int("name_length", 0, "名字长度（不含姓）：1 单字 / 2 双字（默认2）")
	fExcludeRare = flag.Bool("exclude_rare", false, "排除生僻字")
	fWuxingMatch = flag.String("wuxing_match", "", "五行偏好，逗号分隔（如：水,木）")
	fSource      = flag.String("source_classic", "", "典籍来源（shijing/chuci/poetry 等）")
	fMinStrokes  = flag.Int("min_strokes", 0, "候选字最小笔画")
	fMaxStrokes  = flag.Int("max_strokes", 0, "候选字最大笔画")
	fPoetry      = flag.Bool("include_poetry", false, "包含诗词字库")
	fClassic     = flag.Bool("include_classic", false, "包含经典字库")
	fKeywords    = flag.String("meaning_keywords", "", "寓意关键词，逗号分隔")
	fPinyinInit  = flag.String("pinyin_initial", "", "拼音首字母筛选")
	fAvoidElder  = flag.String("avoid_elder_names", "", "避讳长辈姓名，逗号分隔（排除同形/同音字）")

	// 输出控制
	fCount  = flag.Int("count", 0, "展示名字数量上限（默认20）")
	fFormat = flag.String("format", "", "输出格式：text / json（默认text）")
)

func main() {
	flag.Parse()
	if *help {
		printHelp()
		return
	}

	// 1. 收集命令行显式传入的参数名（用于优先级合并）
	explicit := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config", "data", "help":
			// 运行控制参数，不参与起名参数合并
		default:
			explicit[f.Name] = true
		}
	})

	// 2. 加载 TOML 配置（可选）
	params := &Params{}
	if *configFile != "" {
		loaded, err := LoadTOML(*configFile)
		if err != nil {
			fail(err)
		}
		fmt.Printf("已加载配置文件: %s\n", *configFile)
		params = loaded
	}

	// 3. 命令行显式参数覆盖 TOML 配置
	params.ApplyOverrides(buildFlagOverrides(), explicit)

	// 4. 填充默认值并校验
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		fail(err)
	}

	// 5. 初始化数据与服务，执行生成
	resp, err := generate(params.ToRequest(), *dataFlag)
	if err != nil {
		fail(err)
	}

	// 6. 输出结果（JSON 格式输出完整候选，text 按 count 截断展示）
	switch params.Format {
	case "json":
		out, renderErr := RenderJSON(resp)
		if renderErr != nil {
			fail(renderErr)
		}
		fmt.Println(out)
	default:
		fmt.Print(RenderText(resp, params.Count))
	}
}

// buildFlagOverrides 将命令行 flag 值收集为覆盖参数对象
func buildFlagOverrides() *Params {
	return &Params{
		Surname:         strings.TrimSpace(*fSurname),
		Generation:      strings.TrimSpace(*fGeneration),
		GenerationPosition: strings.TrimSpace(*fGenPosition),
		Gender:          strings.TrimSpace(*fGender),
		BirthYear:       *fBirthYear,
		BirthMonth:      *fBirthMonth,
		BirthDay:        *fBirthDay,
		BirthHour:       *fBirthHour,
		BirthMinute:     *fBirthMin,
		BirthLocation:   strings.TrimSpace(*fBirthLoc),
		NameType:        strings.TrimSpace(*fNameType),
		NameLength:      *fNameLength,
		ExcludeRare:     *fExcludeRare,
		WuxingMatch:     ParseCSVList(*fWuxingMatch),
		SourceClassic:   strings.TrimSpace(*fSource),
		MinStrokes:      *fMinStrokes,
		MaxStrokes:      *fMaxStrokes,
		IncludePoetry:   *fPoetry,
		IncludeClassic:  *fClassic,
		MeaningKeywords: ParseCSVList(*fKeywords),
		PinyinInitial:   strings.TrimSpace(*fPinyinInit),
		AvoidElderNames: ParseCSVList(*fAvoidElder),
		Count:           *fCount,
		Format:          strings.TrimSpace(*fFormat),
	}
}

// generate 初始化数据与领域服务后执行生成
// 装配顺序与 cmd/server/main.go 保持一致，确保 CLI 与 Web 链路行为相同：
//
//	data.Init（字表/易经/诗词/生肖）→ word.json → 同步起名索引
//	→ 注入策展好名 → fate 引擎 + NameService
func generate(req *services.GenerateRequest, dataDir string) (*services.GenerateWithAnalysisResponse, error) {
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("解析数据目录失败: %w", err)
	}

	// 1. 加载传统文化数据（hanzi.json → 易经 → 诗词 → 生肖）
	if err := data.Init(absDataDir); err != nil {
		return nil, fmt.Errorf("加载文化数据失败（目录 %s）: %w", absDataDir, err)
	}

	// 2. 加载 word.json 释义（失败不阻断，仅部分字缺释义）
	if loadErr := hanzi.LoadWordData(absDataDir); loadErr != nil {
		fmt.Fprintf(os.Stderr, "警告: 加载 word.json 失败(忽略): %v\n", loadErr)
	}

	// 3. 日志初始化到文件，避免污染终端结果输出
	if logErr := logger.Init(&logger.Config{
		Level:      "info",
		Format:     "console",
		OutputPath: "namer-cli.log",
	}); logErr != nil {
		fmt.Fprintf(os.Stderr, "警告: 日志系统初始化失败(%v)，使用标准输出\n", logErr)
	}
	defer logger.Sync()

	// 4. 同步起名分类索引到 fate 层（前置条件）
	services.SyncNamingIndexFromHanzi()

	// 5. 注入策展好名到 fate 引擎（共现分白名单，与 server 一致）
	if curatedNames := loadCuratedNames(absDataDir); len(curatedNames) > 0 {
		fate.SetCuratedNames(curatedNames)
	}

	// 6. 装配服务（内存存储即可满足测试验证场景，避免与运行中的服务锁冲突）
	cache.Init()
	store := memory.NewStore()

	fateEngine := fate.NewEngine(&services.HanziDataProvider{}, services.NewBaziAnalyzerAdapter(), fate.DefaultRaters())
	svc := services.NewNameService(
		services.WithBaziAnalyzer(&services.BaziAdapter{}),
		services.WithHexagramFinder(&services.HexagramAdapter{}),
		services.WithZiweiAnalyzer(&services.ZiweiAdapter{}),
		services.WithEnhancedAnalyzer(services.NewEnhancedNameAnalyzerAdapter(absDataDir, services.NewCuratedPersisterAdapter(store))),
		services.WithZodiacFinder(&services.ZodiacAdapter{}),
		services.WithCache(cache.GetCache()),
		services.WithFateService(services.NewFateNameService(fateEngine)), // fate 引擎优先（与 server 一致）
	)

	return svc.GenerateWithAnalysis(context.Background(), req)
}

// loadCuratedNames 读取 curated_names.json 的 name 字段列表
func loadCuratedNames(dataDir string) []string {
	path := filepath.Join(dataDir, "curated_names.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names
}

// fail 输出错误信息并以非零码退出
func fail(err error) {
	fmt.Fprintf(os.Stderr, "\n❌ 错误: %v\n", err)
	os.Exit(1)
}

// printHelp 打印中文使用说明
func printHelp() {
	fmt.Println(`起名 CLI（namer-cli）— 命令行方式测试验证起名功能

用法:
  go run ./cmd/namer-cli [flags]
  namer-cli.exe [flags]

参数来源与优先级:
  命令行参数 > TOML 配置文件(-config 指定) > 默认值

常用示例:
  # 纯命令行参数生成
  go run ./cmd/namer-cli -surname 王 -gender male \
    -birth_year 2024 -birth_month 1 -birth_day 15 -birth_hour 12

  # 使用 TOML 配置文件
  go run ./cmd/namer-cli -config cmd/namer-cli/example.toml

  # 配置文件 + 命令行覆盖（姓氏以命令行为准）
  go run ./cmd/namer-cli -config cmd/namer-cli/example.toml -surname 李

  # JSON 输出（结构与 Web API 一致，便于脚本对比验证）
  go run ./cmd/namer-cli -config cmd/namer-cli/example.toml -format json

核心参数:
  -surname              姓氏（必填）
  -generation           字辈
  -generation_position  字辈位置：middle（名中）/ tail（名尾）
  -gender               性别：male / female（必填）
  -birth_year/month/day/hour/minute  出生年月日时分（年月日时必填）

筛选参数:
  -name_length          名字长度（不含姓）：1 单字 / 2 双字，默认 2
  -exclude_rare         排除生僻字
  -wuxing_match         五行偏好，逗号分隔（如 水,木）
  -min_strokes/-max_strokes  候选字笔画范围
  -source_classic       典籍来源（shijing/chuci/poetry 等）
  -meaning_keywords     寓意关键词，逗号分隔
  -pinyin_initial       拼音首字母
  -avoid_elder_names    避讳长辈姓名，逗号分隔（排除同形/同音字）

输出参数:
  -format               text（默认）/ json
  -count                text 展示数量上限，默认 20

其他:
  -data                 文化数据 JSON 目录，默认 ./data（在 backend 目录下运行）
  -config               TOML 配置文件路径`)
}
