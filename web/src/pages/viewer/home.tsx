import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import { formatDecimal, formatTime } from "../../utils/app-helpers";

export default function ViewerHomePage({
  user,
  events,
  telemetry,
  userPermissions,
}: WorkspacePageProps) {
  if (user.role === "viewer") {
    return (
      <section className="grid-3">
        <article className="panel panel-large viewer-welcome">
          <div className="panel-header">
            <div>
              <p className="eyebrow">Viewer</p>
              <h3>Desktop web replacement</h3>
            </div>
          </div>
          <p className="panel-copy">
            Utilise le menu de gauche pour ouvrir directement le live timing, la map, les secteurs, les flags et les
            reports. L&apos;accueil viewer reste volontairement simple et sans informations techniques.
          </p>
        </article>
      </section>
    );
  }

  return (
    <section className="grid-3">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Viewer</p>
            <h3>Desktop web replacement</h3>
          </div>
          <span className="badge">{events.length} updates</span>
        </div>
        <div className="stats-grid">
          <div className="stat-card">
            <span>Messages / sec</span>
            <strong>{formatDecimal(telemetry?.messagesPerS, 1)}</strong>
          </div>
          <div className="stat-card">
            <span>Total messages</span>
            <strong>{telemetry?.totalMsgs ?? 0}</strong>
          </div>
          <div className="stat-card">
            <span>Last update</span>
            <strong>{formatTime(telemetry?.lastAt)}</strong>
          </div>
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Read surfaces</p>
            <h3>Available modules</h3>
          </div>
        </div>
        <div className="chip-list">
          {userPermissions
            .filter((permission) => permission.startsWith("view_") || permission === "view_reports")
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
            <p className="eyebrow">Recent feed</p>
            <h3>Last events</h3>
          </div>
        </div>
        <div className="feed-list">
          {events.slice(-5).reverse().map((event) => (
            <div className="feed-item" key={`${event.time}-${event.message}`}>
              <span>{formatTime(event.time)}</span>
              <div>
                <strong>{event.message}</strong>
                <p>{event.kind}</p>
              </div>
            </div>
          ))}
          {!events.length ? <EmptyState title="No live data" body="Connect the feed to populate viewer surfaces." /> : null}
        </div>
      </article>
    </section>
  );
}
