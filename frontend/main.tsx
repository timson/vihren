import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import "@mantine/core/styles.css";
import "bootstrap-icons/font/bootstrap-icons.css";
import "d3-flame-graph/dist/d3-flamegraph.css";
import "./styles.css";
import App from "./app";

const theme = {
  fontFamily: "Sora, Segoe UI, sans-serif",
  colors: {
    brand: [
      "#eef2ff",
      "#d9e0ff",
      "#b3c0ff",
      "#8da0ff",
      "#677fff",
      "#4e66f5",
      "#3e53d6",
      "#2f40b3",
      "#222f8a",
      "#192369"
    ] as const
  },
  primaryColor: "brand"
};

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider theme={theme}>
      <App />
    </MantineProvider>
  </React.StrictMode>
);
