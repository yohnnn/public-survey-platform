import fs from "node:fs";
import http from "node:http";
import https from "node:https";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const distDir = path.join(__dirname, "dist");
const port = Number(process.env.PORT || 3000);
const apiTarget = process.env.API_PROXY_TARGET || "http://localhost:8080";

const mimeTypes = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
};

const server = http.createServer(async (req, res) => {
  try {
    const url = new URL(req.url || "/", `http://${req.headers.host || "localhost"}`);

    if (url.pathname === "/__upload_proxy" && req.method === "POST") {
      await handleUploadProxy(req, res, url);
      return;
    }

    if (url.pathname === "/healthz" || url.pathname.startsWith("/v1/")) {
      proxyRequest(req, res, apiTarget);
      return;
    }

    serveStatic(url.pathname, res);
  } catch (error) {
    res.writeHead(500, { "Content-Type": "text/plain; charset=utf-8" });
    res.end(error instanceof Error ? error.message : "Internal server error");
  }
});

server.listen(port, "0.0.0.0", () => {
  console.log(`front is serving on http://localhost:${port}`);
});

function serveStatic(urlPath, res) {
  const normalized = path.normalize(decodeURIComponent(urlPath)).replace(/^(\.\.[/\\])+/, "");
  const candidate = path.join(distDir, normalized === "/" ? "index.html" : normalized);
  const filePath = candidate.startsWith(distDir) && fs.existsSync(candidate) && fs.statSync(candidate).isFile() ? candidate : path.join(distDir, "index.html");
  const ext = path.extname(filePath);

  res.writeHead(200, {
    "Content-Type": mimeTypes[ext] || "application/octet-stream",
    "Cache-Control": ext === ".html" ? "no-cache" : "public, max-age=31536000, immutable",
  });
  fs.createReadStream(filePath).pipe(res);
}

async function handleUploadProxy(req, res, url) {
  const uploadUrl = url.searchParams.get("uploadUrl");
  if (!uploadUrl) {
    res.writeHead(400, { "Content-Type": "text/plain; charset=utf-8" });
    res.end("uploadUrl is required");
    return;
  }

  const body = await readBody(req);
  const contentType = String(req.headers["x-upload-content-type"] || "application/octet-stream");
  const result = await forwardUpload(uploadUrl, body, contentType);
  res.writeHead(result.status);
  res.end(result.body);
}

function proxyRequest(req, res, targetBase) {
  const target = new URL(req.url || "/", targetBase);
  const transport = target.protocol === "https:" ? https : http;
  const headers = { ...req.headers, host: target.host };
  delete headers.origin;
  const proxy = transport.request(
    target,
    {
      method: req.method,
      headers,
    },
    (proxyResponse) => {
      res.writeHead(proxyResponse.statusCode || 502, proxyResponse.headers);
      proxyResponse.pipe(res);
    },
  );

  proxy.on("error", (error) => {
    res.writeHead(502, { "Content-Type": "text/plain; charset=utf-8" });
    res.end(error.message);
  });
  req.pipe(proxy);
}

async function forwardUpload(uploadUrl, body, contentType) {
  const primary = await tryUpload(uploadUrl, body, contentType);
  if (primary.status >= 200 && primary.status < 300) {
    return primary;
  }

  const parsed = new URL(uploadUrl);
  if (!parsed.hostname || parsed.hostname === "localhost" || parsed.hostname === "127.0.0.1") {
    return primary;
  }

  const fallback = new URL(uploadUrl);
  fallback.hostname = "localhost";
  return tryUpload(fallback.toString(), body, contentType, parsed.host);
}

function tryUpload(target, body, contentType, hostHeader) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(target);
    const transport = parsed.protocol === "https:" ? https : http;
    const request = transport.request(
      parsed,
      {
        method: "PUT",
        headers: {
          "Content-Type": contentType,
          "Content-Length": body.length,
          ...(hostHeader ? { Host: hostHeader } : {}),
        },
      },
      (response) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
        response.on("end", () => resolve({ status: response.statusCode || 500, body: Buffer.concat(chunks) }));
      },
    );
    request.on("error", reject);
    request.end(body);
  });
}

function readBody(req) {
  return new Promise((resolve) => {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
    req.on("end", () => resolve(Buffer.concat(chunks)));
  });
}
