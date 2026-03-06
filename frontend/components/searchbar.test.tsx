import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import SearchBar from "./searchbar";

describe("SearchBar", () => {
  it("renders input with placeholder", () => {
    render(
      <SearchBar value="" onChange={vi.fn()} onReset={vi.fn()} onSubmit={vi.fn()} />
    );
    const input = screen.getByPlaceholderText("Search for functions or frames");
    expect(input).toBeDefined();
  });

  it("calls onChange when typing", () => {
    const onChange = vi.fn();
    render(
      <SearchBar value="" onChange={onChange} onReset={vi.fn()} onSubmit={vi.fn()} />
    );
    const input = screen.getByPlaceholderText("Search for functions or frames");
    fireEvent.change(input, { target: { value: "malloc" } });
    expect(onChange).toHaveBeenCalledWith("malloc");
  });

  it("calls onSubmit on Enter key", () => {
    const onSubmit = vi.fn();
    render(
      <SearchBar value="test" onChange={vi.fn()} onReset={vi.fn()} onSubmit={onSubmit} />
    );
    const input = screen.getByPlaceholderText("Search for functions or frames");
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onSubmit).toHaveBeenCalledWith("test");
  });

  it("does not show reset button when value is empty", () => {
    render(
      <SearchBar value="" onChange={vi.fn()} onReset={vi.fn()} onSubmit={vi.fn()} />
    );
    expect(screen.queryByText("Reset")).toBeNull();
  });

  it("shows reset button when value is non-empty", () => {
    render(
      <SearchBar value="query" onChange={vi.fn()} onReset={vi.fn()} onSubmit={vi.fn()} />
    );
    expect(screen.getByText("Reset")).toBeDefined();
  });

  it("calls onReset when reset button clicked", () => {
    const onReset = vi.fn();
    render(
      <SearchBar value="query" onChange={vi.fn()} onReset={onReset} onSubmit={vi.fn()} />
    );
    fireEvent.click(screen.getByText("Reset"));
    expect(onReset).toHaveBeenCalled();
  });
});
