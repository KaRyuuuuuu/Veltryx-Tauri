import type { LiveTimingSnapshot } from "../types/app";

export type TmsColor = "yellow" | "white" | "green" | "purple";

export type SnapshotEntry = {
  carNumber: string;
  position: number;
  gapMs: number;
  passageMs: number;
  lastLapMs: number;
  bestLapMs: number;
  team: string;
  car: string;
};

export type ComputedTmsPacket = {
  front: number;
  rear: number;
  lap: number;
  position: number;
  frontColor: TmsColor;
  rearColor: TmsColor;
  lapColor: TmsColor;
  payload: number[];
  testPayload: number[];
};

export type TmsUiConfig = {
  primaryCanId: string;
  testCanId: string;
  selectedCar: string;
  realtimeEnabled: boolean;
  autoAllEnabled: boolean;
  useCarOnTestRear: boolean;
  canId: string;
  frontColor: TmsColor;
  rearColor: TmsColor;
  lapColor: TmsColor;
  deltaFront: string;
  deltaRear: string;
  position: string;
  lapTime: string;
};

export const TMS_STORAGE_KEY = "veltryx:tms-config:v1";

export const defaultTmsUiConfig: TmsUiConfig = {
  primaryCanId: "",
  testCanId: "",
  selectedCar: "",
  realtimeEnabled: false,
  autoAllEnabled: false,
  useCarOnTestRear: true,
  canId: "514",
  frontColor: "yellow",
  rearColor: "yellow",
  lapColor: "yellow",
  deltaFront: "0",
  deltaRear: "0",
  position: "1",
  lapTime: "0",
};

export function loadStoredTmsConfig() {
  if (typeof window === "undefined") return defaultTmsUiConfig;
  try {
    const raw = window.localStorage.getItem(TMS_STORAGE_KEY);
    if (!raw) return defaultTmsUiConfig;
    const parsed = JSON.parse(raw) as Partial<TmsUiConfig>;
    return { ...defaultTmsUiConfig, ...parsed };
  } catch {
    return defaultTmsUiConfig;
  }
}

export function resolveLapMetric(entry: SnapshotEntry | null | undefined) {
  if (!entry) return 0;
  if (entry.lastLapMs > 0) return entry.lastLapMs;
  if (entry.bestLapMs > 0) return entry.bestLapMs;
  return 0;
}

export function toHundredths(valueMs: number) {
  return Math.max(0, Math.trunc(valueMs / 100));
}

function applyDeltaColor(value: number, color: TmsColor) {
  if (color === "white") return value | (1 << 14);
  if (color === "green" || color === "purple") return value | (1 << 15);
  return value;
}

function applyLapColor(value: number, color: TmsColor) {
  if (color === "white") return value | (1 << 22);
  if (color === "green") return value | (1 << 23);
  if (color === "purple") return value | (1 << 22) | (1 << 23);
  return value;
}

export function encodeTmsPayload(
  deltaFront: number,
  frontColor: TmsColor,
  deltaRear: number,
  rearColor: TmsColor,
  position: number,
  lapTime: number,
  lapColor: TmsColor,
) {
  const encodedFront = applyDeltaColor(Math.max(0, deltaFront), frontColor);
  const encodedRear = applyDeltaColor(Math.max(0, deltaRear), rearColor);
  const encodedLap = applyLapColor(Math.max(0, lapTime), lapColor);

  return [
    encodedFront & 0xff,
    (encodedFront >> 8) & 0xff,
    encodedRear & 0xff,
    (encodedRear >> 8) & 0xff,
    Math.max(0, Math.min(255, position)),
    encodedLap & 0xff,
    (encodedLap >> 8) & 0xff,
    (encodedLap >> 16) & 0xff,
  ];
}

export function formatLap(valueMs: number) {
  if (!valueMs) return "-";
  const totalSeconds = valueMs / 1000;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds - minutes * 60;
  return `${minutes}:${seconds.toFixed(3).padStart(6, "0")}`;
}

