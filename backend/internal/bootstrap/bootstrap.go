// Package bootstrap 桌面端装配层
//
// 把「配置 → 存储 → 物料装载 → 服务装配 → Wails 绑定」串成一条启动链，
// 替代 cmd/server/main.go 中面向 HTTP 部署的装配逻辑：
//
//   - 不加载 application.yml / 不启 Gin / 不设静态目录与限流；
//   - SQLite 为用户数据（收藏/历史/分享）持久化载体，缺失可回落内存（测试）；
//   - 物料以 SQLite 为唯一真源（docs/31 ADR-008）；首跑时库为空则从
//     data/ 目录的 JSON 物料自动播种（SeedFromJSON），此后指纹短路幂等。
package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
	"name/internal/application/services"
	"name/internal/desktop"
	"name/internal/domain/classics"
	"name/internal/domain/fate"
	"name/internal/domain/name"
	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/database/sqlite"
	"name/internal/infrastructure/logger"
	"name/internal/paths"
)

// Options 装配选项
type Options struct {
	// DBPath SQLite 库路径；空 = 内存存储（测试/无持久化场景）
	DBPath string
	// DataDir 物料源目录（JSON/CSV）；空 = DATA_DIR 环境变量 > 向上探测仓库根 data/ > cwd/data
	DataDir string
	// SeedFromJSON 首跑时库内物料为空则从 DataDir 的 JSON 物料自动播种
	SeedFromJSON bool
}

// Result 装配结果
type Result struct {
	// Store 数据存储（调用方经 Close 释放）
	Store database.Store
	// App 桌面端绑定层（Wails Bind 入口）
	App *desktop.App
}

// Close 释放存储连接；重复调用安全
func (r *Result) Close() error {
	if r.Store == nil {
		return nil
	}
	if closer, ok := r.Store.(interface{ Close() error }); ok {
		r.Store = nil // 置空后再关闭，重复 Close 直接短路
		return closer.Close()
	}
	return nil
}

// Run 执行装配。任何一步失败即返回错误（fail-fast），不静默降级启动。
func Run(opts Options) (*Result, error) {
	// 1. 解析物料目录（显式指定 > DATA_DIR > 仓库根探测 > cwd/data）
	dataDir, err := ResolveDataDir(opts.DataDir)
	if err != nil {
		return nil, err
	}

	// 1. 初始化数据存储：SQLite 优先，缺失/失败回落内存
	store, err := initStore(opts.DBPath, dataDir)
	if err != nil {
		return nil, err
	}

	// 2. 首跑播种：SQLite 物料库为空时从 JSON 物料导入（幂等，指纹短路）
	if s, ok := store.(*sqlite.Store); ok && opts.SeedFromJSON {
		if err := seedMaterialsIfEmpty(s, dataDir); err != nil {
			_ = closeQuietly(store)
			return nil, fmt.Errorf("物料播种失败: %w", err)
		}
	}

	// 3. 装载传统文化物料（ADR-008：必需物料缺失即拒绝启动）
	if err := loadCulturalData(dataDir, store); err != nil {
		_ = closeQuietly(store)
		return nil, err
	}
	logger.Info("传统文化物料已装载", logger.String("数据目录", dataDir))

	// 诗词索引预热：把首个生成请求的索引构建尖峰前移到启动期
	classics.BuildPoetryIndex()

	// 4. 服务装配（与原 newServices 同口径，面向桌面端裁剪）
	app, err := assembleServices(store, dataDir)
	if err != nil {
		_ = closeQuietly(store)
		return nil, err
	}

	return &Result{Store: store, App: app}, nil
}

// --- 数据目录解析 ---

// DataDirSource 数据目录的解析来源
//
// 区分 dev（仓库结构）与打包态是目录语义的关键分叉：dev 的用户数据
// （namer.db/logs）沿用「物料目录上一级」的仓库根布局；打包态收敛到
// exe 同级 data/ 文件夹内，安装目录整体自包含、可整体拷贝迁移。
type DataDirSource int

const (
	// SourceExplicit 显式参数（-data / Options.DataDir）
	SourceExplicit DataDirSource = iota
	// SourceEnvVar DATA_DIR 环境变量
	SourceEnvVar
	// SourceDevRepo 仓库根 data/（源码结构向上探测命中，仅 dev）
	SourceDevRepo
	// SourceExeDir exe 同级 data/（打包分发布局）
	SourceExeDir
	// SourceCwdDir cwd/data（便携回落）
	SourceCwdDir
)

