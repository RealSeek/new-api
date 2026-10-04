export const meta = {
  apiVersion: 1,
  key: "rs-gateway",
  name: "RS Gateway",
  version: "1.5.1",
  description: { en: "Video tasks managed by RS Gateway", zh: "由 RS Gateway 管理的视频任务" },
  author: { name: "RealSeek" },
  channelTypes: [61],
  models: [
    "seedance-2.0", "seedance-2.5", "doubao-seedance-2-0-260128", "doubao-seedance-2-5-260628",
    "[c]seedance-2.0", "[c]seedance-2.5",
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

function seedanceModel(ctx) {
  const match = /^(?:\[c\])?(?:doubao-)?seedance-2[._-]([05])(?:-?\d{6})?(?:-(?:fast|mini))?(?:-(480p|720p|1080p|4k))?$/i.exec(ctx.upstreamModel || ctx.model || "");
  return match ? [match[0], "seedance-2." + match[1], match[2]] : null;
}

function minimaxH3Model(ctx) {
  return /minimax[-_ ]?h3/i.test(ctx.upstreamModel || ctx.model || "");
}

const VIDEO_OPTIONS = new Set([
  "generate_audio", "watermark", "return_last_frame", "video_format",
  "omni_reference_task_type", "priority", "execution_expires_after",
  "safety_identifier", "tools", "face_required", "callback_url",
]);

function validateCallbackURL(value) {
  if (typeof value !== "string") throw new Error("callback_url must be a valid HTTPS URL");
  const url = value.trim();
  if (!/^https:\/\/[^\s/?#]+(?:[/?][^\s#]*)?$/i.test(url) || url.includes("@") || url.includes("#")) {
    throw new Error("callback_url must be a valid HTTPS URL");
  }
  return url;
}

function normalizeVideoRequest(request) {
  if (!request || typeof request !== "object" || Array.isArray(request)) throw new Error("video request must be a JSON object");
  const canonical = request.contract_version !== undefined || request.prompt !== undefined || request.references !== undefined || request.options !== undefined;
  if (!canonical) return request;
  // Protocol decoders persist this normalized internal shape before the host
  // calls buildSubmitRequest. Keep that second hook invocation idempotent.
  if (request.prompt !== undefined && request.content !== undefined && request.references === undefined && request.options === undefined && request.contract_version === undefined) return request;
  if (request.contract_version !== undefined && request.contract_version !== "video-v1") throw new Error("unsupported video contract_version");
  if (request.content !== undefined) throw new Error("content cannot be combined with prompt, references or options");
  if (typeof request.prompt !== "string" || !request.prompt.trim()) throw new Error("prompt is required");
  if (request.contract_version === "video-v1") {
    const fields = new Set(["contract_version", "model", "prompt", "duration", "resolution", "ratio", "references", "options"]);
    for (const key of Object.keys(request)) if (!fields.has(key)) throw new Error("unsupported video-v1 field: " + key);
  }
  const references = request.references === undefined ? [] : request.references;
  if (!Array.isArray(references) || references.length > 50) throw new Error("references must be an array of at most 50 items");
  const content = [{ type: "text", text: request.prompt }];
  for (const reference of references) {
    if (!reference || typeof reference !== "object" || Array.isArray(reference)) throw new Error("invalid video reference");
    const { type, role, source } = reference;
    const kind = { image: "image_url", video: "video_url", audio: "audio_url" }[type];
    const roles = { image: ["reference_image", "first_frame", "last_frame"], video: ["reference_video"], audio: ["reference_audio"] };
    if (!kind || !roles[type].includes(role)) throw new Error("reference type and role do not match");
    if (typeof source !== "string" || !(source.startsWith("https://") || (type !== "video" && source.startsWith("data:" + type + "/")))) {
      throw new Error("reference source must be a public HTTPS URL or supported image/audio Data URL");
    }
    content.push({ type: kind, role, [kind]: { url: source } });
  }
  const options = request.options === undefined ? {} : request.options;
  if (!options || typeof options !== "object" || Array.isArray(options)) throw new Error("options must be a JSON object");
  const body = Object.assign({}, request, { content });
  delete body.contract_version;
  delete body.prompt;
  delete body.references;
  delete body.options;
  for (let [key, value] of Object.entries(options)) {
    if (!VIDEO_OPTIONS.has(key)) throw new Error("unsupported video option: " + key);
    const target = key === "video_format" ? "output_format" : key;
    if (key === "callback_url") value = validateCallbackURL(value);
    if (body[target] !== undefined && body[target] !== value) throw new Error("conflicting video option: " + key);
    body[target] = value;
  }
  return body;
}

function multipartRequest(ctx) {
  const fields = ctx.body && ctx.body.fields;
  if (!fields || typeof fields !== "object" || Array.isArray(fields)) throw new Error("multipart fields are required");
  const allowedFields = new Set(["contract_version", "model", "prompt", "duration", "resolution", "ratio", "references", "options"]);
  for (const name of Object.keys(fields)) if (!allowedFields.has(name)) throw new Error("unsupported video multipart field: " + name);
  const value = (name) => {
    const values = fields[name];
    if (values === undefined) return undefined;
    if (!Array.isArray(values) || values.length !== 1) throw new Error(name + " must be provided once");
    return values[0];
  };
  const body = {};
  for (const name of ["contract_version", "model", "prompt", "resolution", "ratio"]) {
    const item = value(name);
    if (item !== undefined) body[name] = item;
  }
  const duration = value("duration");
  if (duration !== undefined) {
    const parsed = Number(duration);
    if (!Number.isInteger(parsed)) throw new Error("duration must be an integer");
    body.duration = parsed;
  }
  for (const name of ["references", "options"]) {
    const raw = value(name);
    if (raw === undefined) continue;
    let parsed;
    try { parsed = JSON.parse(raw); } catch { throw new Error(name + " must be valid JSON"); }
    body[name] = parsed;
  }
  if (typeof body.prompt !== "string" || !body.prompt.trim()) throw new Error("prompt is required");
  if (body.model === undefined) body.model = ctx.model;
  return body;
}

function appendMultipartReferences(body, files) {
  if (!Array.isArray(files) || files.length === 0) return body;
  const content = Array.isArray(body.content) ? body.content.slice() : [];
  const roles = { image: ["image", "reference_image"], first_frame: ["image", "first_frame"], last_frame: ["image", "last_frame"], audio: ["audio", "reference_audio"] };
  for (const file of files) {
    const mapping = roles[file.field];
    if (!mapping) {
      if (file.field === "video") throw new Error("multipart reference video must be a public HTTPS URL");
      throw new Error("unsupported video multipart field: " + String(file.field || ""));
    }
    const placeholder = { __fileRef: file.ref, encoding: "dataUrl", mimeType: file.mimeType, maxBytes: 32 * 1024 * 1024 };
    const urlKey = mapping[0] === "image" ? "image_url" : "audio_url";
    content.push({ type: urlKey, role: mapping[1], [urlKey]: { url: placeholder } });
  }
  return Object.assign({}, body, { content });
}

export function buildSubmitRequest(ctx) {
  let body = Object.assign({}, normalizeVideoRequest(ctx.requestBody), { model: ctx.upstreamModel });
  body = appendMultipartReferences(body, ctx.files);
  delete body.callback_url;
  if (/^\[c\]seedance-2[._-]5$/i.test(ctx.model || ctx.requestBody.model || "") && Number(body.duration ?? body.seconds) !== 30) {
    throw new Error("[c]seedance-2.5 requires duration 30 seconds");
  }
  const metadata = typeof body.metadata === "string" ? JSON.parse(body.metadata) : body.metadata || {};
  let duration;
  for (const [index, value] of [body.seconds, body.duration, body.durationSeconds, metadata.seconds, metadata.duration, metadata.durationSeconds].entries()) {
    const automatic = index === 1 && seedanceModel(ctx) && Number(value) === -1;
    if (value !== undefined && !automatic && (!Number.isInteger(Number(value)) || Number(value) <= 0 || Number(value) > 3600)) {
      throw new Error("duration must be between 1 and 3600 seconds");
    }
    if (value !== undefined) {
      if (duration !== undefined && duration !== Number(value)) throw new Error("conflicting video durations");
      duration = Number(value);
    }
  }
  // The gateway reads top-level seconds/duration. Do not charge a metadata
  // duration while forwarding a request that makes it use its default.
  if (duration !== undefined && body.seconds === undefined && body.duration === undefined) {
    throw new Error("video duration requires top-level seconds or duration");
  }
  const upstreamModel = ctx.upstreamModel || ctx.model || "";
  const fullSeedance = Boolean(seedanceModel({ upstreamModel }) && !/^\[c\]/i.test(upstreamModel));
  if (fullSeedance) {
    const routedAlias = /^(seedance-2\.[05])-(?:480p|720p|1080p|4k)$/i.exec(upstreamModel);
    if (routedAlias) body.model = routedAlias[1];
    const resolution = String(body.resolution || "720p").trim().toLowerCase();
    const is25 = /seedance-2[._-]5/i.test(upstreamModel);
    const allowed = is25
      ? ["480p", "720p", "1080p"] : ["480p", "720p", "1080p", "4k"];
    if (!allowed.includes(resolution)) throw new Error("unsupported Seedance resolution");
    if (duration !== undefined && duration !== -1 && (duration < 4 || duration > (is25 ? 30 : 15))) {
      throw new Error("unsupported Seedance duration");
    }
    body.resolution = resolution;
  }
  const base = ctx.baseUrl.replace(/\/$/, "");
  const path = ctx.action === "remix" ? "/v1/videos/" + encodeURIComponent(ctx.originTaskId) + "/remix" : "/v1/videos";
  const headers = { Authorization: "Bearer " + ctx.apiKey };
  for (const name of Object.keys(ctx.requestHeaders || {})) {
    if (name.toLowerCase() === "idempotency-key") headers["Idempotency-Key"] = ctx.requestHeaders[name];
  }
  headers["Content-Type"] = "application/json";
  return { url: base + path, method: "POST", headers, body };
}

export function extractUsage(ctx) {
  // Legacy prices already apply their own seconds multiplier.
  if (ctx.usagePurpose === "billing_ratios") return null;
  const model = seedanceModel(ctx);
  const body = ctx.requestBody || {};
  const metadata = typeof body.metadata === "string" ? JSON.parse(body.metadata) : body.metadata || {};
  const content = typeof body.content === "string" ? JSON.parse(body.content) : body.content || metadata.content || [];
  const references = typeof body.references === "string" ? JSON.parse(body.references) : body.references || [];
  const videoInput = content.some(item => item.type === "video_url") || references.some(item => item.type === "video") || (ctx.files || []).some(file => file.field === "video");
  if (!model && minimaxH3Model(ctx)) {
    const requested = Number(body.duration);
    const duration = Number.isInteger(requested) && requested > 0 && requested <= 15 ? requested : 15;
    return {
      resolution: String(body.resolution || "768P").trim().toLowerCase(),
      requests: 1,
      seconds: duration,
      video_input: videoInput,
      web_search_calls: 0,
    };
  }
  if (!model) return null;
  const hasVideoInput = videoInput || (metadata.reference_videos || []).length > 0;
  // Budget estimate only: the existing Ark formula uses pixels * 24 FPS / 1024.
  // Automatic output and unknown reference-video lengths reserve the documented
  // model duration ceiling. Completion replaces this estimate with real tokens.
  const maxDuration = model[1] === "seedance-2.5" ? 30 : 15;
  const requested = Number(body.duration === undefined ? body.seconds : body.duration);
  const seconds = requested > 0 ? requested : maxDuration;
  const resolution = model[2] || String(body.resolution || metadata.resolution || body.size || "720p").trim().toLowerCase();
  const pixels = { "480p": 854 * 480, "720p": 1280 * 720, "1080p": 1920 * 1080, "4k": 3840 * 2160 }[resolution];
  if (!pixels) throw new Error("unsupported Seedance resolution for token budget");
  // Keep the removed field at zero for expressions saved against older metadata.
  return { resolution, requests: 1, seconds, tokens: Math.ceil((seconds + (hasVideoInput ? maxDuration : 0)) * pixels * 24 / 1024), video_input: hasVideoInput, web_search_calls: 0 };
}

export function extractUsageOnComplete(_ctx, result, body) {
  if (result.status !== "SUCCESS") return null;
  const usage = body.usage || {};
  const facts = {};
  // Missing usage keeps the reservation; explicit zero remains a real zero.
  if (usage.seconds !== undefined) facts.seconds = usage.seconds;
  else if (usage.output_seconds !== undefined) facts.seconds = usage.output_seconds;
  if (usage.completion_tokens !== undefined) facts.tokens = usage.completion_tokens;
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
    headers: { Authorization: "Bearer " + ctx.apiKey },
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
  if (status === "FAILURE") result.reason = (body.error || {}).message || body.fail_reason || "gateway task failed";
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
  const data = ctx.data || {};
  const url = String((data.result || {}).url || (data.metadata || {}).url || "").trim();
  if (url.startsWith("https://")) {
    return { url, method: ctx.clientRequest.method, credentialless: true };
  }
  return {
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos/" + encodeURIComponent(ctx.upstreamTaskId) + "/content",
    method: ctx.clientRequest.method,
    headers: { Authorization: "Bearer " + ctx.apiKey },
  };
}

export const protocols = {
  openai_video: {
    decodeRequest(ctx) {
      if (ctx.body.kind === "multipart") return { kind: "submit", model: ctx.model, requestBody: normalizeVideoRequest(multipartRequest(ctx)) };
      if (ctx.body.kind !== "json") throw new Error("JSON or multipart body required");
      return { kind: "submit", model: ctx.model, requestBody: normalizeVideoRequest(ctx.body.value) };
    },
    render(_ctx, task) {
      const statuses = { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "in_progress", SUCCESS: "completed", FAILURE: "failed" };
      const result = Object.assign({}, task.data || {}, {
        id: task.task_id,
        task_id: task.task_id,
        model: (task.properties || {}).origin_model_name || "",
        status: statuses[task.status] || "unknown",
      });
      if (task.status === "SUCCESS") {
        const details = result.result && typeof result.result === "object" && !Array.isArray(result.result) ? result.result : {};
        const lastFrame = details.last_frame_url || result.last_frame_url || (result.metadata || {}).last_frame_url;
        result.result = Object.assign({}, details, {
          url: "/v1/videos/" + encodeURIComponent(task.task_id) + "/content",
          ...(lastFrame ? { last_frame_url: lastFrame } : {}),
        });
      }
      if (task.status === "FAILURE" && !result.error) result.error = { code: "video_generation_failed", message: task.fail_reason || "gateway task failed" };
      return result;
    },
  },
};
