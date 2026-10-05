本站开放的视频模型统一使用 video-v1 接口：创建、查询、下载使用同一组地址和同一种 JSON 结构，模型只决定能力与价格。调用时只需要本站 API Key，令牌需绑定 `满血视频` 或 `特价视频` 分组，两者的区别见 [分组说明](/docs/groups)。

> [!NOTE]
> 能力核对日期：2026-10-04。文中所有 `example.com` 地址和 Base64 占位文本都要换成真实素材，不能直接用于生成。

## 接口一览

| 操作     | 方法与地址                                |
| -------- | ----------------------------------------- |
| 创建任务 | `POST {{BASE_URL}}/v1/videos`             |
| 查询任务 | `GET {{BASE_URL}}/v1/videos/{id}`         |
| 下载成片 | `GET {{BASE_URL}}/v1/videos/{id}/content` |

- 所有请求都携带 `Authorization: Bearer sk-xxxxxxxx`。
- 创建请求支持 `application/json` 和 `multipart/form-data`。
- 创建成功后保存返回的任务 `id` 并轮询；不要因为客户端超时就重新提交，重新提交会再次付费。
- 已创建任务的进度与结果可在控制台 [使用日志](/usage-logs) 的「任务日志」中查看；创建前就被拒绝的请求，请保留错误响应和请求 ID（响应头 `X-Oneapi-Request-Id`）用于排查。

> [!IMPORTANT]
> 计费：满血 Seedance 和 MiniMax 保留按输出秒数定价，特价模型可按次计费。统一接口不改变账户现有价格，也不会把 Seedance 切换为 Token 定价。实际计费方式、模型和分辨率价格以 [模型广场](/pricing) 中账户可见的价格为准。

## 请求格式

```json
{
  "contract_version": "video-v1",
  "model": "seedance-2.0",
  "prompt": "一只红色纸船在湖面缓缓漂动",
  "duration": 5,
  "resolution": "720p",
  "ratio": "16:9",
  "references": []
}
```

- `contract_version` 填 `video-v1`；`model` 和非空的 `prompt` 必填。
- 明确填写该模型支持的 `duration`（秒）、`resolution` 和 `ratio`，不要依赖默认值。
- `references` 是参考素材数组，可省略；`options` 用于模型高级选项和回调地址 `callback_url`，也可省略。
- 使用下文能力表中的公开模型名，并确认账户已开通相应模型。模型名区分拼写，例如满血 `MiniMax-H3` 与特价 `[c]MiniMaxH3` 不是同一模型。
- 不要给 Seedance 模型名追加 `-720p` 等分辨率后缀，分辨率单独放在 `resolution`。

## 参考素材

`references` 的每一项结构为 `{ "type": "image|video|audio", "role": "...", "source": "..." }`，角色和素材类型必须匹配：

| 能力     | `references` 项                                                           | 说明                                                               |
| -------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| 参考图   | `{ "type": "image", "role": "reference_image", "source": "https://..." }` | 可传一张或多张，数量受模型上限约束                                 |
| 首帧     | `{ "type": "image", "role": "first_frame", "source": "https://..." }`     | 可单独使用；支持该能力的模型最多一张                               |
| 尾帧     | `{ "type": "image", "role": "last_frame", "source": "https://..." }`      | 可单独使用；支持该能力的模型最多一张                               |
| 首尾帧   | 同时传 `first_frame` 和 `last_frame`                                      | 不能混入 `reference_image`、`reference_video` 或 `reference_audio` |
| 参考视频 | `{ "type": "video", "role": "reference_video", "source": "https://..." }` | 仅支持该能力的模型接受；视频不能使用 Data URL                      |
| 参考音频 | `{ "type": "audio", "role": "reference_audio", "source": "https://..." }` | 仅支持该能力的模型接受                                             |

素材来源规则：

- 图片的公网 HTTPS 地址适用于所有当前模型，链接在任务处理期间必须一直可访问。
- 图片和音频的 Data URL 只有部分模型支持。Data URL 必须包含 MIME 类型、`;base64,` 和完整的 Base64 数据，例如 `data:audio/mpeg;base64,...`，不接受单独的 Base64 字符串。
- 满血 `MiniMax-H3` 的所有素材必须使用公网 HTTPS。
- 特价 Seedance 的音频必须使用 `data:audio/...;base64,...` 或 multipart 上传，不能填 HTTPS 音频地址。
- 视频参考一律使用公网 HTTPS。当前公开接口不接受 `asset://` 引用。
- 模型不支持的组合会直接报错；素材预处理失败时，任务也可能在创建后变为 `failed`。

### 人脸与人物参考

