package fate

// NamingCategory 起名用字分类信息
type NamingCategory struct {
	Name  string // 分类名称（如"品德""山水"）
	Desc  string // 分类说明
	Chars string // 该分类包含的汉字字符串
}

// NamingCharInfo 单字在起名库中的信息
type NamingCharInfo struct {
	Char     string   // 汉字
	Gender   string   // 性别标签 male/female/neutral
	Category []string // 所属分类列表（一个字可属多个分类）
}

// NamingCategories 起名用字分类定义表
//
// 作为分类定义（名称+说明），Chars 字段保留精选示例字。
// 实际全量数据来源：HanziData（由 adapter 层通过 SyncNamingIndexFromHanzi 注入）。
// 引擎查询已改为基于 HanziData.NamingCategories 的实时过滤，
// 不再受限于此处硬编码的 ~400 字。详见 adapters.go 的 SyncNamingIndexFromHanzi。
var NamingCategories = []NamingCategory{
	{Name: "品德", Desc: "品德修养", Chars: "仁义德贤慧睿智信礼诚善谦恒毅勤敏达廉温良恭让"},
	{Name: "自然天象", Desc: "自然天象", Chars: "晨曦岚澜霖晗昊旭煜熙辰泽润霞晴朗旷朔皓皎晚暮"},
	{Name: "山水", Desc: "山水意象", Chars: "川峰渊泉溪涧澄清源涟淙淮汀沐洵浔汐淼澈渺汝洛"},
	{Name: "花木", Desc: "花草树木", Chars: "芷蕙桐楠梓萱菲荷莲桃杏柳竹梅芸茗蓉苑葳蔚苒苏"},
	{Name: "美玉", Desc: "美玉珍宝", Chars: "瑶琳璟珩琛瑜璇琪珺琰玥珂瑄琦琬珏瑗玙珑琤玢玳"},
	{Name: "志向", Desc: "志向抱负", Chars: "鸿凌翔鹏程弘博宏杰彦哲逸卓远烁奕昱恺睿淳"},
	{Name: "文雅", Desc: "文采风雅", Chars: "文韵翰诗铭彬彧墨章赋颂斐斯衍思羲衡熠烁煊"},
	{Name: "美好", Desc: "美好祝愿", Chars: "嘉佳妙怡悦欣欢乐宜安宁祥瑞熹喜融和畅舒"},
	{Name: "光明", Desc: "光明璀璨", Chars: "曜昭晖耀灿烨炫焕皓朗映耿晟炜昕晁旻昶炅"},
	{Name: "清新水韵", Desc: "清新水韵（现代流行）", Chars: "沐沅汀淇洇浩淮汐沁湛涵漫溯滢滟潇漪渟沂泓渝沛沣"},
	{Name: "简雅意境", Desc: "简洁现代意境", Chars: "初末言夏秋冬春归远来念思新白青苍素安一之又"},
	{Name: "星宿宇宙", Desc: "星宿宇宙感", Chars: "辰宸昂璇玑霄苍穹廓寥旷渺"},
	{Name: "气质品格", Desc: "独特气质", Chars: "澹泊宁谧幽邃逸疏旷旸霁凛俊飒朗铮朴"},
	{Name: "温柔细腻", Desc: "温柔细腻", Chars: "婉娴娆柔绮绾绒缕绡絮绫锦纹苒软暖融"},
	{Name: "现代男生", Desc: "现代男生风格用字", Chars: "奕晁骁朔霆驰驹駻烺玚"},
	{Name: "现代女生", Desc: "现代女生风格用字", Chars: "栀柠芃芮茜苓茹萌蓁蔓芙葵棠棣橙柚苡芩"},
	{Name: "文学意境", Desc: "文学意境感", Chars: "怀古今斜霜残醒明渐隐烟迟晚墟野静幽"},
}

// namingCharIndex 字符 → 其起名分类信息索引
// lazy-init，线程安全
var (
	namingIdx   map[string]*NamingCharInfo
	namingOnce  syncOnce
)

