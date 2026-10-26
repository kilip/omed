import { describe, expect, it } from "vitest";
import { type Invitation, invitationStatus } from "./lib";

describe("invitationStatus", () => {
  const baseInvitation: Invitation = {
    id: 1,
    token: "test-token-123",
    workspaceId: 10,
    roles: ["member"],
    isRevoked: false,
    createdAt: "2026-01-01T00:00:00Z",
  };

  const fixedNow = new Date("2026-06-01T12:00:00Z").getTime();

  it("returns revoked when isRevoked is true", () => {
    const inv: Invitation = {
      ...baseInvitation,
      isRevoked: true,
      expiresAt: "2026-05-01T00:00:00Z", // already expired too
      maxUses: 1,
      usesCount: 1, // exhausted too
    };

    expect(invitationStatus(inv, fixedNow)).toBe("revoked");
  });

  it("returns expired when expiresAt is in the past and not revoked", () => {
    const inv: Invitation = {
      ...baseInvitation,
      expiresAt: "2026-05-31T23:59:59Z",
      maxUses: 10,
      usesCount: 1,
    };

    expect(invitationStatus(inv, fixedNow)).toBe("expired");
  });

  it("returns exhausted when usesCount reaches or exceeds maxUses", () => {
    const invEqual: Invitation = {
      ...baseInvitation,
      expiresAt: "2026-07-01T00:00:00Z",
      maxUses: 3,
      usesCount: 3,
    };

    const invGreater: Invitation = {
      ...baseInvitation,
      expiresAt: "2026-07-01T00:00:00Z",
      maxUses: 3,
      usesCount: 5,
    };

    expect(invitationStatus(invEqual, fixedNow)).toBe("exhausted");
    expect(invitationStatus(invGreater, fixedNow)).toBe("exhausted");
  });

  it("returns active when invitation is valid and has not expired or exhausted", () => {
    const inv: Invitation = {
      ...baseInvitation,
      expiresAt: "2026-07-01T00:00:00Z",
      maxUses: 5,
      usesCount: 2,
    };

    expect(invitationStatus(inv, fixedNow)).toBe("active");
  });

  it("returns active when invitation has no expiresAt and no maxUses", () => {
    const inv: Invitation = {
      ...baseInvitation,
      expiresAt: undefined,
      maxUses: undefined,
      usesCount: 0,
    };

    expect(invitationStatus(inv, fixedNow)).toBe("active");
  });

  it("treats undefined usesCount as 0 when comparing against maxUses", () => {
    const inv: Invitation = {
      ...baseInvitation,
      maxUses: 1,
      usesCount: undefined,
    };

    expect(invitationStatus(inv, fixedNow)).toBe("active");
  });
});
