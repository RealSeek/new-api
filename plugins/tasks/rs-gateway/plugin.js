export const meta = {
  apiVersion: 1,
  key: "rs-gateway",
  name: "RS Gateway",
  version: "1.8.0",
  description: { en: "Video tasks managed by RS Gateway", zh: "由 RS Gateway 管理的视频任务" },
  author: { name: "RealSeek" },
  channelTypes: [61],
  models: [
    "seedance-2.0", "seedance-2.5", "doubao-seedance-2-0-260128", "doubao-seedance-2-5-260628",
    "[c]seedance-2.0", "[c]seedance-2.5",
    "[NSFW]seedance-2.0", "[NSFW]seedance-2.5",
    "MiniMax-H3", "[c]MiniMaxH3", "grok-imagine-video-1.5",
  ],
  fetchMode: "per_task",
  protocols: ["openai_video"],
  usageSchema: {
    resolution: { enum: ["480p", "720p", "1080p", "4k"], allowCustomValues: true, description: { en: "Output video resolution", zh: "输出视频分辨率" } },
    requests: { type: "number", unit: "count", unitLabel: { en: "video", zh: "次" }, description: { en: "Video generation unit price", zh: "视频生成单价" } },
    seconds: { type: "number", unit: "second", description: { en: "Video generation unit price", zh: "视频生成单价" } },
    tokens: { type: "number", unit: "token", description: { en: "Video generation token unit price", zh: "视频生成 Token 单价" } },
    video_input: { type: "boolean", description: { en: "Reference video present", zh: "存在参考视频" } },
  },
  usageExamples: [{ label: "720p · 5s", facts: { resolution: "720p", requests: 1, seconds: 5, tokens: 108000, video_input: false } }],
};

// Ark 计费 token 估算使用的像素表（tokens ≈ 时长 × 像素 × 24FPS / 1024）。
const SEEDANCE_RESOLUTION_PIXELS = { "480p": 854 * 480, "720p": 1280 * 720, "1080p": 1920 * 1080, "4k": 3840 * 2160 };

function seedanceModel(ctx) {
  const match = /^(?:\[c\]|\[nsfw\])?(?:doubao-)?seedance-2[._-]([05])(?:-?\d{6})?(?:-(?:fast|mini))?(?:-(480p|720p|1080p|4k))?$/i.exec(ctx.upstreamModel || ctx.model || "");
  return match ? [match[0], "seedance-2." + match[1], match[2]] : null;
}

