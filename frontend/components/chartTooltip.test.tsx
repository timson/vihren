import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import ChartTooltip from "./chartTooltip";

describe("ChartTooltip", () => {
  it("renders all lines", () => {
    render(<ChartTooltip x={100} y={200} lines={["Title", "Detail 1", "Detail 2"]} />);
    expect(screen.getByText("Title")).toBeDefined();
    expect(screen.getByText("Detail 1")).toBeDefined();
    expect(screen.getByText("Detail 2")).toBeDefined();
  });

  it("applies correct class names to lines", () => {
    render(<ChartTooltip x={0} y={0} lines={["First", "Second", "Third"]} />);
    expect(screen.getByText("First").className).toBe("chart-tooltip-value");
    expect(screen.getByText("Second").className).toBe("chart-tooltip-time");
    expect(screen.getByText("Third").className).toBe("chart-tooltip-time");
  });

  it("positions with left and top style", () => {
    const { container } = render(
      <ChartTooltip x={150} y={250} lines={["Line"]} />
    );
    const tooltip = container.firstElementChild as HTMLElement;
    expect(tooltip.style.left).toBe("150px");
    expect(tooltip.style.top).toBe("250px");
  });

  it("renders empty when lines is empty", () => {
    const { container } = render(<ChartTooltip x={0} y={0} lines={[]} />);
    const tooltip = container.firstElementChild as HTMLElement;
    expect(tooltip.children).toHaveLength(0);
  });
});
