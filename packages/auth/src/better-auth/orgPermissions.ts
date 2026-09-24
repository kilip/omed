import { createAccessControl } from "better-auth/plugins/access";
import {
  adminAc,
  defaultStatements,
  memberAc,
  ownerAc,
} from "better-auth/plugins/organization/access";

const statement = {
  ...defaultStatements,
  finance: ["create", "read", "update", "delete"],
} as const;

export const ac = createAccessControl(statement);

export const member = ac.newRole({
  ...memberAc.statements,
  finance: ["read"],
});

export const admin = ac.newRole({
  ...adminAc.statements,
  finance: ["create", "update", "read", "delete"],
});

export const owner = ac.newRole({
  ...ownerAc.statements,
  finance: ["create", "update", "read", "delete"],
});

export const roles = {
  owner,
  admin,
  member,
};
