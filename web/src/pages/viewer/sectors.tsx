import type { WorkspacePageProps } from "../../types/app";
import { flagShortLabel } from "../../utils/app-helpers";

export default function ViewerSectorsPage({ sectorStates, status, events }: WorkspacePageProps) {
      const sectors = sectorStates.length
    ? sectorStates.map((sector) => ({
        label: sector.name || `Sector ${sector.id}`,
        status: flagShortLabel(sector.flag),
        time: `ID ${sector.id}`,
      }))
    : [
        { label: "Sector 1", status: status.connected ? "Green" : "Standby", time: "31.204" },
        { label: "Sector 2", status: events.length > 2 ? "Yellow watch" : "Green", time: "29.887" },
        { label: "Sector 3", status: events.length ? "Clear" : "Standby", time: "32.115" },
      ];

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Sectors</p>
            <h3>Sectors board</h3>
          </div>
        </div>
        <div className="sector-list">
          {sectors.map((sector) => (
            <div className="sector-card" key={sector.label}>
              <strong>{sector.label}</strong>
              <span>{sector.status}</span>
              <p>{sector.time}</p>
            </div>
          ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Usage</p>
            <h3>Consultation only</h3>
          </div>
        </div>
        <p className="panel-copy">Cette page sert uniquement a lire les secteurs et leur etat pendant la session.</p>
      </article>
    </section>
  );
}
