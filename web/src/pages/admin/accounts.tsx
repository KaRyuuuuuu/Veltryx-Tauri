import type { WorkspacePageProps } from "../../types/app";
import { formatDateTime, formatRoleLabel } from "../../utils/app-helpers";

export default function AdminAccountsPage({ user, adminPresence, handleDisconnectDesktopSession, busy }: WorkspacePageProps) {
  const sessions = adminPresence?.sessions ?? [];

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Admin</p>
            <h3>Account management</h3>
          </div>
          <span className="badge">{sessions.length} known sessions</span>
        </div>

        <div className="admin-list">
          {sessions.map((entry) => (
            <div className="admin-card" key={entry.sessionKey}>
              <div>
                <strong>{entry.name}</strong>
                <p>
                  @{entry.username} | {formatRoleLabel(entry.role)}
                </p>
              </div>
              <div>
                <strong>{entry.tenantId || "global"}</strong>
                <p>{entry.clientLabel || "desktop"}</p>
              </div>
              <div>
                <strong>{formatDateTime(entry.lastSeenAt)}</strong>
                <p>{entry.userId}</p>
              </div>
              <div>
                <button
                  className="ghost-button"
                  disabled={busy}
                  onClick={() => void handleDisconnectDesktopSession(entry.sessionKey)}
                >
                  Disconnect
                </button>
              </div>
            </div>
          ))}

          {!sessions.length ? (
            <div className="empty-state">
              <strong>No managed accounts online</strong>
              <p>Connected operators will appear here for supervision and forced disconnect.</p>
            </div>
          ) : null}
        </div>
      </article>

      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Current admin</p>
            <h3>Management scope</h3>
          </div>
        </div>

        <div className="stats-stack">
          <div className="stack-row">
            <span>Operator</span>
            <strong>{user.name}</strong>
          </div>
          <div className="stack-row">
            <span>Role</span>
            <strong>{formatRoleLabel(user.role)}</strong>
          </div>
          <div className="stack-row">
            <span>Tenant</span>
            <strong>{user.tenantId || "global"}</strong>
          </div>
          <div className="stack-row">
            <span>Actions</span>
            <strong>View, supervise, disconnect</strong>
          </div>
        </div>
      </article>
    </section>
  );
}