// initNamingIndex 构建字符分类索引
func initNamingIndex() {
	namingIdx = make(map[string]*NamingCharInfo)

	// 性别倾向集合（从 ai4naming 数据移植）
	maleSet := charSet("鸿凌翔鹏弘博宏杰彦哲毅恒刚雷昊旭煜浩烁奕昱恺淳彬彧章衡熠昶炅炜晟宸昂骁朔霆驰烺玚")
	femaleSet := charSet("婉娴娆柔绮绾绒缕绡絮绫锦纹怡静芷蕙萱菲荷莲蓉瑶琳璇琪珺玥婷媛颖妙栀柠芃芮茜苓茹萌蓁蔓芙葵棠棣橙柚苡芩")

	for _, cat := range NamingCategories {
		for _, r := range []rune(cat.Chars) {
			ch := string(r)
			info, exists := namingIdx[ch]
			if !exists {
				gender := "neutral" // 默认中性
				if maleSet[ch] {
					gender = "male"
				} else if femaleSet[ch] {
					gender = "female"
				}
				info = &NamingCharInfo{
					Char:     ch,
					Gender:   gender,
					Category: make([]string, 0, 2),
				}
				namingIdx[ch] = info
			}
			// 去重添加分类
			dup := false
			for _, c := range info.Category {
				if c == cat.Name {
					dup = true
					break
				}
			}
			if !dup {
				info.Category = append(info.Category, cat.Name)
			}
		}
	}
}

// charSet 将字符串转成 map[string]bool 快速查表
func charSet(s string) map[string]bool {
	set := make(map[string]bool, len([]rune(s)))
	for _, r := range []rune(s) {
		set[string(r)] = true
	}
	return set
}

// GetNamingCharInfo 获取某字的起名分类信息
// 如果该字不在精选库中，返回 nil
func GetNamingCharInfo(char string) *NamingCharInfo {
	namingOnce.Do(initNamingIndex)
	return namingIdx[char]
}

// IsNamingChar 判断某字是否在精选起名库中
func IsNamingChar(char string) bool {
	return GetNamingCharInfo(char) != nil
}

// GetNamingCharsByCategory 获取指定分类的起名用字列表
// category 为空则返回所有分类的字
func GetNamingCharsByCategory(category string) []string {
	namingOnce.Do(initNamingIndex)

	if category == "" {
		chars := make([]string, 0, len(namingIdx))
		for ch := range namingIdx {
			chars = append(chars, ch)
		}
		return chars
	}

	var chars []string
	for ch, info := range namingIdx {
		for _, c := range info.Category {
			if c == category {
				chars = append(chars, ch)
				break
			}
		}
	}
	return chars
}

// GetNamingCharsByGender 获取指定性别的起名用字
func GetNamingCharsByGender(gender string) []string {
	namingOnce.Do(initNamingIndex)

	var chars []string
	for ch, info := range namingIdx {
		if info.Gender == gender || info.Gender == "neutral" {
			chars = append(chars, ch)
		}
	}
	return chars
}

// GetNamingCategoryNames 获取所有可用分类名称
func GetNamingCategoryNames() []string {
	names := make([]string, len(NamingCategories))
	for i, cat := range NamingCategories {
		names[i] = cat.Name
	}
	return names
}

// CountNamingChars 获取精选起名库总字数和分类数
func CountNamingChars() (total int, categories int) {
	namingOnce.Do(initNamingIndex)
	return len(namingIdx), len(NamingCategories)
}

// ——— 风格/类别兼容性矩阵 ———

// CategoryCompatibility 定义分类之间的风格兼容性
// 用于引擎中根据用户选择的分类推荐兼容的分类用字
// 例如"品德"兼容"文雅"、"美好"、"志向"
type CategoryCompatibility struct {
	Category    string   // 主分类
	Compatible  []string // 兼容分类列表（风格搭配效果好）
	Description string   // 搭配说明
}

