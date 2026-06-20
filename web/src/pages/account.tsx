import type { WorkspacePageProps } from "../types/app";
import { formatRoleLabel } from "../utils/app-helpers";

function inferPlan(role: string) {
  switch (role) {
    case "super_admin":
    case "admin":
      return "Enterprise Control";
    case "dc":
      return "Race Control Operator";
    case "diffuseur":
      return "Broadcast";
    default:
      return "Viewer Access";
  }
}

export default function AccountPage({ user, userPermissions, adminPresence }: WorkspacePageProps) {
  const planName = inferPlan(user.role);
  const activeSessions = adminPresence?.sessions.filter((session) => session.userId === user.id).length ?? 1;

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Account</p>
            <h3>Profile and plan</h3>
          </div>
          <span className="badge">{planName}</span>
        </div>

        <div className="stats-stack">
          <div className="stack-row">
            <span>Name</span>
            <strong>{user.name}</strong>
          </div>
          <div className="stack-row">
            <span>Username</span>
            <strong>@{user.username}</strong>
          </div>
          <div className="stack-row">
            <span>Email</span>
            <strong>{user.email || "Not set"}</strong>
          </div>
          <div className="stack-row">
            <span>Role</span>
            <strong>{formatRoleLabel(user.role)}</strong>
          </div>
          <div className="stack-row">
            <span>Workspace plan</span>
            <strong>{planName}</strong>
          </div>
          <div className="stack-row">
            <span>Tenant</span>
            <strong>{user.tenantId || "global"}</strong>
          </div>
          <div className="stack-row">
            <span>Known active sessions</span>
            <strong>{activeSessions}</strong>
          </div>
        </div>
      </article>

      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Capabilities</p>
            <h3>Access currently assigned</h3>
          </div>
        </div>

        <div className="chip-list">
          {userPermissions.map((permission) => (
            <span className="chip" key={permission}>
              {permission}
            </span>
          ))}
        </div>
      </article>
    </section>
  );
}
