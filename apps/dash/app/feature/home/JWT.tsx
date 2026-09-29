"use client";
import { Button } from "antd";
import { useAuth } from "@/shared/providers/AuthProvider";

export default function JWT() {
  const { token, user, api } = useAuth();
  return (
    <div>
      <Button
        onClick={async () => {
          const { data: response } = await api.finance.GET("/ping");
          console.log(response?.data?.databaseStatus);
        }}
      >
        Ping
      </Button>
      {token}
      <div>user: {user?.name}</div>
    </div>
  );
}
