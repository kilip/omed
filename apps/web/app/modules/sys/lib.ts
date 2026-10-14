import type { components } from "@omed/openapi";

export type Invitation = components["schemas"]["model.InvitationResponse"];
export type Member = components["schemas"]["model.MemberResponse"];

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
