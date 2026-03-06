import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import RuntimeLegend from "./runtimelegend";
import type { LegendItem, RuntimeKey } from "../types";

const items: LegendItem[] = [
  { key: "java", label: "Java", color: "#ff8b5c" },
  { key: "python", label: "Python", color: "#f5c34d" },
  { key: "go", label: "Go", color: "#61d2f2" },
];

describe("RuntimeLegend", () => {
  it("renders a button per item", () => {
    render(
      <RuntimeLegend items={items} enabled={["java", "python", "go"]} onToggle={vi.fn()} />
    );
    expect(screen.getByText("Java")).toBeDefined();
    expect(screen.getByText("Python")).toBeDefined();
    expect(screen.getByText("Go")).toBeDefined();
  });

  it("marks enabled items as active", () => {
    render(
      <RuntimeLegend items={items} enabled={["java"]} onToggle={vi.fn()} />
    );
    const javaBtn = screen.getByText("Java").closest("button")!;
    const pythonBtn = screen.getByText("Python").closest("button")!;
    expect(javaBtn.className).toContain("active");
    expect(pythonBtn.className).toContain("inactive");
  });

  it("sets aria-pressed correctly", () => {
    render(
      <RuntimeLegend items={items} enabled={["go"]} onToggle={vi.fn()} />
    );
    const goBtn = screen.getByText("Go").closest("button")!;
    const javaBtn = screen.getByText("Java").closest("button")!;
    expect(goBtn.getAttribute("aria-pressed")).toBe("true");
    expect(javaBtn.getAttribute("aria-pressed")).toBe("false");
  });

  it("calls onToggle with the correct key when clicked", () => {
    const onToggle = vi.fn();
    render(
      <RuntimeLegend items={items} enabled={["java", "python", "go"]} onToggle={onToggle} />
    );
    fireEvent.click(screen.getByText("Python").closest("button")!);
    expect(onToggle).toHaveBeenCalledWith("python");
  });

  it("renders colored dots", () => {
    const { container } = render(
      <RuntimeLegend items={items} enabled={["java"]} onToggle={vi.fn()} />
    );
    const dots = container.querySelectorAll(".legend-dot");
    expect(dots).toHaveLength(3);
    expect((dots[0] as HTMLElement).style.background).toBe("rgb(255, 139, 92)"); // #ff8b5c
  });
});
