import { Elysia } from "elysia";
import { CloudflareAdapter } from "elysia/adapter/cloudflare-worker";
import { server } from "./server";

const app = new Elysia({ adapter: CloudflareAdapter }).use(server).compile();

export default app;
