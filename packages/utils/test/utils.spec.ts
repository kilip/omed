import { describe, expect, it } from "vitest";
import { ensureEnv, singleton } from "../src";

describe("utils", () => {
  it("ensureEnv() should ensure env variables configured", () => {
    expect(() => {
      ensureEnv("FOO");
    }).toThrowError();
  });

  it("singleton() should provide singleton object", () => {
    type Foo = {
      foo?: string;
    };

    const foo = singleton<Foo>("foo", () => {
      return {};
    });
    const g = global as unknown as { __singletons: Record<string, unknown> };
    const singletons: any = g.__singletons;

    foo.foo = "hello world";
    expect(singletons).toBeDefined();
    expect(singletons["foo"]).toBeDefined();
    expect(foo.foo).toBe("hello world");
  });
});
