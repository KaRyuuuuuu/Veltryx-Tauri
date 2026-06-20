import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";

export default function RaceControlBaselinksPage({ baseLinks }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">RIStoMylaps</p>
            <h3>Baselinks workspace</h3>
          </div>
        </div>
        <div className="device-grid">
          {baseLinks.map((item) => (
            <div className="device-card" key={item.id}>
              <strong>{item.name || `BL ${item.id}`}</strong>
              <span>{item.status || "online"}</span>
              <p>
                {item.hostname || item.ipv4addr || "unknown"} {item.ipv4port ? `:${item.ipv4port}` : ""}
              </p>
            </div>
          ))}
          {!baseLinks.length ? (
            <EmptyState title="No baselinks yet" body="Aucun baselink n'est encore remonte par X2." />
          ) : null}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Scope</p>
            <h3>Operational visibility</h3>
          </div>
        </div>
        <ul className="plain-list">
          <li>Baselink cards</li>
          <li>Signal and state</li>
          <li>Quick diagnostics</li>
          <li>Restricted to control roles</li>
        </ul>
      </article>
    </section>
  );
}
