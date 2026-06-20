import type {
  DesktopProfileResponse,
  DesktopUser,
  DetachedPanel,
  LiveBoardRow,
  LiveTimingSnapshot,
  NavItem,
  PageId,
  TelemetryResponse,
  TrackPathPayload,
} from "../types/app";

export type FlagMeta = {
  code: number;
  label: string;
  color: string;
  shortLabel?: string;
  controlEnabled?: boolean;
};

export const FLAG_CATALOG: FlagMeta[] = [
  { code: 0, label: "Clear", color: "#8ea5bf", shortLabel: "Clear" },
  { code: 1, label: "Clear", color: "#8ea5bf", shortLabel: "Clear" },
  { code: 2, label: "Yellow", color: "#f4c542", shortLabel: "Yellow" },
  { code: 3, label: "Double Yellow", color: "#ff9f1c", shortLabel: "2 Yellow" },
  { code: 4, label: "White", color: "#f5f7fa", shortLabel: "White" },
  { code: 5, label: "Green", color: "#35d07f", shortLabel: "Green" },
  { code: 6, label: "Blue", color: "#4d96ff", shortLabel: "Blue" },
  { code: 7, label: "Red", color: "#f25555", shortLabel: "Red" },
  { code: 8, label: "Adhesion Change", color: "#c77dff", shortLabel: "Adhesion" },
  { code: 9, label: "Safety Car", color: "#ff8c42", shortLabel: "Safety" },
  { code: 10, label: "Virtual Safety Car", color: "#ffb703", shortLabel: "VSC" },
  { code: 11, label: "FCY", color: "#ffd60a", shortLabel: "FCY" },
  { code: 12, label: "Code 60", color: "#9b5de5", shortLabel: "Code 60" },
  { code: 13, label: "Keep Right", color: "#7bdff2", shortLabel: "Right" },
  { code: 14, label: "Keep Left", color: "#7bdff2", shortLabel: "Left" },
  { code: 15, label: "Red Cross", color: "#ff595e", shortLabel: "Red Cross" },
  { code: 16, label: "Blue 2", color: "#2563eb", shortLabel: "Blue 2" },
  { code: 17, label: "SS", color: "#14b8a6", shortLabel: "SS" },
  { code: 18, label: "Rolling Start", color: "#06b6d4", shortLabel: "Rolling" },
  { code: 19, label: "Technique", color: "#94a3b8", shortLabel: "Tech" },
  { code: 20, label: "Double White", color: "#dfe7ef", shortLabel: "2 White" },
  { code: 21, label: "Black Flag", color: "#111827", shortLabel: "Black" },
  { code: 22, label: "Chequered", color: "#e5e7eb", shortLabel: "Chequered" },
  { code: 23, label: "NS", color: "#64748b", shortLabel: "NS" },
  { code: 24, label: "SZ", color: "#64748b", shortLabel: "SZ" },
  { code: 25, label: "Red Cross On White", color: "#fff1f2", shortLabel: "Cross/White" },
  { code: 26, label: "Red Cross On White With Adhesion Change", color: "#f9d5e5", shortLabel: "Cross/Slip" },
  { code: 27, label: "Logo", color: "#94a3b8", shortLabel: "Logo" },
  { code: 28, label: "Mylaps", color: "#ef4444", shortLabel: "Mylaps" },
  { code: 29, label: "Warning", color: "#fb7185", shortLabel: "Warning" },
  { code: 30, label: "Speedhive", color: "#f97316", shortLabel: "Speedhive" },
  { code: 31, label: "Red Cross On White With Adhesion Change And Yellow", color: "#fbbf24", shortLabel: "Cross/Slip/Y" },
  { code: 32, label: "Red Cross On White With Adhesion Change And Double Yellow", color: "#f59e0b", shortLabel: "Cross/Slip/2Y" },
  { code: 33, label: "Yellow Static", color: "#d4a700", shortLabel: "Y Static" },
  { code: 34, label: "Yellow Attention", color: "#ffd166", shortLabel: "Y Attention" },
  { code: 35, label: "Green Static", color: "#1faa59", shortLabel: "G Static" },
  { code: 38, label: "Slippery Flash", color: "#b5179e", shortLabel: "Slip Flash" },
  { code: 43, label: "Behavior Static", color: "#38bdf8", shortLabel: "Behavior" },
  { code: 44, label: "Behavior Flash", color: "#0ea5e9", shortLabel: "Behavior Flash" },
  { code: 59, label: "FCY Static", color: "#eab308", shortLabel: "FCY Static" },
  { code: 60, label: "FCY Asia", color: "#facc15", shortLabel: "FCY Asia" },
  { code: 61, label: "Do Not Use", color: "#475569", shortLabel: "N/A", controlEnabled: false },
];

