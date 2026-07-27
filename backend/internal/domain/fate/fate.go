package fate

import "context"

// Fate 起名引擎顶层接口
//
// 借鉴 fate-main (github.com/babyname/fate/v4) 的 Fate interface 设计:
//   - NewSession(): 创建默认命名会话
//   - NewSessionWithFilter(filter): 创建带自定义过滤条件的会话
//
// 使用方式:
//
//	f, _ := fate.New(cfg)
//	s := f.NewSessionWithFilter(filter)
//	s.Start(ctx, input)
//	s.Wait()
//	result := s.Result()
type Fate interface {
	// NewSession 创建默认命名会话
	NewSession() Session
	// NewSessionWithFilter 创建带过滤条件的命名会话
	NewSessionWithFilter(filter Filter) Session
}

// Session 命名会话生命周期接口
//
// 提供完整的命名流程控制:
//   - Start(ctx, input):  异步开始生成
//   - Wait():             等待生成完成
//   - Result():           获取生成结果
//   - State():            查询当前状态
//   - Stop():             提前终止
//
// 负面反馈:
//   - ExcludeChar(char):    排除某个名字用字（后续生成不再包含）
//   - ExcludeCombo(c1,c2):  排除某两个字组合出现在名字中
//   - ClearExclusions():    清空所有排除项
//   - ExcludedChars():      获取已排除的字列表
type Session interface {
	// Start 开始名字生成
	// 异步非阻塞，完成后可通过 Wait() 等待
	Start(ctx context.Context, input *Input) error

	// Wait 阻塞等待生成完成
	Wait() error

	// Result 获取生成结果
	// 仅在 State() == SessionStateFinish 后调用有效
	Result() *Output

	// State 返回当前会话状态
	State() SessionState

	// Stop 提前终止生成
	Stop() error

	// ——— 负面反馈（排除不喜欢的结果） ———

	// ExcludeChar 排除指定名字用字
	// 后续重新生成时，所有包含该字的候选将被过滤
	ExcludeChar(char string)

	// ExcludeCombo 排除两个字同时出现在名字中的组合
	// 后续重新生成时，跳过同时包含这两个字的候选
	ExcludeCombo(c1, c2 string)

	// ClearExclusions 清空所有排除项
	ClearExclusions()

	// ExcludedChars 获取已排除的字符列表
	ExcludedChars() []string
}

// Output 命名会话输出结果
type Output struct {
	Input          *Input           `json:"input"`           // 输入参数
	FateData       *FateData        `json:"fate_data"`       // 八字分析数据
	TopNames       []NameResult     `json:"top_names"`       // Top-N 完整分析
	ExcellentTable *ExcellentTable  `json:"-"`               // 全部候选数据（流式 Top-N）
	TotalCount     int              `json:"total_count"`     // 候选总数
}

// EngineFactory 引擎工厂，创建 Fate 实例
// 具体实现由 infrastructure 层提供
type EngineFactory func(raters []Rater) (Fate, error)

var defaultRaters = DefaultRaters()

// NewDefaultFate 使用默认评分器创建 Fate 实例
func NewDefaultFate(factory EngineFactory) (Fate, error) {
	return factory(defaultRaters)
}

// NewFateWithRaters 使用自定义评分器创建 Fate 实例
func NewFateWithRaters(factory EngineFactory, raters []Rater) (Fate, error) {
	return factory(raters)
}
