"use client";
import { token } from "@omed/auth/client";
import { Button } from "antd";
import { useState } from "react";

export default function JWT() {
  const [tkn, setTkn] = useState<string | undefined>(undefined);
  return (
    <div>
      <Button
        onClick={async () => {
          const data = await token();
          setTkn(data.data?.token);
        }}
      >
        Token
      </Button>
      {tkn}
    </div>
  );
}
