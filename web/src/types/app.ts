import type { ReactNode } from "react";
import type { TmsUiConfig } from "../utils/tms";

export type StatusResponse = {
  connected: boolean;
  state: string;
};

export type ActionResponse = StatusResponse & {
  ok?: boolean;
  error?: string;
};

export type TelemetryEvent = {
  time: string;
  kind: string;
  message: string;
  raw?: string;
};

export type TelemetryResponse = {
  connected: boolean;
  state: string;
  totalMsgs: number;
  messagesPerS: number;
  lastMsg: number;
  lastRaw: string;
  lastAt?: string;
  connectedAt?: string;
  welcome?: {
    clientid?: number;
    expirationdate?: string;
    expirationdaysleft?: number;
    protocolversion?: number;
    token?: string;
  };
  auth?: {
    authenticated?: boolean;
    clientname?: string;
    role?: string;
    user?: string;
  };
  track?: {
    version?: number;
    name?: string;
    creator?: string;
    modified?: string;
    paths?: Array<{
      id?: number;
      name?: string;
      width?: number;
      type?: string;
      closed?: boolean;
      coordinates?: number[][];
    }>;
    lines?: Array<{
      id?: number;
      name?: string;
      start?: number[];
      end?: number[];
      type?: string;
    }>;
    sectors?: Array<{
      id?: number;
      name?: string;
      coordinate?: number[];
      path?: number;
    }>;
  };
  gpsLatest?: Array<{
    racelinkId?: number;
    lat?: number;
    lon?: number;
    speed?: number;
    timestamp?: number;
    vehicle?: string;
  }>;
  raceLinks?: Array<{
    id: number;
    name?: string;
    carNumber?: number;
    flag?: number;
    battery?: number;
    rssi?: number;
    status?: string;
    baseLink?: number;
    lastSeen?: number;
  }>;
  baseLinks?: Array<{
    id: number;
    name?: string;
    hostname?: string;
    ipv4addr?: string;
    ipv4port?: number;
    status?: string;
  }>;
  sectors?: Array<{
    id: number;
    name?: string;
    flag?: number;
  }>;
  globalFlag?: number;
  alerts?: Array<{
    code?: string;
    message?: string;
    level?: string;
  }>;
  events: TelemetryEvent[];
};

export type TrackPayload = NonNullable<TelemetryResponse["track"]>;
export type TrackPathPayload = NonNullable<TrackPayload["paths"]>[number];

export type LiveBoardRow = {
  pos: string;
  now: string;
  num: string;
  team: string;
  car: string;
  className: string;
  laps: string;
  lastLap: string;
  bestLap: string;
  s1: string;
  s2: string;
  s3: string;
  gap: string;
  info: string;
};

export type TmsScreenConfig = {
  primaryCanId?: number | null;
  testCanId?: number | null;
  channel: string;
};

export type TmsSendResponse = {
  ok: boolean;
  canId: number;
  data: number[];
  targets: number[];
};

export type TmsUiStatus = {
  sending: boolean;
  feedback: string;
  feedbackTone: "ok" | "error";
  sendLog: string[];
};

export type LiveTimingSnapshot = {
  session: {
    name?: string;
    series?: string;
    sessionType?: string;
    date?: string;
    time?: string;
    circuit?: string;
    durationMs?: number;
  };
  stream: {
    address?: string;
    connected: boolean;
    lastError?: string;
    lastChunkAt?: string;
    bytesReceived?: number;
    messagesReceived?: number;
    lastMessage?: string;
    recentMessages?: string[];
  };
  leaderboard: Array<{
    position?: number;
    carNumber?: string;
    team?: string;
    car?: string;
    class?: string;
    laps?: number;
    passageMs?: number;
    lastLapMs?: number;
    bestLapMs?: number;
    sector1Ms?: number;
    sector2Ms?: number;
    sector3Ms?: number;
    gapMs?: number;
    status?: number;
    pitState?: number;
  }>;
  lastMessage?: {
    tag?: string;
    code?: string;
    text?: string;
  };
  messages?: Array<{
    tag?: string;
    code?: string;
    text?: string;
  }>;
  participants?: number;
};

export type DesktopUser = {
  id: string;
  username: string;
  email: string;
  name: string;
  role: string;
  tenantId?: string;
  permissions: string[];
};

export type DesktopProfileResponse = {
  authenticated: boolean;
  user: DesktopUser | null;
};

export type ErrorBoundaryProps = {
  children: ReactNode;
};

export type ErrorBoundaryState = {
  error: string;
};

export type AdminPresenceEntry = {
  sessionKey: string;
  userId: string;
  username: string;
  name: string;
  role: string;
  tenantId?: string;
  clientLabel?: string;
  lastSeenAt: string;
};

export type AdminPresenceResponse = {
  sessions: AdminPresenceEntry[];
  total: number;
};

export type PageId =
  | "account-profile"
  | "viewer-home"
  | "viewer-live"
  | "viewer-map"
  | "viewer-sectors"
  | "viewer-flags"
  | "viewer-reports"
  | "rc-overview"
  | "rc-flags"
  | "rc-racelinks"
  | "rc-tms"
  | "rc-baselinks"
  | "bridge"
  | "bridge-health"
  | "logs"
  | "admin-presence"
  | "admin-accounts"
  | "admin-users"
  | "admin-events"
  | "admin-settings";

export type DetachedPanel = "viewer-live" | "viewer-map" | null;

export type NavItem = {
  id: PageId;
  label: string;
  section: string;
  permission?: string;
  roles?: string[];
};

export type WorkspacePageProps = {
  user: DesktopUser;
  profile: DesktopProfileResponse | null;
  status: StatusResponse;
  telemetry: TelemetryResponse | null;
  liveTiming: LiveTimingSnapshot | null;
  events: TelemetryEvent[];
  userPermissions: string[];
  viewerPermissions: string[];
  controlPermissions: string[];
  adminPermissions: string[];
  liveBoardRows: LiveBoardRow[];
  liveTitle: string;
  liveTopInfo: string;
  liveBottomMessage: string;
  trackBounds: {
    minLat: number;
    maxLat: number;
    minLon: number;
    maxLon: number;
  } | null;
  trackName: string;
  trackPathCount: number;
  trackGpsPoints: NonNullable<TelemetryResponse["gpsLatest"]>;
  raceLinks: NonNullable<TelemetryResponse["raceLinks"]>;
  baseLinks: NonNullable<TelemetryResponse["baseLinks"]>;
  sectorStates: NonNullable<TelemetryResponse["sectors"]>;
  globalFlagLabel: string;
  adminPresence: AdminPresenceResponse | null;
  tmsConfig: TmsScreenConfig | null;
  tmsState: TmsUiConfig;
  tmsStatus: TmsUiStatus;
  busy: boolean;
  canSeeLogs: boolean;
  isDetachedWindow: boolean;
  handleDetachWindow: (target: DetachedPanel) => Promise<void>;
  handleSetGlobalFlag: (flag: number) => Promise<void>;
  handleSetSectorFlag: (sectorId: number, flag: number) => Promise<void>;
  handleSendTmsCan: (
    data: number[],
    primaryCanId?: number | null,
    testCanId?: number | null,
    canId?: number | null,
  ) => Promise<TmsSendResponse>;
  updateTmsState: (patch: Partial<TmsUiConfig>) => void;
  handleSendTmsManual: () => Promise<void>;
  handleDisconnectDesktopSession: (sessionId: string) => Promise<void>;
};
