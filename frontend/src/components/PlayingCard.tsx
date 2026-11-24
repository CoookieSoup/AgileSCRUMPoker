import type { CSSProperties } from "react";
import { LogoMark } from "./Logo";

interface PlayingCardProps {
  /** Card value; omitted shows the card back. */
  value?: string;
  glow?: boolean;
  className?: string;
  style?: CSSProperties;
}

export function PlayingCard({ value, glow, className = "", style }: PlayingCardProps) {
  return (
    <div className={`playing-card ${glow ? "glow" : ""} ${className}`} style={style}>
      {value === undefined ? (
        <div className="playing-card-back">
          <LogoMark size={30} />
          <strong>Zenith</strong>
          <small>POKER</small>
        </div>
      ) : (
        <span className="playing-card-value">{value}</span>
      )}
    </div>
  );
}
