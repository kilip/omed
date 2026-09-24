import createClient from "openapi-fetch";
import { apiEnv } from "./env";
import type { paths as financePaths } from "./schema/finance";

const api = () => {
  return {
    finance: createClient<financePaths>({ baseUrl: apiEnv.API_FINANCE_URL }),
  };
};

export default api;
