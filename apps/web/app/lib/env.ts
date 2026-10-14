const read = (value: unknown, fallback: string): string =>
  typeof value === "string" && value.length > 0 ? value : fallback;

const apiUrl = read(import.meta.env.VITE_API_URL, "http://localhost:8002");

export const env = {
  apiUrl,
  publicUrl: read(import.meta.env.VITE_PUBLIC_URL, "http://localhost:3001"),
};
