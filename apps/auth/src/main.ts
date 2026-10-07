import { Elysia } from "elysia";
import { server } from "./server";

const port = Number(process.env.AUTH_PORT ?? 8001);

new Elysia().use(server).listen({ port, hostname: "0.0.0.0" });

console.log(`[auth] listening on 0.0.0.0:${port}`);
