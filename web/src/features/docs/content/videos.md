使用本站 API Key，通过统一的 OpenAI 视频入口创建、查询和下载视频。Seedance、MiniMax 等模型的供应商请求由网关转换；调用方填写 `prompt`、`seconds` 和模型支持的扩展，无需构造 `content[]` 或切换供应商原生地址。分组、可用模型与价格以账户可见的 [分组说明](/docs/groups) 和 [模型广场](/pricing) 为准。

## 接口一览

| 操作 | 方法与地址 |
| --- | --- |
| 创建任务 | `POST {{BASE_URL}}/v1/videos` |
| 查询任务 | `GET {{BASE_URL}}/v1/videos/{id}` |
| 下载成片 | `GET {{BASE_URL}}/v1/videos/{id}/content` |
| 查询成片响应头 | `HEAD {{BASE_URL}}/v1/videos/{id}/content` |

所有请求携带 `Authorization: Bearer sk-xxxxxxxx`。创建支持 JSON 和 multipart；保存返回的本站任务 `id`，后续查询与下载均使用这个 ID。任务未结束时不要重新提交，重新创建会产生新的任务和费用。

## 请求格式

```json
{
  "model": "YOUR_VIDEO_MODEL",
  "prompt": "一只红色纸船在湖面缓缓漂动",
  "seconds": "8",
  "size": "1280x720"
}
```

| 字段 | 含义 |
| --- | --- |
| `model` | 必填，替换为账户可用的视频模型名 |
| `prompt` | 必填，非空提示词 |
| `seconds` | 输出时长，整数或十进制整数字符串；固定时长必须符合模型范围，入口上限为 3600 秒。支持自动时长的模型可传 `-1` 或 `"-1"` |
| `size` | 可选输出像素，格式为 `宽x高`，例如 `1280x720`；使用模型支持的尺寸 |
| `input_reference` | 可选参考图；JSON 地址和本地文件的传法见下文 |
| `resolution` | 模型扩展，清晰度或画质档位，例如 `720p`、`768p`、`2k`；以所选模型为准 |
| `aspect_ratio` | 模型扩展，输出比例，例如 `16:9`、`9:16`，或支持模型的 `adaptive` |
| `callback_url` | 本站扩展，可选的公网 HTTPS 任务通知地址 |

建议明确填写 `seconds`，并选择 `size` 或 `resolution` 与 `aspect_ratio` 描述输出规格。模型范围优先于入口上限；使用此入口不要求所有模型遵循 Sora 的时长或尺寸枚举。不要把 `metadata.seconds`、旧 `duration` 或 `ratio` 当成本页标准字段；扩展按下文的明确字段填写。

旧 `video-v1` 契约已停止受理，提交 `contract_version` 会报错。参考素材使用本页的 `input_reference` 或 `images`、`videos`、`audios`，高级参数直接放在顶层，不提交旧 `references`、`options` 或供应商原生 `content[]`。

### 尺寸与比例

- 仅传 `size` 时，网关根据像素读取清晰度与比例。
- 同时传 `size` 和 `resolution: "720p"` 等档位时，像素短边必须与档位一致。例如 `1280x720` 可搭配 `720p`，不能搭配 `1080p`。
- 同时传 `size` 和 `aspect_ratio` 时，比例必须与像素一致。例如 `1280x720` 对应 `16:9`，不能搭配 `9:16` 或 `adaptive`。
- 使用 `adaptive` 或没有统一像素定义的 `2k` 档位时，省略 `size`，只传模型支持的 `resolution` 与 `aspect_ratio`。特惠 MiniMax 的 `2k-pro` 是模型名称中的画质后缀，对应请求的 `resolution` 仍为 `2k`。不要把 `2k` 自行改写成 `1440p`。

字段冲突、不支持的档位或时长会明确报错，不静默修改为其他规格。

## 模型能力与自动时长

以下为当前接入的模型族范围，不代表所有账户都已开通全部模型或档位。模型名称中的连字符、前缀与画质后缀具有实际含义，不要互换。

