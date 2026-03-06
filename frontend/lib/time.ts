export const timeRanges = [
  { label: "15 min", value: "15m" },
  { label: "30 min", value: "30m" },
  { label: "1 hour", value: "1h" },
  { label: "3 hour", value: "3h" },
  { label: "12 hour", value: "12h" },
  { label: "1d", value: "1d" },
  { label: "3d", value: "3d" },
  { label: "7d", value: "7d" }
];

const pad = (value: number) => String(value).padStart(2, "0");

export const formatDateTimeLocal = (date: Date) =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(
    date.getDate()
  )}T${pad(date.getHours())}:${pad(date.getMinutes())}`;

export const parseRangeToMs = (range: string) => {
  const match = /^(\d+)(m|h|d)$/.exec(range);
  if (!match) {
    return 60 * 60 * 1000;
  }
  const value = Number(match[1]);
  const unit = match[2];
  if (unit === "m") {
    return value * 60 * 1000;
  }
  if (unit === "h") {
    return value * 60 * 60 * 1000;
  }
  return value * 24 * 60 * 60 * 1000;
};

// Returns the initial time range state, reading from URL params.
// Relative ranges (?range=1h) are always recomputed from "now" on load.
// Absolute ranges (?start=...&end=...) are restored as-is.
export const getInitialTimeState = (timeRangeValues: string[]) => {
  const params = new URLSearchParams(window.location.search);
  const rangeParam = params.get("range");
  const startParam = params.get("start");
  const endParam = params.get("end");
  const now = new Date();

  if (!rangeParam && startParam && endParam) {
    // Absolute range: restore the fixed timestamps exactly
    return { timeRange: "custom", start: startParam, end: endParam };
  }

  // Relative range: always compute fresh from now
  const rangeStr =
    rangeParam && timeRangeValues.includes(rangeParam) ? rangeParam : "1h";
  const rangeMs = parseRangeToMs(rangeStr);
  return {
    timeRange: rangeStr,
    start: formatDateTimeLocal(new Date(now.getTime() - rangeMs)),
    end: formatDateTimeLocal(now),
  };
};

// Syncs the current time selection to the URL without a full navigation.
// Call after every time range change.
export const syncTimeToUrl = (
  timeRange: string,
  start: string,
  end: string
) => {
  const url = new URL(window.location.href);
  if (timeRange === "custom") {
    url.searchParams.delete("range");
    url.searchParams.set("start", start);
    url.searchParams.set("end", end);
  } else {
    url.searchParams.set("range", timeRange);
    url.searchParams.delete("start");
    url.searchParams.delete("end");
  }
  window.history.replaceState({}, "", url.toString());
};

export const formatForApi = (value: string) => {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(
    date.getUTCDate()
  )}T${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(
    date.getUTCSeconds()
  )}`;
};
