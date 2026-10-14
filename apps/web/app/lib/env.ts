const read = (value: unknown, fallback: string): string =>
  typeof value === "string" && value.length > 0 ? value : fallback;

const apiUrl = read(import.meta.env.VITE_API_URL, "http://localhost:8002");

const rawAuthUrl = read(import.meta.env.VITE_AUTH_URL, apiUrl);

export const env = {
  apiUrl,
  authUrl: rawAuthUrl === "http://localhost:8001" ? apiUrl : rawAuthUrl,
  publicUrl: read(import.meta.env.VITE_PUBLIC_URL, "http://localhost:3001"),
};
