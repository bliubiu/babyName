package fate

// curated_names.go 策展好名集合（共现分白名单依据）
//
// 解决的问题（verify_fate 验证暴露）：
//   - GetBigramScore 依据典籍同句共现给分（《论语》"仲尼曰"→10分），
//     但"同句共现"不等于"好名字"——「忘莫/至律/值接/灼唐/荆拂/百亩」
//     等无实义组合仅因典籍高频共现就被推上 85+ 高分。
//   - 数据层 curated_names.json 是经过人工策展的优质名字库（候选名库），
//     只有策展认可的搭配才值得获得满分共现加成。
//
// 设计：
//   - fate 层维护一个包级策展好名集合，由上层（verify_fate/server 启动）注入。
//   - IsCuratedName 正反序检测：命中策展库的组合，共现分全额加成；
//     未命中的典籍共现组合，共现分降权（见 WenHuaRater）。

import "sync"

var (
	curatedNameMu sync.RWMutex
	// curatedNameSet 策展好名集合（key 为 2 字名字原文）
	curatedNameSet = map[string]bool{}
	// curatedCharSet 策展好名中出现过的单字集合（统计型门禁依据）
	// 由 SetCuratedNames 同步构建；用于判断某字是否有策展认可的命名价值。
	curatedCharSet = map[string]bool{}
)

// SetCuratedNames 注入策展好名集合（幂等，可多次调用覆盖）
// 由上层从 curated_names.json 加载 name 字段后调用
func SetCuratedNames(names []string) {
	curatedNameMu.Lock()
	defer curatedNameMu.Unlock()
	curatedNameSet = make(map[string]bool, len(names))
	curatedCharSet = make(map[string]bool, len(names)*2)
	for _, n := range names {
		if n == "" {
			continue
		}
		curatedNameSet[n] = true
		// 同时收录名字中的每个字，构建策展字符池
		for _, r := range n {
			curatedCharSet[string(r)] = true
		}
	}
}

// CuratedNameCount 返回当前策展好名数量（调试/验证用）
func CuratedNameCount() int {
	curatedNameMu.RLock()
	defer curatedNameMu.RUnlock()
	return len(curatedNameSet)
}

// IsCuratedChar 判断单字是否出现在策展好名集合中（统计型门禁依据）
// 「姐/赌/货/了」等荒谬字从未出现在策展库，应得 0 笔策展命名认可；
// 而「浩/然/文/子」等策展常用好字在池内，不受惩罚。
// 未调用 SetCuratedNames 时返回 false（不影响既有逻辑）。
func IsCuratedChar(ch string) bool {
	if ch == "" {
		return false
	}
	curatedNameMu.RLock()
	defer curatedNameMu.RUnlock()
	return curatedCharSet[ch]
}

// IsCuratedName 判断双字组合是否为策展认可的好名（正反序均检测）
// 未调用 SetCuratedNames 时返回 false（不影响既有逻辑）
func IsCuratedName(c1, c2 string) bool {
	if c1 == "" || c2 == "" {
		return false
	}
	curatedNameMu.RLock()
	defer curatedNameMu.RUnlock()
	if curatedNameSet[c1+c2] {
		return true
	}
	return curatedNameSet[c2+c1]
}