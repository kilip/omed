const read = (value: unknown, fallback: string): string =>
  typeof value === "string" && value.length > 0 ? value : fallback;

export const env = {
  authUrl: read(import.meta.env.VITE_AUTH_URL, "http://localhost:8001"),
  apiUrl: read(import.meta.env.VITE_API_URL, "http://localhost:8002"),
  publicUrl: read(import.meta.env.VITE_PUBLIC_URL, "http://localhost:3001"),
};
