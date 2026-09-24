import {
  adminClient,
  inferAdditionalFields,
  inferOrgAdditionalFields,
  jwtClient,
  organizationClient,
} from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";
import type { auth } from "./auth";
import { ac as orgAc, roles as orgRoles } from "./orgPermissions";

export const { signIn, signOut, token, organization, admin } = createAuthClient(
  {
    plugins: [
      adminClient(),
      inferAdditionalFields<typeof auth>(),
      organizationClient({
        schema: inferOrgAdditionalFields<typeof auth>(),
        ac: orgAc,
        roles: orgRoles,
      }),
      jwtClient(),
    ],
  },
);
