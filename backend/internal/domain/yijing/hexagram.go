package yijing

import (
	"math/rand"
	"time"
)

type YaoResult struct {
	Yao    int    `json:"yao"`
	Name   string `json:"name"`
	YaoName string `json:"yao_name"`
	Meaning string `json:"meaning"`
}

var YaoMap = map[int]YaoResult{
	6: {Yao: 6, Name: "老阴", YaoName: "六", Meaning: "阴爻发动，事物向相反方向转化"},
	7: {Yao: 7, Name: "少阳", YaoName: "七", Meaning: "阳爻静止，事物保持稳定发展"},
	8: {Yao: 8, Name: "少阴", YaoName: "八", Meaning: "阴爻静止，事物需要等待时机"},
	9: {Yao: 9, Name: "老阳", YaoName: "九", Meaning: "阳爻发动，事物向前发展"},
}

// Hexagram represents an I Ching hexagram
type Hexagram struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Number         int    `json:"number"`
	Symbol         string `json:"symbol"`
	UpperTrigram   int    `json:"upper_trigram"`   // 上卦 (1-8)
	LowerTrigram   int    `json:"lower_trigram"`   // 下卦 (1-8)
	GuaCi          string `json:"gua_ci"`          // 卦辞
	XiangCi        string `json:"xiang_ci"`       // 象辞
	YaoCi          []string `json:"yao_ci"`        // 爻辞
	Interpretation string `json:"interpretation"` // 解读
}

// TrigramMap 八卦对应表
var TrigramMap = map[int]string{
	1: "☰", 2: "☱", 3: "☲", 4: "☳",
	5: "☴", 6: "☵", 7: "☶", 8: "☷",
}

// TrigramNameMap 八卦名称表
var TrigramNameMap = map[int]string{
	1: "乾", 2: "兑", 3: "离", 4: "震",
	5: "巽", 6: "坎", 7: "艮", 8: "坤",
}

// TrigramWuxingMap 八卦五行对应表
var TrigramWuxingMap = map[int]string{
	1: "金", 2: "金", 3: "火", 4: "木",
	5: "木", 6: "水", 7: "土", 8: "土",
}

// TrigramMeaningMap 八卦象征意义
var TrigramMeaningMap = map[int]string{
	1: "天、君、父、刚健",
	2: "泽、悦、口、柔",
	3: "火、日、离、文明",
	4: "雷、震、动、起始",
	5: "风、木、巽、入",
	6: "水、雨、坎、险",
	7: "山、止、艮、稳",
	8: "地、坤、顺、柔",
}

