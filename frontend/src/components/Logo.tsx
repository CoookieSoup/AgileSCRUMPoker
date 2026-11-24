import { Link } from "react-router-dom";

export function LogoMark({ size = 28 }: { size?: number }) {
  return <img src="/logo.svg" width={size} height={size} alt="" aria-hidden="true" />;
}

export function Logo({ title = "Zenith Poker" }: { title?: string }) {
  return (
    <Link to="/" className="logo">
      <LogoMark />
      <span>{title}</span>
    </Link>
  );
}
