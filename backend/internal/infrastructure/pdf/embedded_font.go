package pdf

import (
	_ "embed"
	"fmt"
	"sync"
)

// embeddedFontData 是随二进制分发的开源中文字体（Noto Sans SC，SIL OFL 1.1）。
//
// 它让 PDF 报告在任何机器开箱可用：不依赖系统是否安装中文字体，也不需要设置
// NAMER_PDF_FONT。字体为 TrueType（glyf）轮廓，可被本包解析器直接处理；
// 子集化时可变轴会被固化成默认轮廓，渲染结果与静态字体一致。
//
// 法律合规：Noto Sans SC 采用 SIL Open Font License 1.1，允许随应用程序
// 嵌入与再分发，但须随附版权声明与许可证全文（见同目录 OFL.txt）。
//
//go:embed fonts/NotoSansSC.ttf
var embeddedFontData []byte

// EmbeddedFontName 用于日志与诊断展示的内置字体标识。
const EmbeddedFontName = "NotoSansSC.ttf (built-in, SIL OFL 1.1)"

var (
	embeddedOnce sync.Once
	embeddedFont *Font
	embeddedErr  error
)

// EmbeddedFontBytes 返回内置字体的原始字节。
//
// 返回的是 go:embed 持有的只读切片，调用方不得修改；parseTTF / subsetFor
// 全程只读该数据（子集化会先拷贝到新缓冲再改写），可安全并发使用。
func EmbeddedFontBytes() []byte { return embeddedFontData }

// LoadEmbeddedFont 解析并返回内置的开源中文字体句柄，进程内只解析一次。
//
// CJK 字体体积大（约 17 MB）、解析需重建 cmap（约 3 万项），因此用 sync.Once
// 缓存不可变的 *Font；后续每次生成报告仅按用到的 rune 子集化，不再重解析。
func LoadEmbeddedFont() (*Font, error) {
	embeddedOnce.Do(func() {
		embeddedFont, embeddedErr = ParseFont(embeddedFontData)
		if embeddedErr != nil {
			embeddedErr = fmt.Errorf("解析内置中文字体失败: %w", embeddedErr)
		}
	})
	return embeddedFont, embeddedErr
}
