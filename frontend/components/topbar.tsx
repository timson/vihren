import {
  Button,
  Checkbox,
  Menu,
  Popover,
  Select,
  TextInput
} from "@mantine/core";
import { useEffect, useMemo, useState } from "react";
import TimePicker from "./timepicker";
import SearchBar from "./searchbar";
import type { TimeRangeOption } from "../types";

type FilterGroup = {
  group: string;
  lookupFor: string;
  items: { value: string; label: string }[];
};

export type TopBarProps = {
  loadingServices: boolean;
  loadingGraph: boolean;
  loadingFilters: boolean;
  serviceOptions: { value: string; label: string }[];
  filterOptions: FilterGroup[];
  selectedService: string;
  selectedFilters: string[];
  onFiltersChange: (value: string[]) => void;
  onResetFilters: () => void;
  searchValue: string;
  onSearchChange: (value: string) => void;
  onSearchReset: () => void;
  onSearchSubmit: (value: string) => void;
  graphView: "samples" | "cpu" | "memory";
  onGraphViewChange: (value: "samples" | "cpu" | "memory") => void;
  rootFrameOptions: { value: string; label: string }[];
  selectedRootFrame: string;
  onRootFrameChange: (value: string) => void;
  onServiceChange: (value: string) => void;
  onRefreshServices: () => void;
  onRefreshFlamegraph: () => void;
  timeRanges: TimeRangeOption[];
  timeRange: string;
  timePickerOpen: boolean;
  onToggleTimePicker: () => void;
  onCloseTimePicker: () => void;
  onTimeRangeChange: (value: string) => void;
  onApplyTimeSelection: (
    nextStart?: string,
    nextEnd?: string,
    nextRange?: string
  ) => void;
  startTime: string;
  endTime: string;
  onStartTimeChange: (value: string) => void;
  onEndTimeChange: (value: string) => void;
};