// ResolveDataDir 解析传统文化数据目录（与 cmd/server 的 cwdDataDir 同口径）：
// 显式指定 > DATA_DIR 环境变量 > 向上探测仓库根 data/ > exe 同级 data/ > cwd/data。
// 皆无结果时返回错误——物料目录是启动的硬前提，不猜测。
//
// 供桌面入口 main.go 复用，保证入口与装配层用同一套目录语义。
func ResolveDataDir(explicit string) (string, error) {
	dir, _, err := ResolveDataDirDetailed(explicit)
	return dir, err
}

// ResolveDataDirDetailed 解析数据目录并报告来源（main.go 据此选择 db/logs 默认位置）
func ResolveDataDirDetailed(explicit string) (string, DataDirSource, error) {
	if dir := strings.TrimSpace(explicit); dir != "" {
		return dir, SourceExplicit, nil
	}
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir, SourceEnvVar, nil
	}
	if dir := findDataDir(); dir != "" {
		return dir, SourceDevRepo, nil
	}
	if dir := findDataDirFromExe(); dir != "" {
		return dir, SourceExeDir, nil
	}
	if dir := findDataDirFromCwd(); dir != "" {
		return dir, SourceCwdDir, nil
	}
	return "", SourceExplicit, fmt.Errorf("未找到传统文化数据目录，请通过 DATA_DIR 环境变量或 Options.DataDir 指定")
}

// findDataDir 以本包源文件位置为起点向上探测仓库根 data/（与测试侧共用 paths.FindDataDir）
//
// 提取为包级变量是为了可测：模拟「无仓库结构、探测无果」场景时可替换为临时路径。
var findDataDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return paths.FindDataDir(file)
}

// findDataDirFromExe 探测 exe 同级 data/（打包分发布局：物料随安装器落在 exe 旁）
//
// 提取为包级变量是为了可测：测试注入临时目录布局模拟打包态。
var findDataDirFromExe = func() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	return paths.FindDataDirFromExe(exePath)
}

// findDataDirFromCwd 回落 cwd/data（便携兜底：从任意工作目录启动时随遇而安）
func findDataDirFromCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cwd, "data")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return ""
}

// DefaultDBPath 按数据目录来源推导 SQLite 库默认路径（flag/env 覆盖优先于本默认值）
//
// dev 仓库结构：沿用「物料目录上一级」= 仓库根 namer.db（历史行为不变）；
// 打包态：收敛到 data 文件夹内，与物料同处 exe 同级 data/，整体自包含。
func DefaultDBPath(dataDir string, source DataDirSource) string {
	if source == SourceDevRepo {
		return filepath.Join(filepath.Dir(dataDir), "namer.db")
	}
	return filepath.Join(dataDir, "namer.db")
}

// DefaultLogDir 按数据目录来源推导日志目录默认位置（flag/env 覆盖优先于本默认值）
//
// 分叉规则与 DefaultDBPath 一致：dev 沿用仓库根 logs/，打包态收敛 data/logs。
func DefaultLogDir(dataDir string, source DataDirSource) string {
	if source == SourceDevRepo {
		return filepath.Join(filepath.Dir(dataDir), "logs")
	}
	return filepath.Join(dataDir, "logs")
}

// --- 存储 ---

// initStore 初始化数据存储：SQLite 优先，失败回落内存（不阻断启动）
func initStore(dbPath, dataDir string) (database.Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		logger.Warn("未指定 SQLite 库路径，使用内存存储（收藏/历史不持久化）")
		return memory.NewStore(), nil
	}
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}
	dbStore, err := sqlite.NewStore(dbPath, dataDir)
	if err != nil {
		logger.Warn("打开 SQLite 失败，回落内存存储", logger.ErrField(err))
		return memory.NewStore(), nil
	}
	logger.Info("使用 SQLite 存储", logger.String("db", dbPath))
	return dbStore, nil
}

// seedMaterialsIfEmpty 库内物料为空时从 JSON 物料播种（幂等：已导入的物料指纹短路）
func seedMaterialsIfEmpty(s *sqlite.Store, dataDir string) error {
	rows, err := s.HanziRowCount()
	if err != nil {
		return fmt.Errorf("查询字库行数失败: %w", err)
	}
	if rows > 0 {
		logger.Info("SQLite 物料库已有数据，跳过播种", logger.Int("字库行数", rows))
		return nil
	}
	logger.Info("SQLite 物料库为空，开始从 JSON 物料播种", logger.String("数据目录", dataDir))
	report, err := s.Import(sqlite.ImportOptions{DataDir: dataDir})
	if err != nil {
		return err
	}
	if report.Failed > 0 {
		var details []string
		for _, r := range report.Results {
			if r.Err != "" {
				details = append(details, r.Material+": "+r.Err)
			}
		}
		return fmt.Errorf("有 %d 项物料导入失败: %s", report.Failed, strings.Join(details, "；"))
	}
	logger.Info("物料播种完成", logger.Int("物料数", len(report.Results)))
	return nil
}

