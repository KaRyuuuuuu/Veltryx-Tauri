import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import { buildTrackPolyline, flagColor, flagShortLabel, isCoord, projectTrackPoint } from "../../utils/app-helpers";

type SectorSegment = {
  id: number;
  name: string;
  flag: number;
  coordinates: number[][];
};

function normalizeCoords(coords?: number[][]): number[][] {
  return (coords ?? []).filter((c): c is [number, number] => isCoord(c));
}

function pickMainTrackCoords(paths?: Array<{ type?: string; coordinates?: number[][] }>): number[][] {
  const list = paths ?? [];
  const trackOnly = list.filter((p) => (p.type ?? "").toLowerCase().includes("track"));
  const source = trackOnly.length ? trackOnly : list;
  const best = source.reduce<{ size: number; coords: number[][] }>(
    (acc, item) => {
      const coords = normalizeCoords(item.coordinates);
      if (coords.length > acc.size) {
        return { size: coords.length, coords };
      }
      return acc;
    },
    { size: 0, coords: [] },
  );
  return best.coords;
}

function nearestPointIndex(pathCoords: number[][], target: number[]) {
  let bestIndex = -1;
  let bestDist = Number.POSITIVE_INFINITY;

  for (let i = 0; i < pathCoords.length; i += 1) {
    const p = pathCoords[i];
    const dLat = p[0] - target[0];
    const dLon = p[1] - target[1];
    const dist = dLat * dLat + dLon * dLon;
    if (dist < bestDist) {
      bestDist = dist;
      bestIndex = i;
    }
  }

  return bestIndex;
}

function segmentBetween(pathCoords: number[][], startIndex: number, endIndex: number) {
  if (startIndex < 0 || endIndex < 0 || pathCoords.length < 2) return [];
  if (startIndex <= endIndex) return pathCoords.slice(startIndex, endIndex + 1);
  return [...pathCoords.slice(startIndex), ...pathCoords.slice(0, endIndex + 1)];
}

function buildSectorSegments(
  pathCoords: number[][],
  trackSectors: Array<{ id?: number; name?: string; coordinate?: number[] }>,
  sectorFlagById: Map<number, number>,
): SectorSegment[] {
  if (pathCoords.length < 2) return [];

  const indexed = trackSectors
    .filter((s) => Number.isInteger(s.id) && (s.id ?? 0) > 0 && isCoord(s.coordinate))
    .map((s) => ({
      id: s.id as number,
      name: s.name || `S${s.id}`,
      flag: sectorFlagById.get(s.id as number) ?? 0,
      index: nearestPointIndex(pathCoords, s.coordinate as number[]),
    }))
    .filter((s) => s.index >= 0)
    .sort((a, b) => a.index - b.index);

  if (indexed.length < 2) return [];

  const segments: SectorSegment[] = [];
  for (let i = 0; i < indexed.length; i += 1) {
    const current = indexed[i];
    const next = indexed[(i + 1) % indexed.length];
    const coords = segmentBetween(pathCoords, current.index, next.index);
    if (coords.length >= 2) {
      segments.push({
        id: current.id,
        name: current.name,
        flag: current.flag,
        coordinates: coords,
      });
    }
  }
  return segments;
}

