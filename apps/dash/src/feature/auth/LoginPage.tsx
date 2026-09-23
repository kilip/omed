import type { Metadata } from "next";
import LoginForm from "./component/LoginForm";

export const metadata: Metadata = {
  title: "Login",
};

export default function LoginPage() {
  return <LoginForm />;
}
