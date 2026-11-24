import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./styles/theme.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main style={{ padding: 32 }}>Zenith Poker</main>
  </StrictMode>,
);
