package database

import (
	"time"

	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
)

// PageResult 通用分页结果
type PageResult[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// HistoryStore 历史记录存储接口
//
// 读/删方法均以 sessionID 作用域：历史记录含生辰与完整生成结果，
// 任何跨会话读取或删除都必须被存储层拒绝（不能只靠上层过滤）。
type HistoryStore interface {
	SaveHistory(record *HistoryRecord) string
	GetHistory(sessionID string) []*HistoryRecord
	GetHistoryPage(sessionID string, page, limit int) ([]*HistoryRecord, int, error)
	GetHistoryByID(sessionID, id string) *HistoryRecord
	DeleteHistory(sessionID, id string) error
	BatchDeleteHistory(sessionID string, ids []string) error
}

// ShareStore 分享快照存储接口（2026-11-10 结果页动作区接线）
//
// 与 history/favorites 不同：分享快照不含会话归属，任何拿到随机 id 的人都能读，
// 因此只允许存放前端裁剪过的**展示摘要**（不含出生地、长辈姓名等输入信息）。
type ShareStore interface {
	SaveShare(record *ShareRecord) error
	GetShare(id string) (*ShareRecord, error)
}

// ShareRecord 分享快照记录。Payload 为 JSON 字符串，由应用层做结构与大小校验。
type ShareRecord struct {
	ID        string    `json:"id"`
	Payload   string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// FavoriteStore 收藏存储接口
type FavoriteStore interface {
	// SaveFavorite 保存收藏；写入失败返回空 ID 与错误（此前吞错返回假 ID，
	// 服务层误判为成功，收藏静默丢失——docs/29 A6）
	SaveFavorite(record *FavoriteRecord) (string, error)
	GetFavorites(sessionID string) []*FavoriteRecord
	GetFavoritesPage(sessionID string, page, limit int) ([]*FavoriteRecord, int, error)
	DeleteFavorite(sessionID, id string) error
	BatchDeleteFavorite(sessionID string, ids []string) error
	GetFavoriteByName(sessionID, surname, givenName string) *FavoriteRecord
}

// HanziStore 汉字查询接口
type HanziStore interface {
	GetHanziByChar(char string) *Hanzi
	GetHanziByWuxing(wuxing string) []*Hanzi
	GetHanziByStrokes(min, max int) []*Hanzi
	SearchHanzi(keyword string, limit int) []*Hanzi
}

// FeedbackStore 名字反馈存储接口
type FeedbackStore interface {
	SaveNameRequest(record *NameRequest) int64
	GetNameRequestByHash(hash string) *NameRequest
	SaveNameFeedback(record *NameFeedback) int64
	GetNameFeedbackByRequestID(requestID int64) []*NameFeedback
	GetAlgorithmPerformance() ([]*AlgorithmPerformance, error)
}

// YijingStore 易经存储接口
type YijingStore interface {
	GetHexagramByNumber(id int) *yijing.Hexagram
	GetHexagrams() []yijing.Hexagram
}

// ZodiacStore 生肖存储接口
type ZodiacStore interface {
	GetZodiacByName(name string) *zodiac.Zodiac
	GetZodiacs() []zodiac.Zodiac
}

// CuratedStore 精选名存储接口
type CuratedStore interface {
	SaveCuratedName(name string, pinyin string, gender string, score float64, source string) error
	LoadAllCuratedNames() ([]CuratedNameEntry, error)
	IsCurated(name string) (bool, error)
	DeleteCuratedName(name string) error
}

// CuratedNameEntry 精选名条目
//
// 由「单字库」扩列至携带 meaning/wuxing/styles/tags（docs/31 ADR-008）：
// 这些字段在 curated_names.json 中确有数据，只落 5 列会在「运行时改读库」后
// 丢失策展释义/五行/风格标签，使 SearchCuratedNames / GetStyles 等退化。
type CuratedNameEntry struct {
	Name    string   `json:"name"`
	Pinyin  string   `json:"pinyin,omitempty"`
	Gender  string   `json:"gender,omitempty"`
	Score   float64  `json:"score"`
	Source  string   `json:"source,omitempty"`
	Meaning string   `json:"meaning,omitempty"`
	Wuxing  string   `json:"wuxing,omitempty"`
	Styles  []string `json:"styles,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// NameStatStore 姓名统计存储接口
type NameStatStore interface {
	GetSurnameStats(limit int) ([]SurnameStat, error)
	GetSurnameStat(surname string) (*SurnameStat, error)
	GetGivenNameStats(surname string, limit int) ([]GivenNameStat, error)
	GetFullNameStats(surname string, limit int) ([]FullNameStat, error)
	GetFullNameStat(fullName string) (*FullNameStat, error)
	GetNameGenderStats(name string) (*NameGenderStat, error)
	GetTopFullNames(limit int) ([]FullNameStat, error)
	GetTotalNameCount() (int, error)
}

// SurnameStat 姓氏统计
type SurnameStat struct {
	Surname     string  `json:"surname"`
	Count       int     `json:"count"`
	Ratio       float64 `json:"ratio"`
	Rank        int     `json:"rank"`
	MaleRatio   float64 `json:"male_ratio"`
	FemaleRatio float64 `json:"female_ratio"`
}

// GivenNameStat 名字部分统计（按姓氏分组）
type GivenNameStat struct {
	Surname      string `json:"surname"`
	GivenName    string `json:"given_name"`
	Count        int    `json:"count"`
	Rank         int    `json:"rank"`
	MaleCount    int    `json:"male_count"`
	FemaleCount  int    `json:"female_count"`
	UnknownCount int    `json:"unknown_count"`
}

// FullNameStat 全名统计
type FullNameStat struct {
	FullName     string  `json:"full_name"`
	Surname      string  `json:"surname"`
	GivenName    string  `json:"given_name"`
	Count        int     `json:"count"`
	MaleCount    int     `json:"male_count"`
	FemaleCount  int     `json:"female_count"`
	UnknownCount int     `json:"unknown_count"`
	Rank         int     `json:"rank"`
	MaleRatio    float64 `json:"male_ratio,omitempty"`
	FemaleRatio  float64 `json:"female_ratio,omitempty"`
}

// NameGenderStat 名字性别统计
type NameGenderStat struct {
	Name         string  `json:"name"`
	MaleCount    int     `json:"male_count"`
	FemaleCount  int     `json:"female_count"`
	UnknownCount int     `json:"unknown_count"`
	TotalCount   int     `json:"total_count"`
	MaleRatio    float64 `json:"male_ratio"`
	FemaleRatio  float64 `json:"female_ratio"`
}

// Store 定义完整存储接口（向后兼容，组合所有子接口）
type Store interface {
	HistoryStore
	FavoriteStore
	ShareStore
	HanziStore
	FeedbackStore
	YijingStore
	ZodiacStore
	CuratedStore
	NameStatStore

	// ReadPing 探活持久化存储的读路径（docs/40 A5）。
	//
	// 语义与 GetHanziByChar 不同：后者探的是运行时内存 map（预载快照），
	// 持久层损坏照样返回非 nil；ReadPing 直接在持久层执行轻量查询，
	// 用于 /readyz 真探活。内存库实现恒返回 nil（无磁盘依赖）。
	ReadPing() error
}

// HistoryRecord 历史记录结构
type HistoryRecord struct {
	ID string `json:"id"`
	// SessionID 记录归属的访客会话标识；不对外序列化（json:"-"），
	// 避免把 bearer 标识随列表/详情响应泄露给其他调用方。
	SessionID     string    `json:"-"`
	Surname       string    `json:"surname"`
	Gender        string    `json:"gender"`
	BirthDate     string    `json:"birth_date"`
	BirthTime     string    `json:"birth_time"`
	BirthLocation string    `json:"birth_location"`
	Results       string    `json:"results"`
	CreatedAt     time.Time `json:"created_at"`
}

// FavoriteRecord 收藏记录结构
type FavoriteRecord struct {
	ID string `json:"id"`
	// SessionID 记录归属的访客会话标识；不对外序列化。
	SessionID string    `json:"-"`
	Surname   string    `json:"surname"`
	GivenName string    `json:"given_name"`
	Pinyin    string    `json:"pinyin"`
	Gender    string    `json:"gender"`
	Score     float64   `json:"score"` // 综合评分是浮点（如 92.7），不能用 int 收
	Source    string    `json:"source"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// Hanzi 汉字数据结构
type Hanzi struct {
	Char    string `json:"char"`
	Pinyin  string `json:"pinyin"`
	Wuxing  string `json:"wuxing"`
	Strokes int    `json:"strokes"`

	// SQLite 下沉扩展（2026.09.01.4）：通用过滤需要的字段
	// UsageLevel 常用等级（1-5），与 namer.json 同源
	UsageLevel int `json:"usage_level,omitempty"`
	// PositiveScore 寓意评分（0-100），与 namer.json 同源
	PositiveScore int `json:"positive_score,omitempty"`

	// 以下字段为 P2（ADR-003「SQL 与内存结果等价」）补齐：
	// 原先这些维度只存在于内存 hanzi.HanziData，SQL 查询必须回查内存二次过滤，
	// 导致 SQL 成为「粗筛 + Go 端精筛」的两段式实现。现全部下沉到 hanzi_data，
	// 使 services 层可只依赖 SQL 结果构造 Character（docs/31 §11.9 第 2 项）。
	//
	// Level 通用规范汉字表分级（1=一级 3500 / 2=二级 3000 / 3=三级 1605 / 0=表外）
	Level int `json:"level,omitempty"`
	// KangxiStrokes 康熙笔画（0 表示未收录，回落简体笔画）
	KangxiStrokes int `json:"kangxi_strokes,omitempty"`
	// Gender 性别倾向（male/female/neutral），与 namer.json 同源
	Gender string `json:"gender,omitempty"`
	// Radical 偏旁部首，与 namer.json 同源
	Radical string `json:"radical,omitempty"`
	// NamingCategories 该字所属候选分类（namer.json namingCategories，JSON 数组）
	NamingCategories []string `json:"naming_categories,omitempty"`
	// Meaning 词典释义（word.json 首字释义）
	Meaning string `json:"meaning,omitempty"`
	// IsRare 生僻字标记（namer.json is_rare）
	IsRare bool `json:"is_rare,omitempty"`
	// IsNegative 不宜入名标记（namer.json isNegative，实测 14 字为真）
	IsNegative bool `json:"is_negative,omitempty"`
	// NamePenalty 起名扣分（namer.json namePenalty，0-100，实测 54 字为真）
	NamePenalty int `json:"name_penalty,omitempty"`
	// NameFreqTier 人名频率档位（1-5，由 name_frequency 语料合并进 namer_loader，
	// 实测 1118 字有值；注意其上游是语料合并而非 namer.json 字段）
	NameFreqTier int `json:"name_freq_tier,omitempty"`
}

// NameRequest 名字生成请求记录
type NameRequest struct {
	ID          int64     `json:"id"`
	Surname     string    `json:"surname"`
	Gender      string    `json:"gender"`
	BirthYear   int       `json:"birth_year"`
	BirthMonth  int       `json:"birth_month"`
	BirthDay    int       `json:"birth_day"`
	BirthHour   int       `json:"birth_hour"`
	Preferences string    `json:"preferences"`
	NameType    string    `json:"name_type"`
	RequestHash string    `json:"request_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

// NameFeedback 名字反馈记录
type NameFeedback struct {
	ID             int64     `json:"id"`
	RequestID      int64     `json:"request_id"`
	FullName       string    `json:"full_name"`
	GivenName      string    `json:"given_name"`
	IsLiked        bool      `json:"is_liked"`
	IsSelected     bool      `json:"is_selected"`
	UserRating     *int      `json:"user_rating,omitempty"`
	FeedbackText   string    `json:"feedback_text,omitempty"`
	AlgorithmScore float64   `json:"algorithm_score"`
	WuxingMatch    string    `json:"wuxing_match,omitempty"`
	DeviceInfo     string    `json:"device_info,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// AlgorithmPerformance 算法性能统计
type AlgorithmPerformance struct {
	Date          string  `json:"date"`
	TotalRequests int     `json:"total_requests"`
	AvgScore      float64 `json:"avg_score"`
	LikeRate      float64 `json:"like_rate"`
	SelectRate    float64 `json:"select_rate"`
}
