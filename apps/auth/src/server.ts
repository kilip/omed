import { cors } from "@elysia/cors";
import { openapi } from "@elysia/openapi";
import { auth, authEnv } from "@omed/better-auth/auth";
import { Elysia } from "elysia";
import { OpenAPI } from "./openapi";

const betterAuth = new Elysia({ name: "better-auth" })
  .mount(auth.handler)
  .macro({
    auth: {
      async resolve({ status, request: { headers } }) {
        const session = await auth.api.getSession({
          headers,
        });
        if (!session) return status(401);
        return {
          user: session.user,
          session: session.session,
        };
      },
    },
  });

export const server = new Elysia({})
  .onRequest(({ request }) => {
    console.log(`[REQ] ${request.method} ${request.url}`);
  })
  .use(
    cors({
      origin: authEnv.OMED_TRUSTED_ORIGINS,
      methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      credentials: true,
      allowedHeaders: ["Content-Type", "Authorization"],
      // preflight: true,
    }),
  )
  .use(
    openapi({
      documentation: {
        components: await OpenAPI.components,
        paths: await OpenAPI.getPaths(),
      },
    }),
  )
  .get("/", ({ redirect }) => redirect("/openapi"))
  .get("/health", () => ({
    status: "ok",
    timestamp: new Date().toISOString(),
  }))
  .use(betterAuth);