// HexagramList 64卦数据
var HexagramList = []Hexagram{
	{1, "乾", 1, "☰☰", 1, 1, "元亨利贞", "天行健，君子以自强不息", []string{"潜龙勿用", "见龙在田，利见大人", "君子终日乾乾，夕惕若厉，无咎", "或跃在渊，无咎", "飞龙在天，利见大人", "亢龙有悔", "见群龙无首，吉"}, "象征天，象征刚健有力。事业腾飞，名利双收。但需注意过刚易折，需知进退。"},
	{2, "坤", 2, "☷☷", 8, 8, "元亨，利牝马之贞", "地势坤，君子以厚德载物", []string{"履霜，坚冰至", "直方大，不习无不利", "含章可贞，或从王事，无成有终", "括囊，无咎无誉", "黄裳，元吉", "龙战于野，其血玄黄", "利永贞"}, "象征地，象征柔顺包容。宜稳扎稳打，积累德行。适合五行需土或木的宝宝。"},
	{3, "屯", 3, "☵☳", 6, 4, "元亨利贞，勿用有攸往，利建侯", "云雷屯，君子以经纶", []string{"磐桓，利居贞，利建侯", "屯如邅如，乘马班如，匪寇婚媾", "即鹿无虞，惟入于林中，君子几不如舍", "乘马班如，求婚媾，往吉无不利", "屯其膏，小贞吉，大贞凶", "乘马班如，泣血涟如"}, "象征事物初生，困难与机遇并存。需耐心积累，不可急于求成。"},
	{4, "蒙", 4, "☶☵", 7, 6, "亨，匪我求童蒙，童蒙求我", "山下出泉，蒙，君子以果行育德", []string{"发蒙，利用刑人，用说桎梏", "包蒙吉，纳妇吉，子克家", "勿用取女，见金夫，不有躬，无攸利", "困蒙，吝", "童蒙，吉", "击蒙，不利为寇，利御寇"}, "象征蒙昧，需要启发教育。适合教育得当的宝宝，前景光明。"},
	{5, "需", 5, "☰☵", 1, 6, "有孚，光亨，贞吉，利涉大川", "云上于天，需，君子以饮食宴乐", []string{"需于郊，利用恒，无咎", "需于沙，小有言，终吉", "需于泥，致寇至", "需于血，出自穴", "需于酒食，贞吉", "入于穴，有不速之客三人来"}, "象征等待时机。有耐心地等待，必能获得成功。需注意饮食健康。"},
	{6, "讼", 6, "☵☰", 6, 1, "有孚窒惕，中吉，终凶", "天与水违行，讼，君子以作事谋始", []string{"不永所事，小有言，终吉", "不克讼，归而逋，其邑人三百户，无眚", "食旧德，贞厉，终吉", "不克讼，复即命，渝安贞，吉", "讼元吉", "或锡之鞶带，终朝三褫之"}, "象征争执诉讼。需谨慎处理人际关系，以和为贵。避免与人发生冲突。"},
	{7, "师", 7, "☷☵", 8, 6, "贞丈人吉，无咎", "地中有水，师，君子以容民畜众", []string{"师出以律，否臧凶", "在师中吉，无咎，王三锡命", "师或舆尸，凶", "师左次，无咎", "田有禽，利执言，无咎，长子帅师", "大君有命，开国承家，小人勿用"}, "象征军队战争。适合武职或领导岗位。但需注意以德服人。"},
	{8, "比", 8, "☵☷", 6, 8, "吉，原筮元永贞，无咎", "地上有水，比，先王以建万国，亲诸侯", []string{"有孚比之，无咎，有孚盈缶，终来有他吉", "比之自内，贞吉", "比之匪人", "显比，王用三驱，失前禽", "邑人不诫，吉", "比之无首，凶"}, "象征亲近依附。善于团结协作，人缘好。适合团队合作的工作。"},
	{9, "小畜", 9, "☴☰", 5, 1, "亨，密云不雨，自我西郊", "风行天上，小畜，君子以懿文德", []string{"复自道，何其咎，吉", "牵复，吉", "舆说辐，夫妻反目", "有孚，血去惕出，无咎", "有孚挛如，富以其邻", "既雨既处，尚德载，妇贞厉"}, "象征小有积累。财富逐步增长，但需防小人破坏。"},
	{10, "履", 10, "☰☱", 1, 2, "履虎尾，不咥人，亨", "上天下泽，履，君子以辨上下，定民志", []string{"素履往，无咎", "履道坦坦，幽人贞吉", "眇能视，跛能履，履虎尾咥人凶", "履虎尾，愬愬终吉", "夬履，贞厉", "视履考祥，其旋元吉"}, "象征小心行走。需谨慎行事，不可冒进。终能化险为夷。"},
	{11, "泰", 11, "☰☷", 1, 8, "小往大来，吉亨", "天地交泰，后以财成天地之道", []string{"拔茅茹以其汇，征吉", "包荒，用冯河，不遐遗", "无平不陂，无往不复，艰贞无咎", "翩翩不富以其邻，不戒以孚", "帝乙归妹，以祉元吉", "城复于隍，勿用师"}, "象征天地相交，阴阳调和。大吉之卦，万事如意。适合八字需调和的宝宝。"},
	{12, "否", 12, "☷☰", 8, 1, "否之匪人，不利君子贞", "天地不交，否，君子以俭德辟难", []string{"拔茅茹以其汇，贞吉亨", "包承，小人吉，大人否亨", "包羞", "有命无咎，畴离祉", "休否，大人吉", "倾否，先否后喜"}, "象征闭塞不通。需耐心等待时机，不宜冒进。否极泰来。"},
	{13, "同人", 13, "☰☲", 1, 3, "同人于野，亨，利涉大川", "天与火，同人，君子以类族辨物", []string{"同人于门，无咎", "同人于宗，吝", "伏戎于莽，升其高陵，三岁不兴", "乘其墉，弗克攻，吉", "同人先号咷而后笑，大师克相遇", "同人于郊，无悔"}, "象征与人同心。善于交际，人脉广。适合需要广泛人脉的宝宝。"},
	{14, "大有", 14, "☲☰", 3, 1, "元亨", "火在天上，大有，君子以遏恶扬善", []string{"无交害，匪咎，艰则无咎", "大车以载，有攸往，无咎", "公用亨于天子，小人弗克", "匪其彭，无咎", "厥孚交如，威如，吉", "自天祐之，吉无不利"}, "象征大有所获。财运亨通，名利双收。但需保持诚信。"},
	{15, "谦", 15, "☶☷", 7, 8, "亨，君子有终", "地中有山，谦，君子以裒多益寡", []string{"谦谦君子，用涉大川，吉", "鸣谦，贞吉", "劳谦君子，有终吉", "无不利，撝谦", "不富以其邻，利用侵伐，无不利", "鸣谦，利用行师征邑国"}, "象征谦虚。越谦虚越有福报。适合低调做人的宝宝。"},
	{16, "豫", 16, "☳☷", 4, 8, "利建侯行师", "雷出地奋，豫，先王以作乐崇德", []string{"鸣豫，凶", "介于石，不终日，贞吉", "盱豫悔，迟有悔", "由豫，大有得，勿疑朋盍簪", "贞疾，恒不死", "冥豫成，有渝无咎"}, "象征欢乐喜庆。性格开朗，人缘好。但需防乐极生悲。"},
	{17, "随", 17, "☱☳", 2, 4, "元亨利贞，无咎", "泽中有雷，随，君子以向晦入宴息", []string{"官有渝，贞吉，出门交有功", "系小子，失丈夫", "系丈夫，失小子，随有求得", "随有获，贞凶，有孚在道以明", "孚于嘉，吉", "拘系之，乃从维之，王用亨于西山"}, "象征随和顺从。善于跟随他人，学习进步。适合跟对导师。"},
	{18, "蛊", 18, "☴☶", 5, 7, "元亨，利涉大川", "山下有风，蛊，君子以振民育德", []string{"干父之蛊，有子考无咎，厉终吉", "干母之蛊，不可贞", "干父之蛊，小有悔，无大咎", "裕父之蛊，往见吝", "干父之蛊，用誉", "不事王侯，高尚其事"}, "象征事物败坏后需要整治。经历困难后必有收获。"},
	{19, "临", 19, "☷☳", 8, 4, "元亨利贞，至于八月有凶", "泽上有地，临，君子以教思无穷", []string{"咸临，贞吉", "咸临，吉无不利", "甘临，无攸利，既忧之，无咎", "至临，无咎", "知临，大君之宜，吉", "敦临，吉无咎"}, "象征君临天下。领导能力强，适合管理岗位。但需防小人对付。"},
	{20, "观", 20, "☴☷", 5, 8, "盥而不荐，有孚颙若", "风行地上，观，先王以省方观民设教", []string{"童观，小人无咎，君子吝", "窥观，利女贞", "观我生，进退", "观国之光，利用宾于王", "观我生，君子无咎", "观其生，君子无咎"}, "象征观察。有敏锐的观察力，适合研究分析工作。"},
	{21, "噬嗑", 21, "☲☳", 3, 4, "亨，利用狱", "雷电噬嗑，先王以明罚敕法", []string{"屦校灭趾，无咎", "噬肤灭鼻，无咎", "噬腊肉，遇毒，小吝无咎", "噬干胏，得金矢，利艰贞，吉", "噬干肉，得黄金，贞厉无咎", "何校灭耳，凶"}, "象征咀嚼刑罚。需果断坚决，不可优柔寡断。"},
	{22, "贲", 22, "☶☲", 7, 3, "亨，小利有攸往", "山下有火，贲，君子以明庶政", []string{"贲其趾，舍车而徒", "贲其须", "贲如濡如，永贞吉", "贲如皤如，白马翰如，匪寇婚媾", "贲于丘园，束帛戋戋，吝终吉", "白贲，无咎"}, "象征文饰美化。外表光鲜，有艺术气质。适合文艺工作。"},
	{23, "剥", 23, "☷☶", 8, 7, "不利有攸往", "山地剥，不利有攸往", []string{"剥床以足，蔑贞凶", "剥床以辨，蔑贞凶", "剥之，无咎", "剥床以肤，凶", "贯鱼以宫人宠，无不利", "硕果不食，君子得舆，小人剥庐"}, "象征衰落。运势不佳，需静待时机。否极泰来。"},
	{24, "复", 24, "☳☷", 4, 8, "亨，出入无疾，朋来无咎", "雷在地中，复，先王以至日闭关", []string{"不远复，无祗悔，元吉", "休复，吉", "频复，厉无咎", "中行独复", "敦复，无悔", "迷复，凶，有灾眚"}, "象征复归。运势转好，否极泰来。适合重新开始的宝宝。"},
	{25, "无妄", 25, "☰☳", 1, 4, "元亨利贞，其匪正有眚", "天下雷行，物与无妄，先王以茂对时育万物", []string{"无妄往，吉", "不耕获，不菑畬，则利有攸往", "无妄之灾，或系之牛，行人之得", "可贞，无咎", "无妄之疾，勿药有喜", "无妄行，有眚，无攸利"}, "象征不妄为。顺应天道，自然会有收获。需保持正直。"},
	{26, "大畜", 26, "☶☰", 7, 1, "利贞，不家食吉，利涉大川", "天在山中，大畜，君子以多识前言往行", []string{"有厉利已", "舆说輹", "良马逐，利艰贞，曰闲舆卫，利有攸往", "童牛之牿，元吉", "豶豕之牙，吉", "何天之衢，亨"}, "象征大为积蓄。财富积累，地位提升。适合长期发展。"},
	{27, "颐", 27, "☶☳", 7, 4, "贞吉，观颐，自求口实", "山下有雷，颐，君子以慎言语，节饮食", []string{"舍尔灵龟，观我朵颐，凶", "颠颐，拂经于丘颐，征凶", "拂颐，贞凶，十年勿用，无攸利", "颠颐吉，虎视眈眈，其欲逐逐，无咎", "拂经，居贞吉，不可涉大川", "由颐，厉吉，利涉大川"}, "象征颐养。注重修养身心，饮食有节。适合养生之道。"},
	{28, "大过", 28, "☴☱", 5, 2, "栋桡，利有攸往，亨", "泽灭木，大过，君子以独立不惧，遁世无闷", []string{"藉用白茅，无咎", "枯杨生稊，老夫得其女妻，无不利", "栋桡，凶", "栋隆，吉，有它吝", "枯杨生华，老妇得其士夫，无咎无誉", "过涉灭顶，凶，无咎"}, "象征过度。需防过度行为，保持中庸之道。"},
	{29, "坎", 29, "☵☵", 6, 6, "习坎，有孚，维心亨，行有尚", "水洊至，习坎，君子以常德行，习教事", []string{"习坎，入于坎窞，凶", "坎有险，求小得", "来之坎坎，险且枕，入于坎窞，勿用", "樽酒簋贰，用缶，纳约自牖，终无咎", "坎不盈，祗既平，无咎", "系用徽纆，寘于丛棘，三岁不得，凶"}, "象征坎险。经历困难，但终能脱险。需坚韧不拔。"},
	{30, "离", 30, "☲☲", 3, 3, "利贞，亨，畜牝牛吉", "明两作，离，大人以继明照于四方", []string{"履错然，敬之无咎", "黄离，元吉", "日昃之离，不鼓缶而歌，则大耋之嗟，凶", "突如其来如，焚如，死如，弃如", "出涕沱若，戚嗟若，吉", "王用出征，有嘉折首，获匪其丑，无咎"}, "象征光明。聪明智慧，前途光明。但需防过于锋芒毕露。"},
	{31, "咸", 31, "☱☶", 2, 7, "亨利贞，取女吉", "山上有泽，咸，君子以虚受人", []string{"咸其拇", "咸其腓，凶，居吉", "咸其股，执其随，往吝", "贞吉悔亡，憧憧往来，朋从尔思", "咸其脢，无悔", "咸其辅颊舌"}, "象征感应。感情丰富，人际和谐。适合需要沟通的工作。"},
	{32, "恒", 32, "☳☴", 4, 5, "亨无咎，利贞，利有攸往", "雷风相与，恒，君子以立不易方", []string{"浚恒，贞凶，无攸利", "悔亡", "不恒其德，或承之羞，贞吝", "田无禽", "恒其德，贞，妇人吉，夫子凶", "振恒，凶"}, "象征恒久。持之以恒，必有收获。需坚持不懈。"},
	{33, "遁", 33, "☰☶", 1, 7, "亨，小利贞", "天下有山，遁，君子以远小人，不恶而严", []string{"遁尾，厉，勿用有攸往", "执之用黄牛之革，莫之胜说", "系遁，有疾厉，畜臣妾吉", "好遁君子吉，小人否", "嘉遁，贞吉", "肥遁，无不利"}, "象征退隐。适时的退让是为了更好的前进。需审时度势。"},
	{34, "大壮", 34, "☳☰", 4, 1, "利贞", "雷在天上，大壮，君子以非礼勿履", []string{"壮于趾，征凶，有孚", "贞吉", "小人用壮，君子用罔，贞厉", "贞吉悔亡，藩决不羸，壮于大舆之輹", "丧羊于易，无悔", "羝羊触藩，不能退，不能遂"}, "象征强大。实力强盛，但需防过刚易折。"},
	{35, "晋", 35, "☲☷", 3, 8, "康侯用锡马蕃庶，昼日三接", "明出地上，晋，君子以自昭明德", []string{"晋如摧如，贞吉，罔孚裕无咎", "晋如愁如，贞吉，受兹介福于其王母", "众允，悔亡", "晋如鼫鼠，贞厉", "悔亡，失得勿恤，往吉无不利", "晋其角，维用伐邑，厉吉无咎，贞吝"}, "象征晋升。事业上升，地位提升。但需防小人嫉妒。"},
	{36, "明夷", 36, "☷☲", 8, 3, "利艰贞", "明入地中，明夷，君子以莅众，用晦而明", []string{"明夷于飞，垂其翼，君子于行，三日不食", "明夷，夷于左股，用拯马壮，吉", "明夷于南狩，得其大首，不可疾贞", "入于左腹，获明夷之心，于出门庭", "箕子之明夷，利贞", "不明晦，初登于天，后入于地"}, "象征光明受损。运势不佳，需韬光养晦。"},
	{37, "家人", 37, "☴☲", 5, 3, "利女贞", "风自火出，家人，君子以言有物而行有恒", []string{"闲有家，悔亡", "无攸遂，在中馈，贞吉", "家人嗃嗃，悔厉吉，妇子嘻嘻，终吝", "富家，大吉", "王假有家，勿恤吉", "有孚威如，终吉"}, "象征家庭。家庭和睦，万事兴隆。适合稳定的工作生活。"},
	{38, "睽", 38, "☲☱", 3, 2, "小事吉", "上火下泽，睽，君子以同而异", []string{"悔亡，丧马勿逐自复，见恶人无咎", "遇主于巷，无咎", "见舆曳，其牛掣，其人天且劓，无初有终", "睽孤遇元夫，交孚，厉无咎", "悔亡，厥宗噬肤，往何咎", "睽孤见豕负涂，载鬼一车"}, "象征乖离离散。需化解矛盾，寻找共同点。"},
	{39, "蹇", 39, "☶☵", 7, 6, "利西南，不利东北，利见大人，贞吉", "山上有水，蹇，君子以反身修德", []string{"往蹇来誉", "王臣蹇蹇，匪躬之故", "往蹇来反", "往蹇来连", "大蹇朋来", "往蹇来硕，吉，利见大人"}, "象征困难险阻。经历困难后必有收获。需坚持不懈。"},
	{40, "解", 40, "☵☳", 6, 4, "利西南，无所往，其来复吉", "雷雨作，解，君子以赦过宥罪", []string{"无咎", "田获三狐，得黄矢，贞吉", "负且乘，致寇至，贞吝", "解而拇，朋至斯孚", "君子维有解，吉，有孚于小人", "公用射隼于高墉之上，获之，无不利"}, "象征解除困难。困难解除，运势转好。适合重新开始。"},
	{41, "损", 41, "☶☱", 7, 2, "有孚，元吉，无咎可贞，利有攸往", "山下有泽，损，君子以惩忿窒欲", []string{"已事遄往，无咎，酌损之", "利贞，征凶，弗损益之", "三人行则损一人，一人行则得其友", "损其疾，使遄有喜，无咎", "或益之十朋之龟，弗克违，元吉", "弗损益之，无咎，贞吉，利有攸往"}, "象征减损。有舍有得，需要付出才能收获。"},
	{42, "益", 42, "☴☳", 5, 4, "利有攸往，利涉大川", "风雷益，君子以见善则迁，有过则改", []string{"利用为大作，元吉，无咎", "或益之十朋之龟，弗克违，永贞吉", "益之用凶事，无咎，有孚中行，告公用圭", "中行告公从，利用为依迁国", "有孚惠心，勿问元吉，有孚惠我德", "莫益之，或击之，立心勿恒，凶"}, "象征增益。不断进步，收益增加。适合学习成长。"},
	{43, "夬", 43, "☰☱", 1, 2, "扬于王庭，孚号有厉，告自邑，不利即戎", "泽上于天，夬，君子以施禄及下，居德则忌", []string{"壮于前趾，往不胜为咎", "惕号，莫夜有戎，勿恤", "壮于頄，有凶，君子夬夬独行遇雨", "臀无肤，其行次且，牵羊悔亡，闻言不信", "苋陆夬夬，中行无咎", "无号，终有凶"}, "象征决断。需要果断决策，不可犹豫不决。"},
	{44, "姤", 44, "☴☰", 5, 1, "女壮，勿用取女", "天下有风，姤，后以施命诰四方", []string{"系于金柅，贞吉，有攸往，见凶", "包有鱼，无咎，不利宾", "臀无肤，其行次且，厉，无大咎", "包无鱼，起凶", "以杞包瓜，含章，有陨自天", "姤其角，吝，无咎"}, "象征相遇。机遇出现，但需谨慎选择。"},
	{45, "萃", 45, "☷☱", 8, 2, "亨，王假有庙，利见大人，亨利贞", "泽上于地，萃，君子以除戎器，戒不虞", []string{"有孚不终，乃乱乃萃，若号一握为笑", "引吉，无咎，孚乃利用禴", "萃如嗟如，无攸利，往无咎，小吝", "大吉无咎", "萃有位，无咎，匪孚，元永贞，悔亡", "赍咨涕洟，无咎"}, "象征聚集。人才济济，众志成城。适合团队合作。"},
	{46, "升", 46, "☴☷", 5, 8, "元亨，用见大人，勿恤，南征吉", "地中生木，升，君子以顺德，积小以高大", []string{"允升，大吉", "孚乃利用禴，无咎", "升虚邑", "王用亨于岐山，吉无咎", "贞吉，升阶", "冥升，利于不息之贞"}, "象征上升。职位提升，地位提高。适合发展事业。"},
	{47, "困", 47, "☵☱", 6, 2, "亨，贞大人吉，无咎，有言不信", "泽无水，困，君子以致命遂志", []string{"臀困于株木，入于幽谷，三岁不觌", "困于酒食，朱绂方来，利用亨祀", "困于石，据于蒺藜，入于其宫，不见其妻，凶", "来徐徐，困于金车，吝，有终", "劓刖，困于赤绂，乃徐有说，利用祭祀", "困于葛藟，于臲卼，曰动悔有悔，征吉"}, "象征困顿。经历困难，但需保持信念。"},
	{48, "井", 48, "☵☴", 6, 5, "改邑不改井，无丧无得，往来井井", "木上有水，井，君子以劳民劝相", []string{"井泥不食，旧井无禽", "井谷射鲋，瓮敝漏", "井渫不食，为我心恻，可用汲，王明并受其福", "井甃，无咎", "井冽寒泉，食", "井收勿幕，有孚元吉"}, "象征井德。取之不尽，用之不竭。适合有稳定收入的工作。"},
	{49, "革", 49, "☱☲", 2, 3, "己日乃孚，元亨利贞，悔亡", "泽中有火，革，君子以治历明时", []string{"巩用黄牛之革", "己日乃革之，征吉，无咎", "征凶，贞厉，革言三就，有孚", "悔亡，有孚改命，吉", "大人虎变，未占有孚", "君子豹变，小人革面，征凶，居贞吉"}, "象征变革。改革创新，与时俱进。适合需要创新的工作。"},
	{50, "鼎", 50, "☲☱", 3, 2, "元吉，亨", "木上有火鼎，君子以正位凝命", []string{"鼎颠趾，利出否，得妾以其子，无咎", "鼎有实，我仇有疾，不我能即，吉", "鼎耳革，其行塞，雉膏不食，方雨亏悔，终吉", "鼎折足，覆公餗，其形渥，凶", "鼎黄耳金铉，利贞", "鼎玉铉，大吉，无不利"}, "象征鼎新。革故鼎新，焕然一新。适合改革创新的宝宝。"},
	{51, "震", 51, "☳☳", 4, 4, "亨，震来虩虩，笑言哑哑", "洊雷震，君子以恐惧修省", []string{"震来虩虩，后笑言哑哑，吉", "震来厉，亿丧贝，跻于九陵，勿逐，七日得", "震苏苏，震行无眚", "震遂泥", "震往来厉，亿无丧有事", "震索索，视矍矍，征凶"}, "象征震动。经历波折，但能化险为夷。"},
	{52, "艮", 52, "☶☶", 7, 7, "艮其背，不获其身，行其庭，不见其人，无咎", "兼山艮，君子以思不出其位", []string{"艮其趾，无咎，利永贞", "艮其腓，不拯其随，其心不快", "艮其限，列其夤，厉薰心", "艮其身，无咎", "艮其辅，言有序，悔亡", "敦艮，吉"}, "象征静止。保持定力，不为外物所动。适合需要专注的工作。"},
	{53, "渐", 53, "☴☶", 5, 7, "女归吉，利贞", "山上有木，渐，君子以居贤德善俗", []string{"鸿渐于干，小子厉，有言，无咎", "鸿渐于磐，饮食衎衎，吉", "鸿渐于陆，夫征不复，妇孕不育，凶", "鸿渐于木，或得其桷，无咎", "鸿渐于陵，妇三岁不孕，终莫之胜，吉", "鸿渐于陆，其羽可用为仪，吉"}, "象征渐进。循序渐进，逐步发展。"},
	{54, "归妹", 54, "☱☳", 2, 4, "征凶，无攸利", "泽上有雷，归妹，君子以永终知敝", []string{"归妹以娣，跛能履，征吉", "眇能视，利幽人之贞", "归妹以须，反归以娣", "归妹愆期，迟归有时", "帝乙归妹，其君之袂不如其娣之袂良", "女承筐无实，士刲羊无血，无攸利"}, "象征嫁娶。感情婚姻方面需谨慎。"},
	{55, "丰", 55, "☲☳", 3, 4, "亨，王假之，勿忧，宜日中", "雷电皆至，丰，君子以折狱致刑", []string{"遇其配主，虽旬无咎，往有尚", "丰其蔀，日中见斗，往得疑疾，有孚发若，吉", "丰其沛，日中见沫，折其右肱，无咎", "丰其蔀，日中见斗，遇其夷主，吉", "来章，有庆誉，吉", "丰其屋，蔀其家，窥其户，阒其无人，三岁不觌，凶"}, "象征丰盛。事业鼎盛，财运丰厚。但需防盛极必衰。"},
	{56, "旅", 56, "☶☲", 7, 3, "小亨，旅贞吉", "山上有火，旅，君子以明慎用刑，而不留狱", []string{"旅琐琐，斯其所取灾", "旅即次，怀其资，得童仆贞", "旅焚其次，丧其童仆，贞厉", "旅于处，得其资斧，我心不快", "射雉一矢亡，终以誉命", "鸟焚其巢，旅人先笑后号咷，丧牛于易，凶"}, "象征旅行或漂泊。适合外出闯荡，但需防意外。"},
	{57, "巽", 57, "☴☴", 5, 5, "小亨，利有攸往，利见大人", "随风巽，君子以申命行事", []string{"进退，利武人之贞", "巽在床下，用史巫纷若，吉无咎", "频巽，吝", "悔亡，田获三品", "贞吉悔亡，无不利，无初有终", "巽在床下，丧其资斧，贞凶"}, "象征顺从。善于服从和配合。适合需要协调的工作。"},
	{58, "兑", 58, "☱☱", 2, 2, "亨，利贞", "丽泽兑，君子以朋友讲习", []string{"和兑，吉", "孚兑，吉，悔亡", "来兑，凶", "商兑未宁，介疾有喜", "孚于剥，有厉", "引兑"}, "象征喜悦。心情愉快，人际和谐。适合需要沟通的工作。"},
	{59, "涣", 59, "☵☴", 6, 5, "亨，王假有庙，利涉大川", "风水涣，先王以享于帝立庙", []string{"用拯马壮，吉", "涣奔其机，悔亡", "涣其躬，无悔", "涣其群，元吉，涣有丘，匪夷所思", "涣汗其大号，涣王居，无咎", "涣其血去逖出，无咎"}, "象征离散涣散。需化解分歧，团结一致。"},
	{60, "节", 60, "☵☱", 6, 2, "亨，苦节不可贞", "泽上有水，节，君子以制数度，议德行", []string{"不出户庭，无咎", "不出门庭，凶", "不节若，则嗟若，无咎", "安节，亨", "甘节，吉，往有尚", "苦节，贞凶，悔亡"}, "象征节制。有所节制，不可过度。适合需要自律的宝宝。"},
	{61, "中孚", 61, "☴☱", 5, 2, "豚鱼吉，利涉大川，利贞", "泽上有风，中孚，君子以议狱缓死", []string{"虞吉，有他不燕", "鸣鹤在阴，其子和之，我有好爵，吾与尔靡之", "得敌，或鼓或罢，或泣或歌", "月几望，马匹亡，无咎", "有孚挛如，无咎", "翰音登于天，贞凶"}, "象征诚信。诚实守信，必有福报。适合需要诚信的工作。"},
	{62, "小过", 62, "☶☳", 7, 4, "亨利贞，可小事，不可大事", "山上有雷，小过，君子以行过乎恭，丧过乎哀", []string{"飞鸟以凶", "过其祖，遇其妣，不及其君，遇其臣，无咎", "弗过防之，从或戕之，凶", "无咎，弗过遇之，往厉必戒，勿用永贞", "密云不雨，自我西郊，公弋取彼在穴", "弗遇过之，飞鸟离之，凶，是谓灾眚"}, "象征小有过失。小有过错，但无伤大雅。"},
	{63, "既济", 63, "☵☲", 6, 3, "亨小利贞，初吉终乱", "水在火上，既济，君子以思患而豫防之", []string{"曳其轮，濡其尾，无咎", "妇丧其茀，勿逐，七日得", "高宗伐鬼方，三年克之，小人勿用", "繻有衣袽，终日戒", "东邻杀牛，不如西邻之禴祭，实受其福", "濡其首，厉"}, "象征成功。功成名就，但需防盛极必衰。"},
	{64, "未济", 64, "☲☵", 3, 6, "亨，小狐汔济，濡其尾，无攸利", "火在水上，未济，君子以慎辨物居方", []string{"濡其尾，吝", "曳其轮，贞吉", "未济，征凶，利涉大川", "贞吉悔亡，震用伐鬼方，三年有赏于大国", "贞吉无悔，君子之光，有孚吉", "有孚于饮酒，无咎，濡其首，有孚失是"}, "象征未完成。事物未成，仍需努力。"},
}