- 满血 `seedance-2.0`、`seedance-2.5` 和 `MiniMax-H3` 支持人脸参考：直接传入人物图片即可，使用普通 `reference_image`，或在关键帧模式中使用 `first_frame` / `last_frame`，不需要额外的人脸开关。支持人物参考不代表承诺身份完全一致，仍须遵守内容审核规则。
- 特价 `[c]seedance-2.0`、`[c]seedance-2.5` 不支持人脸参考，需要人脸素材时请改用满血模型。
- 特价 MiniMax 和 Grok 当前未声明人脸支持，不保证身份保持。

### 参考图示例

普通参考图请求如下。满血 Seedance 采用相同的 `references` 写法；Grok 只能保留一张图片。

```json
{
  "contract_version": "video-v1",
  "model": "[c]seedance-2.0",
  "prompt": "让参考图片中的纸船在水面缓缓漂动",
  "duration": 5,
  "resolution": "720p",
  "ratio": "16:9",
  "references": [
    {
      "type": "image",
      "role": "reference_image",
      "source": "https://example.com/boat.png"
    }
  ]
}
```

## 当前模型与能力

| 公开模型名               | 分组     | 参考图     | 首尾帧  | 参考视频   | 参考音频   | 人脸参考       | 特别限制                                                 |
| ------------------------ | -------- | ---------- | ------- | ---------- | ---------- | -------------- | -------------------------------------------------------- |
| `seedance-2.0`           | 满血视频 | 最多 9 张  | 各 1 张 | 最多 3 段  | 最多 3 段  | 支持           | 素材总数最多 15；时长 4～15 秒                           |
| `seedance-2.5`           | 满血视频 | 最多 30 张 | 各 1 张 | 最多 10 段 | 最多 10 段 | 支持           | 素材总数最多 50；时长 4～30 秒                           |
| `MiniMax-H3`             | 满血视频 | 最多 9 张  | 各 1 张 | 最多 3 段  | 最多 3 段  | 支持           | 素材总数最多 12；固定 768P，整数 1～15 秒；仅 HTTPS 素材 |
| `grok-imagine-video-1.5` | 满血视频 | 最多 1 张  | 不支持  | 不支持     | 不支持     | 未声明，不保证 | 不支持模型高级选项                                       |
| `[c]seedance-2.0`        | 特价视频 | 最多 9 张  | 不支持  | 不支持     | 最多 3 段  | 不支持         | 音频仅 Data URL / multipart；时长按账户档位              |
| `[c]seedance-2.5`        | 特价视频 | 最多 30 张 | 不支持  | 不支持     | 最多 3 段  | 不支持         | 固定 30 秒；音频仅 Data URL / multipart                  |
| `[c]MiniMaxH3`           | 特价视频 | 最多 9 张  | 不支持  | 不支持     | 最多 3 段  | 未声明，不保证 | 素材总数最多 12；分辨率和时长以账户档位为准              |

- 上表是本站当前开放的受理上限，实际可用能力以账户权限和素材校验结果为准；并非所有媒体格式、尺寸和时长都能通过审核。
- 满血 Seedance、MiniMax 的首尾帧模式不能与其他 `reference_*` 素材混用。
- 支持参考音频不等于支持纯音频请求：Seedance 2.0 建议同时提供视觉参考；满血 MiniMax 明确支持纯音频参考，但此时须使用固定比例。
- 特价 Seedance 不支持通过 `resolution`、`ratio` 自定义成片规格，填写更高分辨率或不同画幅也不能强制改变输出。
- 只使用账户已开通且已定价的模型和档位；素材编码、尺寸、时长及组合仍须符合模型限制。

### Seedance 参数

- 满血版的首尾帧必须通过 `first_frame` / `last_frame` 显式指定，不能当作普通参考图代发。
- `seedance-2.0` 支持 4～15 秒和 `480p`、`720p`、`1080p`、`4k`；`seedance-2.5` 支持 4～30 秒和 `480p`、`720p`、`1080p`。两者都支持 `duration: -1` 自动时长；特价 `[c]seedance-2.5` 固定 30 秒。
- 满血 Seedance 可在 `options` 中传 `generate_audio`、`watermark`、`return_last_frame`、`priority`、`execution_expires_after`、`safety_identifier` 和 `tools`。
- `seedance-2.5` 还支持 `video_format`（`mp4` / `mov`）和 `omni_reference_task_type`（`auto`、`reference`、`edit`、`extend`）。`edit` 要求参考视频、`ratio: adaptive` 和 `duration: -1`；`extend` 要求参考视频和 `ratio: adaptive`。
- 其他模型只传该模型已声明的选项，不要把某个模型的高级参数复制给所有模型。

### MiniMax 参数

