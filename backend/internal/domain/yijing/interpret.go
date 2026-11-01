package yijing

// interpret.go InterpretHexagram 的 64 卦差异化口径（docs/42 P1-⑦）。
//
// 旧实现的事业/爱情/健康/财运四句为全卦共用模板句，任何卦解读结果雷同。
// 现按卦名/卦象逐卦给出「卦象四象限」文案：每卦四句均取意于该卦卦辞、
// 大象传与上下卦卦德（trigramGuaDe），同一卦内四句独立成文，跨卦互不重复。
// 宜忌（Auspicious/Inauspicious）暂维持通用口径，未在本轮差异化范围。

// HexagramInterpretation 六爻卦综合解读结果
type HexagramInterpretation struct {
	Overall      string `json:"overall"`
	Career       string `json:"career"`
	Love         string `json:"love"`
	Health       string `json:"health"`
	Fortune      string `json:"fortune"`
	Auspicious   string `json:"auspicious"`
	Inauspicious string `json:"inauspicious"`
}

// hexagramVoices 单卦四象限文案（按下标 = 卦序-1 存取）
type hexagramVoices struct {
	Career  string // 事业
	Love    string // 感情
	Health  string // 健康
	Fortune string // 财运
}

// hexagramVoiceTable 64 卦四象限文案表，下标 = Number-1
var hexagramVoiceTable = [64]hexagramVoices{
	{Career: "刚健进取，自强不息，宜开创引领，忌刚愎自用", Love: "主动热忱，宜刚柔相济，勿以强势压人", Health: "阳气偏盛，注意头首与肺，劳逸有节", Fortune: "财势旺而宜正途经营，过刚易折"},
	{Career: "厚德载物，宜稳步积累、辅佐执行，不宜急进", Love: "宽容体谅，以柔顺经营可长久", Health: "脾胃为先，忌思虑过度伤脾", Fortune: "财运平稳，积少成多，宜守成"},
	{Career: "创业维艰，宜蓄力结援，勿轻举妄动", Love: "起步多磨，宜耐心求婚媾，诚意可通", Health: "初生之伤，注意调养勿冒进", Fortune: "财源未开，宜守宜聚，不宜扩张"},
	{Career: "启蒙待教，宜虚心求师，学而后行", Love: "懵懂易生猜疑，宜坦率沟通引路", Health: "起居有常，防小疾拖成沉疴", Fortune: "财识未足，宜谨慎问途，防受诳骗"},
	{Career: "蓄势待时，宜静候机缘，不宜强求", Love: "感情需时间酝酿，心诚则灵", Health: "饮食有节，耐心调养可自愈", Fortune: "财运将临，静待佳机，不宜躁进"},
	{Career: "争讼非成之象，宜求和解，避讼趋和", Love: "口角是非多，退一步海阔天空", Health: "气郁化火，留意心神与肝胆", Fortune: "财务易起纠纷，明算账目，忌争利"},
	{Career: "行师有律，宜有组织有纪律，服从大局", Love: "关系需明确定位，忌纠缠不清", Health: "防积劳成疾，作息规律为要", Fortune: "财需规划，宜预算节制，忌挥霍"},
	{Career: "亲密相比，合作为贵，宜择善而从", Love: "两情相依，真诚相待则水乳交融", Health: "人气和顺，注意饮食卫生", Fortune: "合伙生财，宜众利共赢"},
	{Career: "力量尚小，宜积小成大，以柔畜刚", Love: "情意升温而未定，宜循序渐进", Health: "小恙调理，防微杜渐", Fortune: "小有积蓄，宜储蓄理财，忌大手笔"},
	{Career: "如履虎尾，谨慎行事则有惊无险", Love: "关系微妙，宜以礼相待、恭敬持守", Health: "行事小心，防意外磕碰损伤", Fortune: "理财如履薄冰，步步为营为上"},
	{Career: "天地交泰，上下通达，事业亨通畅达", Love: "心意相通，和合美满之象", Health: "气血通畅，身心安泰", Fortune: "财运亨通，正当兴盛之时"},
	{Career: "天地不交，闭塞之象，宜守静待变", Love: "沟通不畅，宜主动破冰释疑", Health: "气机郁滞，宜疏通调畅", Fortune: "财运闭塞，宜守不宜投"},
	{Career: "同心协力，与志同道合者共事则成", Love: "志趣相投，坦诚相见则情谊笃定", Health: "心火偏亢，宜清凉平和", Fortune: "众利共创，忌独占其成"},
	{Career: "大有所获，事业盛大，名利可期", Love: "感情丰盈，宜以谦和守成", Health: "火势炎上，防心脑血管之亢", Fortune: "财运丰盛，宜谦守，忌骄奢"},
	{Career: "谦谦君子，谦虚受益，步步稳升", Love: "谦和相待，情谊细水长流", Health: "心态平和，百病少侵", Fortune: "财运平顺，谦让之间反得利"},
	{Career: "顺时而豫，宜谋定后动，共享成果", Love: "相悦相豫，防乐极生悲", Health: "心情舒畅，防安逸过度", Fortune: "财运和乐，宜量入为出"},
	{Career: "随时而动，宜顺势而为、择善而从", Love: "相互依随，贵在真心不盲从", Health: "起居随时调摄，顺天应时", Fortune: "随行就市，宜灵活应变"},
	{Career: "整饬积弊，宜革新治理、正本清源", Love: "情有积弊，需用心梳理治理", Health: "慢性之疾，宜彻底调理勿拖延", Fortune: "账有蛀漏，宜清查整顿"},
	{Career: "以上临下，事业渐进，宜宽厚待人", Love: "真诚相临，诚意自能动人", Health: "调养有道，康复在望", Fortune: "财运渐旺，宜把握当下"},
	{Career: "观仰化育，宜观察学习、以身作则", Love: "细心观人，知彼而后合", Health: "留察身体信号，防患未然", Fortune: "观望行情，看准而后动"},
	{Career: "咬合除梗，排除障碍，赏罚分明", Love: "直面矛盾，化解隔阂则亲", Health: "防口齿咽喉与积食之患", Fortune: "明辨是非财，忌贪不义之利"},
	{Career: "文饰其表，重形象包装，须表里如一", Love: "浪漫有情调，重外表更重心意", Health: "重养生修饰，防外强中干", Fortune: "勿为体面排场破费，务实为要"},
	{Career: "剥落消残，势不可用，宜守正待复", Love: "情意转淡，宜固本维系", Health: "元气耗损，宜静养少耗", Fortune: "财运消剥，宜守财节流"},
	{Career: "一阳来复，否极泰来，宜重新出发", Love: "旧情可复，重修旧好有望", Health: "元气渐复，宜养勿耗", Fortune: "财运回暖，稳扎稳打"},
	{Career: "无妄而正，遵循正道，不妄为自有成", Love: "真诚无妄，缘分自来", Health: "顺其自然，勿乱进补妄治", Fortune: "意外之财勿贪，守正得财"},
	{Career: "大有蓄养，厚积薄发，宜蓄贤储能", Love: "情感深厚，蓄势而发", Health: "体魄强健，仍须节制有度", Fortune: "财富累积，宜谋长远再生产"},
	{Career: "颐养有道，宜自力更生、涵养人才", Love: "慎言节欲，涵养感情", Health: "饮食养生为要，节制口腹之欲", Fortune: "节流养财，细水长流"},
	{Career: "非常之时行非常之事，量力而行防过载", Love: "情炽易失衡，防过犹不及", Health: "防积劳过用，张弛有度", Fortune: "大进大出，严防透支"},
	{Career: "重险叠陷，守信行险可济，忌侥幸", Love: "情路坎坷，以诚涉险则济", Health: "防肾水泌尿之疾与风寒水厄", Fortune: "财途多险，宜稳守正财"},
	{Career: "附丽光明，才华外显，宜依托平台而明", Love: "热烈明亮，宜持久经营勿骤冷骤热", Health: "防心火眼目之疾与燥热", Fortune: "财自明来，宜正大光明取之"},
	{Career: "交感相悦，宜广结善缘、交流互动", Love: "两情相悦，婚娶之吉兆", Health: "防过敏感应之症，气机宜和", Fortune: "情谊生财，人脉即财脉"},
	{Career: "恒久有守，久于其道则事业可常", Love: "白头之约，贵在恒心持守", Health: "锻炼贵在有恒，持之以恒", Fortune: "细水长流，忌见异思迁"},
	{Career: "退避保身，宜敛锋蓄势，避其锐气", Love: "保持距离，冷静审视关系", Health: "宜静养，远离耗神之事", Fortune: "收缩战线，宜守财观望"},
	{Career: "声势正壮，忌恃强蛮干，壮而守礼", Love: "热情高涨，宜以礼节之", Health: "体格强壮，防过用致伤", Fortune: "财力正盛，忌冒进豪投"},
	{Career: "晋升向明，顺而上行，宜进取有为", Love: "感情升温，进展明朗", Health: "状态向好，防上火发炎", Fortune: "财运上扬，宜乘势进取"},
	{Career: "晦而转明，韬光养晦，藏才待时", Love: "心有隐伤，宜守心自持", Health: "防目力心神之耗，宜静养", Fortune: "财不外露，低调守持"},
	{Career: "齐家有道，内治有序，分工分明", Love: "谈婚论嫁，家和万事兴", Health: "家居养生，规律生活即良药", Fortune: "持家有方，节俭持盈"},
	{Career: "睽异乖离，宜求同存异、以和化睽", Love: "聚散参商，宜互谅互让", Health: "上火下寒，阴阳失谐宜调", Fortune: "合伙易分歧，账目宜分明"},
	{Career: "行止蹇滞，宜反躬自省、借助贵人之力", Love: "相守不易，贵在坚守", Health: "防腿足关节之疾，行动宜慎", Fortune: "财路受阻，宜固守本业"},
	{Career: "险难得解，宜速办速决，趁势舒困", Love: "误会冰释，宽容则和", Health: "病去身松，宜舒筋活络", Fortune: "财困缓解，宜把握时机"},
	{Career: "损益有道，舍小取大，宜减负聚焦", Love: "以付出换长久，先损后益", Health: "戒除耗损嗜好，有舍有得", Fortune: "或有破费，破财消灾莫强求"},
	{Career: "损上益下，得助增益，宜乘势拓展", Love: "相互滋养，共同成长", Health: "进补得益，宜适度勿滥", Fortune: "财运增益，宜适度扩大投入"},
	{Career: "决而能和，果断决策，清除积弊障碍", Love: "当断则断，明快处理情感纠葛", Health: "急症宜速治，勿讳疾忌医", Fortune: "果断出手，忌拖泥带水失机"},
	{Career: "不期而遇，机遇与暗耗并存，防小人", Love: "艳遇之象，宜慎择勿滥", Health: "防外感风邪，起居宜慎", Fortune: "有意外之财，亦防暗亏"},
	{Career: "聚集成势，宜聚人聚财、组织资源", Love: "情缘汇聚，喜结连理之兆", Health: "人气旺身，防聚集染疾", Fortune: "财源汇聚，宜广蓄厚储"},
	{Career: "积小成高，步步高升，宜循序而进", Love: "感情升温，稳步而登堂入室", Health: "状态上行，宜培元固本", Fortune: "财运攀升，宜持续投入"},
	{Career: "处困守正，身困志不困，坚忍则亨", Love: "情逢困顿，困境中方见真心", Health: "困乏疲惫，宜休养生息", Fortune: "财用拮据，宜节用度困"},
	{Career: "井养不穷，宜修内功、泽被于人", Love: "井然有序，情意绵长不改", Health: "调养根本，如井之养人", Fortune: "财源稳定，宜深耕细作"},
	{Career: "顺天应人，变革当时，破旧立新", Love: "关系升级，破旧局立新约", Health: "除旧布新，宜改易不良起居", Fortune: "转型得财，宜创新求变"},
	{Career: "鼎新养贤，器局宏大，稳中求新", Love: "鼎盛成家，安稳之兆", Health: "固本培元，进补得宜", Fortune: "财基稳固，宜置业置产"},
	{Career: "震惊而动，变动骤起，临危不乱者胜", Love: "惊喜与波折并至，镇定应对", Health: "防心神受惊与肝气之疾", Fortune: "动荡中有财机，宜快进快出"},
	{Career: "止于当止，宜守不宜进，适可而止", Love: "驻足沉淀，静观其变", Health: "宜静养安神，防腰背劳损", Fortune: "守财止贪，落袋为安"},
	{Career: "循序渐进，按部就班，徐图发展", Love: "水到渠成，正礼婚嫁之象", Health: "康复渐进，勿求速效", Fortune: "财运渐积，宜长期主义"},
	{Career: "名分未正，宜守本分，不宜僭进", Love: "情动易偏，宜正名定分而后安", Health: "防内分泌失调之象，宜调顺", Fortune: "财有隐患，忌急功近利"},
	{Career: "丰大如日中天，宜明断守成防盛极", Love: "情浓意盛，防盛极生隙", Health: "精力旺盛，防亢奋过耗", Fortune: "财运丰隆，趁势而须守正"},
	{Career: "旅居在外，宜异地发展、随遇而安", Love: "聚少离多，宜珍惜相处之时", Health: "防水土不服，起居自慎", Fortune: "财在旅途，宜流动灵活取之"},
	{Career: "随风入伏，以柔渗透，宜沟通协调", Love: "温柔相随，含蓄经营见真章", Health: "防风气之疾与呼吸道之患", Fortune: "财随人气，宜顺势借力"},
	{Career: "和悦以言，口才社交见长，宜悦近人", Love: "喜悦相亲，言语传情最灵", Health: "防口舌咽喉之疾，过悦亦伤身", Fortune: "和气生财，谈笑间得利"},
	{Career: "涣散之象，宜凝聚人心、重整旗鼓", Love: "若即若离，宜重新聚拢心意", Health: "气机涣散，宜收敛心神", Fortune: "财有涣散之虞，宜归拢集中"},
	{Career: "节而有度，制度约束，节流增效", Love: "感情有度，张弛相宜则久", Health: "节制饮食起居，过节亦伤", Fortune: "开源节流，宜预算有度"},
	{Career: "诚信相孚，以诚立信，口碑致胜", Love: "真诚互信，心灵相契", Health: "心平气和，诚信即是养生", Fortune: "信誉为财，守诺者得"},
	{Career: "小事可为，大事勿担，宜谨守本位", Love: "小别胜新婚，细节处见真情", Health: "小恙勿拖，谨防过用", Fortune: "小财可求，大投资宜慎"},
	{Career: "功成守成，初吉后乱，防懈怠生变", Love: "有情人终成眷属，成而后守", Health: "康复在望，防旧疾复发", Fortune: "财已成局，宜守不宜贪"},
	{Career: "事未竟成，火水未交，仍需努力不懈", Love: "缘分未到，静待花开", Health: "调理未竟，坚持为要", Fortune: "财运未定，宜蓄势待时"},
}