// CategoryCompatMatrix 类别兼容性矩阵
//
// 设计原则：
// - 同一大类的互相关联（品德↔志向↔文雅）
// - 意象互补（山水↔花木、星宿↔光明）
// - 风格呼应（美玉↔光明↔美好、清新水韵↔简雅意境）
var CategoryCompatMatrix = []CategoryCompatibility{
	{
		Category:    "品德",
		Compatible:  []string{"文雅", "美好", "志向", "气质品格"},
		Description: "品德搭配文雅或美好，寓意品德高尚、气质优雅",
	},
	{
		Category:    "自然天象",
		Compatible:  []string{"山水", "光明", "星宿宇宙", "清新水韵"},
		Description: "自然天象搭配山水或星宿，气象万千",
	},
	{
		Category:    "山水",
		Compatible:  []string{"自然天象", "花木", "清新水韵", "文学意境"},
		Description: "山水搭配花木或清新水韵，画面感强",
	},
	{
		Category:    "花木",
		Compatible:  []string{"山水", "清新水韵", "温柔细腻", "现代女生"},
		Description: "花木搭配山水或清新水韵，自然清新",
	},
	{
		Category:    "美玉",
		Compatible:  []string{"光明", "美好", "品德", "文雅"},
		Description: "美玉搭配光明或美好，光彩照人",
	},
	{
		Category:    "志向",
		Compatible:  []string{"品德", "文雅", "美好", "现代男生"},
		Description: "志向搭配品德或文雅，志存高远",
	},
	{
		Category:    "文雅",
		Compatible:  []string{"品德", "志向", "文学意境", "美好"},
		Description: "文雅搭配品德或文学意境，书卷气浓",
	},
	{
		Category:    "美好",
		Compatible:  []string{"品德", "美玉", "光明", "温柔细腻"},
		Description: "美好搭配品德或光明，吉祥如意",
	},
	{
		Category:    "光明",
		Compatible:  []string{"美玉", "星宿宇宙", "自然天象", "志向"},
		Description: "光明搭配星宿或自然天象，前程似锦",
	},
	{
		Category:    "清新水韵",
		Compatible:  []string{"山水", "花木", "简雅意境", "自然天象"},
		Description: "清新水韵搭配山水或简雅意境，清秀灵动",
	},
	{
		Category:    "简雅意境",
		Compatible:  []string{"文学意境", "清新水韵", "气质品格", "美好"},
		Description: "简雅意境搭配文学意境或清新水韵，简约不简单",
	},
	{
		Category:    "星宿宇宙",
		Compatible:  []string{"光明", "自然天象", "志向", "文学意境"},
		Description: "星宿宇宙搭配光明或自然天象，浩瀚无边",
	},
	{
		Category:    "气质品格",
		Compatible:  []string{"品德", "文雅", "简雅意境", "志向"},
		Description: "气质品格搭配品德或文雅，不落俗套",
	},
	{
		Category:    "温柔细腻",
		Compatible:  []string{"花木", "美好", "现代女生", "清新水韵"},
		Description: "温柔细腻搭配花木或美好，温婉可人",
	},
	{
		Category:    "现代男生",
		Compatible:  []string{"志向", "品德", "光明", "星宿宇宙"},
		Description: "现代男生搭配志向或光明，阳光大气",
	},
	{
		Category:    "现代女生",
		Compatible:  []string{"温柔细腻", "花木", "美好", "清新水韵"},
		Description: "现代女生搭配温柔细腻或花木，活泼可爱",
	},
	{
		Category:    "文学意境",
		Compatible:  []string{"文雅", "简雅意境", "山水", "自然天象"},
		Description: "文学意境搭配文雅或山水，诗意盎然",
	},
}

// GetCompatibleCategories 获取与某分类兼容的推荐分类
// 返回兼容分类名称列表，不含主分类本身
func GetCompatibleCategories(category string) []string {
	for _, entry := range CategoryCompatMatrix {
		if entry.Category == category {
			result := make([]string, len(entry.Compatible))
			copy(result, entry.Compatible)
			return result
		}
	}
	return nil
}

// GetCompatibleCharsByCategory 获取指定分类及其兼容分类的起名用字
// includeSelf: 是否包含主分类本身的字
// maxCompat: 最多引入几个兼容分类（0 表示全部引入）
func GetCompatibleCharsByCategory(category string, includeSelf bool, maxCompat int) []string {
	namingOnce.Do(initNamingIndex)

	// 收集要包含的分类
	catSet := make(map[string]bool)
	if includeSelf {
		catSet[category] = true
	}
	compat := GetCompatibleCategories(category)
	count := 0
	for _, c := range compat {
		if maxCompat > 0 && count >= maxCompat {
			break
		}
		catSet[c] = true
		count++
	}

	// 找属于这些分类的字
	var chars []string
	for ch, info := range namingIdx {
		for _, c := range info.Category {
			if catSet[c] {
				chars = append(chars, ch)
				break
			}
		}
	}
	return chars
}

// syncOnce 简易 sync.Once 实现，避免导入 sync 包外的类型冲突
type syncOnce struct {
	done bool
}

func (o *syncOnce) Do(f func()) {
	if !o.done {
		o.done = true
		f()
	}
}

// AddNamingChar 向起名索引中添加或补充一个字的分类信息
// 由适配层在加载汉字数据后调用，用于将 HanziData 的分类同步到 fate 层
func AddNamingChar(char string, categories []string, gender string) {
	namingOnce.Do(initNamingIndex)
	info, exists := namingIdx[char]
	if !exists {
		info = &NamingCharInfo{
			Char:     char,
			Gender:   gender,
			Category: make([]string, 0, len(categories)),
		}
		namingIdx[char] = info
	}
	// 合并分类（去重）
	for _, cat := range categories {
		found := false
		for _, c := range info.Category {
			if c == cat {
				found = true
				break
			}
		}
		if !found {
			info.Category = append(info.Category, cat)
		}
	}
	// 性别：如果已有性别且为默认中性，用传入值覆盖
	if gender != "" && info.Gender == "neutral" {
		info.Gender = gender
	}
}