function TopBar({
  loadingServices,
  loadingGraph,
  loadingFilters,
  serviceOptions,
  selectedService,
  filterOptions,
  selectedFilters,
  onFiltersChange,
  onResetFilters,
  searchValue,
  onSearchChange,
  onSearchReset,
  onSearchSubmit,
  graphView,
  onGraphViewChange,
  rootFrameOptions,
  selectedRootFrame,
  onRootFrameChange,
  onServiceChange,
  onRefreshServices,
  onRefreshFlamegraph,
  timeRanges,
  timeRange,
  timePickerOpen,
  onToggleTimePicker,
  onCloseTimePicker,
  onTimeRangeChange,
  onApplyTimeSelection,
  startTime,
  endTime,
  onStartTimeChange,
  onEndTimeChange
}: TopBarProps) {
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [activeEntity, setActiveEntity] = useState("");
  const [filterSearch, setFilterSearch] = useState("");
  const [draftFilters, setDraftFilters] = useState<string[]>([]);

  useEffect(() => {
    if (filtersOpen) {
      setDraftFilters(selectedFilters);
    }
  }, [filtersOpen, selectedFilters]);

  useEffect(() => {
    if (!filtersOpen) {
      setDraftFilters(selectedFilters);
    }
  }, [selectedFilters, filtersOpen]);

  useEffect(() => {
    if (!filterOptions.length) {
      setActiveEntity("");
      return;
    }
    const isActiveValid = filterOptions.some(
      (group) => group.lookupFor === activeEntity
    );
    if (!isActiveValid) {
      setActiveEntity(filterOptions[0].lookupFor);
    }
  }, [filterOptions, activeEntity]);

  const activeGroup = useMemo(
    () => filterOptions.find((group) => group.lookupFor === activeEntity),
    [filterOptions, activeEntity]
  );

  const filteredItems = useMemo(() => {
    if (!activeGroup) {
      return [];
    }
    const query = filterSearch.trim().toLowerCase();
    if (!query) {
      return activeGroup.items;
    }
    return activeGroup.items.filter((item) =>
      item.label.toLowerCase().includes(query)
    );
  }, [activeGroup, filterSearch]);

  const toggleFilter = (value: string, checked: boolean) => {
    if (checked) {
      setDraftFilters((prev) => [...prev, value]);
      return;
    }
    setDraftFilters((prev) => prev.filter((item) => item !== value));
  };

  const hasFilterChanges = useMemo(() => {
    if (draftFilters.length !== selectedFilters.length) {
      return true;
    }
    const draftSet = new Set(draftFilters);
    return selectedFilters.some((value) => !draftSet.has(value));
  }, [draftFilters, selectedFilters]);

  const filtersLabel =
    selectedFilters.length > 0 ? `Filters (${selectedFilters.length})` : "Filters";

  return (
    <div className="topbar">
      <div className="topbar-row">
        <div className="topbar-group">
          <Select
            placeholder={
              loadingServices ? "Loading services..." : "Pick a service"
            }
            data={serviceOptions}
            value={selectedService}
            searchable
            onChange={(value) => onServiceChange(value || "")}
            disabled={loadingServices || serviceOptions.length === 0}
            w={320}
          />
          <Popover
            opened={filtersOpen}
            onChange={setFiltersOpen}
            position="bottom-start"
            shadow="md"
            width={520}
          >
            <Popover.Target>
              <Button
                variant="light"
                className="icon-action ghost-button"
                onClick={() => setFiltersOpen((prev) => !prev)}
                disabled={
                  loadingFilters || filterOptions.length === 0 || !selectedService
                }
              >
                <i className="bi bi-funnel" aria-hidden="true"></i>
              </Button>
            </Popover.Target>
            <Popover.Dropdown>
              <div className="filter-panel">
                <div className="filter-entities">
                  {filterOptions.map((group) => (
                    <button
                      key={group.lookupFor}
                      type="button"
                      className={`filter-entity ${
                        group.lookupFor === activeEntity ? "active" : ""
                      }`}
                      onClick={() => {
                        setActiveEntity(group.lookupFor);
                        setFilterSearch("");
                      }}
                    >
                      {group.group}
                    </button>
                  ))}
                </div>
                <div className="filter-values">
                  <TextInput
                    placeholder="Search values"
                    value={filterSearch}
                    onChange={(event) => setFilterSearch(event.currentTarget.value)}
                    className="filter-search"
                  />
                  <div className="filter-list">
                    {filteredItems.map((item) => {
                      const checked = draftFilters.includes(item.value);
                      return (
                        <label key={item.value} className="filter-item">
                          <Checkbox
                            checked={checked}
                            onChange={(event) =>
                              toggleFilter(item.value, event.currentTarget.checked)
                            }
                          />
                          <span>{item.label}</span>
                        </label>
                      );
                    })}
                  </div>
                  <div className="filter-actions">
                    <button
                      type="button"
                      className="link-button filter-reset"
                      onClick={() => setDraftFilters([])}
                      disabled={draftFilters.length === 0}
                    >
                      Reset selection
                    </button>
                    <Button
                      size="xs"
                      className="filter-apply"
                      onClick={() => {
                        onFiltersChange(draftFilters);
                        setFiltersOpen(false);
                      }}
                      disabled={!hasFilterChanges}
                    >
                      Apply
                    </Button>
                  </div>
                </div>
              </div>
            </Popover.Dropdown>
          </Popover>
          <Button
            variant="light"
            className="icon-action ghost-button"
            onClick={onRefreshServices}
            disabled={loadingServices}
          >
            <i className="bi bi-arrow-repeat" aria-hidden="true"></i>
          </Button>
          <span className="topbar-divider" aria-hidden="true"></span>
          <SearchBar
            value={searchValue}
            onChange={onSearchChange}
            onReset={onSearchReset}
            onSubmit={onSearchSubmit}
          />
          <Select
            placeholder="All apps"
            data={rootFrameOptions}
            value={selectedRootFrame}
            searchable
            onChange={(value) => onRootFrameChange(value || "")}
            disabled={rootFrameOptions.length === 0}
            w={220}
          />
          <Menu position="bottom-end" width={200} shadow="md">
            <Menu.Target>
              <Button
                variant="light"
                className="icon-action ghost-button"
                aria-label="Select graph view"
              >
                <i className="bi bi-graph-up" aria-hidden="true"></i>
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                onClick={() => onGraphViewChange("samples")}
                leftSection={
                  graphView === "samples" ? (
                    <i className="bi bi-check2" aria-hidden="true"></i>
                  ) : null
                }
              >
                Samples
              </Menu.Item>
              <Menu.Item
                onClick={() => onGraphViewChange("cpu")}
                leftSection={
                  graphView === "cpu" ? (
                    <i className="bi bi-check2" aria-hidden="true"></i>
                  ) : null
                }
              >
                CPU Utilization
              </Menu.Item>
              <Menu.Item
                onClick={() => onGraphViewChange("memory")}
                leftSection={
                  graphView === "memory" ? (
                    <i className="bi bi-check2" aria-hidden="true"></i>
                  ) : null
                }
              >
                Memory Utilization
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </div>
        <div className="topbar-right">
          <TimePicker
            timeRanges={timeRanges}
            timeRange={timeRange}
            open={timePickerOpen}
            onToggle={onToggleTimePicker}
            onClose={onCloseTimePicker}
            onTimeRangeChange={onTimeRangeChange}
            onApply={onApplyTimeSelection}
            startTime={startTime}
            endTime={endTime}
            onStartTimeChange={onStartTimeChange}
            onEndTimeChange={onEndTimeChange}
          />
          <Button
            className="icon-action"
            onClick={onRefreshFlamegraph}
            disabled={!selectedService || loadingGraph}
          >
            <i className="bi bi-arrow-clockwise" aria-hidden="true"></i>
          </Button>
        </div>
      </div>
    </div>
  );
}

export default TopBar;
