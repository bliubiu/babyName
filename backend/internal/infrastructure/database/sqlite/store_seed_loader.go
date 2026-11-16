package sqlite

// 种子库只读装载器（ADR-010「A + 单库」形态）
//
// 背景：物料的唯一真源是 namer.db（docs/31 ADR-008）。离线工具与门禁原本
// 走 data.Init → initFromJSON 的纯 JSON 内存路径，导致 CI 必须入库十几 MB
// 频率 JSON，且 golden 快照会随入库物料子集漂移。单库路线下，测试、门禁、
// 运行期、安装包共用同一份二进制库，JSON 只保留为「构建期导入输入」。
//
// 为何不直接 NewStore(dbPath)：NewStore 会 PRAGMA journal_mode=WAL、建表、
// 执行迁移，属于写路径。多个 go test 包是独立进程，若同时打开入库的种子库，
// 会争用写锁并改脏 git 工作区。MaterialLoader 只开一条 mode=ro 连接，
// 复用 *Store 的全部 DB 装载方法（LoadMaterialsFromDB / LoadClassicsFromDB
// 只读 s.db），因此：
//   - 并发安全：任意多个测试进程可同时装载；
//   - 零污染：不会生成 -wal / -shm，也不会改写库头；
//   - 零重复代码：物料读取逻辑仍只有 store_materials_load.go 一份。

import (
	"database/sql"
	"fmt"
)

// MaterialLoader 只读种子库装载器，实现 data.DBLoader / data.ClassicsLoader
//
// 内嵌 *Store 是为了复用其物料装载方法；writeDB 指向同一条只读连接，
// 使意外调用到写方法时得到「attempt to write a readonly database」的明确
// 错误，而不是 nil 解引用 panic。物料装载路径不会触碰 writeDB。
type MaterialLoader struct {
	*Store
}

// NewMaterialLoader 以只读方式打开种子库
//
// dbPath 必须是已封存的单文件库（journal_mode=DELETE，无 -wal/-shm 残留），
// 由 tools/build_seed 生成；只读连接不做任何 PRAGMA 写入。
func NewMaterialLoader(dbPath string) (*MaterialLoader, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("种子库路径为空")
	}
	db, err := sql.Open("sqlite", dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("打开只读种子库失败: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接只读种子库失败: %w", err)
	}
	return &MaterialLoader{Store: &Store{db: db, writeDB: db}}, nil
}

// LoadCoreCharDataFromDB 只装载字库与康熙笔画，**不覆盖**其他领域数据
//
// 用途：yijing 包的 docs 金标准对拍针对的是代码内置 HexagramList，若调用
// LoadMaterialsFromDB 会用库内卦象把它替换掉，导致对拍对象从「内置数据」
// 漂移成「物料数据」，测试语义被破坏。这里只灌字库（梅花易数取笔画所需），
// 保留内置卦象不动。
//
// 注意：这是测试专用窄入口，生产装配仍必须走 data.InitWithDB 全量装载。
func (l *MaterialLoader) LoadCoreCharDataFromDB() {
	l.loadKangxiFromDB()
	l.loadHanziFromDB()
}

// Close 释放只读连接（重复调用安全，由 database/sql 保证）
func (l *MaterialLoader) Close() error {
	if l == nil || l.Store == nil || l.Store.db == nil {
		return nil
	}
	return l.Store.db.Close()
}
