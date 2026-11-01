package schema

import (
	"path"
	"sort"
)

// 本文件固化「哪些 data 源文件必须进入 SQLite」这一事实（docs/40）。
//
// 清单的唯一真源是 schema.Definitions：
//   - 单文件物料用 Source；
//   - 多文件物料用 Files（当前仅 classics 的 23 个 JSON）。
//
// `materials manifest` 直接消费本文件的 Manifest/AllSourceFiles，部署与 CI
// 无需再维护第二份文件列表；classics 导入也以 ClassicsFiles 派生，避免
// 「目录里多一个 JSON 就静默多导一份」与「清单漏登记就静默少导一份」。

// ClassicsFiles 经典语料的权威源文件清单（相对 dataDir，字典序）
//
// 新增/删除经典文件必须同时更新本清单与 classicsFileCategory，
// 由 sqlite 包的守护测试双向校验，防止只改一处。
var ClassicsFiles = []string{
	"baijiaxing.json",
	"chuci.json",
	"cifu.json",
	"daxue.json",
	"dizigui.json",
	"guwenguanzhi.json",
	"lunyu.json",
	"mengzi.json",
	"qianjiashi.json",
	"qianziwen.json",
	"sanzijing-new.json",
	"sanzijing-traditional.json",
	"shenglvqimeng.json",
	"shici.json",
	"shijing.json",
	"wenzimengqiu.json",
	"yijing.json",
	"youxueqionglin.json",
	"yuanqu.json",
	"yuefu.json",
	"zengguangxianwen.json",
	"zhongyong.json",
	"zhuzijiaxun.json",
}

// SourceFiles 返回该物料的源文件清单（相对 dataDir）
//
// Files 非空时以 Files 为准，否则回退单文件 Source。返回值是副本，
// 调用方修改不会污染注册表。
func (d Definition) SourceFiles() []string {
	if len(d.Files) > 0 {
		out := make([]string, len(d.Files))
		copy(out, d.Files)
		return out
	}
	return []string{d.Source}
}

// ManifestEntry 单个物料的入库源清单条目
type ManifestEntry struct {
	// Material 物料名（与 materials_state.material 一致）
	Material string `json:"material"`
	// Required 源文件缺失时是否必须失败
	Required bool `json:"required"`
	// Files 源文件路径（相对 dataDir）
	Files []string `json:"files"`
}

// Manifest 返回按注册顺序排列的全部物料源清单
//
// 注册顺序即导入语义顺序；同一文件可能出现在多个物料下（如 yijing.json
// 既是 yijing 物料源，也是 classics 的一部），去重由 AllSourceFiles 负责。
func Manifest() []ManifestEntry {
	out := make([]ManifestEntry, 0, len(Definitions))
	for _, d := range Definitions {
		out = append(out, ManifestEntry{
			Material: d.Name,
			Required: d.Required,
			Files:    d.SourceFiles(),
		})
	}
	return out
}

// AllSourceFiles 返回去重并排序后的全部入库源文件（相对 dataDir）
func AllSourceFiles() []string {
	seen := make(map[string]bool, len(Definitions))
	out := make([]string, 0, len(Definitions))
	for _, d := range Definitions {
		for _, f := range d.SourceFiles() {
			f = path.Clean(f)
			if seen[f] {
				continue
			}
			seen[f] = true
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}