// GetHexagramByNumber 根据卦序获取卦象
func GetHexagramByNumber(num int) *Hexagram {
	for i := range HexagramList {
		if HexagramList[i].Number == num {
			return &HexagramList[i]
		}
	}
	return nil
}

// GetHexagramByStrokes 根据笔画数起卦
func GetHexagramByStrokes(strokes int) *Hexagram {
	lowerTrigram := ((strokes - 1) % 8) + 1
	upperTrigram := (((strokes / 8) - 1) % 8) + 1
	if upperTrigram < 1 {
		upperTrigram = 1
	}

	number := (upperTrigram-1)*8 + lowerTrigram
	return GetHexagramByNumber(number)
}

// GetHexagramByName 根据卦名获取卦象
func GetHexagramByName(name string) *Hexagram {
	for i := range HexagramList {
		if HexagramList[i].Name == name {
			return &HexagramList[i]
		}
	}
	return nil
}

// GetHexagramSymbol 获取卦象符号
func GetHexagramSymbol(upper, lower int) string {
	upperSymbol := TrigramMap[upper]
	lowerSymbol := TrigramMap[lower]
	return lowerSymbol + upperSymbol
}

// GetAllHexagrams 获取所有卦象
func GetAllHexagrams() []Hexagram {
	return HexagramList
}

