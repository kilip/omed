import {
  adminClient,
  inferAdditionalFields,
  inferOrgAdditionalFields,
  jwtClient,
  organizationClient,
} from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";
import type { auth } from "./auth";

export const { signIn, signOut, token } = createAuthClient({
  plugins: [
    adminClient(),
    inferAdditionalFields<typeof auth>(),
    organizationClient({
      schema: inferOrgAdditionalFields<typeof auth>(),
    }),
    jwtClient(),
  ],
});
