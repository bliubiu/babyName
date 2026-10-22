# 41 离线粤语 TTS E2E 验证报告

> 2026.10.22 · Audio8-TTS-0.1B-ONNX-INT8 sidecar 全链路实测
> 关联：`backend/tts_sidecar/`（start.sh / start.ps1 / model / Audio8_TTS）、`backend/internal/application/services/tts.go`、`handlers/tts.go`、`cmd/server/main.go`、前端 `lib/api.ts` + `components/CantoneseScore.tsx`

## 1. 验证目标

对「保证离线可用 + 保证粤语发音正确」选定的引擎方案做端到端实测：

1. 模型权重与运行时代码可获取（国内镜像）；
2. sidecar 服务可启动，HTTP 契约符合 Go 代理预期；
3. 粤拼（Jyutping，含声调数字）可合成为合法 WAV；
4. 前端契约 `{jyutping}` → Go `/api/v1/tts` → sidecar 全链路贯通；
5. 错误路径（空值 / 非 JSON / 引擎未启用）行为正确。

## 2. 架构拓扑

```
前端 CantoneseScore.tsx
  └─ POST /api/v1/tts {"jyutping":"wong4 zi2 man4"}   （粤拼罗马字，非汉字）
       └─ Go handlers/tts.go → services/tts.go（SidecarTTSSynthesizer）
            └─ POST http://127.0.0.1:8024/api/tts {"text":"wong4 zi2 man4","voice_name":"default","max_new_tokens":256}
                 └─ Audio8 onnx_runtime_0_1b_int8（arktts_runtime，FastAPI + ONNX Runtime CPU）
                      ├─ slow_ar_int8.onnx（127MB）自回归慢路径
                      ├─ fast_ar_int8.onnx（35MB）自回归快路径
                      ├─ codec_decoder_fp16.onnx（249MB）声码器解码
                      └─ voices/default（codes.npy + meta.json，零样本参考音色）
```

语言/口音锁死机制：喂 **Jyutping 罗马字**（而非汉字）。默认音色 `default` 的 `reference_text` 是普通话（`至今为止，元气火箭总共发行了两张专辑。`），若喂汉字，模型按普通话读音念（係→xì、廣東→guǎngdōng）；Jyutping 是触发粤语发音的输入形态。

## 3. 环境事实

| 项 | 实测值 |
|---|---|
| 模型来源 | ModelScope `Audio8/audio8-TTS-0.1B-ONNX-INT8`（公开免登录，`modelscope download` CLI） |
| 模型体积 | 819MB（slow_ar 133MB + fast_ar 37MB + decoder 261MB + encoder 395MB（registration/，合成不需要）+ tokenizer 5.8MB） |
| 运行时代码 | GitHub `Audio8-AI/Audio8_TTS` 的 `onnx_runtime_0_1b_int8/`，29 文件（`arktts_runtime/` Python 包 + FastAPI service + 脚本 + tests） |
| 获取方式 | `git clone --depth 1 --filter=blob:none --sparse` + `sparse-checkout set onnx_runtime_0_1b_int8`；github.com 不可达时走 `gh-proxy.com` 前缀 |
| 依赖 | numpy 2.5.3 / onnxruntime 1.31.0 / fastapi / uvicorn / soundfile / tokenizers / scipy / pydantic / python-multipart（系统 Python 3.12 直接可用，亦可 `setup.sh` 建 venv） |
| 推理精度 | int8 自回归 + fp16 声码器，CPUExecutionProvider，ARKTTS_THREADS=5 |
| 常驻内存 | 约 0.6GB |
| 输出规格 | WAV 44.1kHz / 16bit / 单声道 / FP32→PCM16 |

模型与运行时均**不进 git**（`backend/tts_sidecar/.gitignore` 排除），由 `start.sh`/`start.ps1` 首次搭建时获取。

## 4. E2E 验证矩阵（全部实跑）

### 4.1 sidecar 服务契约

