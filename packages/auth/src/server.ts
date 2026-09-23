import { cors } from "@elysia/cors";
import { openapi } from "@elysia/openapi";
import { Elysia } from "elysia";
import { healthcheckPlugin } from "elysia-healthcheck";
import { auth } from "./auth";
import { authEnv } from "./env";

export const authServer = new Elysia({
  name: "auth-server",
  prefix: authEnv.AUTH_BASE_PATH,
})
  .use(
    cors({
      origin: "http://localhost:3000",
      methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      credentials: true,
      allowedHeaders: ["Content-Type", "Authorization"],
    }),
  )
  .use(openapi({}))
  .use(
    healthcheckPlugin({
      //prefix: "/health",
      paths: {
        liveness: "/liveness",
        readiness: "/readiness",
      },
    }),
  )
  .mount(auth.handler)
  .get("/hello", "Hello World");
