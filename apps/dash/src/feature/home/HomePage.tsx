import { auth } from "@omed/auth";
import type { Metadata } from "next";
import { headers } from "next/headers";
import LogoutButton from "../auth/component/LogoutButton";
import JWT from "./JWT";

export const metadata: Metadata = {
  title: "Home",
};

export default async function HomePage() {
  const session = await auth.api.getSession({ headers: await headers() });
  return (
    <div>
      <h1>Home</h1>
      <LogoutButton />
      <div>
        {session?.user.name} - {session?.user.email} {session?.user.image}
      </div>
      <div>active org: {session?.session.activeOrganizationId}</div>
      <div>active team: {session?.session.activeTeamId}</div>
      <JWT />
    </div>
  );
}