type HexagramMatch struct {
	Hexagram     *Hexagram
	Score        int    `json:"score"`
	Reason       string `json:"reason"`
	WuxingCompat string `json:"wuxing_compat"`
}

func GetHexagramsByWuxing(wuxing string) []Hexagram {
	var results []Hexagram

	for _, h := range HexagramList {
		upperWuxing := TrigramWuxingMap[h.UpperTrigram]
		lowerWuxing := TrigramWuxingMap[h.LowerTrigram]
		if upperWuxing == wuxing || lowerWuxing == wuxing {
			results = append(results, h)
		}
	}

	if len(results) == 0 {
		return HexagramList[:8]
	}
	return results
}

func MatchHexagramWithXiyongshen(hexagram *Hexagram, xiyongshen []string) *HexagramMatch {
	if hexagram == nil {
		return nil
	}

	score := 50
	reason := ""
	wuxingCompat := "一般"

	upperWuxing := TrigramWuxingMap[hexagram.UpperTrigram]
	lowerWuxing := TrigramWuxingMap[hexagram.LowerTrigram]

	shengMap := map[string]string{
		"木": "火",
		"火": "土",
		"土": "金",
		"金": "水",
		"水": "木",
	}

	for _, xy := range xiyongshen {
		if upperWuxing == xy || lowerWuxing == xy {
			score += 30
			reason = "卦象五行补益喜用神"
			wuxingCompat = "最佳"
			break
		}

		if shengMap[xy] == upperWuxing || shengMap[xy] == lowerWuxing {
			score += 15
			reason = "卦象五行相生喜用神"
			wuxingCompat = "良好"
		}
	}

	if score == 50 {
		reason = "卦象五行与喜用神无直接关联"
		wuxingCompat = "一般"
	}

	if score > 100 {
		score = 100
	}

	return &HexagramMatch{
		Hexagram:     hexagram,
		Score:        score,
		Reason:       reason,
		WuxingCompat: wuxingCompat,
	}
}