function upstreamMediaURL(ctx, data) {
  if (!ctx.preserveUpstreamMediaURL) return "";
  const candidates = [(data.metadata || {}).content_url, (data.result || {}).url, (data.metadata || {}).url, data.url];
  return candidates.find(value => typeof value === "string" &&
    /^https?:\/\/[^/?#\s]+(?:[/?#]|$)/i.test(value)) || "";
}

function validateCallbackURL(value) {
  if (typeof value !== "string") throw new Error("callback_url must be a valid HTTPS URL");
  const url = value.trim();
  if (!/^https:\/\/[^\s/?#]+(?:[/?][^\s#]*)?$/i.test(url) || url.includes("@") || url.includes("#")) {
    throw new Error("callback_url must be a valid HTTPS URL");
  }
  return url;
}

function validateVideoRequest(request) {
  if (!request || typeof request !== "object" || Array.isArray(request)) throw new Error("video request must be a JSON object");
  if (request.contract_version !== undefined) throw new Error("video-v1 is no longer supported; use the native OpenAI video request");
  if (typeof request.prompt !== "string" || !request.prompt.trim()) throw new Error("prompt is required");
  if (request.callback_url !== undefined) validateCallbackURL(request.callback_url);
  return request;
}

function multipartRequest(ctx) {
  const fields = ctx.body && ctx.body.fields;
  if (!fields || typeof fields !== "object" || Array.isArray(fields)) throw new Error("multipart fields are required");
  const body = {};
  for (const [name, values] of Object.entries(fields)) {
    if (!Array.isArray(values) || values.length === 0) throw new Error("invalid multipart field: " + name);
    if (["model", "prompt", "seconds", "callback_url"].includes(name) && values.length !== 1) throw new Error(name + " must be provided once");
    body[name] = values.length === 1 ? values[0] : values;
  }
  if (body.model === undefined) body.model = ctx.model;
  return validateVideoRequest(body);
}

function automaticDurationLimit(ctx) {
  const modelName = ctx.upstreamModel || ctx.model || "";
  const seedance = seedanceModel(ctx);
  if (seedance && !/^\[c\]/i.test(modelName)) return seedance[1] === "seedance-2.5" ? 30 : 15;
  // HHCoding's quality aliases use a different contract from native MiniMax-H3.
  if (/^(?:\[c\])?MiniMaxH3(?:-(?:480p|720p|2k(?:-pro)?))?(?:-sec)?$/i.test(modelName)) return 15;
  return 0;
}

function videoDurationSeconds(ctx) {
  const seconds = (ctx.requestBody || {}).seconds;
  if (seconds !== undefined &&
      (!(typeof seconds === "number" || (typeof seconds === "string" && /^(?:-1|\d+)$/.test(seconds.trim()))) ||
       !Number.isInteger(Number(seconds)) || (Number(seconds) !== -1 && (Number(seconds) <= 0 || Number(seconds) > 3600)))) {
    throw new Error("duration must be between 1 and 3600 seconds, or -1 for supported automatic duration");
  }
  if (Number(seconds) === -1 && !automaticDurationLimit(ctx)) throw new Error("automatic duration is not declared for this model");
  return seconds === undefined ? undefined : Number(seconds);
}

export function buildSubmitRequest(ctx) {
  const body = Object.assign({}, validateVideoRequest(ctx.requestBody), { model: ctx.upstreamModel || ctx.model });
  delete body.callback_url;
  videoDurationSeconds(ctx);
  const base = ctx.baseUrl.replace(/\/$/, "");
  const path = ctx.action === "remix" ? "/v1/videos/" + encodeURIComponent(ctx.originTaskId) + "/remix" : "/v1/videos";
  const headers = { Authorization: "Bearer " + ctx.apiKey };
  if (ctx.preserveUpstreamMediaURL) headers["X-Media-Result-Url"] = "upstream";
  for (const name of Object.keys(ctx.requestHeaders || {})) {
    if (name.toLowerCase() === "idempotency-key") headers["Idempotency-Key"] = ctx.requestHeaders[name];
  }
  if ((ctx.files || []).length || Object.entries(ctx.requestHeaders || {}).some(([name, value]) => name.toLowerCase() === "content-type" && String(value).toLowerCase().startsWith("multipart/form-data"))) {
    const parts = [];
    for (const [name, value] of Object.entries(body)) {
      for (const item of Array.isArray(value) ? value : [value]) {
        parts.push({ name, value: item !== null && typeof item === "object" ? JSON.stringify(item) : item });
      }
    }
    for (const file of ctx.files || []) parts.push({ name: file.field, fileRef: file.ref, filename: file.filename });
    return { url: base + path, method: "POST", headers, bodyType: "multipart", parts };
  }
  headers["Content-Type"] = "application/json";
  return { url: base + path, method: "POST", headers, body };
}

export function extractUsage(ctx) {
  // Legacy prices already apply their own seconds multiplier.
  if (ctx.usagePurpose === "billing_ratios") return null;
  const model = seedanceModel(ctx);
  const body = ctx.requestBody || {};
  const requestedSeconds = videoDurationSeconds(ctx) ?? (model ? (model[1] === "seedance-2.5" ? 30 : 15) : 4);
  const seconds = requestedSeconds === -1 ? automaticDurationLimit(ctx) : requestedSeconds;
  const dimensions = /^(\d+)x(\d+)$/i.exec(String(body.size || ""));
  const edge = dimensions ? Math.min(Number(dimensions[1]), Number(dimensions[2])) : 0;
  const resolution = typeof body.resolution === "string" ? body.resolution.toLowerCase() : edge > 0 ? (edge <= 480 ? "480p" : edge <= 720 ? "720p" : edge <= 1080 ? "1080p" : "4k") : "720p";
  const facts = { resolution, requests: 1, seconds, video_input: ctx.action === "remix" || Array.isArray(body.videos) && body.videos.length > 0, web_search_calls: Array.isArray(body.tools) && body.tools.some(tool => tool && tool.type === "web_search") ? 1 : 0 };
  if (!model) return facts;
  // Retain the existing Seedance pricing reservation, not its provider wire
  // conversion. Completion replaces estimated tokens with upstream usage.
  const pixels = dimensions ? Number(dimensions[1]) * Number(dimensions[2]) : SEEDANCE_RESOLUTION_PIXELS[resolution] || SEEDANCE_RESOLUTION_PIXELS["4k"];
  return { ...facts, tokens: Math.ceil(seconds * pixels * 24 / 1024) };
}

export function extractUsageOnComplete(_ctx, result, body) {
  if (result.status !== "SUCCESS") return null;
  const usage = body.usage || {};
  const facts = {};
  // Missing usage keeps the reservation; explicit zero remains a real zero.
  const actualSeconds = usage.seconds ?? usage.output_seconds ?? body.seconds;
  if (actualSeconds !== undefined && Number.isFinite(Number(actualSeconds)) && Number(actualSeconds) >= 0) facts.seconds = Number(actualSeconds);
  if (usage.completion_tokens !== undefined) facts.tokens = usage.completion_tokens;
  else if (usage.output_tokens !== undefined) facts.tokens = usage.output_tokens;
  const dimensions = /^(\d+)x(\d+)$/i.exec(String(usage.size || body.size || ""));
  if (dimensions) {
    const edge = Math.min(Number(dimensions[1]), Number(dimensions[2]));
    if (edge > 0) facts.resolution = edge <= 480 ? "480p" : edge <= 720 ? "720p" : edge <= 1080 ? "1080p" : "4k";
  }
  return facts;
}

export function parseSubmitResponse(_ctx, resp) {
  const body = resp.body || {};
  const taskId = body.id || body.task_id;
  if (!taskId) throw new Error("gateway task id is missing");
  const callbackUrl = _ctx && _ctx.requestBody && _ctx.requestBody.callback_url;
  return {
    taskId,
    taskData: body,
    ...(callbackUrl ? { state: { callback: { url: callbackUrl, notified_status: "", attempts: 0, exhausted_status: "" } } } : {}),
  };
}

export function buildQueryRequest(ctx) {
  return {
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos/" + encodeURIComponent(ctx.taskId),
    method: "GET",
    headers: { Authorization: "Bearer " + ctx.apiKey, ...(ctx.preserveUpstreamMediaURL ? { "X-Media-Result-Url": "upstream" } : {}) },
  };
}

export function parseTaskResult(_ctx, body) {
  const statuses = {
    queued: "QUEUED",
    pending: "QUEUED",
    processing: "IN_PROGRESS",
    running: "IN_PROGRESS",
    in_progress: "IN_PROGRESS",
    completed: "SUCCESS",
    succeeded: "SUCCESS",
    failed: "FAILURE",
    cancelled: "FAILURE",
  };
  const status = statuses[body.status] || "UNKNOWN";
  const result = { status };
  if (status === "UNKNOWN") result.reason = "unrecognized gateway task status: " + String(body.status || "");
  if (status === "FAILURE") result.reason = (body.error || {}).message || body.fail_reason || "视频生成失败，请稍后重试";
  if (Number(body.progress) >= 0 && Number(body.progress) <= 100) result.progress = Number(body.progress) + "%";
  const prior = _ctx && _ctx.state && typeof _ctx.state === "object" && !Array.isArray(_ctx.state) ? _ctx.state : null;
  if (prior && prior.callback && typeof prior.callback === "object") {
    result.state = { ...prior, callback: { ...prior.callback, observed_status: status } };
  }
  return result;
}

export function listArtifacts(task) {
  return task.status === "SUCCESS" ? [{ key: "video", type: "video" }] : [];
}

export function buildContentRequest(ctx) {
  if (ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  const directURL = upstreamMediaURL(ctx, ctx.data || {});
  if (directURL) return { url: directURL, method: "GET", credentialless: true };
  return {
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos/" + encodeURIComponent(ctx.upstreamTaskId) + "/content",
    method: ctx.clientRequest.method,
    headers: { Authorization: "Bearer " + ctx.apiKey },
  };
}

// Preserve native media URLs only under host policy; otherwise use the host's
// ownership-checked artifact endpoint. Preview links need their own artifact.
function publicVideoSnapshot(value, contentURL, preserveUpstreamMediaURL) {
  if (Array.isArray(value)) return value.map(item => publicVideoSnapshot(item, contentURL, preserveUpstreamMediaURL));
  if (!value || typeof value !== "object") return value;
  const result = {};
  for (const [key, item] of Object.entries(value)) {
    if (["last_frame_url", "thumbnail_url", "preview_url"].includes(key)) continue;
    if (["url", "video_url", "download_url", "content_url"].includes(key) && typeof item === "string") {
      if (contentURL) result[key] = preserveUpstreamMediaURL ? item : contentURL;
    } else result[key] = publicVideoSnapshot(item, contentURL, preserveUpstreamMediaURL);
  }
  return result;
}

export const protocols = {
  openai_video: {
    decodeRequest(ctx) {
      if (ctx.body.kind === "multipart") return { kind: "submit", model: ctx.model, requestBody: multipartRequest(ctx) };
      if (ctx.body.kind !== "json") throw new Error("JSON or multipart body required");
      return { kind: "submit", model: ctx.model, requestBody: validateVideoRequest(ctx.body.value) };
    },
    render(ctx, task) {
      const statuses = { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "in_progress", SUCCESS: "completed", FAILURE: "failed" };
      const properties = task.properties || {};
      const directURL = upstreamMediaURL(ctx, task.data || {});
      const contentURL = directURL || ((ctx.artifacts || {}).video || {}).url;
      const result = Object.assign(publicVideoSnapshot(task.data || {}, contentURL, Boolean(directURL)), {
        id: task.task_id,
        model: ctx.model || properties.origin_model_name || "",
        status: statuses[task.status] || "unknown",
      });
      delete result.task_id;
      if (task.status === "SUCCESS") {
        if (!contentURL) throw new Error("video content URL is unavailable");
        result.result = Object.assign({}, result.result || {}, { url: contentURL });
      }
      if (task.status === "FAILURE") result.error = { code: "video_generation_failed", message: task.fail_reason || "视频生成失败，请稍后重试" };
      return result;
    },
  },
};