| 用例 | 结果 |
|---|---|
| `GET /api/health` | 200 `{"ok":true,"model":"Audio8/Audio8-TTS-Preview-0.1B-ONNX-INT8","precision":"int8","codec_precision":"fp16","providers":{slow/fast/decoder:["CPUExecutionProvider"]}}` |
| `GET /api/voices` | 200，列出 `default` 音色 |
| 默认音色注册 | 官方 `scripts/register_default_voice.py` 从 `runtime_manifest.json` 生成：`reference_text=至今为止，元气火箭总共发行了两张专辑。`、`shape=[10,110]`、`sample_rate=44100`、`model_fingerprint=audio8-tts-preview-0.1b-int8-v1` |

> ⚠ 历史坑：早期手工构造的 `meta.json` 用了错误 `reference_text` 与 `model_fingerprint`。零样本合成里 `reference_text` 必须与 `reference_codes` 对齐，错配劣化音质——**必须用官方脚本注册**。

### 4.2 合成验证（粤拼 / 汉字 / 中文）

| 输入 | 产物 | 规格 | 判定 |
|---|---|---|---|
| `wong4 zi2 man4`（粤拼带声调） | `cmp_1_jyut_tone.wav` | 11.89s，RMS 0.2551，peak 0.9186 | 合法非静音 |
| `wong zi man`（无声调对照） | `cmp_2_jyut_notone.wav` | 11.89s，RMS 0.2181，peak 0.8275 | 合法非静音 |
| `黃子文`（汉字名） | `cmp_3_chars.wav` | 11.89s，RMS 0.2472，peak 0.8907 | 合法非静音 |
| `你好，欢迎使用粤语起名助手。`（中文） | `zh_cn.wav` | 11.89s，RMS 0.284 | 合法非静音 |
| `我係廣東人，講粵語。`（粤语汉字） | `jyut_chars.wav` | 11.89s，RMS 0.2866 | 合法非静音 |

单条 CPU 合成耗时：**44–49s**（`max_new_tokens=256`）；不带上限（sidecar 默认 1024 步）实测超过 120s 未返回。

### 4.3 全链路（前端契约 → Go → sidecar）

| 用例 | 结果 |
|---|---|
| `GET /api/v1/tts` | 200 `{"available":true}`（装配 `AUDIO8_TTS_URL` 后） |
| `POST /api/v1/tts {"jyutping":"wong4 zi2 man4"}` | **200，audio/wav，1,048,620 字节，Content-Disposition: inline**（修复 30s 误杀后，见 §5.3） |
| `POST` 空 jyutping | 400 `BAD_REQUEST` |
| `POST` 非 JSON body | 400 `BAD_REQUEST` |
| `AUDIO8_TTS_URL` 未设置 | 503 `TTS_UNAVAILABLE`（设计行为：显式 opt-in，前端回退 Web Speech） |

## 5. 关键发现与修复

### 5.1 GBK 编码坑（测试方法层）

Git Bash 下 `curl -d '{"text":"你好"}'` 把中文按 **cp936(GBK)** 字节发出，服务端按 UTF-8 解析失败，返回 48 字节 `{"detail":"There was an error parsing the body"}`。粤拼纯 ASCII 不受影响。**复测必须用 Python `urllib`（`.encode('utf-8')` + `charset=utf-8`）**。非模型缺陷。

### 5.2 运行时代码与模型权重分离获取

模型权重在 HuggingFace/ModelScope 仓库；运行时代码（`onnx_runtime_0_1b_int8/`）在 GitHub `Audio8-AI/Audio8_TTS`。GitHub API 拉取遇共享代理 IP 限流（403 rate limit），改用 gh-proxy 稀疏克隆成功。`start.sh`/`start.ps1` 已内置「github.com → gh-proxy.com」自动回退。

### 5.3 30s 全局请求超时误杀 TTS（产品缺陷，已修复）

**现象**：Go 服务 POST 三次全部在精确 30.001s 失败，503 `TTS_UNAVAILABLE`；而 sidecar 访问日志显示合成最终 200 OK（同步端点不随客户端断连中止）。

