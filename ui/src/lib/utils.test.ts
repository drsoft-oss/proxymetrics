import { describe, it, expect } from "vitest";
import { cn } from "./utils";

describe("cn", () => {
  it("joins class names", () => {
    expect(cn("a", "b")).toBe("a b");
  });
  it("dedupes Tailwind utilities via twMerge", () => {
    expect(cn("p-2", "p-4")).toBe("p-4");
  });
  it("filters falsy", () => {
    expect(cn("a", false && "b", undefined, "c")).toBe("a c");
  });
});