- `MiniMax-H3` 使用 `resolution: "768P"`；画幅可选 `21:9`、`16:9`、`4:3`、`1:1`、`3:4`、`9:16`，有图片或视频素材时也可用 `adaptive`。
- 除回调地址 `callback_url` 外，不要给 `MiniMax-H3` 传 Seedance 的音频开关、水印、返回尾帧等高级字段；特价 MiniMax 的多分辨率也不能套用到满血版。
- 特价 `[c]MiniMaxH3` 的分辨率档位为 `480p`、`720p`、`2k`、`2k-pro`，同样通过 `resolution` 选择，不修改公开模型名。`2k-pro` 是质量档位名称，不代表支持人脸；可用价格和时长以账户档位为准。
- MiniMax 按每种媒体在 `references` 中的出现顺序分别编号，提示词中用 `<Picture 1>`、`<Video 1>`、`<Audio 1>` 引用。
- 视频和音频每段至少 2 秒；超过 15 秒只取片头 15 秒，裁切后视频累计、独立音频累计各不得超过 15 秒。
- 素材格式：图片为静态 JPG/PNG/WebP/HEIC/HEIF（最多 30,000,000 字节），视频为 MP4/MOV（H.264/H.265，最多 50,000,000 字节），音频为 WAV/MP3/M4A/AAC（最多 15,000,000 字节）。
- 满血 MiniMax 首尾帧的宽高比差异不得超过 2%。

## 请求示例

### 首尾帧

满血 Seedance 2.0 / 2.5 的写法如下，只需要首帧或尾帧时删除另一项即可。满血 MiniMax 使用相同的角色，但要改为 `"model": "MiniMax-H3"`、`"resolution": "768P"`、1～15 秒的 `duration`，并删去 Seedance 专用的 `options`。

```json
{
  "contract_version": "video-v1",
  "model": "seedance-2.0",
  "prompt": "从第一帧平滑过渡到最后一帧",
  "duration": 4,
  "resolution": "480p",
  "ratio": "16:9",
  "references": [
    {
      "type": "image",
      "role": "first_frame",
      "source": "https://example.com/start.png"
    },
    {
      "type": "image",
      "role": "last_frame",
      "source": "https://example.com/end.png"
    }
  ],
  "options": { "generate_audio": false, "return_last_frame": true }
}
```

### 参考视频与参考音频

下面的完整请求只适用于满血 Seedance，包含一张参考图、一段参考视频和一段参考音频。不需要的素材可以删除，但不能再加入首尾帧角色。`generate_audio` 控制成片是否生成音频，不是上传参考音频的开关。

```json
{
  "contract_version": "video-v1",
  "model": "seedance-2.0",
  "prompt": "保持参考图片的人物外观，参考视频中的动作和参考音频的节奏，生成人物舞蹈视频",
  "duration": 5,
  "resolution": "720p",
  "ratio": "16:9",
  "references": [
    {
      "type": "image",
      "role": "reference_image",
      "source": "https://example.com/person.png"
    },
    {
      "type": "video",
      "role": "reference_video",
      "source": "https://example.com/motion.mp4"
    },
    {
      "type": "audio",
      "role": "reference_audio",
      "source": "https://example.com/music.mp3"
    }
  ],
  "options": { "generate_audio": true }
}
```

满血 MiniMax 的混合参考请求使用同样的 `references`（素材须为 HTTPS），把模型、分辨率和提示词换成：

```json
{
  "model": "MiniMax-H3",
  "prompt": "让 <Picture 1> 的人物模仿 <Video 1> 的动作，并参考 <Audio 1> 的声音",
  "duration": 5,
  "resolution": "768P",
  "ratio": "16:9"
}
```

### 特价 Seedance 音频

特价 Seedance 不接受参考视频，但可以把参考图和参考音频组合使用。音频 `source` 必须是 Data URL，不能填写 HTTPS 地址；使用 `[c]seedance-2.5` 时要把 `duration` 改为 `30`。示例省略了实际的 Base64 数据：

```json
{
  "contract_version": "video-v1",
  "model": "[c]seedance-2.0",
  "prompt": "参考图片中的花朵跟随音频节奏轻轻摇摆",
  "duration": 5,
  "references": [
    {
      "type": "image",
      "role": "reference_image",
      "source": "https://example.com/flowers.png"
    },
    {
      "type": "audio",
      "role": "reference_audio",
      "source": "data:audio/mpeg;base64,REPLACE_WITH_COMPLETE_BASE64"
    }
  ]
}
```

## 文件上传（multipart）

创建接口也接受 `multipart/form-data`，可以直接上传本地文件。满血 `MiniMax-H3` 不接受文件上传，需要先把素材放到自己的公网 HTTPS 存储，再通过 `references` 传地址。

