import * as d3 from "d3";
import { flamegraph as flamegraphFactory } from "d3-flame-graph";
import type { FlamegraphNode, FlamegraphResponse, LegendItem, RuntimeKey } from "../types";

export const runtimeColors = {
  appid: "#8893E5",
  java: "#ff8b5c",
  python: "#f5c34d",
  php: "#b973ff",
  ruby: "#ef6b6b",
  node: "#34c38f",
  dotnet: "#4aa3ff",
  cpp: "#65B6F2",
  kernel: "#f29aca",
  go: "#61d2f2",
  rust: "#dea584",
  other: "#B375FD"
} as const;

export const legendItems: LegendItem[] = [
  { key: "appid", label: "Appid", color: runtimeColors.appid },
  { key: "java", label: "Java", color: runtimeColors.java },
  { key: "python", label: "Python", color: runtimeColors.python },
  { key: "php", label: "PHP", color: runtimeColors.php },
  { key: "ruby", label: "Ruby", color: runtimeColors.ruby },
  { key: "node", label: "Node", color: runtimeColors.node },
  { key: "dotnet", label: ".NET", color: runtimeColors.dotnet },
  { key: "cpp", label: "C++", color: runtimeColors.cpp },
  { key: "kernel", label: "Kernel", color: runtimeColors.kernel },
  { key: "go", label: "Go", color: runtimeColors.go },
  { key: "rust", label: "Rust", color: runtimeColors.rust },
  { key: "other", label: "Other", color: runtimeColors.other }
];

export const allRuntimeKeys = legendItems.map((item) => item.key);

let currentFlamegraph: ReturnType<typeof flamegraphFactory> | null = null;
let currentSearchTerm = "";
let currentData: FlamegraphResponse | null = null;
let contextMenuVisible = false;
let lastMousePosition: { x: number; y: number } | null = null;
let hasMouseTracker = false;

const runtimeColorMap: Record<string, string> = {
  appid: runtimeColors.appid,
  java: runtimeColors.java,
  "java (inl)": runtimeColors.java,
  "java (c1)": runtimeColors.java,
  "java (interpreted)": runtimeColors.java,
  python: runtimeColors.python,
  php: runtimeColors.php,
  ruby: runtimeColors.ruby,
  node: runtimeColors.node,
  ".net": runtimeColors.dotnet,
  "c++": runtimeColors.cpp,
  kernel: runtimeColors.kernel,
  go: runtimeColors.go,
  rust: runtimeColors.rust,
  other: runtimeColors.other
};

const runtimeKeyMap: Record<string, RuntimeKey> = {
  appid: "appid",
  java: "java",
  "java (inl)": "java",
  "java (c1)": "java",
  "java (interpreted)": "java",
  python: "python",
  php: "php",
  ruby: "ruby",
  node: "node",
  ".net": "dotnet",
  "c++": "cpp",
  kernel: "kernel",
  go: "go",
  rust: "rust",
  other: "other"
};

const formatPercent = (value: number) => {
  if (!Number.isFinite(value)) {
    return "0.0%";
  }
  return `${value.toFixed(1)}%`;
};

const getRuntimeInfo = (node: any) => {
  const rawLanguage =
    node?.data?.language ??
    node?.language ??
    node?.data?.Language ??
    node?.Language;
  const label = rawLanguage ? String(rawLanguage) : "Other";
  const key = rawLanguage ? String(rawLanguage).toLowerCase() : "other";
  return {
    label,
    color: runtimeColorMap[key] ?? runtimeColors.other
  };
};

const getRuntimeKey = (node: FlamegraphNode) => {
  if (!node.language) {
    return null;
  }
  const key = String(node.language).toLowerCase();
  return runtimeKeyMap[key] ?? "other";
};

export const filterFlamegraphData = (
  data: FlamegraphResponse,
  enabledRuntimes: RuntimeKey[]
): FlamegraphResponse | null => {
  const normalizeRoot = (node: FlamegraphResponse) => {
    const children = node.children ?? [];
    if (children.length === 0) {
      return node;
    }
    const value = children.reduce((sum, child) => sum + (child.value ?? 0), 0);
    return { ...node, value };
  };
  if (enabledRuntimes.length === allRuntimeKeys.length) {
    const nodeName = data.name?.toLowerCase?.() ?? "";
    const normalized = normalizeRoot(data);
    return nodeName === "root" ? { ...normalized, name: "All" } : normalized;
  }
  const enabled = new Set(enabledRuntimes);
  const filterNode = (node: FlamegraphNode): FlamegraphNode | null => {
    const nodeName = node.name?.toLowerCase?.() ?? "";
    const isRoot = nodeName === "all" || nodeName === "root";
    const runtimeKey = getRuntimeKey(node);
    const children = node.children
      ?.map(filterNode)
      .filter((child): child is FlamegraphNode => Boolean(child));
    const hasChildren = Boolean(children && children.length > 0);
    const childrenValue = children?.reduce(
      (sum, child) => sum + (child.value ?? 0),
      0
    );
    const nodeValue =
      hasChildren && typeof childrenValue === "number"
        ? childrenValue
        : node.value ?? 0;
    if (isRoot) {
      return {
        ...node,
        name: node.name === "root" ? "All" : node.name,
        children,
        value: nodeValue
      };
    }
    if (runtimeKey && enabled.has(runtimeKey)) {
      return { ...node, children, value: nodeValue };
    }
    if (hasChildren) {
      return { ...node, children, value: nodeValue };
    }
    return null;
  };

  const filtered = filterNode(data);
  if (!filtered) {
    return null;
  }
  const normalized = normalizeRoot(filtered);
  return normalized;
};

