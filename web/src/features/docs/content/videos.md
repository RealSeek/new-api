视频生成使用 OpenAI 视频接口的请求格式。调用时使用本站 API Key，选择账户已开通的视频模型；分组与可用模型见 [分组说明](/docs/groups) 和 [模型广场](/pricing)。

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
| `seconds` | 输出时长，使用正整数或十进制整数字符串；本站上限为 3600 秒，模型自身的限制可能更小 |
| `size` | 输出尺寸，格式为 `宽x高`，例如 `1280x720`；使用模型支持的尺寸 |
| `input_reference` | 可选参考图；JSON 地址和本地文件的传法见下文 |
| `callback_url` | 本站扩展，可选的公网 HTTPS 任务通知地址 |

明确填写模型支持的 `seconds` 和 `size`，便于预估价格与确认输出规格。`metadata` 及其他扩展字段会保留，但不作为本站视频时长或尺寸的计费来源；不要用 `metadata.seconds`、`duration`、`resolution` 或 `ratio` 替代原生字段。

旧 `video-v1` 契约已停止受理，提交 `contract_version` 会报错。原来的 `references`、`options` 和 `content[]` 角色数组不再由 new-api 转译；提供商支持某项高级能力，不代表本页的 OpenAI 视频入口已经开放该能力。

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

## 参考图片

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

本地图片通过 multipart 的 `input_reference` 文件字段上传。文件保持二进制形式，由网关处理对应的提供商协议；不要再使用旧 `image`、`first_frame`、`last_frame` 或 `audio` 字段的角色映射。

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

不要手动设置 multipart 的 `Content-Type`，客户端会生成与请求体一致的 boundary。参考图格式、尺寸与文件大小仍须满足所选模型和本站上传限制。

## 查询与下载

```bash
curl '{{BASE_URL}}/v1/videos/{id}' \
  -H 'Authorization: Bearer sk-xxxxxxxx'
```

状态包括 `queued`、`in_progress`、`completed` 和 `failed`。失败原因见 `error`；任务与费用也可在 [使用日志](/usage-logs) 中查看。

完成后通过本站内容接口下载，支持 `HEAD` 和单段 `Range`。返回的 `result.url` 也是本站授权内容链接；授权链接本身具有访问权限，请妥善保管。提供商原始下载地址不会直接暴露给调用方。

```bash
curl '{{BASE_URL}}/v1/videos/{id}/content' \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -o video.mp4
```

任务只能由所属用户读取；更换 API Key 不会转移任务归属。已持久化的历史任务仍沿用原来保存的网关任务和密钥查询、下载。

## 回调通知

需要通知时在请求顶层填写 `"callback_url": "https://example.com/video-callback"`。这是本站扩展，由 new-api 保存和发送，不会转发给模型供应商。

通知正文使用本站任务号和 `event_id`，请求头 `X-MAI-Event-ID` 携带同一个事件 ID；接收方按该 ID 去重并返回 2xx。回调失败会重试，仍可随时使用查询接口获取状态。

## 计费与任务职责

视频可按次、秒数、分辨率或实际 Token 用量定价，具体以 [模型广场](/pricing) 中账户可见的配置为准。创建时预扣，完成后按保存的价格快照和可信实际用量调整；缺少实际用量时保留已有预扣估算。失败任务按既有规则退款，重复结算或退款不会重复调整余额。

new-api 负责用户认证、任务归属、价格与余额、令牌额度、任务记录和退款。RS Gateway 负责供应商选择、协议转换、上游任务持久化和内容读取。供应商采购倍率与账户售价倍率分别计算。

创建前就被拒绝时，保留错误响应和响应头 `X-Oneapi-Request-Id`，按 [错误处理](/docs/errors) 排查。客户端断开不代表上游任务未创建，先查任务和日志，再决定是否重新提交。