const FLAG_META_BY_CODE = new Map<number, FlagMeta>(FLAG_CATALOG.map((item) => [item.code, item]));

export function hasPermission(profile: DesktopProfileResponse | null, permission: string) {
  return Boolean(
    profile?.authenticated &&
      Array.isArray(profile.user?.permissions) &&
      profile.user.permissions.includes(permission),
  );
}

export function normalizeRole(role?: string) {
  if (role === "super_admin") {
    return "admin";
  }
  return role ?? "";
}

export function formatRoleLabel(role?: string) {
  switch (normalizeRole(role)) {
    case "admin":
      return "Administration";
    case "dc":
      return "Direction de course";
    case "diffuseur":
      return "Diffusion";
    case "viewer":
      return "Viewer";
    default:
      return "Guest";
  }
}

export function formatTime(value?: string) {
  return value ? new Date(value).toLocaleTimeString("fr-BE") : "--:--:--";
}

export function formatDateTime(value?: string) {
  return value ? new Date(value).toLocaleString("fr-BE") : "--";
}

export function formatPermissionLabel(value: string) {
  return value.split("_").join(" ");
}

export function trimTo(value: string, max: number) {
  if (value.length <= max) {
    return value;
  }
  if (max <= 1) {
    return value.slice(0, max);
  }
  return `${value.slice(0, max - 1)}.`;
}

export function formatDurationMs(value?: number) {
  if (!value || value <= 0) {
    return "-";
  }
  const totalSeconds = value / 1000;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds - minutes * 60;
  return minutes > 0 ? `${minutes}:${seconds.toFixed(3).padStart(6, "0")}` : seconds.toFixed(3);
}

export function formatDecimal(value: number | null | undefined, digits = 1) {
  return typeof value === "number" && Number.isFinite(value) ? value.toFixed(digits) : "0.0";
}

export function getFlagMeta(flag?: number) {
  if (typeof flag !== "number" || Number.isNaN(flag)) {
    return FLAG_META_BY_CODE.get(0)!;
  }
  return FLAG_META_BY_CODE.get(flag) ?? {
    code: flag,
    label: `Flag ${flag}`,
    color: "#8ea5bf",
    shortLabel: `Flag ${flag}`,
  };
}

export function flagLabel(flag?: number) {
  return getFlagMeta(flag).label;
}

export function flagShortLabel(flag?: number) {
  const meta = getFlagMeta(flag);
  return meta.shortLabel || meta.label;
}

export function flagColor(flag?: number) {
  return getFlagMeta(flag).color;
}

export function isCoord(value: number[] | undefined): value is [number, number] {
  return Boolean(
    Array.isArray(value) &&
      value.length >= 2 &&
      typeof value[0] === "number" &&
      Number.isFinite(value[0]) &&
      typeof value[1] === "number" &&
      Number.isFinite(value[1]),
  );
}

export function isTrackPath(path: TrackPathPayload | undefined) {
  return Boolean(path && Array.isArray(path.coordinates) && path.coordinates.some((point: number[]) => isCoord(point)));
}

export function computeTrackBounds(track?: TelemetryResponse["track"]) {
  const points: Array<[number, number]> = [];

  for (const path of track?.paths ?? []) {
    if (!isTrackPath(path)) continue;
    for (const point of path.coordinates ?? []) {
      if (isCoord(point)) {
        points.push([point[0], point[1]]);
      }
    }
  }

  for (const line of track?.lines ?? []) {
    if (isCoord(line.start)) points.push([line.start[0], line.start[1]]);
    if (isCoord(line.end)) points.push([line.end[0], line.end[1]]);
  }

  for (const sector of track?.sectors ?? []) {
    if (isCoord(sector.coordinate)) {
      points.push([sector.coordinate[0], sector.coordinate[1]]);
    }
  }

  if (!points.length) {
    return null;
  }

  let minLat = Infinity;
  let maxLat = -Infinity;
  let minLon = Infinity;
  let maxLon = -Infinity;

  for (const [lat, lon] of points) {
    minLat = Math.min(minLat, lat);
    maxLat = Math.max(maxLat, lat);
    minLon = Math.min(minLon, lon);
    maxLon = Math.max(maxLon, lon);
  }

  return { minLat, maxLat, minLon, maxLon };
}

