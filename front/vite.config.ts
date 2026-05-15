import react from "@vitejs/plugin-react";
import http from "node:http";
import https from "node:https";
import { defineConfig, type Plugin } from "vite";

function uploadProxyPlugin(): Plugin {
  return {
    name: "public-survey-upload-proxy",
    configureServer(server) {
      server.middlewares.use("/__upload_proxy", (req, res) => {
        if (req.method !== "POST") {
          res.statusCode = 405;
          res.end("Method not allowed");
          return;
        }

        const requestUrl = new URL(req.url || "", "http://localhost");
        const uploadUrl = requestUrl.searchParams.get("uploadUrl");
        if (!uploadUrl) {
          res.statusCode = 400;
          res.end("uploadUrl is required");
          return;
        }

        const chunks: Buffer[] = [];
        req.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
        req.on("end", () => {
          forwardUpload(uploadUrl, Buffer.concat(chunks), String(req.headers["x-upload-content-type"] || "application/octet-stream"))
            .then(({ status, body }) => {
              res.statusCode = status;
              res.end(body);
            })
            .catch((error: unknown) => {
              res.statusCode = 502;
              res.end(error instanceof Error ? error.message : "Upload failed");
            });
        });
      });
    },
  };
}

async function forwardUpload(uploadUrl: string, body: Buffer, contentType: string): Promise<{ status: number; body: Buffer }> {
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

function tryUpload(target: string, body: Buffer, contentType: string, hostHeader?: string): Promise<{ status: number; body: Buffer }> {
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
        const chunks: Buffer[] = [];
        response.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
        response.on("end", () => resolve({ status: response.statusCode || 500, body: Buffer.concat(chunks) }));
      },
    );
    request.on("error", reject);
    request.end(body);
  });
}

export default defineConfig({
  plugins: [react(), uploadProxyPlugin()],
  server: {
    proxy: {
      "/v1": {
        target: "http://localhost:8080",
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyReq", (proxyReq) => {
            proxyReq.removeHeader("origin");
          });
        },
      },
      "/healthz": {
        target: "http://localhost:8080",
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyReq", (proxyReq) => {
            proxyReq.removeHeader("origin");
          });
        },
      },
    },
  },
});
