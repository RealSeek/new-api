export const meta = {
  apiVersion: 1,
  key: "rs-gateway",
  name: "RS Gateway",
  version: "1.0.0",
  description: { en: "Video tasks managed by RS Gateway", zh: "由 RS Gateway 管理的视频任务" },
  author: { name: "RealSeek" },
  channelTypes: [61],
  models: [],
  fetchMode: "per_task",
  protocols: ["openai_video"],
};

export function buildSubmitRequest(ctx) {
  const body = Object.assign({}, ctx.requestBody, { model: ctx.upstreamModel });
  const metadata = typeof body.metadata === "string" ? JSON.parse(body.metadata) : body.metadata || {};
  let duration;
  for (const value of [body.seconds, body.duration, body.durationSeconds, metadata.seconds, metadata.duration, metadata.durationSeconds]) {
    if (value !== undefined && (!Number.isInteger(Number(value)) || Number(value) <= 0 || Number(value) > 3600)) {
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
      if (task.status === "FAILURE") result.error = { code: "video_generation_failed", message: task.fail_reason || "gateway task failed" };
      return result;
    },
  },
};
