import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ExportBar from "./exportbar";

describe("ExportBar", () => {
  it("renders three export buttons", () => {
    render(
      <ExportBar onExportPng={vi.fn()} onExportSvg={vi.fn()} onExportCollapsed={vi.fn()} disabled={false} />
    );
    expect(screen.getByText("PNG")).toBeDefined();
    expect(screen.getByText("SVG")).toBeDefined();
    expect(screen.getByText("Collapsed")).toBeDefined();
  });

  it("calls the correct callback on click", () => {
    const onPng = vi.fn();
    const onSvg = vi.fn();
    const onCollapsed = vi.fn();
    render(
      <ExportBar onExportPng={onPng} onExportSvg={onSvg} onExportCollapsed={onCollapsed} disabled={false} />
    );
    fireEvent.click(screen.getByText("PNG"));
    expect(onPng).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByText("SVG"));
    expect(onSvg).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByText("Collapsed"));
    expect(onCollapsed).toHaveBeenCalledOnce();
  });

  it("disables all buttons when disabled is true", () => {
    render(
      <ExportBar onExportPng={vi.fn()} onExportSvg={vi.fn()} onExportCollapsed={vi.fn()} disabled={true} />
    );
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(3);
    buttons.forEach((btn) => {
      expect(btn).toHaveProperty("disabled", true);
    });
  });

  it("does not fire callbacks when disabled", () => {
    const onPng = vi.fn();
    render(
      <ExportBar onExportPng={onPng} onExportSvg={vi.fn()} onExportCollapsed={vi.fn()} disabled={true} />
    );
    fireEvent.click(screen.getByText("PNG"));
    expect(onPng).not.toHaveBeenCalled();
  });
});
