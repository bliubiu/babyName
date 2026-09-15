package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// probeRune 用于校验候选字体确实含中文字形。
//
// 只检查文件能否解析是不够的：macOS 的 PingFang.ttc 首个字体是西文，
// Linux 上也存在只有拉丁字形的 .ttf，用它们渲染会得到一整页空白/方框。
const probeRune = '中'

// fontCandidates 返回按当前平台优先排序的候选字体路径。
var fontCandidates = func() []string {
	win := []string{
		`C:\Windows\Fonts\simhei.ttf`,
		`C:\Windows\Fonts\msyh.ttc`,
		`C:\Windows\Fonts\msyh.ttf`,
		`C:\Windows\Fonts\simsun.ttc`,
		`C:\Windows\Fonts\simkai.ttf`,
		`C:\Windows\Fonts\Deng.ttf`,
	}
	linux := []string{
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
		"/usr/share/fonts/truetype/arphic/uming.ttc",
		"/usr/share/fonts/truetype/arphic/ukai.ttc",
	}
	darwin := []string{
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/Library/Fonts/Arial Unicode.ttf",
	}
	// 本平台优先，其余作为兜底（例如 WSL / 多平台共享的挂载目录）
	order := [][]string{win, linux, darwin}
	switch runtime.GOOS {
	case "darwin":
		order = [][]string{darwin, linux, win}
	case "linux":
		order = [][]string{linux, darwin, win}
	}
	out := make([]string, 0, len(win)+len(linux)+len(darwin))
	for _, group := range order {
		out = append(out, group...)
	}
	return out
}()

// FindSystemCJKFont 在当前系统里寻找一个可用的中文 TrueType 字体。
//
// 查找顺序：环境变量 NAMER_PDF_FONT → 平台常见路径。
// 每个候选都会被真实解析并校验含 probeRune 字形，避免选中「能解析但无中文」的字体。
//
// 若均不可用，返回带排查指引的错误——调用方应把它当作明确的配置问题上报，
// 而不是退化成输出空白文字的报告。
func FindSystemCJKFont() (string, error) {
	candidates := make([]string, 0, len(fontCandidates)+1)
	if env := os.Getenv("NAMER_PDF_FONT"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, fontCandidates...)

	var tried []string
	for _, p := range candidates {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		f, err := parseTTF(data)
		if err != nil {
			tried = append(tried, fmt.Sprintf("%s（%v）", p, err))
			continue
		}
		if gid, ok := f.cmap[probeRune]; !ok || gid == 0 {
			tried = append(tried, fmt.Sprintf("%s（不含中文字形）", p))
			continue
		}
		return p, nil
	}

	msg := "未找到可用的中文 TrueType 字体"
	if len(tried) > 0 {
		msg += "；已尝试但不可用的候选：" + fmt.Sprint(tried)
	}
	return "", fmt.Errorf("%s。请安装中文字体，或设置环境变量 NAMER_PDF_FONT 指向一个"+
		"含中文的 .ttf/.ttc 文件（例：%s）", msg, filepath.FromSlash("/usr/share/fonts/truetype/wqy/wqy-microhei.ttc"))
}
