import { Welcome } from "~/welcome/welcome";
import type { Route } from "./+types/home";
import { useOutletContext } from "react-router";
import type { RootOutletContext } from "~/types";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Dashboard" },
    { name: "description", content: "Welcome to Omed!" },
  ];
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const { user } = useOutletContext<RootOutletContext>();

  return (
    <div>
      <h3>Hello {user.name}!</h3>
      <Welcome />
    </div>
  );
}
