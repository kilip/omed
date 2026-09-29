import { useState } from "react";

export function useFinance(token: string) {
  const [foo, setFoo] = useState<string>(token);
  return {
    foo,
    setFoo,
  };
}
