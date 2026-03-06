export type ExportBarProps = {
  onExportPng: () => void;
  onExportSvg: () => void;
  onExportCollapsed: () => void;
  disabled: boolean;
};

function ExportBar({ onExportPng, onExportSvg, onExportCollapsed, disabled }: ExportBarProps) {
  return (
    <div className="export-bar">
      <button type="button" className="export-button" disabled={disabled} onClick={onExportPng}>
        PNG
      </button>
      <button type="button" className="export-button" disabled={disabled} onClick={onExportSvg}>
        SVG
      </button>
      <button type="button" className="export-button" disabled={disabled} onClick={onExportCollapsed}>
        Collapsed
      </button>
    </div>
  );
}

export default ExportBar;