func GetBestHexagramsForXiyongshen(xiyongshen []string, limit int) []HexagramMatch {
	var matches []HexagramMatch

	for _, h := range HexagramList {
		match := MatchHexagramWithXiyongshen(&h, xiyongshen)
		if match != nil {
			matches = append(matches, *match)
		}
	}

	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches
}

func AnalyzeHexagramWuxing(hexagram *Hexagram) string {
	if hexagram == nil {
		return ""
	}

	upperWuxing := TrigramWuxingMap[hexagram.UpperTrigram]
	lowerWuxing := TrigramWuxingMap[hexagram.LowerTrigram]

	shengMap := map[string]string{
		"木": "火",
		"火": "土",
		"土": "金",
		"金": "水",
		"水": "木",
	}

	result := "上卦" + TrigramNameMap[hexagram.UpperTrigram] + "（" + upperWuxing + "），"
	result += "下卦" + TrigramNameMap[hexagram.LowerTrigram] + "（" + lowerWuxing + "）"

	if upperWuxing == lowerWuxing {
		result += "，上下卦五行相同"
	} else if shengMap[upperWuxing] == lowerWuxing {
		result += "，上卦生下卦"
	} else if shengMap[lowerWuxing] == upperWuxing {
		result += "，下卦生上卦"
	}

	return result
}

