import { cors } from "@elysia/cors";
import { openapi } from "@elysia/openapi";
import { Elysia } from "elysia";
import { auth } from "@/auth";
import { finance } from "./modules";

const app = new Elysia({ prefix: "/api" })
  .use(openapi())
  .use(
    cors({
      origin: "http://localhost:3000",
      methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      credentials: true,
      allowedHeaders: ["Content-Type", "Authorization"],
    }),
  )
  .mount(auth.handler)
  .use(finance);

export const createNextHandler = () => {
  return {
    GET: app.fetch,
    POST: app.fetch,
  };
};