// --- 物料装载 ---

// loadCulturalData 装载传统文化物料：SQLite 为唯一真源（ADR-008），
// 库为内存存储（无 DBLoader）时走纯 JSON 路径（测试/无库场景）。
func loadCulturalData(dataDir string, store database.Store) error {
	if loader, ok := store.(data.DBLoader); ok {
		if err := data.InitWithDB(dataDir, loader); err != nil {
			return fmt.Errorf("%w；恢复命令: go run ./cmd/materials import --data %q --db %q", err, dataDir, "")
		}
		return nil
	}
	return data.Init(dataDir)
}

// warnClassicsExtractGaps 经典字集提取完整性校验（docs/44 遗留④）：
// 已登记字集物料在库中行数为 0 时打 ERROR 日志暴露静默遗漏，不阻断启动。
func warnClassicsExtractGaps(store database.Store) {
	sc, ok := store.(*sqlite.Store)
	if !ok {
		return
	}
	stats := sc.ClassicsStats()
	if stats.TotalExtracts == 0 {
		return
	}
	var missing []string
	for _, m := range classics.RegisteredMaterials() {
		if stats.ExtractByMaterial[m] == 0 {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		logger.Error("经典字集提取存在静默遗漏，出典加分与来源筛选将降级",
			logger.String("缺失来源", strings.Join(missing, ", ")),
			logger.String("修复指引", "重新执行物料播种后重启应用"),
		)
	}
}

// --- 服务装配 ---

// assembleServices 装配领域服务并接入桌面端绑定层
// （与 cmd/server 的 newServices 同口径：fate 引擎优先 + 自学习 + 门禁闭环）
func assembleServices(store database.Store, dataDir string) (*desktop.App, error) {
	baziAdapter := &services.BaziAdapter{}
	hexagramAdapter := &services.HexagramAdapter{}
	ziweiAdapter := &services.ZiweiAdapter{}
	zodiacAdapter := &services.ZodiacAdapter{}

	// 候选名库管理器（精选名 + 自学习 + 标准字表）
	ndb, ndbErr := name.NewNameDB(dataDir, name.WithCuratedPersister(services.NewCuratedPersisterAdapter(store)))
	if ndbErr != nil {
		logger.Warn("构造候选名库失败，收藏自学习与候选名功能降级", logger.ErrField(ndbErr))
	}

	// fate 引擎（新一代引擎：完整 Rater 评分链、避讳长辈、负面反馈、更大候选池）
	services.SyncNamingIndexFromHanzi()
	if curatedNames := loadCuratedNamesForFate(store); len(curatedNames) > 0 {
		fate.SetCuratedNames(curatedNames)
		logger.Info("已注入策展好名到 fate 引擎", zap.Int("count", len(curatedNames)))
	}

	// SQLite 下沉：库有字库行时启用 SQL 过滤，否则回落 Go 端全表过滤（库空甄别见 docs/29 B8）
	hanziProvider := &services.HanziDataProvider{}
	if store != nil && services.HasSQLiteCharStore(store) {
		charStore := services.AsSQLiteCharStore(store)
		switch rows, err := charStore.HanziRowCount(); {
		case err != nil:
			logger.Warn("查询 SQLite 字库行数失败，SQLite 汉字过滤未启用（走 Go 端过滤）", logger.ErrField(err))
		case rows == 0:
			logger.Warn("SQLite 字库为空，SQLite 汉字过滤未启用，回退 Go 端全表过滤")
		default:
			hanziProvider.SetSQLFilter(services.NewSQLiteHanziFilter(charStore))
			logger.Info("SQLite 汉字过滤已启用", logger.Int("字库行数", rows))
		}
	}

	fateEngine := fate.NewEngine(hanziProvider, services.NewBaziAnalyzerAdapter(), fate.DefaultRaters())
	fateSvc := services.NewFateNameService(fateEngine,
		services.WithFateBaziAnalyzer(baziAdapter),
		services.WithFateHexagramFinder(hexagramAdapter),
		services.WithFateZiweiAnalyzer(ziweiAdapter),
	)

	nameSvc := services.NewNameService(
		services.WithBaziAnalyzer(baziAdapter),
		services.WithHexagramFinder(hexagramAdapter),
		services.WithZiweiAnalyzer(ziweiAdapter),
		services.WithNameDB(ndb),
		services.WithZodiacFinder(zodiacAdapter),
		services.WithFateService(fateSvc), // fate 引擎优先（新引擎更完善）
	)

	// 异步生成任务（长任务体验）
	taskSvc := services.NewTaskService(nameSvc)

	// 收藏服务（NameDB 自学习 + 门禁复审回流 + 指标闭环）
	favSvc := services.NewFavoriteService(store)
	if n := nameSvc.GetNameDB(); n != nil {
		favSvc.SetNameDB(n)
	}
	if r, ok := store.(services.GateReviewRecorder); ok {
		favSvc.SetGateReviewRecorder(r)
	}
	if p, ok := store.(fate.GateMetricProvider); ok {
		fate.SetGateMetricProvider(p)
	}

	warnClassicsExtractGaps(store)

	// 物料管理器（SQLite 专属：状态查看 + 手动重导；内存存储降级为 nil）
	var materials desktop.MaterialManager
	if s, ok := store.(*sqlite.Store); ok {
		materials = newMaterialManager(s, s, dataDir)
	}

	return desktop.New(
		nameSvc,
		desktop.NewTaskRunner(taskSvc),
		services.NewHistoryService(store),
		favSvc,
		services.NewReportService(),
		services.NewShareService(store),
		desktop.NewCharacterSource(),
		desktop.NewNameStatSource(),
		materials,
	), nil
}

// --- 物料管理适配器 ---

// materialStateReader 物料库内状态只读面
type materialStateReader interface {
	MaterialStates() ([]sqlite.MaterialState, error)
}

// materialImporter 物料重新导入面
type materialImporter interface {
	Import(sqlite.ImportOptions) (*sqlite.ImportReport, error)
}

// sqliteMaterialManager sqlite.Store 物料面到 desktop.MaterialManager 的适配
type sqliteMaterialManager struct {
	reader   materialStateReader
	importer materialImporter
	dataDir  string
}

// newMaterialManager 构造物料管理适配器
func newMaterialManager(reader materialStateReader, importer materialImporter, dataDir string) desktop.MaterialManager {
	return sqliteMaterialManager{reader: reader, importer: importer, dataDir: dataDir}
}

// MaterialStates 物料库内状态列表（物料/版本/指纹/行数/导入时间）
func (m sqliteMaterialManager) MaterialStates() ([]desktop.MaterialState, error) {
	states, err := m.reader.MaterialStates()
	if err != nil {
		return nil, err
	}
	result := make([]desktop.MaterialState, 0, len(states))
	for _, s := range states {
		result = append(result, desktop.MaterialState{
			Material:   s.Material,
			Version:    s.Version,
			Checksum:   s.Checksum,
			RowCount:   s.RowCount,
			ImportedAt: s.ImportedAt,
		})
	}
	return result, nil
}

// Reimport 按装配时确定的物料目录重新导入（指纹短路幂等）
func (m sqliteMaterialManager) Reimport() (*desktop.MaterialImportReport, error) {
	report, err := m.importer.Import(sqlite.ImportOptions{DataDir: m.dataDir})
	if err != nil {
		return nil, err
	}
	items := make([]desktop.MaterialImportItem, 0, len(report.Results))
	for _, r := range report.Results {
		items = append(items, desktop.MaterialImportItem{
			Material: r.Material,
			Version:  r.Version,
			Rows:     r.Rows,
			Skipped:  r.Skipped,
			Err:      r.Err,
		})
	}
	return &desktop.MaterialImportReport{
		Items:   items,
		Written: report.Written,
		Skipped: report.Skipped,
		Failed:  report.Failed,
		Elapsed: report.Elapsed,
	}, nil
}

// loadCuratedNamesForFate 从库读取策展名注入 fate 引擎共现分白名单（运行时只读 SQLite）
func loadCuratedNamesForFate(store database.Store) []string {
	if store == nil {
		return nil
	}
	entries, err := store.LoadAllCuratedNames()
	if err != nil {
		logger.Warn("从库读取策展名失败，fate 引擎无策展白名单", logger.ErrField(err))
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

// closeQuietly 关闭存储连接（装配失败路径的清理，错误仅记日志）
func closeQuietly(store database.Store) error {
	if store == nil {
		return nil
	}
	if closer, ok := store.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Warn("装配失败后关闭存储失败", logger.ErrField(err))
		}
	}
	return nil
}