// InterpretHexagram 六爻卦综合解读（64 卦差异化口径，docs/42 P1-⑦）：
// Overall 取卦自带解读；事业/感情/健康/财运按卦序取 hexagramVoiceTable
// 专属文案；喜用神命中上下卦五行时附「与命格相合」吉断。
// 卦序越界（未收录卦）时回落旧通用模板，保证函数总可用。
func InterpretHexagram(hexagram *Hexagram, xiyongshen []string) *HexagramInterpretation {
	interp := &HexagramInterpretation{
		Overall:      hexagram.Interpretation,
		Career:       "事业平稳发展，需把握时机",
		Love:         "感情需要耐心经营",
		Health:       "注意调理身体，保持平衡",
		Fortune:      "财运平稳，不宜冒险",
		Auspicious:   "祭祀、祈福、嫁娶",
		Inauspicious: "动土、破土",
	}

	if hexagram.Number >= 1 && hexagram.Number <= 64 {
		v := hexagramVoiceTable[hexagram.Number-1]
		interp.Career = v.Career
		interp.Love = v.Love
		interp.Health = v.Health
		interp.Fortune = v.Fortune
	}

	upperWuxing := TrigramWuxingMap[hexagram.UpperTrigram]
	lowerWuxing := TrigramWuxingMap[hexagram.LowerTrigram]

	for _, xy := range xiyongshen {
		if upperWuxing == xy || lowerWuxing == xy {
			interp.Fortune += "；五行相合，财运亨通"
			interp.Overall += "此卦与命格相合，大吉。"
			break
		}
	}

	return interp
}
