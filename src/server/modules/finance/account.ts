import Elysia from "elysia";
import { createAccountSchema } from "@/database/schema";
import { betterAuthMacro } from "@/server/better-auth";

export const account = new Elysia({ prefix: "/accounts" }).use(betterAuthMacro);

account.post(
  "/",
  ({ user }) => {
    console.log(user);
    return user;
  },
  {
    auth: true,
    body: createAccountSchema,
  },
);