export default function ViewerMapPage({
  telemetry,
  trackBounds,
  trackName,
  trackPathCount,
  trackGpsPoints,
  raceLinks,
  sectorStates,
  status,
}: WorkspacePageProps) {
  const raceLinkById = new Map((raceLinks ?? []).map((entry) => [entry.id, entry]));
  const sectorFlagById = new Map((sectorStates ?? []).map((entry) => [entry.id, entry.flag ?? 0]));
  const mainTrackCoords = pickMainTrackCoords(telemetry?.track?.paths);
  const sectorSegments = buildSectorSegments(mainTrackCoords, telemetry?.track?.sectors ?? [], sectorFlagById);

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Map</p>
            <h3>Track map</h3>
          </div>
        </div>
        {trackBounds && trackPathCount ? (
          <div className="track-map-shell">
            <svg className="track-map-svg" viewBox="0 0 1000 680" role="img" aria-label="Track map">
              <rect x="0" y="0" width="1000" height="680" rx="24" fill="rgba(8, 16, 26, 0.9)" />

              {(telemetry?.track?.paths ?? []).map((path) => {
                const coords = normalizeCoords(path.coordinates);
                if (!coords.length) return null;
                const isMainTrack = (path.type ?? "").toLowerCase().includes("track");
                if (isMainTrack && mainTrackCoords.length > 1) {
                  return null;
                }
                const points = buildTrackPolyline(path.coordinates, trackBounds);
                if (!points) return null;
                return (
                  <polyline
                    key={`path-${path.id ?? path.name}`}
                    points={points}
                    fill="none"
                    stroke="#4f6f90"
                    strokeWidth={3}
                    strokeLinejoin="round"
                    strokeLinecap="round"
                    opacity={0.72}
                  />
                );
              })}

              {mainTrackCoords.length > 1 ? (
                <polyline
                  points={buildTrackPolyline(mainTrackCoords, trackBounds)}
                  fill="none"
                  stroke="#202f40"
                  strokeWidth={11}
                  strokeLinejoin="round"
                  strokeLinecap="round"
                  opacity={0.95}
                />
              ) : null}

              {sectorSegments.map((segment) => (
                <polyline
                  key={`sector-segment-${segment.id}`}
                  points={buildTrackPolyline(segment.coordinates, trackBounds)}
                  fill="none"
                  stroke={flagColor(segment.flag)}
                  strokeWidth={8}
                  strokeLinejoin="round"
                  strokeLinecap="round"
                  opacity={0.95}
                />
              ))}

              {(telemetry?.track?.lines ?? []).map((line) => {
                if (!isCoord(line.start) || !isCoord(line.end)) return null;
                const start = projectTrackPoint(line.start[0], line.start[1], trackBounds);
                const end = projectTrackPoint(line.end[0], line.end[1], trackBounds);
                const intermediate = (line.type ?? "").toLowerCase().includes("intermediate");
                return (
                  <line
                    key={`line-${line.id ?? line.name}`}
                    x1={start.x}
                    y1={start.y}
                    x2={end.x}
                    y2={end.y}
                    stroke={intermediate ? "#52b2ff" : "#f4c76a"}
                    strokeWidth={2}
                    strokeDasharray={intermediate ? "10 8" : "0"}
                    opacity={0.9}
                  />
                );
              })}

              {(telemetry?.track?.sectors ?? []).map((sector) => {
                if (!isCoord(sector.coordinate)) return null;
                const point = projectTrackPoint(sector.coordinate[0], sector.coordinate[1], trackBounds);
                const currentFlag = sectorFlagById.get(sector.id ?? 0) ?? 0;
                const color = flagColor(currentFlag);
                return (
                  <g key={`sector-${sector.id ?? sector.name}`}>
                    <circle cx={point.x} cy={point.y} r="9" fill={color} />
                    <circle cx={point.x} cy={point.y} r="16" fill={color} opacity={0.16} />
                    <text x={point.x + 12} y={point.y + 4} className="track-map-label">
                      {sector.name || `S${sector.id ?? ""}`} ({flagShortLabel(currentFlag)})
                    </text>
                  </g>
                );
              })}

              {trackGpsPoints.map((point) => {
                const projected = projectTrackPoint(point.lat!, point.lon!, trackBounds);
                const raceLink = point.racelinkId ? raceLinkById.get(point.racelinkId) : undefined;
                const markerColor = flagColor(raceLink?.flag);
                const carLabel = raceLink?.carNumber
                  ? `#${raceLink.carNumber}`
                  : raceLink?.name || point.vehicle || point.racelinkId || "car";
                const speedKmh = typeof point.speed === "number" && Number.isFinite(point.speed) ? Math.round(point.speed) : null;
                return (
                  <g key={`car-${point.racelinkId ?? point.vehicle ?? `${point.lat}-${point.lon}`}`}>
                    <circle cx={projected.x} cy={projected.y} r="6.5" fill={markerColor} />
                    <circle cx={projected.x} cy={projected.y} r="14" fill={markerColor} opacity={0.18} />
                    <text x={projected.x + 10} y={projected.y - 10} className="track-map-car-label">
                      {speedKmh ? `${carLabel} ${speedKmh} km/h` : carLabel}
                    </text>
                  </g>
                );
              })}
            </svg>
          </div>
        ) : (
          <EmptyState
            title="Tracking unavailable"
            body="Connecte le bridge X2 et attends la trackconfiguration pour afficher la piste et les positions."
          />
        )}
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Track</p>
            <h3>Map status</h3>
          </div>
        </div>
        <div className="stats-stack">
          <div className="stack-row">
            <span>Track name</span>
            <strong>{trackName}</strong>
          </div>
          <div className="stack-row">
            <span>Track paths</span>
            <strong>{trackPathCount}</strong>
          </div>
          <div className="stack-row">
            <span>Sector markers</span>
            <strong>{telemetry?.track?.sectors?.length ?? 0}</strong>
          </div>
          <div className="stack-row">
            <span>Active sectors</span>
            <strong>{sectorStates.length}</strong>
          </div>
          <div className="stack-row">
            <span>Cars on map</span>
            <strong>{trackGpsPoints.length}</strong>
          </div>
          <div className="stack-row">
            <span>Tracking source</span>
            <strong>{status.connected ? "X2 live" : "Offline"}</strong>
          </div>
        </div>
        <div className="tag-list">
              {sectorStates.slice(0, 8).map((sector) => (
                <span
                  key={`sector-legend-${sector.id}`}
                  className="chip"
              style={{
                borderColor: flagColor(sector.flag),
                color: "#dce8f7",
                boxShadow: `inset 0 0 0 1px ${flagColor(sector.flag)}55`,
              }}
                >
                  {sector.name || `S${sector.id}`} - {flagShortLabel(sector.flag)}
                </span>
              ))}
        </div>
      </article>
    </section>
  );
}
