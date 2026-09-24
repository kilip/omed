"use client";
import { token as genToken, organization } from "@omed/auth/client";
import { Button } from "antd";
import { useState } from "react";

export default function JWT() {
  const [token, setToken] = useState<string | null>(null);
  return (
    <div>
      <Button
        onClick={async () => {
          const data = await genToken();
          setToken(data.data?.token ?? null);
        }}
      >
        Token
      </Button>
      <Button
        onClick={async () => {
          const canCreateFinance = await organization.hasPermission({
            permissions: {
              finance: ["create"],
            },
          });
        }}
      >
        Check Permission
      </Button>
      {token}
    </div>
  );
}