export function deriveMetricColor(
  current: number,
  previous: number | undefined,
  carBest: number | undefined,
  sessionBest: number | undefined,
) {
  if (current <= 0) return "white" as const;
  if (typeof sessionBest === "number" && sessionBest > 0 && current <= sessionBest) return "purple" as const;
  if (typeof carBest === "number" && carBest > 0 && current <= carBest) return "green" as const;
  if (typeof previous === "number" && previous > 0 && current < previous) return "yellow" as const;
  return "white" as const;
}

export function buildFrontDelta(entry: SnapshotEntry, ahead: SnapshotEntry | null) {
  if (ahead && entry.passageMs > 0 && ahead.passageMs > 0 && entry.passageMs >= ahead.passageMs) {
    return toHundredths(entry.passageMs - ahead.passageMs);
  }
  return toHundredths(Math.max(0, entry.gapMs - (ahead?.gapMs || 0)));
}

export function buildTmsLeaderboard(snapshot: LiveTimingSnapshot | null) {
  return (snapshot?.leaderboard ?? [])
    .map((entry, index) => ({
      carNumber: entry.carNumber || "",
      position: entry.position || index + 1,
      gapMs: entry.gapMs || 0,
      passageMs: entry.passageMs || 0,
      lastLapMs: entry.lastLapMs || 0,
      bestLapMs: entry.bestLapMs || 0,
      team: entry.team || "-",
      car: entry.car || "-",
    }))
    .filter((entry) => entry.carNumber !== "")
    .sort((a, b) => a.position - b.position);
}

export function computeTmsPacket(
  entry: SnapshotEntry,
  currentBoard: SnapshotEntry[],
  previousSnapshot: Record<string, SnapshotEntry>,
  bestFrontByCar: Record<string, number>,
  bestRearByCar: Record<string, number>,
  bestLapByCar: Record<string, number>,
  bestFrontSession: number,
  bestRearSession: number,
  bestLapSession: number,
  useCarOnTestRear: boolean,
): ComputedTmsPacket {
  const entryIndex = currentBoard.findIndex((candidate) => candidate.carNumber === entry.carNumber);
  const ahead = entryIndex > 0 ? currentBoard[entryIndex - 1] : null;
  const behind = entryIndex >= 0 && entryIndex < currentBoard.length - 1 ? currentBoard[entryIndex + 1] : null;
  const previous = previousSnapshot[entry.carNumber];
  const previousAhead =
    previous && previous.position > 1
      ? Object.values(previousSnapshot).find((candidate) => candidate.position === previous.position - 1) ?? null
      : null;

  const front = buildFrontDelta(entry, ahead);
  const rear = behind ? toHundredths(resolveLapMetric(behind)) : 0;
  const lap = Math.max(0, Math.trunc(resolveLapMetric(entry)));
  const positionValue = entry.position || 1;
  const carNumberValue = Number(entry.carNumber || 0);

  const resolvedFrontColor = deriveMetricColor(
    front,
    previous ? buildFrontDelta(previous, previousAhead) : undefined,
    bestFrontByCar[entry.carNumber],
    Number.isFinite(bestFrontSession) ? bestFrontSession : undefined,
  );
  const resolvedRearColor = deriveMetricColor(
    rear,
    previous ? toHundredths(resolveLapMetric(previous)) : undefined,
    bestRearByCar[entry.carNumber],
    Number.isFinite(bestRearSession) ? bestRearSession : undefined,
  );
  const resolvedLapColor = deriveMetricColor(
    lap,
    resolveLapMetric(previous),
    bestLapByCar[entry.carNumber],
    Number.isFinite(bestLapSession) ? bestLapSession : undefined,
  );

  const payload = encodeTmsPayload(
    front,
    resolvedFrontColor,
    rear,
    resolvedRearColor,
    positionValue,
    lap,
    resolvedLapColor,
  );
  const testPayload = useCarOnTestRear
    ? encodeTmsPayload(front, resolvedFrontColor, carNumberValue, "yellow", positionValue, lap, resolvedLapColor)
    : payload;

  return {
    front,
    rear,
    lap,
    position: positionValue,
    frontColor: resolvedFrontColor,
    rearColor: resolvedRearColor,
    lapColor: resolvedLapColor,
    payload,
    testPayload,
  };
}
