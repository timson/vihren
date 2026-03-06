import type { ChangeEvent } from "react";
import { formatDateTimeLocal, parseRangeToMs } from "../lib/time";
import type { TimeRangeOption } from "../types";

export type TimePickerProps = {
  timeRanges: TimeRangeOption[];
  timeRange: string;
  open: boolean;
  onToggle: () => void;
  onClose: () => void;
  onTimeRangeChange: (value: string) => void;
  onApply: (nextStart?: string, nextEnd?: string, nextRange?: string) => void;
  startTime: string;
  endTime: string;
  onStartTimeChange: (value: string) => void;
  onEndTimeChange: (value: string) => void;
};

function TimePicker({
  timeRanges,
  timeRange,
  open,
  onToggle,
  onClose,
  onTimeRangeChange,
  onApply,
  startTime,
  endTime,
  onStartTimeChange,
  onEndTimeChange
}: TimePickerProps) {
  const currentLabel =
    timeRanges.find((range) => range.value === timeRange)?.label || "";
  const formatCustomLabel = (value: string) => value.replace("T", " ");
  const customLabel =
    startTime && endTime
      ? `${formatCustomLabel(startTime)} – ${formatCustomLabel(endTime)}`
      : "Custom range";
  const buttonLabel =
    timeRange === "custom" || !currentLabel
      ? customLabel
      : `Last ${currentLabel}`;

  return (
    <div className="time-picker">
      <button type="button" className="time-picker-button" onClick={onToggle}>
        {buttonLabel}
        <span className="time-caret"></span>
      </button>
      {open ? (
        <div className="time-picker-panel">
          <div className="time-picker-header">
            <span>Relative time</span>
            <button className="time-picker-close" type="button" onClick={onClose}>
              ×
            </button>
          </div>
          <div className="time-picker-grid">
            {timeRanges.map((range) => (
              <button
                key={range.value}
                type="button"
                className={`time-pill ${timeRange === range.value ? "active" : ""}`}
                onClick={() => {
                  onTimeRangeChange(range.value);
                  const now = new Date();
                  const rangeMs = parseRangeToMs(range.value);
                  const startValue = formatDateTimeLocal(
                    new Date(now.getTime() - rangeMs)
                  );
                  const endValue = formatDateTimeLocal(now);
                  onApply(startValue, endValue, range.value);
                  onClose();
                }}
              >
                {range.label}
              </button>
            ))}
          </div>
          <div className="time-picker-divider"></div>
          <div className="time-picker-fields">
            <label className="time-field">
              <span>From</span>
              <input
                type="datetime-local"
                value={startTime}
                onChange={(event: ChangeEvent<HTMLInputElement>) =>
                  onStartTimeChange(event.target.value)
                }
              />
            </label>
            <label className="time-field">
              <span>To</span>
              <input
                type="datetime-local"
                value={endTime}
                onChange={(event: ChangeEvent<HTMLInputElement>) =>
                  onEndTimeChange(event.target.value)
                }
              />
            </label>
          </div>
          <div className="time-picker-actions">
            <button
              type="button"
              className="time-apply-button"
              onClick={() => {
                onApply();
                onClose();
              }}
            >
              Apply
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

export default TimePicker;