| 模型族 | 固定时长 | `seconds: -1` | 清晰度与主要参考能力 |
| --- | --- | --- | --- |
| Seedance 2.0 | 4–15 秒 | 支持；按 15 秒上限预留 | 480p、720p、1080p、4k；最多 9 图、3 视频、3 音频，总数最多 15 |
| Seedance 2.5 | 4–30 秒 | 支持；按 30 秒上限预留 | 480p、720p、1080p；最多 30 图、10 视频、10 音频，总数最多 50 |
| 特惠 `MiniMaxH3-*` | 5–15 秒，具体按渠道档位 | 声明支持的特惠别名可用；按 15 秒上限预留 | 480p、720p、2k 等由模型与渠道绑定；普通参考图及音频，不支持首尾帧或参考视频 |
| 原生 `MiniMax-H3` | 当前接入服务为 1–15 秒 | 不支持 | 当前接入服务固定 768p；最多 9 图、3 视频、3 音频，总数最多 12 |

`seedance-2.0`、`seedance-2.5` 在本项目是公开别名，映射到已接入的 `doubao-seedance-2-0-260128`、`doubao-seedance-2-5-260628`。这不等于另外接入同名的供应商简化模型；调用方仍按本页统一格式提交。

特惠 MiniMax 名称中 `MiniMax` 与 `H3` 之间没有连字符，例如 `MiniMaxH3-720p-sec`。`MiniMaxH3`、`[c]MiniMaxH3`、带 480p / 720p / 2k / 2k-pro 及可选 `-sec` 的公开别名，只有在账户渠道配置了对应映射和自动时长能力时才能使用 `-1`。原生 `MiniMax-H3` 与这些别名不是同一协议。带 `[c]` 的特惠 Seedance 也不能照搬满血模型的自动时长、参考或画质能力；例如特惠 2.5 可被限定为固定 30 秒、720p。

原生 MiniMax 的 1–15 秒、768p 是当前接入服务的约定，不能据此推断其他 MiniMax 官方托管模型也采用相同范围。

自动时长的最小示例：

```json
{
  "model": "seedance-2.5",
  "prompt": "完整呈现纸船穿过拱桥的过程",
  "seconds": "-1",
  "resolution": "720p",
  "aspect_ratio": "16:9"
}
```

`-1` 表示由支持的服务决定成片时长，不代表负秒数费用。完成后按可信实际用量结算；没有实际用量时保留已有预留估算，不把估算 Token 冒充真实用量。

## 创建任务

将 `YOUR_VIDEO_MODEL` 替换为可用模型，并按该模型调整时长和尺寸：

<!-- code-group -->

```bash
curl '{{BASE_URL}}/v1/videos' \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "YOUR_VIDEO_MODEL",
    "prompt": "A red paper boat on a calm lake",
    "seconds": "8",
    "size": "1280x720"
  }'
```

```powershell
$body = @{
  model = "YOUR_VIDEO_MODEL"
  prompt = "一只红色纸船在湖面缓缓漂动"
  seconds = "8"
  size = "1280x720"
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "{{BASE_URL}}/v1/videos" `
  -Headers @{ Authorization = "Bearer sk-xxxxxxxx" } `
  -ContentType "application/json; charset=utf-8" `
  -Body ([Text.Encoding]::UTF8.GetBytes($body))
