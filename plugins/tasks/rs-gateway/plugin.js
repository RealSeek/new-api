export const meta = {
  apiVersion: 1,
  key: "rs-gateway",
  name: "RS Gateway",
  version: "1.2.1",
  description: { en: "Video tasks managed by RS Gateway", zh: "由 RS Gateway 管理的视频任务" },
  author: { name: "RealSeek" },
  channelTypes: [61],
  models: [],
  fetchMode: "per_task",
  protocols: ["openai_video"],
  usageSchema: {
    resolution: { enum: ["480p", "720p", "1080p", "4k"], allowCustomValues: true, description: { en: "Output video resolution", zh: "输出视频分辨率" } },
    seconds: { type: "number", unit: "second", description: { en: "Video generation unit price", zh: "视频生成单价" } },
    tokens: { type: "number", unit: "token", description: { en: "Video generation token unit price", zh: "视频生成 Token 单价" } },
    video_input: { type: "boolean", description: { en: "Reference video present", zh: "存在参考视频" } },
  },
  usageExamples: [{ label: "720p · 5s", facts: { resolution: "720p", seconds: 5, tokens: 108000, video_input: false } }],
};

function seedanceModel(ctx) {
  const model = /^(seedance-2\.[05])(?:-(480p|720p|1080p|4k))?$/.exec(ctx.upstreamModel || ctx.model || "");
  return model && !(model[1] === "seedance-2.5" && model[2] === "4k") ? model : null;
}

export function buildSubmitRequest(ctx) {
  const body = Object.assign({}, ctx.requestBody, { model: ctx.upstreamModel });
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
  const base = ctx.baseUrl.replace(/\/$/, "");
  const path = ctx.action === "remix" ? "/v1/videos/" + encodeURIComponent(ctx.originTaskId) + "/remix" : "/v1/videos";
  const headers = { Authorization: "Bearer " + ctx.apiKey };
  if ((ctx.files || []).length || String((ctx.requestHeaders || {})["Content-Type"] || "").includes("multipart/form-data")) {
    const parts = [];
    for (const key of Object.keys(body)) {
      if (body[key] !== undefined && body[key] !== null) {
        parts.push({ name: key, value: typeof body[key] === "object" ? JSON.stringify(body[key]) : body[key] });
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
  if (!model) return null;
  const body = ctx.requestBody || {};
  const metadata = typeof body.metadata === "string" ? JSON.parse(body.metadata) : body.metadata || {};
  const content = typeof body.content === "string" ? JSON.parse(body.content) : body.content || metadata.content || [];
  const references = typeof body.references === "string" ? JSON.parse(body.references) : body.references || [];
  const videoInput = content.some(item => item.type === "video_url") || references.some(item => item.type === "video") || (metadata.reference_videos || []).length > 0 || (ctx.files || []).some(file => file.field === "video");
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
  return { resolution, seconds, tokens: Math.ceil((seconds + (videoInput ? maxDuration : 0)) * pixels * 24 / 1024), video_input: videoInput, web_search_calls: 0 };
}

export function extractUsageOnComplete(_ctx, result, body) {
  if (result.status !== "SUCCESS") return null;
  const usage = body.usage || {};
  const facts = {};
  // Missing usage keeps the reservation; explicit zero remains a real zero.
  if (usage.seconds !== undefined) facts.seconds = usage.seconds;
  if (usage.completion_tokens !== undefined) facts.tokens = usage.completion_tokens;
  return facts;
}

export function parseSubmitResponse(_ctx, resp) {
  const body = resp.body || {};
  const taskId = body.id || body.task_id;
  if (!taskId) throw new Error("gateway task id is missing");
  return { taskId, taskData: body };
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
  return result;
}

export function listArtifacts(task) {
  return task.status === "SUCCESS" ? [{ key: "video", type: "video" }] : [];
}

export function buildContentRequest(ctx) {
  if (ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  return {
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos/" + encodeURIComponent(ctx.upstreamTaskId) + "/content",
    method: ctx.clientRequest.method,
    headers: { Authorization: "Bearer " + ctx.apiKey },
  };
}

export const protocols = {
  openai_video: {
    decodeRequest(ctx) {
      if (ctx.body.kind !== "json") throw new Error("JSON body required");
      return { kind: "submit", model: ctx.model, requestBody: ctx.body.value };
    },
    render(_ctx, task) {
      const statuses = { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "in_progress", SUCCESS: "completed", FAILURE: "failed" };
      const result = Object.assign({}, task.data || {}, {
        id: task.task_id,
        task_id: task.task_id,
        model: (task.properties || {}).origin_model_name || "",
        status: statuses[task.status] || "unknown",
      });
      if (task.status === "FAILURE" && !result.error) result.error = { code: "video_generation_failed", message: task.fail_reason || "gateway task failed" };
      return result;
    },
  },
};
