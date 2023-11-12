import invariant from "tiny-invariant";

export function ensureEnv(name: string) {
  const value = process.env[name];

  invariant(value, `${name} environment variables not configured`);

  return value;
}
