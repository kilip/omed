import invariant from "tiny-invariant";

export function ensureEnv(name: string) {
  const value = process.env[name];

  invariant(value, `${name} environment variables not configured`);

  return value;
}

export function singleton<Value>(
  name: string,
  valueFactory: () => Value
): Value {
  const g = global as unknown as { __singletons: Record<string, unknown> };
  g.__singletons ??= {};
  g.__singletons[name] ??= valueFactory();
  return g.__singletons[name] as Value;
}
