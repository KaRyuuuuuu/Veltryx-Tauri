import type { WorkspacePageProps } from "../../types/app";
import { FLAG_CATALOG, flagLabel, flagShortLabel } from "../../utils/app-helpers";

const CONTROL_FLAGS = FLAG_CATALOG.filter((item) => item.controlEnabled !== false && item.code !== 0 && item.code !== 61);
const QUICK_GLOBAL_FLAGS = [1, 2, 3, 5, 7, 9, 10, 11, 12, 22];
const QUICK_SECTOR_FLAGS = [1, 2, 3, 4, 5, 6, 8, 20, 33, 34, 35, 38, 43, 44, 59, 60];

export default function RaceControlFlagsPage({
  userPermissions,
  status,
  telemetry,
  sectorStates,
  globalFlagLabel,
  handleSetGlobalFlag,
  handleSetSectorFlag,
}: WorkspacePageProps) {
  return (
    <section className="grid-3">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Race Control</p>
            <h3>Flag operations</h3>
          </div>
        </div>
        <div className="action-grid">
          {[0, ...QUICK_GLOBAL_FLAGS].map((flag) => (
            <button className="action-tile" key={`global-${flag}`} onClick={() => void handleSetGlobalFlag(flag)}>
              {flag === 0 ? "Clear" : flagShortLabel(flag)}
            </button>
          ))}
        </div>
        <div className="chip-list" style={{ marginTop: 16 }}>
          {CONTROL_FLAGS.map((item) => (
            <button className="chip" key={`global-palette-${item.code}`} onClick={() => void handleSetGlobalFlag(item.code)}>
              {item.code} - {item.label}
            </button>
          ))}
          <button className="chip" onClick={() => void handleSetGlobalFlag(0)}>
            0 - Clear
          </button>
        </div>
        {sectorStates.length ? (
          <div className="sector-list">
            {sectorStates.map((sector) => (
              <div className="sector-card" key={sector.id}>
                <strong>{sector.name || `S${sector.id}`}</strong>
                <span>{flagLabel(sector.flag)}</span>
                <div className="chip-list">
                  {[0, ...QUICK_SECTOR_FLAGS].map((flag) => (
                    <button className="chip" key={`${sector.id}-${flag}`} onClick={() => void handleSetSectorFlag(sector.id, flag)}>
                      {flag === 0 ? "Clear" : flagShortLabel(flag)}
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Access</p>
            <h3>Control rights</h3>
          </div>
        </div>
        <div className="chip-list">
          {userPermissions
            .filter((permission) => permission.startsWith("control_") || permission === "clear_flags")
            .map((permission) => (
              <span className="chip" key={permission}>
                {permission}
              </span>
            ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Status</p>
            <h3>X2 link</h3>
          </div>
        </div>
        <div className="stats-stack">
          <div className="stack-row">
            <span>Bridge state</span>
            <strong>{status.connected ? "Ready" : "Offline"}</strong>
          </div>
          <div className="stack-row">
            <span>Global flag</span>
            <strong>{globalFlagLabel}</strong>
          </div>
          <div className="stack-row">
            <span>Last payload</span>
            <strong>{telemetry?.lastMsg ?? 0}</strong>
          </div>
        </div>
      </article>
    </section>
  );
}