```

<!-- /code-group -->

创建响应包含本站公开任务 ID，例如：

```json
{
  "id": "task_example",
  "object": "video",
  "model": "YOUR_VIDEO_MODEL",
  "status": "queued"
}
```

## 参考图片、视频与音频

| 字段 | 格式与用途 |
| --- | --- |
| `input_reference` | 单图通用参考；不承诺供应商未声明的首帧锁定 |
| `images` | 有序数组，元素为 `{"url":"https://...","role":"reference_image"}`；支持模型还可使用 `first_frame`、`last_frame` |
| `videos` | 有序的参考视频 URL 数组 |
| `audios` | 有序的参考音频 URL 数组 |

素材顺序决定引用编号。建议使用任务期间可公开访问的 HTTPS 素材地址，并只使用一套图片输入；不要把同一图片同时放入 `input_reference` 与 `images`。首帧、尾帧各最多一张，帧模式不能混入普通参考图、参考视频或参考音频。当前 Seedance 尾帧需搭配首帧；原生 MiniMax 可单独使用尾帧。

下面使用 Seedance 2.0 支持的图片、视频与音频组合；请替换示例素材地址：

```json
{
  "model": "seedance-2.0",
  "prompt": "参考图中的纸船沿参考视频的运动轨迹前进，配合参考音频的节奏",
  "seconds": "5",
  "resolution": "720p",
  "aspect_ratio": "16:9",
  "images": [
    { "url": "https://example.com/boat.png", "role": "reference_image" }
  ],
  "videos": ["https://example.com/motion.mp4"],
  "audios": ["https://example.com/rhythm.mp3"]
}
```

Seedance 2.0 的音频参考需搭配图片或视频；特惠 MiniMax 音频需搭配图片。原生 MiniMax 可以只传音频，但纯文本或纯音频必须选择固定比例，有图片或视频时才可用 `adaptive`。素材编码、文件大小、分辨率与累计参考时长还须符合所选服务要求。

### 单图与本地文件

JSON 请求使用 `input_reference.image_url`；将示例地址换成任务期间可访问的 HTTPS 图片地址：

```json
{
  "model": "YOUR_VIDEO_MODEL",
  "prompt": "让参考图片中的纸船缓缓漂动",
  "seconds": "8",
  "size": "1280x720",
  "input_reference": {
    "image_url": "https://example.com/boat.png"
  }
}
```

支持文件输入的渠道可通过 multipart 的 `input_reference` 文件字段上传单张图片。文件保持二进制形式；明确首尾帧角色时使用上文的 `images[].role`，不使用旧 `first_frame`、`last_frame` 或 `audio` 文件字段。

<!-- code-group -->

```bash
curl '{{BASE_URL}}/v1/videos' \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -F 'model=YOUR_VIDEO_MODEL' \
  -F 'prompt=Animate the paper boat in the reference image' \
  -F 'seconds=8' \
  -F 'size=1280x720' \
  -F 'input_reference=@boat.png;type=image/png'
```

```powershell
curl.exe "{{BASE_URL}}/v1/videos" `
  -H "Authorization: Bearer sk-xxxxxxxx" `
  -F "model=YOUR_VIDEO_MODEL" `
  -F "prompt=Animate the paper boat in the reference image" `
  -F "seconds=8" `
  -F "size=1280x720" `
  -F "input_reference=@boat.png;type=image/png"
