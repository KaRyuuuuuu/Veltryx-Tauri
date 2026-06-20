import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import { formatDateTime, formatRoleLabel } from "../../utils/app-helpers";

export default function AdminConnectedPage({ adminPresence, busy, handleDisconnectDesktopSession }: WorkspacePageProps) {
  return (
    <section className="grid-1">
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Admin</p>
            <h3>Connected desktops</h3>
          </div>
          <span className="badge">{adminPresence?.total ?? 0} active</span>
        </div>
        <div className="admin-list">
          {(adminPresence?.sessions ?? []).map((entry) => (
            <div className="admin-card" key={entry.sessionKey}>
              <div>
                <strong>{entry.name}</strong>
                <p>
                  @{entry.username} | {formatRoleLabel(entry.role)}
                </p>
              </div>
              <div>
                <strong>{entry.clientLabel ?? "desktop"}</strong>
                <p>{entry.tenantId ?? "global"}</p>
              </div>
              <div>
                <strong>{formatDateTime(entry.lastSeenAt)}</strong>
                <p>{entry.sessionKey}</p>
              </div>
              <div>
                <button className="ghost-button" disabled={busy} onClick={() => void handleDisconnectDesktopSession(entry.sessionKey)}>
                  Disconnect
                </button>
              </div>
            </div>
          ))}
          {!(adminPresence?.sessions ?? []).length ? (
            <EmptyState title="No connected desktop" body="Operators will appear here as soon as they authenticate." />
          ) : null}
        </div>
      </article>
    </section>
  );
}