| 文件字段      | 对应角色          | 当前限制                                                     |
| ------------- | ----------------- | ------------------------------------------------------------ |
| `image`       | `reference_image` | 满血 Seedance、特价 Seedance、特价 MiniMax；数量仍按模型限制 |
| `first_frame` | `first_frame`     | 满血 Seedance，最多一张                                      |
| `last_frame`  | `last_frame`      | 满血 Seedance，最多一张                                      |
| `audio`       | `reference_audio` | 满血 Seedance、特价 Seedance、特价 MiniMax                   |
| `video`       | 不接受上传        | 在 `references` 文本字段中传公网 HTTPS 视频地址              |

- 多个同类文件重复使用同一个字段名，例如两个 `image` 字段，不要写成 `image[]`。
- `references` 和 `options` 作为文本字段传入时必须是 JSON 字符串。
- 不要手动设置 multipart 的 `Content-Type`，由客户端自动生成 boundary。
- 单个文件内联上限 32 MiB，同时受模型更小的文件限制和平台总上传限制约束。

特价 Seedance 上传图片和音频的示例：

<!-- code-group -->

```bash
curl '{{BASE_URL}}/v1/videos' \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -F 'contract_version=video-v1' \
  -F 'model=[c]seedance-2.0' \
  -F 'prompt=让参考图片中的花朵跟随参考音频节奏轻轻摇摆' \
  -F 'duration=5' \
  -F 'image=@flowers.png;type=image/png' \
  -F 'audio=@music.mp3;type=audio/mpeg'
```

```powershell
[IO.File]::WriteAllText("$PWD\prompt.txt", "让参考图片中的花朵跟随参考音频节奏轻轻摇摆")
curl.exe "{{BASE_URL}}/v1/videos" `
  -H "Authorization: Bearer sk-xxxxxxxx" `
  -F "contract_version=video-v1" `
  -F "model=[c]seedance-2.0" `
  -F "prompt=<prompt.txt" `
  -F "duration=5" `
  -F "image=@flowers.png;type=image/png" `
  -F "audio=@music.mp3;type=audio/mpeg"
```

<!-- /code-group -->

PowerShell 示例先把中文提示词写入 UTF-8 文件，再用 `prompt=<prompt.txt` 读取，避免命令行参数中的中文乱码。

## 查询与下载

创建响应中的 `id` 是本站任务号，用它轮询任务状态：

```bash
curl '{{BASE_URL}}/v1/videos/{id}' \
  -H 'Authorization: Bearer sk-xxxxxxxx'
```

- 状态取值包括 `queued`、`in_progress`、`completed`、`failed` 等；`failed` 时查看 `error` 字段。
- 完成后 `result.url` 统一指向本站相对地址 `/v1/videos/{id}/content`，下载时同样携带本站 API Key。
- `result.last_frame_url` 只在模型返回尾帧时出现。
- 任务完成前不要重新提交同一请求。

下载成片：

```bash
curl -L '{{BASE_URL}}/v1/videos/{id}/content' \
  -H 'Authorization: Bearer sk-xxxxxxxx' \
  -o video.mp4
```

下载接口可能返回重定向或流式内容，客户端需要支持重定向。重定向到其他域名时，不要向目标域名发送本站 API Key；curl 的 `-L` 默认不会把 `Authorization` 头带到其他域名，自己编写下载逻辑时请注意这一点。

## 回调通知

在 `options.callback_url` 填写公网 HTTPS 地址，任务状态变化时会收到通知：

```json
{
  "contract_version": "video-v1",
  "model": "seedance-2.0",
  "prompt": "一只红色纸船在湖面缓缓漂动",
  "duration": 5,
  "resolution": "720p",
  "ratio": "16:9",
  "options": { "callback_url": "https://example.com/video-callback" }
}
```

- 服务端向该地址 `POST` JSON，内容使用本站任务号和确定性的 `event_id`，`status` 取值为 `queued`、`in_progress`、`completed`、`failed`。
- 请求头 `X-MAI-Event-ID` 重复携带 `event_id`，接收方应按 `event_id` 去重。
- 接收方返回 2xx 表示接收成功；失败的回调会在后续轮询中重试，连续失败最多重试 8 次。
- 回调不可用时仍可通过查询接口获取结果，不要重新创建任务。

## 目前不支持

当前统一契约支持 JSON 或 multipart 创建、任务查询、成片下载和 HTTPS 回调，以下用法暂不支持：

- `asset://` 素材引用。参考视频请放在自己的存储中，提供任务期间可访问的 HTTPS 链接。
- `content[]` 格式。新接入统一使用本页的 `references` 格式。

调用出错时可参考 [错误处理](/docs/errors)，并在反馈时附上响应头 `X-Oneapi-Request-Id`。
