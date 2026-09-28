import { cors } from "@elysia/cors";
import { openapi } from "@elysia/openapi";
import { auth, authEnv } from "@omed/better-auth";
import { Elysia } from "elysia";
import { CloudflareAdapter } from "elysia/adapter/cloudflare-worker";
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
export const app = new Elysia({
  adapter: CloudflareAdapter,
})
  .use(
    cors({
      origin: authEnv.AUTH_BASE_URL,
      methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      credentials: true,
      allowedHeaders: ["Content-Type", "Authorization"],
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
  .get("/", () => `Hello World ${authEnv.AUTH_BASE_URL}`)
  .get("/hello", () => "Hello World")
  .compile();

export default app;