```

<!-- /code-group -->

不要手动设置 multipart 的 `Content-Type`，客户端会生成与请求体一致的 boundary。multipart 中的 `images`、`videos`、`audios`、`tools` 等数组或对象需填写完整 JSON 文本，布尔扩展填写 `true` / `false`。

文件或 Data URL 能否传给上游由所选渠道决定。原生 MiniMax 当前接入服务仅接受公开 HTTPS 素材，不能假定上传本地文件后会自动获得公网地址；可先托管文件，ART 会为要求公开 URL 的模型临时托管本机素材。OpenAI 的 `file_id` 不会自动成为其他供应商可读的文件，跨供应商调用请使用受支持的文件或图片 URL。参考图格式、尺寸与文件大小仍须满足所选模型和本站上传限制。

## 高级扩展

扩展字段放在请求顶层。只有所选模型和渠道声明支持的字段才能使用；显式 `false` 会保留，不会被默认值覆盖。不支持的字段或素材组合会报错，不通过静默丢弃参数或修改时长、清晰度来继续生成。

| 字段 | 类型与用途 |
| --- | --- |
| `generate_audio` | 布尔值，生成音频；Seedance 支持，不能直接套用于原生 MiniMax |
| `watermark` | 布尔值，按支持服务控制水印 |
| `return_last_frame` | 布尔值，请求服务返回生成视频的尾帧；公共下载能力见下文 |
| `output_format` | 支持模型的 `mp4` / `mov`，当前 Seedance 2.5 可用 |
| `omni_reference_task_type` | Seedance 2.5 的 `auto`、`reference`、`edit`、`extend` |
| `priority`、`execution_expires_after`、`safety_identifier` | 按所选服务支持范围设置优先级、执行期限与调用方标识 |
| `tools` | 支持模型的工具数组，例如 `[{"type":"web_search"}]`；可能产生额外费用 |

Seedance 2.5 的首帧或首尾帧、编辑和延长使用 `aspect_ratio: "adaptive"`，同时省略 `size`。编辑必须提供参考视频并使用 `seconds: "-1"`；延长必须提供参考视频，时长按模型支持范围填写。

```json
{
  "model": "seedance-2.5",
  "prompt": "保留人物动作，将参考视频的背景改为蓝色摄影棚",
  "seconds": "-1",
  "resolution": "720p",
  "aspect_ratio": "adaptive",
  "videos": ["https://example.com/source.mp4"],
  "omni_reference_task_type": "edit",
  "output_format": "mov",
  "generate_audio": false,
  "watermark": false
}
```

`return_last_frame` 用于请求支持的服务生成尾帧。当前本站公共视频接口只提供成片下载，没有单独的尾帧下载地址，任务查询也不返回独立尾帧链接。输入尾帧角色 `last_frame` 与请求输出尾帧是不同功能。

## 查询与下载

```bash
curl '{{BASE_URL}}/v1/videos/{id}' \
  -H 'Authorization: Bearer sk-xxxxxxxx'
```

状态包括 `queued`、`in_progress`、`completed` 和 `failed`。失败原因见 `error`；任务与费用也可在 [使用日志](/usage-logs) 中查看。

完成后通过本站内容接口下载，支持 `HEAD` 和单段 `Range`。返回的 `result.url` 也是本站授权内容链接；授权链接本身具有访问权限，请妥善保管。提供商原始下载地址不会直接暴露给调用方。

```bash
curl --location '{{BASE_URL}}/v1/videos/{id}/content' \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -o video.mp4
```

任务只能由所属用户读取；更换 API Key 不会转移任务归属。已持久化的历史任务沿用创建时的上游绑定，不因切换模型、渠道或凭证轮换重新创建。下载客户端应支持重定向；保留返回链接的完整访问参数，不要手动拼接供应商下载地址或将本站 API Key 发送给外部媒体域名。

## 回调通知

需要通知时在请求顶层填写 `"callback_url": "https://example.com/video-callback"`。这是本站扩展，由 new-api 保存和发送，不会转发给模型供应商。

通知正文使用本站任务号和 `event_id`，请求头 `X-MAI-Event-ID` 携带同一个事件 ID；接收方按该 ID 去重并返回 2xx。回调失败会重试，仍可随时使用查询接口获取状态。回调重试不会重新创建视频任务。

## 计费与任务职责

视频可按次、秒数、分辨率或实际 Token 用量定价，具体以 [模型广场](/pricing) 中账户可见的配置为准。创建时预扣，完成后按保存的价格快照和可信实际用量调整；支持自动时长的模型按其允许上限预留。缺少实际用量时保留已有预扣估算，不虚构真实 Token；失败任务按既有规则退款，重复结算或退款不会重复调整余额。

new-api 负责用户认证、任务归属、价格与余额、令牌额度、任务记录和退款。RS Gateway 负责供应商选择、协议转换、上游任务持久化和内容读取。供应商采购倍率与账户售价倍率分别计算。

参数或模型权限错误时，保留错误响应和响应头 `X-Oneapi-Request-Id`，按 [错误处理](/docs/errors) 排查后修正请求。出现创建超时、连接中断、服务端错误或无法解析的创建响应时，结果可能不明；系统不会自动换供应商重建，调用方也应先查任务和日志。客户端断开不代表上游任务未创建，没有确定结果前不要重新提交付费任务。
