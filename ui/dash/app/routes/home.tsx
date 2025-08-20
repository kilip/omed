import type { Route } from "./+types/home";
import { Welcome } from "../welcome/welcome";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Welcome to Omed" },
    { name: "description", content: "Welcome to omed!" },
  ];
}

export default function Home() {
  return <Welcome />;
}
