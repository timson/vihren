import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import TimePicker from "./timepicker";
import { timeRanges } from "../lib/time";

const defaultProps = () => ({
  timeRanges,
  timeRange: "1h",
  open: false,
  onToggle: vi.fn(),
  onClose: vi.fn(),
  onTimeRangeChange: vi.fn(),
  onApply: vi.fn(),
  startTime: "2026-03-05T11:00",
  endTime: "2026-03-05T12:00",
  onStartTimeChange: vi.fn(),
  onEndTimeChange: vi.fn(),
});

describe("TimePicker", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 2, 5, 12, 0, 0));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows relative label when a preset is selected", () => {
    render(<TimePicker {...defaultProps()} />);
    expect(screen.getByText("Last 1 hour")).toBeDefined();
  });

  it("shows custom label when timeRange is custom", () => {
    render(
      <TimePicker
        {...defaultProps()}
        timeRange="custom"
        startTime="2026-03-04T10:00"
        endTime="2026-03-04T18:00"
      />
    );
    expect(screen.getByText("2026-03-04 10:00 – 2026-03-04 18:00")).toBeDefined();
  });

  it("does not show panel when closed", () => {
    render(<TimePicker {...defaultProps()} />);
    expect(screen.queryByText("Relative time")).toBeNull();
  });

  it("shows panel with range pills when open", () => {
    render(<TimePicker {...defaultProps()} open={true} />);
    expect(screen.getByText("Relative time")).toBeDefined();
    expect(screen.getByText("15 min")).toBeDefined();
    expect(screen.getByText("7d")).toBeDefined();
  });

  it("calls onToggle when button clicked", () => {
    const props = defaultProps();
    render(<TimePicker {...props} />);
    fireEvent.click(screen.getByText("Last 1 hour"));
    expect(props.onToggle).toHaveBeenCalled();
  });

  it("clicking a range pill calls onApply with computed times and closes", () => {
    const props = defaultProps();
    render(<TimePicker {...props} open={true} />);
    fireEvent.click(screen.getByText("3 hour"));
    expect(props.onTimeRangeChange).toHaveBeenCalledWith("3h");
    expect(props.onApply).toHaveBeenCalledWith(
      "2026-03-05T09:00",
      "2026-03-05T12:00",
      "3h"
    );
    expect(props.onClose).toHaveBeenCalled();
  });

  it("marks the active range pill", () => {
    render(<TimePicker {...defaultProps()} open={true} />);
    const pill = screen.getByText("1 hour");
    expect(pill.className).toContain("active");
    const otherPill = screen.getByText("3 hour");
    expect(otherPill.className).not.toContain("active");
  });

  it("clicking Apply calls onApply with no args and closes", () => {
    const props = defaultProps();
    render(<TimePicker {...props} open={true} />);
    fireEvent.click(screen.getByText("Apply"));
    expect(props.onApply).toHaveBeenCalledWith();
    expect(props.onClose).toHaveBeenCalled();
  });

  it("clicking close button calls onClose", () => {
    const props = defaultProps();
    render(<TimePicker {...props} open={true} />);
    fireEvent.click(screen.getByText("×"));
    expect(props.onClose).toHaveBeenCalled();
  });
});
