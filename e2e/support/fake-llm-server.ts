import http from "node:http";

export const FAKE_LLM_PORT = 4010;
export const FAKE_LLM_URL = `http://127.0.0.1:${FAKE_LLM_PORT}/v1`;

let server: http.Server | null = null;

/**
 * Starts a minimal fake OpenAI-compatible HTTP server.
 * Handles chat completions and model listing for connection testing.
 */
export function startFakeLlmServer(port = FAKE_LLM_PORT): Promise<void> {
  if (server) return Promise.resolve();

  return new Promise((resolve, reject) => {
    const srv = http.createServer((req, res) => {
      res.setHeader("Access-Control-Allow-Origin", "*");
      res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
      res.setHeader("Access-Control-Allow-Headers", "*");

      if (req.method === "OPTIONS") {
        res.writeHead(204);
        res.end();
        return;
      }

      res.setHeader("Content-Type", "application/json");

      const url = req.url ?? "";
      if (url.includes("/models")) {
        res.writeHead(200);
        res.end(
          JSON.stringify({
            object: "list",
            data: [
              {
                id: "gpt-4o",
                object: "model",
                created: 1700000000,
                owned_by: "openai",
              },
            ],
          }),
        );
        return;
      }

      // Default response for chat/completions and other endpoints
      res.writeHead(200);
      res.end(
        JSON.stringify({
          id: "chatcmpl-test",
          object: "chat.completion",
          created: 1700000000,
          model: "gpt-4o",
          choices: [
            {
              index: 0,
              message: {
                role: "assistant",
                content: "ok",
              },
              finish_reason: "stop",
            },
          ],
          usage: {
            prompt_tokens: 1,
            completion_tokens: 1,
            total_tokens: 2,
          },
        }),
      );
    });

    srv.once("error", (err: NodeJS.ErrnoException) => {
      if (err.code === "EADDRINUSE") {
        console.log(
          `[fake-llm] port ${port} already in use, assuming server is running`,
        );
        resolve();
      } else {
        reject(err);
      }
    });

    srv.listen(port, "127.0.0.1", () => {
      server = srv;
      console.log(`[fake-llm] server listening on http://127.0.0.1:${port}`);
      resolve();
    });
  });
}

/**
 * Stops the fake LLM server if running.
 */
export function stopFakeLlmServer(): Promise<void> {
  if (!server) return Promise.resolve();
  return new Promise((resolve) => {
    server?.close(() => {
      server = null;
      resolve();
    });
  });
}
