import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  formatDateTimeLocal,
  parseRangeToMs,
  getInitialTimeState,
  syncTimeToUrl,
  formatForApi,
  timeRanges,
} from "./time";

describe("parseRangeToMs", () => {
  it("parses minutes", () => {
    expect(parseRangeToMs("15m")).toBe(15 * 60 * 1000);
    expect(parseRangeToMs("30m")).toBe(30 * 60 * 1000);
  });

  it("parses hours", () => {
    expect(parseRangeToMs("1h")).toBe(60 * 60 * 1000);
    expect(parseRangeToMs("3h")).toBe(3 * 60 * 60 * 1000);
  });

  it("parses days", () => {
    expect(parseRangeToMs("1d")).toBe(24 * 60 * 60 * 1000);
    expect(parseRangeToMs("7d")).toBe(7 * 24 * 60 * 60 * 1000);
  });

  it("returns 1h default for invalid input", () => {
    expect(parseRangeToMs("invalid")).toBe(60 * 60 * 1000);
    expect(parseRangeToMs("")).toBe(60 * 60 * 1000);
  });
});

describe("formatDateTimeLocal", () => {
  it("formats a date as YYYY-MM-DDTHH:MM", () => {
    const date = new Date(2026, 2, 5, 14, 30); // March 5, 2026 14:30
    expect(formatDateTimeLocal(date)).toBe("2026-03-05T14:30");
  });

  it("pads single-digit values", () => {
    const date = new Date(2026, 0, 3, 5, 7); // Jan 3, 2026 05:07
    expect(formatDateTimeLocal(date)).toBe("2026-01-03T05:07");
  });
});

describe("formatForApi", () => {
  it("formats a date in UTC", () => {
    // Create a known UTC time
    const date = new Date(Date.UTC(2026, 2, 5, 14, 30, 45));
    const input = formatDateTimeLocal(date);
    // formatForApi parses the local string and outputs UTC
    const result = formatForApi(input);
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$/);
  });

  it("returns empty string for empty input", () => {
    expect(formatForApi("")).toBe("");
  });
});

describe("getInitialTimeState", () => {
  const rangeValues = timeRanges.map((r) => r.value);

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 2, 5, 12, 0, 0)); // March 5, 2026 12:00
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("defaults to 1h relative range when no URL params", () => {
    // No URL params
    const state = getInitialTimeState(rangeValues);
    expect(state.timeRange).toBe("1h");
    expect(state.start).toBe("2026-03-05T11:00");
    expect(state.end).toBe("2026-03-05T12:00");
  });

  it("uses range param from URL when valid", () => {
    const url = new URL("http://localhost?range=3h");
    vi.stubGlobal("location", url);

    const state = getInitialTimeState(rangeValues);
    expect(state.timeRange).toBe("3h");
    expect(state.start).toBe("2026-03-05T09:00");
    expect(state.end).toBe("2026-03-05T12:00");

    vi.unstubAllGlobals();
  });

  it("restores absolute range from URL params", () => {
    const url = new URL(
      "http://localhost?start=2026-03-04T10:00&end=2026-03-04T18:00"
    );
    vi.stubGlobal("location", url);

    const state = getInitialTimeState(rangeValues);
    expect(state.timeRange).toBe("custom");
    expect(state.start).toBe("2026-03-04T10:00");
    expect(state.end).toBe("2026-03-04T18:00");

    vi.unstubAllGlobals();
  });

  it("recomputes relative range from current time (sliding window)", () => {
    // First call at 12:00
    const state1 = getInitialTimeState(rangeValues);
    expect(state1.end).toBe("2026-03-05T12:00");

    // Advance time by 30 minutes
    vi.setSystemTime(new Date(2026, 2, 5, 12, 30, 0));
    const state2 = getInitialTimeState(rangeValues);

    // The window should have slid forward
    expect(state2.end).toBe("2026-03-05T12:30");
    expect(state2.start).toBe("2026-03-05T11:30");

    // Key assertion: the two calls produce DIFFERENT time windows
    expect(state2.end).not.toBe(state1.end);
    expect(state2.start).not.toBe(state1.start);
  });
});

describe("syncTimeToUrl", () => {
  it("sets range param for relative ranges", () => {
    syncTimeToUrl("1h", "2026-03-05T11:00", "2026-03-05T12:00");
    const params = new URLSearchParams(window.location.search);
    expect(params.get("range")).toBe("1h");
    expect(params.has("start")).toBe(false);
    expect(params.has("end")).toBe(false);
  });

  it("sets start/end params for custom ranges", () => {
    syncTimeToUrl("custom", "2026-03-04T10:00", "2026-03-04T18:00");
    const params = new URLSearchParams(window.location.search);
    expect(params.has("range")).toBe(false);
    expect(params.get("start")).toBe("2026-03-04T10:00");
    expect(params.get("end")).toBe("2026-03-04T18:00");
  });
});
