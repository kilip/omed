import { describe, expect, it } from "vitest";
import { AGENT_ROLES, evaluateAgentPermissions } from "./use-agent-permissions";

describe("evaluateAgentPermissions", () => {
  it("gives full permissions to workspace owner", () => {
    const perms = evaluateAgentPermissions({ owner: true });
    expect(perms.canRead).toBe(true);
    expect(perms.canManage).toBe(true);
  });

  it("gives full permissions to agent admin", () => {
    const perms = evaluateAgentPermissions({ roles: [AGENT_ROLES.ADMIN] });
    expect(perms.canRead).toBe(true);
    expect(perms.canManage).toBe(true);
  });

  it("gives read-only permissions to agent viewer", () => {
    const perms = evaluateAgentPermissions({ roles: [AGENT_ROLES.VIEWER] });
    expect(perms.canRead).toBe(true);
    expect(perms.canManage).toBe(false);
  });

  it("handles granular permissions", () => {
    const perms = evaluateAgentPermissions({
      roles: [],
      permissions: [{ object: "agent:provider", action: "read" }],
    });
    expect(perms.canRead).toBe(true);
    expect(perms.canManage).toBe(false);
  });

  it("handles empty / default permissions", () => {
    const perms = evaluateAgentPermissions({ roles: [] });
    expect(perms.canRead).toBe(false);
    expect(perms.canManage).toBe(false);
  });
});
