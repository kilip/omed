"use client";
import { useAuth } from "@/shared/providers/AuthProvider";

export default function Status() {
  const { token } = useAuth();
  return <div>{token}</div>;
}
