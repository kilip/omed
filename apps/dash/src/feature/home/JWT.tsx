"use client";
import { useApi } from "@/shared/hooks/useApi";
import { token as genToken, organization } from "@omed/auth/client";
import { Button } from "antd";

export default function JWT() {
  const { token } = useApi();
  return (
    <div>
      <Button>Check Permission</Button>
      {token}
    </div>
  );
}
