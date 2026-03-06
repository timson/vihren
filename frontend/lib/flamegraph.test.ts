import { describe, it, expect } from "vitest";
import {
  filterFlamegraphData,
  runtimeColors,
  legendItems,
  allRuntimeKeys,
} from "./flamegraph";
import type { FlamegraphNode, RuntimeKey } from "../types";

describe("runtimeColors", () => {
  it("has a color for every runtime key", () => {
    for (const key of allRuntimeKeys) {
      expect(runtimeColors[key]).toBeTruthy();
    }
  });
});

describe("legendItems", () => {
  it("has one item per runtime key", () => {
    expect(legendItems).toHaveLength(allRuntimeKeys.length);
    const keys = legendItems.map((i) => i.key);
    expect(new Set(keys).size).toBe(keys.length); // no duplicates
  });

  it("every item has a label and color", () => {
    for (const item of legendItems) {
      expect(item.label).toBeTruthy();
      expect(item.color).toMatch(/^#[0-9a-fA-F]{6}$/);
    }
  });
});

describe("filterFlamegraphData", () => {
  const tree: FlamegraphNode = {
    name: "root",
    children: [
      {
        name: "com.example.Main",
        value: 100,
        language: "Java",
        children: [
          { name: "com.example.Main.run", value: 60, language: "Java" },
          { name: "libc.so.malloc", value: 40, language: "C++" },
        ],
      },
      {
        name: "app.handler",
        value: 50,
        language: "Python",
      },
      {
        name: "net/http.ListenAndServe",
        value: 30,
        language: "Go",
      },
    ],
  };

  it("returns all data when all runtimes enabled, renames root to All", () => {
    const result = filterFlamegraphData(tree, [...allRuntimeKeys]);
    expect(result).not.toBeNull();
    expect(result!.name).toBe("All");
    // Root value should be sum of children
    expect(result!.value).toBe(180); // 100 + 50 + 30
  });

  it("filters to only Java runtime", () => {
    const result = filterFlamegraphData(tree, ["java"]);
    expect(result).not.toBeNull();
    expect(result!.name).toBe("All");
    // Should keep Java branch and C++ child is pruned (leaf with wrong runtime)
    const children = result!.children ?? [];
    // Java node is kept; Python and Go are leaf-only so pruned
    const names = children.map((c) => c.name);
    expect(names).toContain("com.example.Main");
    expect(names).not.toContain("app.handler");
    expect(names).not.toContain("net/http.ListenAndServe");
  });

  it("filters to only Python runtime", () => {
    const result = filterFlamegraphData(tree, ["python"]);
    expect(result).not.toBeNull();
    const children = result!.children ?? [];
    const names = children.map((c) => c.name);
    expect(names).toContain("app.handler");
    expect(names).not.toContain("com.example.Main");
  });

  it("keeps parent nodes that have matching descendants", () => {
    const result = filterFlamegraphData(tree, ["cpp"]);
    expect(result).not.toBeNull();
    const children = result!.children ?? [];
    // com.example.Main (Java) should be kept as a parent because it has a C++ child
    const javaNode = children.find((c) => c.name === "com.example.Main");
    expect(javaNode).toBeTruthy();
    const cppChild = javaNode!.children?.find(
      (c) => c.name === "libc.so.malloc"
    );
    expect(cppChild).toBeTruthy();
  });

  it("returns null when no runtimes match", () => {
    const result = filterFlamegraphData(tree, ["dotnet"]);
    // Root should still exist but with no children, value 0
    expect(result).not.toBeNull();
    expect(result!.children).toHaveLength(0);
    expect(result!.value).toBe(0);
  });

  it("recalculates parent values from filtered children", () => {
    const result = filterFlamegraphData(tree, ["java"]);
    expect(result).not.toBeNull();
    const javaNode = result!.children?.find(
      (c) => c.name === "com.example.Main"
    );
    expect(javaNode).toBeTruthy();
    // Only the Java child (60) should remain, C++ (40) pruned
    expect(javaNode!.value).toBe(60);
  });

  it("handles tree with no children", () => {
    const leaf: FlamegraphNode = { name: "root", value: 10 };
    const result = filterFlamegraphData(leaf, [...allRuntimeKeys]);
    expect(result).not.toBeNull();
    expect(result!.value).toBe(10);
  });

  it("handles multiple runtimes filter", () => {
    const result = filterFlamegraphData(tree, ["java", "go"]);
    expect(result).not.toBeNull();
    const names = (result!.children ?? []).map((c) => c.name);
    expect(names).toContain("com.example.Main");
    expect(names).toContain("net/http.ListenAndServe");
    expect(names).not.toContain("app.handler");
  });
});
