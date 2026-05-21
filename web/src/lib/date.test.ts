import { describe, expect, it } from "vitest";
import { safeFormatDate } from "./date";

describe("safeFormatDate", () => {
  it("formats valid timestamps", () => {
    expect(safeFormatDate("2026-05-21T05:20:52Z", "yyyy-MM-dd")).toBe("2026-05-21");
  });

  it("returns fallback for missing or invalid timestamps", () => {
    expect(safeFormatDate(null, "yyyy-MM-dd")).toBe("-");
    expect(safeFormatDate(undefined, "yyyy-MM-dd")).toBe("-");
    expect(safeFormatDate("", "yyyy-MM-dd")).toBe("-");
    expect(safeFormatDate("not-a-date", "yyyy-MM-dd")).toBe("-");
    expect(safeFormatDate(null, "yyyy-MM-dd", "从未")).toBe("从未");
  });
});
