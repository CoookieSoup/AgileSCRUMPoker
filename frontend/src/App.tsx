import { BrowserRouter, Route, Routes } from "react-router-dom";
import { CreateGamePage } from "./pages/CreateGamePage";
import { GamePage } from "./pages/GamePage";
import { LandingPage } from "./pages/LandingPage";
import { NotFoundPage } from "./pages/NotFoundPage";

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/new" element={<CreateGamePage />} />
        <Route path="/game/:code" element={<GamePage />} />
        <Route path="*"element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  );
}
