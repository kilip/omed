import type { components } from "./specs/finance";

export * from "./specs/finance";

// Common Type
export type Meta = components["schemas"]["model.Meta"];
export type ErrorResponse = components["schemas"]["model.ErrorResponse"];
export type ErrorBody = components["schemas"]["model.ErrorBody"];
export type FieldError = components["schemas"]["model.FieldError"];

// Account Types
export type AccountResponse =
  components["schemas"]["model.WebResponse-model_AccountResponse"];
export type AccountsResponse =
  components["schemas"]["model.WebResponse-array_model_AccountResponse"];
export type Account = components["schemas"]["model.AccountResponse"];
export type AccountType = components["schemas"]["model.AccountType"];
export type AccountStatus = components["schemas"]["model.AccountStatus"];
export type SeedAccountRequest =
  components["schemas"]["model.SeedAccountRequest"];
export type SeedTemplate = components["schemas"]["model.SeedTemplateResponse"];
export type SeedTemplatesResponse =
  components["schemas"]["model.WebResponse-array_model_SeedTemplateResponse"];
export type SeedPreviewResponse =
  components["schemas"]["model.WebResponse-model_SeedPreviewResponse"];
export type SeedPreviewAccount =
  components["schemas"]["model.SeedPreviewAccount"];
export type CreateAccountRequest =
  components["schemas"]["model.CreateAccountRequest"];
export type UpdateAccountRequest =
  components["schemas"]["model.UpdateAccountRequest"];
