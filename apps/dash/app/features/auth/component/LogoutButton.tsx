"use client";
import { LogoutOutlined } from "@ant-design/icons";
import { Button } from "antd";
import { useNavigate } from "react-router";
import { signOut } from "~/shared/auth";

export default function LogoutButton() {
  const navigate = useNavigate();
  return (
    <Button
      type="primary"
      icon={<LogoutOutlined />}
      onClick={async () => {
        await signOut({
          fetchOptions: {
            onSuccess: () => {
              navigate("/login", { replace: true });
            },
          },
        });
      }}
    >
      Logout
    </Button>
  );
}