const cloneFlamegraphData = <T extends FlamegraphNode>(node: T): T => {
  const { children, ...rest } = node;
  return {
    ...(rest as Omit<T, "children">),
    children: children ? children.map((child) => cloneFlamegraphData(child)) : undefined
  } as T;
};

export const renderFlamegraph = (data: FlamegraphResponse) => {
  const container = document.getElementById("flamegraph");
  if (!container) {
    return;
  }
  container.innerHTML = "";
  const width = Math.max(container.clientWidth, 320);
  const adjustedData =
    data?.name?.toLowerCase() === "root" ? { ...data, name: "All" } : data;
  const sanitizedData = cloneFlamegraphData(adjustedData);
  currentData = sanitizedData;
  const flamegraph = flamegraphFactory()
    .width(width)
    .cellHeight(18)
    .minFrameSize(3)
    .inverted(true)
    .transitionDuration(300)
    .tooltip(createFlamegraphTooltip(container) as unknown as boolean);

  flamegraph.setColorMapper((node: any) => {
    const searchTerm = currentSearchTerm.trim().toLowerCase();
    const nodeName = String(node?.data?.name ?? node?.name ?? "").toLowerCase();
    if (searchTerm && nodeName !== "all" && nodeName !== "root") {
      const matches = nodeName.includes(searchTerm);
      if (!matches) {
        return "#d7dbe7";
      }
    }
    if (nodeName === "all" || nodeName === "root") {
      return "#373A4B";
    }
    const parentName = String(
      node?.parent?.data?.name ?? node?.parent?.name ?? ""
    ).toLowerCase();
    if (parentName === "all" || parentName === "root") {
      return runtimeColors.appid;
    }
    const rawLanguage =
      node?.data?.language ??
      node?.language ??
      node?.data?.Language ??
      node?.Language;
    if (!rawLanguage) {
      return runtimeColors.other;
    }
    const key = String(rawLanguage).toLowerCase();
    return runtimeColorMap[key] ?? runtimeColors.other;
  });

  d3.select("#flamegraph").datum(sanitizedData).call(flamegraph);
  currentFlamegraph = flamegraph;
  d3.select(container).selectAll("title").remove();
  if (!hasMouseTracker) {
    container.addEventListener("mousemove", (event) => {
      lastMousePosition = { x: event.clientX, y: event.clientY };
    });
    hasMouseTracker = true;
  }
  setupContextMenu(container);
};

let contextMenuCleanup: (() => void) | null = null;

const setupContextMenu = (container: HTMLElement) => {
  if (contextMenuCleanup) {
    contextMenuCleanup();
  }

  let menu: HTMLDivElement | null = null;
  let longPressTimer: ReturnType<typeof setTimeout> | null = null;
  let pendingFrameName: string | null = null;

  const removeMenu = () => {
    if (menu) {
      menu.remove();
      menu = null;
    }
    pendingFrameName = null;
    contextMenuVisible = false;
  };

  const getFrameName = (target: EventTarget | null): string | null => {
    let el = target as HTMLElement | null;
    while (el && el !== container) {
      const datum = (d3.select(el).datum() as any);
      const name = datum?.data?.name ?? datum?.name;
      if (name && name !== "All" && name !== "root") return String(name);
      el = el.parentElement;
    }
    return null;
  };

  const showMenu = (x: number, y: number, frameName: string) => {
    removeMenu();
    contextMenuVisible = true;
    const existingTooltip = container.querySelector(".flamegraph-tooltip") as HTMLElement | null;
    if (existingTooltip) existingTooltip.style.display = "none";
    menu = document.createElement("div");
    menu.className = "flamegraph-context-menu";
    const item = document.createElement("button");
    item.textContent = "Copy frame name";
    item.addEventListener("click", () => {
      navigator.clipboard.writeText(frameName);
      removeMenu();
    });
    menu.appendChild(item);
    const rect = container.getBoundingClientRect();
    menu.style.left = `${x - rect.left}px`;
    menu.style.top = `${y - rect.top}px`;
    container.appendChild(menu);
  };

  const onContextMenu = (e: MouseEvent) => {
    const name = getFrameName(e.target);
    if (!name) return;
    e.preventDefault();
    showMenu(e.clientX, e.clientY, name);
  };

  const onTouchStart = (e: TouchEvent) => {
    const name = getFrameName(e.target);
    if (!name) return;
    pendingFrameName = name;
    const touch = e.touches[0];
    const x = touch.clientX;
    const y = touch.clientY;
    longPressTimer = setTimeout(() => {
      if (pendingFrameName) {
        e.preventDefault();
        showMenu(x, y, pendingFrameName);
      }
    }, 500);
  };

  const cancelLongPress = () => {
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    pendingFrameName = null;
  };

  const onClickOutside = (e: MouseEvent) => {
    if (menu && !menu.contains(e.target as Node)) {
      removeMenu();
    }
  };

  container.addEventListener("contextmenu", onContextMenu);
  container.addEventListener("touchstart", onTouchStart, { passive: false });
  container.addEventListener("touchend", cancelLongPress);
  container.addEventListener("touchmove", cancelLongPress);
  document.addEventListener("click", onClickOutside);

  contextMenuCleanup = () => {
    container.removeEventListener("contextmenu", onContextMenu);
    container.removeEventListener("touchstart", onTouchStart);
    container.removeEventListener("touchend", cancelLongPress);
    container.removeEventListener("touchmove", cancelLongPress);
    document.removeEventListener("click", onClickOutside);
    removeMenu();
  };
};