export function projectTrackPoint(lat: number, lon: number, bounds: NonNullable<ReturnType<typeof computeTrackBounds>>) {
  const width = 1000;
  const height = 680;
  const padding = 40;
  const lonSpan = Math.max(bounds.maxLon - bounds.minLon, 0.000001);
  const latSpan = Math.max(bounds.maxLat - bounds.minLat, 0.000001);
  const usableWidth = width - padding * 2;
  const usableHeight = height - padding * 2;
  const x = padding + ((lon - bounds.minLon) / lonSpan) * usableWidth;
  const y = height - padding - ((lat - bounds.minLat) / latSpan) * usableHeight;
  return { x, y };
}

export function buildTrackPolyline(
  coordinates: number[][] | undefined,
  bounds: NonNullable<ReturnType<typeof computeTrackBounds>>,
) {
  return (coordinates ?? [])
    .filter((point): point is [number, number] => isCoord(point))
    .map(([lat, lon]) => {
      const projected = projectTrackPoint(lat, lon, bounds);
      return `${projected.x},${projected.y}`;
    })
    .join(" ");
}

export function buildLiveBoardRows(snapshot: LiveTimingSnapshot | null): LiveBoardRow[] {
  if (!snapshot || !Array.isArray(snapshot.leaderboard)) {
    return [];
  }

  return snapshot.leaderboard.slice(0, 16).map((entry, index) => ({
    pos: String(entry.position || index + 1),
    now: entry.pitState ? "PIT" : entry.status && entry.status > 1 ? "STP" : "RUN",
    num: entry.carNumber || "--",
    team: trimTo(entry.team || "-", 24),
    car: trimTo(entry.car || "-", 18),
    className: entry.class || "-",
    laps: String(entry.laps || 0),
    lastLap: formatDurationMs(entry.lastLapMs),
    bestLap: formatDurationMs(entry.bestLapMs),
    s1: formatDurationMs(entry.sector1Ms),
    s2: formatDurationMs(entry.sector2Ms),
    s3: formatDurationMs(entry.sector3Ms),
    gap: entry.gapMs ? `+${formatDurationMs(entry.gapMs)}` : "-",
    info: entry.position === 1 ? "LEAD" : "",
  }));
}

export function getShellTone(role?: string) {
  switch (normalizeRole(role)) {
    case "viewer":
      return "tone-viewer";
    case "diffuseur":
      return "tone-broadcast";
    case "dc":
      return "tone-control";
    case "admin":
      return "tone-admin";
    default:
      return "tone-guest";
  }
}

export function getLandingTitle(user: DesktopUser | null) {
  if (!user) return "Veltryx Desktop";
  const role = normalizeRole(user.role);
  if (role === "viewer") return "Viewer Workspace";
  if (role === "diffuseur") return "Broadcast Workspace";
  if (role === "dc") return "Race Control Workspace";
  return "Admin Workspace";
}

export function getAllowedPages(profile: DesktopProfileResponse | null, navItems: NavItem[]) {
  return navItems.filter((item) => (item.permission ? hasPermission(profile, item.permission) : true));
}

export function findDefaultPage(profile: DesktopProfileResponse | null, navItems: NavItem[]): PageId {
  const allowed = getAllowedPages(profile, navItems);
  return allowed[0]?.id ?? "viewer-home";
}

export function groupedNav(items: NavItem[]) {
  return items.reduce<Record<string, NavItem[]>>((acc, item) => {
    acc[item.section] ??= [];
    acc[item.section].push(item);
    return acc;
  }, {});
}

export function buildExpandedSections(groups: Record<string, NavItem[]>, currentPage: PageId) {
  const next: Record<string, boolean> = {};
  for (const [section, items] of Object.entries(groups)) {
    next[section] = items.some((item) => item.id === currentPage);
  }
  return next;
}

export function sanitizeProfile(input: DesktopProfileResponse | null): DesktopProfileResponse {
  if (!input?.authenticated || !input.user) {
    return { authenticated: false, user: null };
  }

  return {
    authenticated: true,
    user: {
      id: input.user.id ?? "",
      username: input.user.username ?? "",
      email: input.user.email ?? "",
      name: input.user.name ?? input.user.username ?? "Desktop user",
      role: input.user.role ?? "viewer",
      tenantId: input.user.tenantId,
      permissions: Array.isArray(input.user.permissions) ? input.user.permissions : [],
    },
  };
}

export function getDetachedPanel(): DetachedPanel {
  const value = new URLSearchParams(window.location.search).get("panel");
  if (value === "live") return "viewer-live";
  if (value === "map") return "viewer-map";
  return null;
}
