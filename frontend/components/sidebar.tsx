import { useState } from "react";
import { Modal, Text } from "@mantine/core";
import flameIcon from "../assets/flame.png";

function Sidebar() {
  const [aboutOpen, setAboutOpen] = useState(false);

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
          aria-label="About"
          title="About"
          onClick={() => setAboutOpen(true)}
        >
          <i className="bi bi-question-circle" aria-hidden="true"></i>
        </button>
      </div>

      <Modal
        opened={aboutOpen}
        onClose={() => setAboutOpen(false)}
        title="Vihren"
        centered
        size="sm"
      >
        <div className="about-dialog">
          <img src={flameIcon} alt="Vihren" className="about-logo" />
          <Text size="lg" fw={600}>Vihren</Text>
          <Text size="sm" c="dimmed">Continuous Profiling UI</Text>
          <Text size="sm" mt="md">
            Collect, store and visualize CPU flamegraphs from Intel gProfiler.
            Powered by embedded ClickHouse.
          </Text>
          <Text size="xs" c="dimmed" mt="md">
            github.com/timson/vihren
          </Text>
        </div>
      </Modal>
    </aside>
  );
}

export default Sidebar;
