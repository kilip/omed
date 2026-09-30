import type { components } from "./specs/finance";

export * from "./specs/finance";

export type AccountResponse =
  components["schemas"]["model.WebResponse-model_Account"];
export type CreateAccountRequest =
  components["schemas"]["model.CreateAccountRequest"];
