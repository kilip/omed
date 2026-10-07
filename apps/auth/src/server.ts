import { cors } from "@elysia/cors";
import { openapi } from "@elysia/openapi";
import { auth, authEnv } from "@omed/better-auth";
import { Elysia } from "elysia";
import { OpenAPI } from "./openapi";

const betterAuth = new Elysia({ name: "better-auth" })
  .all("/*", ({ request }) => auth.handler(request))
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
      origin: [authEnv.AUTH_URL, ...authEnv.AUTH_TRUSTED_ORIGINS],
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
  .use(betterAuth)
  .get("/", () => `Hello World ${authEnv.AUTH_URL}`)
  .get("/hello", () => "Hello World Lagi");
