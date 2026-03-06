type SearchBarProps = {
  value: string;
  onChange: (value: string) => void;
  onReset: () => void;
  onSubmit: (value: string) => void;
};

function SearchBar({ value, onChange, onReset, onSubmit }: SearchBarProps) {
  return (
    <div className="search-inline">
      <span className="search-icon"></span>
      <input
        className="search-input"
        placeholder="Search for functions or frames"
        type="search"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            onSubmit(value);
          }
        }}
      />
      {value ? (
        <button className="search-reset" type="button" onClick={onReset}>
          Reset
        </button>
      ) : null}
    </div>
  );
}

export default SearchBar;