type DayanResult struct {
	Hexagram      *Hexagram
	OriginalHex   *Hexagram
	YaoLines      []int   `json:"yao_lines"`
	DayanNumbers  []int   `json:"dayan_numbers"`
	ChangeYao     int     `json:"change_yao"`
	Interpretation string `json:"interpretation"`
}

func CalculateDayanNumber() int {
	total := 49

	sum := 0
	for i := 0; i < 3; i++ {
		part1 := randInt(1, total-1)
		part2 := total - part1

		part1 = part1 - 1

		remainder := part2 % 4
		if remainder == 0 {
			remainder = 4
		}

		sum += remainder
	}

	result := sum
	if result == 0 {
		result = 8
	}
	result = result % 4
	if result == 0 {
		result = 8
	}

	return result
}

func randInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return min + rand.Intn(max-min+1)
}

func GetYaoInfo(yao int) YaoResult {
	if result, ok := YaoMap[yao]; ok {
		return result
	}
	return YaoResult{Yao: yao, Name: "未知", YaoName: "", Meaning: ""}
}

func GetYaoName(yao int) string {
	names := map[int]string{
		6: "初六", 7: "初九", 8: "六二", 9: "九二",
	}
	if name, ok := names[yao]; ok {
		return name
	}
	return ""
}

