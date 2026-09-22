import type { Metadata } from "next";
import LoginForm from "./component/LoginForm";

export const metadata: Metadata = {
  title: "Login to Omed",
};

export default function LoginPage() {
  return (
    <div>
      <LoginForm />
    </div>
  );
}
