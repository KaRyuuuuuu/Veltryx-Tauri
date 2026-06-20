import type { WorkspacePageProps } from "../../types/app";
import { formatPermissionLabel, formatRoleLabel } from "../../utils/app-helpers";

export default function AdminUsersPage({ user, adminPresence, adminPermissions }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Users</p>
            <h3>Role and permission overview</h3>
          </div>
        </div>
        <div className="report-list">
          <div className="report-card">
            <div>
              <strong>Current operator</strong>
              <p>{user.email}</p>
            </div>
            <span className="badge">{formatRoleLabel(user.role)}</span>
          </div>
          <div className="report-card">
            <div>
              <strong>Connected operators</strong>
              <p>Presence sessions visible to admins</p>
            </div>
            <span className="badge">{adminPresence?.total ?? 0}</span>
          </div>
        </div>
        <div className="chip-list">
          {adminPermissions.map((permission) => (
            <span className="chip" key={permission}>
              {formatPermissionLabel(permission)}
            </span>
          ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Next</p>
            <h3>Admin user management</h3>
          </div>
        </div>
        <ul className="plain-list">
          <li>Users list with role edition</li>
          <li>Session limits and plan values</li>
          <li>Tenant assignment</li>
          <li>Password reset and lock controls</li>
        </ul>
      </article>
    </section>
  );
}
