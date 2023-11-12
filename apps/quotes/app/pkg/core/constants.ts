import { ensureEnv } from "@omed/utils";

export const constants = {
  SECRET: ensureEnv("SECRET"),
} as const;

export default constants;
