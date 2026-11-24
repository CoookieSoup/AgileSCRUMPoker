const PALETTE = ["#e8a33d", "#e5675f", "#5aa2f5", "#2fb57d", "#b07cf0", "#e57fb3", "#4cc3c9", "#c9b458"];

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  const letters = parts.length > 1 ? parts[0]![0]! + parts[1]![0]! : name.trim().slice(0, 2);
  return letters.toUpperCase() || "?";
}

export function colorFor(seed: string): string {
  let hash = 0;
  for (const char of seed) hash = (hash * 31 + char.charCodeAt(0)) | 0;
  return PALETTE[Math.abs(hash) % PALETTE.length]!;
}

export function Avatar({ name, seed = name, size = 40 }: { name: string; seed?: string; size?: number }) {
  return (
    <span
      className="avatar"
      style={{ width: size, height: size, fontSize: size * 0.38, background: colorFor(seed) }}
      aria-hidden="true"
    >
      {initials(name)}
    </span>
  );
}
