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

      // Collect request body to determine if tools or specific scenarios are requested
      let bodyStr = "";
      req.on("data", (chunk) => {
        bodyStr += chunk;
      });
      req.on("end", () => {
        let body: Record<string, unknown> = {};
        try {
          body = JSON.parse(bodyStr) as Record<string, unknown>;
        } catch {
          // ignore
        }

        const messages: Array<{
          role?: string;
          content?: unknown;
          tool_call_id?: string;
        }> = Array.isArray(body.messages)
          ? (body.messages as Array<{
              role?: string;
              content?: unknown;
              tool_call_id?: string;
            }>)
          : [];
        const lastMessage = messages[messages.length - 1];

        // If the last message is a tool response, finish with assistant reply
        if (lastMessage?.role === "tool") {
          res.writeHead(200);
          res.end(
            JSON.stringify({
              id: `chatcmpl-${Date.now()}`,
              object: "chat.completion",
              created: 1700000000,
              model: "gpt-4o",
              choices: [
                {
                  index: 0,
                  message: {
                    role: "assistant",
                    content:
                      "I have prepared the draft transaction for your review.",
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
          return;
        }

        // Check user prompt context (exclude system prompt to avoid keyword collision)
        const nonSystemMessages = (
          messages as Array<{ role?: string; content?: unknown }>
        ).filter((m) => m.role !== "system");

        let userText = JSON.stringify(nonSystemMessages).toLowerCase();
        const base64Matches =
          userText.match(/data:image\/[^;]+;base64,([a-zA-Z0-9+/=]+)/g) || [];
        for (const match of base64Matches) {
          const b64 = match.split("base64,")[1];
          if (b64) {
            try {
              userText +=
                " " +
                Buffer.from(b64, "base64").toString("utf-8").toLowerCase();
            } catch {}
          }
        }

        const isAtm =
          userText.includes("atm") ||
          userText.includes("kaltimtara") ||
          userText.includes("withdrawal");
        const isReceipt =
          userText.includes("receipt") ||
          userText.includes("shop") ||
          userText.includes("stationery") ||
          userText.includes("expense");
        const tools: Array<{ function?: { name?: string } }> = Array.isArray(
          body.tools,
        )
          ? (body.tools as Array<{ function?: { name?: string } }>)
          : [];
        const hasProposeDraft = tools.some(
          (t) => t.function?.name === "propose_draft",
        );

        if (hasProposeDraft && isAtm) {
          // Reference Case 1: ATM slip journal draft
          const proposal = {
            kind: "journal",
            transactionDate: "2026-01-15",
            description: "Tarik Tunai ATM Bank Kaltimtara",
            reference: "ATM-KALTIMTARA-001",
            currency: "IDR",
            totalAmount: "2000000.00",
            entries: [
              {
                accountCode: "1101",
                direction: "DEBIT",
                amount: "2000000.00",
                description: "Kas Utama",
              },
              {
                accountCode: "1102",
                direction: "CREDIT",
                amount: "2000000.00",
                description: "Bank Kaltimtara",
              },
            ],
          };

          res.writeHead(200);
          res.end(
            JSON.stringify({
              id: `chatcmpl-${Date.now()}`,
              object: "chat.completion",
              created: 1700000000,
              model: "gpt-4o",
              choices: [
                {
                  index: 0,
                  message: {
                    role: "assistant",
                    content: null,
                    tool_calls: [
                      {
                        id: `call_${Date.now()}`,
                        type: "function",
                        function: {
                          name: "propose_draft",
                          arguments: JSON.stringify(proposal),
                        },
                      },
                    ],
                  },
                  finish_reason: "tool_calls",
                },
              ],
              usage: {
                prompt_tokens: 1,
                completion_tokens: 1,
                total_tokens: 2,
              },
            }),
          );
          return;
        }

        if (hasProposeDraft && isReceipt) {
          // Reference Case 2: Multi-item shop receipt
          const proposal = {
            kind: "expense",
            paymentAccountCode: "1101",
            transactionDate: "2026-01-15",
            description: "Toko Alat Tulis Kantor",
            reference: "RCPT-SHOP-001",
            currency: "IDR",
            totalAmount: "70000.00",
            items: [
              {
                accountCode: "5101",
                description: "Kertas HVS A4",
                quantity: "1",
                unitPrice: "50000.00",
                amount: "50000.00",
              },
              {
                accountCode: "5101",
                description: "Pulpen Gel Hitam",
                quantity: "2",
                unitPrice: "10000.00",
                amount: "20000.00",
              },
            ],
          };

          res.writeHead(200);
          res.end(
            JSON.stringify({
              id: `chatcmpl-${Date.now()}`,
              object: "chat.completion",
              created: 1700000000,
              model: "gpt-4o",
              choices: [
                {
                  index: 0,
                  message: {
                    role: "assistant",
                    content: null,
                    tool_calls: [
                      {
                        id: `call_${Date.now()}`,
                        type: "function",
                        function: {
                          name: "propose_draft",
                          arguments: JSON.stringify(proposal),
                        },
                      },
                    ],
                  },
                  finish_reason: "tool_calls",
                },
              ],
              usage: {
                prompt_tokens: 1,
                completion_tokens: 1,
                total_tokens: 2,
              },
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
