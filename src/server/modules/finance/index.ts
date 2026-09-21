import Elysia from "elysia";
import { account } from "./account";

export const finance = new Elysia().use(account);
