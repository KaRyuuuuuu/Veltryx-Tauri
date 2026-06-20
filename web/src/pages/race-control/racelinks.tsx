import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import { flagLabel } from "../../utils/app-helpers";

export default function RaceControlRacelinksPage({ raceLinks }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">RIStoMylaps</p>
            <h3>Racelinks workspace</h3>
          </div>
        </div>
        <div className="device-grid">
          {raceLinks.map((item) => (
            <div className="device-card" key={item.id}>
              <strong>RL {item.id}</strong>
              <span>{item.carNumber ? `#${item.carNumber}` : item.status || "active"}</span>
              <p>Flag {flagLabel(item.flag)} | RSSI {item.rssi || 0} | Batt {item.battery || 0}</p>
            </div>
          ))}
          {!raceLinks.length ? (
            <EmptyState title="No racelinks yet" body="Le bridge X2 est connecte, mais aucun racelink n'a encore ete remonte." />
          ) : null}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Data source</p>
            <h3>X2 dependent</h3>
          </div>
        </div>
        <p className="panel-copy">
          {raceLinks.length
            ? `${raceLinks.length} racelink(s) visibles depuis X2, sur le meme principe que RIStoMylaps.`
            : "Cette page passera automatiquement sur des cartes reelles des que le flux X2 annoncera les racelinks."}
        </p>
      </article>
    </section>
  );
}
