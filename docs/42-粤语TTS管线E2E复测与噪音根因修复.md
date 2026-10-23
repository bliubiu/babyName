# 42 粤语 TTS 管线 E2E 复测：噪音根因与修复

> 2026.10.23 · 关联 `docs/41-离线粤语TTS-E2E验证报告.md`、前端 `lib/api.ts` + `components/CantoneseScore.tsx`、后端 `handlers/tts.go` + `services/tts.go`
> 复现脚本与产物：`backend/tts_sidecar/` 之外的 `.workbuddy/tmp/tts_verify/`（3 条 WAV + 频谱图 + `summary.json`）

## 1. 复测目标

前次（doc 41）E2E 报告在 §6 明确把"粤语声调是否正确"留作**人耳判定待办**，未做听辨。本次用户实测反馈：**「生成的音频全是噪音，听不清内容」**。本次目标：

1. 用真实模型**实证复现**该"噪音"现象，定位根因；
2. 给出可量化的客观判别（而非只靠人耳）；
3. 修复并验证。

## 2. 实证复现（直接调用真实 ArkTtsRuntime，绕过 HTTP 层）

对三种输入各合成一条 WAV（44.1kHz / 16bit / 单声道，max_new_tokens=256，与线上一致），计算客观指标：

| 样本 | 输入 | 含义 | RMS | **过零率 ZCR** | 谱平坦度 | 时长 |
|---|---|---|---|---|---|---|
| A | `wong4 zi2 man4` | **现网路径**：前端发给离线引擎的内容（粤拼） | 0.255 | **0.1107** | 5.0e-05 | 11.89s |
| B | `黃子文` | **修复后**：改为汉字 | 0.247 | **0.0437** | 9.1e-06 | 11.89s |
| C | `你好` | 健康对照：简单普通话 | 0.281 | **0.0435** | 8.0e-06 | 11.89s |

> 判别依据：**过零率（ZCR）**。干净语音（B/C）ZCR≈0.043；而 A（粤拼输入）ZCR=0.1107，是正常语音的 **2.5 倍**。高 ZCR = 高频、非音调结构、类噪声——即"听不清的噪音"的客观指纹。谱平坦度三者都极低（均为有结构的语音信号，非白噪声），说明 A 不是白噪声，而是**模型把 `wong4 zi2 man4` 当成英文字母+数字乱读成的无意义 gibberish**（"wong four zi two man four"），人耳听感即"噪音"。

频谱对比图见 `.workbuddy/tmp/tts_verify/spectrograms.png`：A 的高频能量明显更碎、更"毛刺"，B/C 能量集中在清晰的共振峰带上。

**三份 WAV 可由用户直接试听比对**：`A_jyutping_input.wav`（= 之前听到的噪音）、`B_chars_input.wav`（修复后）、`C_control_mandarin.wav`（健康对照）。

## 3. 根因分析

### 3.1 直接根因：前端把「粤拼罗马字」喂给了无 G2P 层的引擎

调用链：`CantoneseScore.tsx` → `synthesizeCantoneseAudio(name.jyutping)` → `POST /api/v1/tts {"jyutping":"wong4 zi2 man4"}` → `services/tts.go` 原样转发 `text` → 引擎 `arktts_runtime/prompt.py` 仅做 `self.tokenizer.encode(text)`。

而该 tokenizer 是**中文训练的 HF tokenizer，无任何 Jyutping→音素（G2P）转换层**（全仓 grep `jyutping|cantonese|pinyin|g2p|phonem` 零命中）。`wong4 zi2 man4` 这类拉丁字母+数字进入中文词表即 **OOD**，声码 LM 生成错乱的声码 token → 解码成高 ZCR 的乱读语音 = 噪音。

> 注意：doc 41 §2 的"喂 Jyutping 罗马字触发粤语发音"是**未经人耳验证的假设**，本次实测证伪。英文 README 与全部示例输入都是**汉字**（如 `--text "这是一个中文测试"`）。

### 3.2 更深的底层限制：离线引擎默认音色是**普通话**

`voices/default/meta.json` 的 `reference_text = "至今为止，元气火箭总共发行了两张专辑。"`（普通话）。即便把输入改成汉字（样本 B），引擎也只是按**普通话口音**念，并不是粤语。Audio8-TTS-0.1B 是零样本 TTS：文本决定内容、参考音色决定**口音/音色**。默认音色为普通话 → 输出必为普通话。

**结论**：当前离线引擎**无法产出真正的粤语发音**。要得到粤语，必须二选一：
- 方案甲：向 sidecar **注册一个粤语参考音色**（提供一段粤语人声+文本，走 `/api/voices/register`，依赖已落地的 `model/registration` 编解码器），再喂**粤语汉字**；
- 方案乙：**优先用 Web Speech `zh-HK`**（前端已实现兜底，本机装有粤语嗓音时即为真粤语），离线引擎退居降级。

## 4. 本次已落地修复（消除"噪音"症状）

- **前端** `api.ts`：`synthesizeCantoneseAudio(jyutping)` → `synthesizeCantoneseAudio(text)`，请求体 `{"jyutping":...}` → `{"text":...}`。
- **前端** `CantoneseScore.tsx`：离线引擎调用改为传入**汉字原名 `fullName`**（`synthesizeCantoneseAudio(fullName)`），不再传 `name.jyutping`。
- **后端** `handlers/tts.go`：入站结构体 `Jyutping` → `Text`（`json:"text"`），错误信息与注释同步修正。
- **后端** `services/tts.go` + 注释：明确"引擎无 Jyutping→音素层，必须由调用方传汉字"。

> 修复后，离线引擎输出从"高 ZCR 乱读噪音"（样本 A）变为"可懂语音"（样本 B，ZCR 0.044）。但口音仍是**普通话**（见 §3.2）。

## 5. 验证

- `go build ./...` → exit 0
- `go test ./internal/application/handlers/... ./internal/application/services/...` → ok（exit 0）
- 前端 `node node_modules/typescript/lib/tsc.js --noEmit` → exit 0
- 全仓无残留旧 `jyutping` TTS 契约引用（`Name.Jyutping` 域名属性与本次契约无关，保留）
- 实证复现见 §2（3 条 WAV + 频谱图 + `summary.json`）

## 6. 遗留事项与下一步建议

1. **真·粤语发音**：当前离线引擎只到"可懂普通话"。请确认走方案甲（注册粤语音色）还是方案乙（优先 Web Speech zh-HK）。我可据此实现。
2. doc 41 §2 "Jyutping 触发粤语" 论断已证伪，建议在其顶部加勘误说明。
3. 复现命令（需先 `cd backend/tts_sidecar && powershell -File start.ps1` 起 sidecar）：
   ```bash
   python .workbuddy/tmp/tts_verify/verify_tts.py   # 直调引擎，出 3 条 WAV + 指标
   python .workbuddy/tmp/tts_verify/make_spec.py     # 出频谱对比图
   ```