type HexagramInterpretation struct {
	Overall     string   `json:"overall"`
	Career      string   `json:"career"`
	Love        string   `json:"love"`
	Health      string   `json:"health"`
	Fortune     string   `json:"fortune"`
	Auspicious  string   `json:"auspicious"`
	Inauspicious string `json:"inauspicious"`
}

func InterpretHexagram(hexagram *Hexagram, xiyongshen []string) *HexagramInterpretation {
	interp := &HexagramInterpretation{
		Overall:    hexagram.Interpretation,
		Career:     "事业平稳发展，需把握时机",
		Love:       "感情需要耐心经营",
		Health:     "注意调理身体，保持平衡",
		Fortune:    "财运平稳，不宜冒险",
		Auspicious: "祭祀、祈福、嫁娶",
		Inauspicious: "动土、破土",
	}

	upperWuxing := TrigramWuxingMap[hexagram.UpperTrigram]
	lowerWuxing := TrigramWuxingMap[hexagram.LowerTrigram]

	for _, xy := range xiyongshen {
		if upperWuxing == xy || lowerWuxing == xy {
			interp.Fortune = "五行相合，财运亨通"
			interp.Overall += "此卦与命格相合，大吉。"
			break
		}
	}

	return interp
}

func CastHexagramByTime(year, month, day, hour int) *DayanResult {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	seed := year*1000000 + month*10000 + day*100 + hour
	r.Seed(int64(seed))

	var yaoLines []int
	var changeYao int

	for i := 0; i < 6; i++ {
		num := r.Intn(4) + 6
		yaoLines = append(yaoLines, num)

		if num == 9 || num == 6 {
			changeYao = i + 1
		}
	}

	lowerTrigram := ((yaoLines[0] + yaoLines[1] + yaoLines[2]) % 8) + 1
	upperTrigram := ((yaoLines[3] + yaoLines[4] + yaoLines[5]) % 8) + 1

	hexagram := GetHexagramByNumber((upperTrigram-1)*8 + lowerTrigram)

	interpretation := ""
	if hexagram != nil {
		interpretation = hexagram.Interpretation
	}

	if changeYao > 0 {
		interpretation += "\n变爻：" + string(rune('一'+changeYao-1)) + "，"
		if yaoLines[changeYao-1] == 6 {
			interpretation += "老阴变阳，推荐关注变卦"
		} else {
			interpretation += "老阳变阴，推荐关注变卦"
		}
	}

	return &DayanResult{
		Hexagram:      hexagram,
		YaoLines:      yaoLines,
		ChangeYao:     changeYao,
		Interpretation: interpretation,
	}
}

