import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import { formatTime } from "../../utils/app-helpers";

export default function BridgeLogsPage({ events, canSeeLogs }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Logs</p>
            <h3>Technical event stream</h3>
          </div>
        </div>
        <div className="feed-list">
          {events.slice().reverse().map((event) => (
            <div className="feed-item" key={`${event.time}-${event.message}`}>
              <span>{formatTime(event.time)}</span>
              <div>
                <strong>{event.message}</strong>
                <p>{event.kind}</p>
              </div>
            </div>
          ))}
          {!events.length ? <EmptyState title="No logs yet" body="Bridge and X2 diagnostics will be shown here." /> : null}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Visibility</p>
            <h3>Permission scope</h3>
          </div>
        </div>
        {canSeeLogs ? (
          <div className="chip-list">
            <span className="chip">view_logs</span>
            <span className="chip">x2_connection</span>
          </div>
        ) : (
          <EmptyState title="No log access" body="This module is reserved to operational and admin roles." />
        )}
      </article>
    </section>
  );
}
