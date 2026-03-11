import { useState } from "react";
import { Modal, NumberInput, Text } from "@mantine/core";
import flameIcon from "../assets/flame.png";

interface SidebarProps {
  stacksNum: number;
  onStacksNumChange: (value: number) => void;
}

function Sidebar({ stacksNum, onStacksNumChange }: SidebarProps) {
  const [aboutOpen, setAboutOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <aside className="sidebar-rail">
      <div className="rail-top">
        <button
          className="rail-logo"
          type="button"
          aria-label="Vihren"
          title="Vihren"
        >
          <img src={flameIcon} alt="" />
        </button>
      </div>
      <div className="rail-middle">
        <button
          className="rail-item active"
          type="button"
          aria-label="Flamegraph"
          title="Flamegraph"
        >
          <i className="bi bi-bar-chart-steps" aria-hidden="true"></i>
        </button>
      </div>
      <div className="rail-wordmark" aria-hidden="true">
        VIHREN
      </div>
      <div className="rail-bottom">
        <button
          className="rail-item"
          type="button"
          aria-label="Settings"
          title="Settings"
          onClick={() => setSettingsOpen(true)}
        >
          <i className="bi bi-gear" aria-hidden="true"></i>
        </button>
        <button
          className="rail-item"
          type="button"
          aria-label="About"
          title="About"
          onClick={() => setAboutOpen(true)}
        >
          <i className="bi bi-question-circle" aria-hidden="true"></i>
        </button>
      </div>

      <Modal
        opened={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        title="Settings"
        centered
        size="sm"
      >
        <NumberInput
          label="Max stacks"
          description="Number of top stack frames to fetch per query"
          value={stacksNum}
          onChange={(val) => onStacksNumChange(typeof val === "number" ? val : 10000)}
          min={1000}
          max={100000}
          step={1000}
        />
      </Modal>

      <Modal
        opened={aboutOpen}
        onClose={() => setAboutOpen(false)}
        title=""
        centered
        size="sm"
      >
        <div className="about-dialog">
          <img src={flameIcon} alt="Vihren" className="about-logo" />

          <Text size="lg" fw={600}>
            <a
              href="https://github.com/timson/vihren"
              target="_blank"
              rel="noopener noreferrer"
              style={{ color: "inherit", textDecoration: "none" }}
            >
              Vihren
            </a>
          </Text>

          <Text size="sm" c="dimmed">Continuous Profiling UI</Text>

          <Text size="sm" mt="md">
            Collect, store and visualize CPU flamegraphs from{" "}
            <a
              href="https://github.com/intel/gProfiler"
              target="_blank"
              rel="noopener noreferrer"
            >
              Intel gProfiler
            </a>
            . Powered by embedded{" "}
            <a
              href="https://github.com/ClickHouse/ClickHouse"
              target="_blank"
              rel="noopener noreferrer"
            >
              ClickHouse
            </a>
            .
          </Text>
        </div>
      </Modal>
    </aside>
  );
}

export default Sidebar;