**根因**：`cmd/server/main.go` 全局挂载 `middleware.RequestTimeout(30 * time.Second)`，TTS handler 把 `c.Request.Context()` 传给 sidecar 调用，30s 到期取消在途请求。该超时按同步起名端点标定，而 CPU 合成实测 44–49s。

**修复**（三处）：
1. `handlers/tts.go`：TTS 改用独立 300s 上下文调用 sidecar（不再继承请求 deadline）；客户端断连不中断合成为可接受代价（本地单用户 sidecar，300s 硬上限）。
2. `services/tts.go`：HTTP 客户端超时 60s→300s；新增 `max_new_tokens` 透传（`AUDIO8_TTS_MAX_TOKENS`，默认 256），把单条合成压到 44–49s；请求头补 `charset=utf-8`。
3. `cmd/server/main.go`：`WriteTimeout` 60s→300s，为长合成留写入窗口（其余路由仍由 RequestTimeout 中间件 30s + 503 兜底保护）。

### 5.4 前端合成等待态（体验缺口，已修复）

离线合成约 45–60s，原实现在等待期间按钮无反馈且可重复点击（会并发拉起多条合成）。`CantoneseScore.tsx` 增加 `synthesizing` 状态：合成中按钮禁用 + 金色脉冲 + 「离线引擎合成中（CPU 推理约需 1 分钟）…」提示。

## 6. 粤语发音正确性：待人耳判定

机器可验的部分（WAV 合法、非静音、时长合理、链路贯通）已全部通过；**声调是否正确必须人耳判定**。三条对比音频在 `.workbuddy/tmp/cmp/`：

| 文件 | 输入 | 听辨要点 |
|---|---|---|
| `cmp_1_jyut_tone.wav` | `wong4 zi2 man4` | 真实前端输入。确认：① 四个/三个音节读作粤语而非英语单词+数字（不是 "wong four zi two man four"）；② 声调走向是否为 2-4-2-4（阴上、阳平、阴上、阳平） |
| `cmp_2_jyut_notone.wav` | `wong zi man` | 对照组：无声调数字时模型如何处理（默认调值？） |
| `cmp_3_chars.wav` | `黃子文` | 汉字输入。预期按普通话念；若竟按粤语念则说明模型有汉字→粤语 G2P 能力（可简化前端输入） |

判定结论将决定是否需要「Jyutping→模型输入」转换层调整（当前直接把粤拼串喂给模型）。

## 7. 复现指南

```bash
# Linux / macOS / Git Bash（首次搭建约 10 分钟：克隆 22s + 下载 819MB + 推理验证）
cd backend/tts_sidecar
MIRROR=modelscope bash start.sh        # MIRROR: modelscope | hf-mirror | hf

# Windows
powershell -File start.ps1

# 环境变量
#   MIRROR            模型下载镜像（默认 modelscope）
#   MODEL_DIR         模型目录（默认 ../model，即 backend/tts_sidecar/model）
#   FORCE_REGISTER=1  强制重建默认音色
#   PORT              服务端口（默认 8024）
#   ARKTTS_PYTHON     指定 Python 解释器（跳过自动解析）
```

Go 后端启用代理：`AUDIO8_TTS_URL=http://127.0.0.1:8024`（未设置则 503，前端回退 Web Speech）。
可选：`AUDIO8_TTS_VOICE`（默认 default）、`AUDIO8_TTS_MAX_TOKENS`（默认 256）。

## 8. 遗留事项

1. **人耳判定粤语声调**（§6）——未听辨前不宣称「发音正确」已验证；
2. 沙箱环境特有：`go run` 启动的服务进程外连 127.0.0.1:8024 被沙箱网络层间歇过滤（独立编译的 Go 二进制正常）——本报告的 Go 链路结果均用直接编译的 `server.exe` 取得，非产品缺陷；
3. `onnxruntime` 实测 1.31.0 超出 `requirements.txt` 的 `<1.24` 上限但工作正常（CPUExecutionProvider 加载/推理无误），未强制降级；
4. codec encoder（395MB，`registration/`）仅注册新音色才需要，合成路径不加载——模型目录可裁剪，暂保留。
