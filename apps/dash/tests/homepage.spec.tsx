import { describe, expect, test } from "vitest";
import { render, screen } from "@testing-library/react";
import Home from "@/page";

describe("Homepage", () => {
  test("Homepage", () => {
    render(<Home />);
    expect(
      screen.getByRole("heading", { level: 1, name: "Home" }),
    ).toBeDefined();
  });
});
