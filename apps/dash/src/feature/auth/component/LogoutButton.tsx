"use client";
import { LogoutOutlined } from "@ant-design/icons";
import { signOut } from "@omed/auth/client";
import { Button } from "antd";
import { useRouter } from "next/navigation";

export default function LogoutButton() {
  const router = useRouter();
  return (
    <Button
      type="primary"
      icon={<LogoutOutlined />}
      onClick={async () => {
        await signOut({
          fetchOptions: {
            onSuccess: () => {
              router.push("/login");
            },
          },
        });
      }}
    >
      Logout
    </Button>
  );
}
