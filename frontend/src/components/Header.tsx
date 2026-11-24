import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { lastDisplayName } from "../api/session";
import { Avatar } from "./Avatar";
import { Logo } from "./Logo";

export function Header({ title, nav = false, children }: { title?: string; nav?: boolean; children?: ReactNode }) {
  const name = lastDisplayName();

  return (
    <header className="site-header">
      <div className="site-header-inner">
        <Logo title={title} />
        {nav && (
          <nav className="site-nav">
            <a href="#features">Features</a>
            <a href="#how">How it works</a>
          </nav>
        )}
        <div className="site-header-actions">
          {children}
          {name && (
            <span className="user-chip">
              <Avatar name={name} size={24} />
              {name}
            </span>
          )}
          <Link to="/new" className="button button-primary button-small">
            Start new game
          </Link>
        </div>
      </div>
    </header>
  );
}
