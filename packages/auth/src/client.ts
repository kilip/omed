import { adminClient, inferAdditionalFields, organizationClient } from "better-auth/client/plugins";
import { createAuthClient } from "better-auth/react";
import type { auth } from "./auth";

export const { signIn, signOut } = createAuthClient({
  plugins: [adminClient(), organizationClient(), inferAdditionalFields<typeof auth>()],
});
