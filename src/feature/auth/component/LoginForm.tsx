"use client";

import { GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { Button, Space } from "antd";
import { useRouter, useSearchParams } from "next/navigation";
import { signIn } from "@/libs/better-auth/client";

export default function LoginForm() {
  const searchParams = useSearchParams();
  const ref = searchParams.get("ref") ?? "/";
  const router = useRouter();

  const signInWith = async (provider: string) => {
    await signIn.social({
      provider,
      callbackURL: ref,
      fetchOptions: {
        onSuccess: () => {
          router.push(ref);
        },
      },
    });
  };

  return (
    <Space orientation="vertical" style={{ width: 300 }} size={"middle"}>
      <Button
        block
        icon={<GoogleOutlined />}
        size="large"
        onClick={async () => await signInWith("google")}
      >
        Sign in with google
      </Button>
      <Button
        block
        icon={<GithubOutlined />}
        size="large"
        onClick={async () => await signInWith("github")}
      >
        Sign in with google
      </Button>
    </Space>
  );
}