const createFlamegraphTooltip = (container: HTMLElement) => {
  let tooltip: HTMLDivElement | null = null;
  let showTimer: ReturnType<typeof setTimeout> | null = null;

  const ensureTooltip = () => {
    if (tooltip) {
      return;
    }
    tooltip = document.createElement("div");
    tooltip.className = "chart-tooltip flamegraph-tooltip";
    tooltip.style.display = "none";
    container.appendChild(tooltip);
  };

  const cancelShowTimer = () => {
    if (showTimer) {
      clearTimeout(showTimer);
      showTimer = null;
    }
  };

  const tip = (() => {
    ensureTooltip();
  }) as (() => void) & {
    show: (d: any) => void;
    hide: () => void;
    destroy: () => void;
  };

  tip.show = (d: any) => {
    cancelShowTimer();
    if (contextMenuVisible) return;
    ensureTooltip();
    const datum = d?.data ?? d;
    if (!datum || !tooltip) {
      return;
    }
    const value = Number(d?.value ?? datum.value ?? datum.data?.value ?? 0);
    const children = d?.children ?? datum.children ?? datum.data?.children ?? [];
    const childrenSum = Array.isArray(children)
      ? children.reduce((sum: number, child: any) => {
          const childValue = Number(
            child?.value ?? child?.data?.value ?? child?.data?.data?.value ?? 0
          );
          return sum + childValue;
        }, 0)
      : 0;
    const selfValue = Math.max(0, value - childrenSum);
    const totalSamples =
      Number(currentData?.value ?? (currentData as any)?.data?.value ?? 0) ||
      (currentData?.children
        ? currentData.children.reduce((sum, child) => sum + (child.value ?? 0), 0)
        : 0);
    const percentAll =
      totalSamples > 0 ? (value / totalSamples) * 100 : 0;
    const selfPercentAll =
      totalSamples > 0 ? (selfValue / totalSamples) * 100 : 0;
    const runtimeInfo = getRuntimeInfo(datum);
    const name = String(datum.name ?? "");
    showTimer = setTimeout(() => {
      if (!tooltip || contextMenuVisible) return;
      tooltip.innerHTML = `
        <div class="chart-tooltip-value">${name}</div>
        <div class="chart-tooltip-time">Samples: ${value} (${formatPercent(
          percentAll
        )})</div>
        <div class="chart-tooltip-time">Self: ${selfValue} (${formatPercent(
          selfPercentAll
        )})</div>
        <div class="chart-tooltip-time">
          <span class="runtime-dot" style="background:${runtimeInfo.color}"></span>
          ${runtimeInfo.label}
        </div>
      `;
      const rect = container.getBoundingClientRect();
      const clientX = lastMousePosition?.x ?? rect.left + rect.width / 2;
      const clientY = lastMousePosition?.y ?? rect.top + rect.height / 2;
      tooltip.style.left = `${clientX - rect.left}px`;
      tooltip.style.top = `${clientY - rect.top}px`;
      tooltip.style.display = "block";
    }, 1000);
  };

  tip.hide = () => {
    cancelShowTimer();
    if (tooltip) {
      tooltip.style.display = "none";
    }
  };

  tip.destroy = () => {
    if (tooltip) {
      tooltip.remove();
      tooltip = null;
    }
  };

  return tip;
};

export const applyFlamegraphSearch = (term: string) => {
  currentSearchTerm = term;
  if (!currentData) {
    return;
  }
  renderFlamegraph(currentData);
};
