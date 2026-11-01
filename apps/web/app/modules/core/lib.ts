import type { components } from "@omed/openapi";

export type Invitation = components["schemas"]["model.InvitationResponse"];
export type Member = components["schemas"]["model.MemberResponse"];
export type AccessToken = components["schemas"]["model.AccessTokenResponse"];
export type CreatedAccessToken =
  components["schemas"]["model.CreatedAccessTokenResponse"];

export const ACCESS_TOKEN_EXPIRY_DAYS = [30, 90, 365] as const;
export const DEFAULT_ACCESS_TOKEN_EXPIRY_DAYS = 90;

export type InvitationStatus = "active" | "revoked" | "expired" | "exhausted";

export function invitationStatus(
  invitation: Invitation,
  now = Date.now(),
): InvitationStatus {
  if (invitation.isRevoked) return "revoked";
  if (invitation.expiresAt && new Date(invitation.expiresAt).getTime() < now) {
    return "expired";
  }
  if (invitation.maxUses && (invitation.usesCount ?? 0) >= invitation.maxUses) {
    return "exhausted";
  }
  return "active";
}
