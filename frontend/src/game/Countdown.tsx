import { useEffect, useState } from "react";

/** Time left until the room expires. */
export function Countdown({ until }: { until: string }) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);

  const seconds = Math.max(0, Math.floor((new Date(until).getTime() - now) / 1000));
  const text = `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;

  return (
    <span className={`countdown ${seconds < 300 ? "warning" : ""}`} title="Time until this room expires">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
        <circle cx="12" cy="13" r="8" />
        <path d="M12 9v4l2 2M9 2h6M12 2v3" strokeLinecap="round" />
      </svg>
      {text}
    </span>
  );
}
