import type { WorkspacePageProps } from "../../types/app";
import { flagShortLabel } from "../../utils/app-helpers";

export default function ViewerFlagsPage({ sectorStates, globalFlagLabel, status }: WorkspacePageProps) {
  const tiles = [
    `Global ${globalFlagLabel}`,
    ...sectorStates.slice(0, 4).map((sector) => `${sector.name || `S${sector.id}`} ${flagShortLabel(sector.flag)}`),
  ];

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Flags</p>
            <h3>Flags board</h3>
          </div>
        </div>
        <div className="action-grid">
          {tiles.map((label) => (
            <div className="status-tile" key={label}>
              <strong>{label}</strong>
              <p>{status.connected ? "Visible" : "Awaiting feed"}</p>
            </div>
          ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Usage</p>
            <h3>What you can do</h3>
          </div>
        </div>
        <ul className="plain-list">
          <li>Voir l etat de course</li>
          <li>Suivre les changements de flag</li>
          <li>Aucune action de controle</li>
        </ul>
      </article>
    </section>
  );
}
