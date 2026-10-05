图片接口兼容 OpenAI Images API，用于文生图和图片编辑。请求发往 `{{BASE_URL}}/v1/images/generations` 和 `{{BASE_URL}}/v1/images/edits`，使用「生图」分组的令牌。

## 准备工作

- 在 [API 密钥](/keys) 页面创建一个令牌，分组选择「生图」。该分组支持 gpt、gemini、grok 的生图模型，倍率为 1。
- 在 [模型广场](/pricing) 筛选「生图」分组，找到要用的模型名和单价。下文示例中的 `<生图模型名>` 需要替换成真实模型名。
- 图片按张计费，请求里的 `n` 越大，扣费越多。

## 文生图

`POST /v1/images/generations`，请求体为 JSON。

<!-- code-group -->

```bash
curl "{{BASE_URL}}/v1/images/generations" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "<生图模型名>",
    "prompt": "一只在窗台上晒太阳的橘猫，水彩风格",
    "n": 1,
    "size": "1024x1024"
  }'
```

```powershell
$body = @{
  model  = "<生图模型名>"
  prompt = "一只在窗台上晒太阳的橘猫，水彩风格"
  n      = 1
  size   = "1024x1024"
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "{{BASE_URL}}/v1/images/generations" `
  -Headers @{ Authorization = "Bearer sk-xxxxxxxx" } `
  -ContentType "application/json; charset=utf-8" -Body $body
```

```python
from openai import OpenAI

client = OpenAI(base_url="{{BASE_URL}}/v1", api_key="sk-xxxxxxxx")
result = client.images.generate(
    model="<生图模型名>",
    prompt="一只在窗台上晒太阳的橘猫，水彩风格",
    n=1,
    size="1024x1024",
)
print(result.data[0].url or result.data[0].b64_json[:80])
```

```javascript
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: '{{BASE_URL}}/v1', apiKey: 'sk-xxxxxxxx' })
const result = await client.images.generate({
  model: '<生图模型名>',
  prompt: '一只在窗台上晒太阳的橘猫，水彩风格',
  n: 1,
  size: '1024x1024',
})
console.log(result.data[0].url ?? result.data[0].b64_json?.slice(0, 80))
```

<!-- /code-group -->

### 常用参数

| 参数                                                | 说明                                         |
| --------------------------------------------------- | -------------------------------------------- |
| `model`                                             | 必填，生图模型名                             |
| `prompt`                                            | 必填，图片描述                               |
| `n`                                                 | 生成张数，不填或填 0 时按 1 张处理，按张计费 |
| `size`                                              | 尺寸，例如 `1024x1024`，可选值取决于模型     |
| `quality`                                           | 画质，可选值取决于模型                       |
| `response_format`                                   | `url` 或 `b64_json`                          |
| `background`、`output_format`、`output_compression` | 背景、输出格式、压缩率，仅部分模型支持       |

模型不支持的参数可能被上游忽略或报错，以模型广场的「支持的参数」为准。

## 图片编辑

`POST /v1/images/edits`，用 `multipart/form-data` 上传原图，可选上传 `mask` 指定要修改的区域。

<!-- code-group -->

```bash
curl "{{BASE_URL}}/v1/images/edits" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -F model="<生图模型名>" \
  -F prompt="把背景换成海边日落" \
  -F image=@input.png \
  -F n=1
```

```powershell
curl.exe "{{BASE_URL}}/v1/images/edits" `
  -H "Authorization: Bearer sk-xxxxxxxx" `
  -F model="<生图模型名>" `
  -F prompt="把背景换成海边日落" `
  -F image=@input.png `
  -F n=1
```

```python
from openai import OpenAI

client = OpenAI(base_url="{{BASE_URL}}/v1", api_key="sk-xxxxxxxx")
with open("input.png", "rb") as f:
    result = client.images.edit(
        model="<生图模型名>",
        prompt="把背景换成海边日落",
        image=f,
        n=1,
    )
print(result.data[0].url or result.data[0].b64_json[:80])
```

<!-- /code-group -->

PowerShell 里要写 `curl.exe`，否则会调用 `Invoke-WebRequest` 别名，参数格式不同。

## 响应格式

成功时返回 `created` 时间戳和 `data` 数组，每张图片一项：

```json
{
  "created": 1767225600,
  "data": [{ "url": "https://..." }]
}
```

`response_format` 为 `b64_json` 时，图片以 `b64_json` 字段返回 Base64 内容，需要自行解码保存。返回的 `url` 可能有有效期，请及时下载。部分模型会额外返回 `revised_prompt` 或 `usage`。

## 计费与排错

- 费用按模型单价乘以张数计算，在 [使用日志](/usage-logs) 中可以查看每次请求的实际扣费。
- 生成较慢的模型可能需要几十秒才返回，客户端超时建议设为 120 秒以上。
- 报错时按 [错误处理](/docs/errors) 排查。常见原因是令牌分组不是「生图」，或模型名写错。
- `/v1/images/variations` 当前未实现，调用会返回错误。

想生成视频请看 [视频接口](/docs/videos)。