func CastHexagramByDayan() *DayanResult {
	var yaoLines []int
	var dayanNumbers []int
	var changeYao int

	for i := 0; i < 6; i++ {
		num := CalculateDayanNumber()
		dayanNumbers = append(dayanNumbers, num)

		yao := 0
		switch num {
		case 6:
			yao = 6
		case 7:
			yao = 7
		case 8:
			yao = 8
		case 9:
			yao = 9
		default:
			yao = 7
		}

		yaoLines = append(yaoLines, yao)

		if num == 9 || num == 6 {
			changeYao = i + 1
		}
	}

	upperTrigram := 0
	lowerTrigram := 0

	lowerNum := yaoLines[0]*100 + yaoLines[1]*10 + yaoLines[2]
	upperNum := yaoLines[3]*100 + yaoLines[4]*10 + yaoLines[5]

	lowerTrigram = (lowerNum % 8) + 1
	upperTrigram = (upperNum % 8) + 1

	if lowerTrigram < 1 {
		lowerTrigram = 1
	}
	if upperTrigram < 1 {
		upperTrigram = 1
	}

	hexagram := GetHexagramByNumber((upperTrigram-1)*8 + lowerTrigram)

	var originalHex *Hexagram
	if changeYao > 0 {
		originalHex = hexagram
	}

	interpretation := ""
	if hexagram != nil {
		interpretation = hexagram.Interpretation
	}

	if changeYao > 0 && originalHex != nil {
		interpretation += "\n变爻：" + string(rune('一'+changeYao-1)) + "，"
		if yaoLines[changeYao-1] == 6 {
			interpretation += "老阴变阳，推荐关注变卦"
		} else {
			interpretation += "老阳变阴，推荐关注变卦"
		}
	}

	return &DayanResult{
		Hexagram:      hexagram,
		OriginalHex:   originalHex,
		YaoLines:      yaoLines,
		DayanNumbers:  dayanNumbers,
		ChangeYao:     changeYao,
		Interpretation: interpretation,
	}
}
