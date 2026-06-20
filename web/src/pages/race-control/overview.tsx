import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import { formatPermissionLabel, formatTime } from "../../utils/app-helpers";

export default function RaceControlOverviewPage({ status, events, profile, controlPermissions }: WorkspacePageProps) {
  return (
    <section className="grid-3">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Race Control</p>
            <h3>Operational overview</h3>
          </div>
        </div>
        <div className="stats-grid">
          <div className="stat-card">
            <span>Control state</span>
            <strong>{status.connected ? "Armed" : "Offline"}</strong>
            <p>X2 control path</p>
          </div>
          <div className="stat-card">
            <span>Telemetry events</span>
            <strong>{events.length}</strong>
            <p>Recent raw entries</p>
          </div>
          <div className="stat-card">
            <span>Stream Deck</span>
            <strong>{profile?.user?.permissions.includes("streamdeck_access") ? "Enabled" : "Hidden"}</strong>
            <p>Permission-gated surface</p>
          </div>
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Control rights</p>
            <h3>Operational permissions</h3>
          </div>
        </div>
        <div className="chip-list">
          {controlPermissions.map((permission) => (
            <span className="chip" key={permission}>
              {formatPermissionLabel(permission)}
            </span>
          ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Alerts</p>
            <h3>Recent events</h3>
          </div>
        </div>
        <div className="feed-list">
          {events.slice(-5).reverse().map((event) => (
            <div className="feed-item" key={`${event.time}-${event.message}`}>
              <span>{formatTime(event.time)}</span>
              <div>
                <strong>{event.kind}</strong>
                <p>{event.message}</p>
              </div>
            </div>
          ))}
          {!events.length ? <EmptyState title="No alerts" body="Race control alerts will appear here." /> : null}
        </div>
      </article>
    </section>
  );
}
