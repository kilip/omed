import type { SocialProviders } from "better-auth/social-providers";
import { appEnv } from "@/config";

export const initSocialProviders = () => {
  const socialProviders = {
    ...(appEnv.AUTH_GOOGLE_ID && appEnv.AUTH_GOOGLE_SECRET
      ? {
          google: {
            clientId: appEnv.AUTH_GOOGLE_ID,
            clientSecret: appEnv.AUTH_GOOGLE_SECRET,
          },
        }
      : {}),
  } satisfies SocialProviders;

  return { socialProviders };
};
