import type { WorkspacePageProps } from "../../types/app";
import { formatPermissionLabel } from "../../utils/app-helpers";

export default function AdminEventsPage({ userPermissions }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Events</p>
            <h3>Operational event supervision</h3>
          </div>
        </div>
        <div className="device-grid">
          {["EWC", "FUN", "LOCAL TEST"].map((eventName, index) => (
            <div className="device-card" key={eventName}>
              <strong>{eventName}</strong>
              <span>{index === 2 ? "Draft" : "Active"}</span>
              <p>{index === 0 ? "Race control live" : "Desktop-ready"}</p>
            </div>
          ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Permissions</p>
            <h3>Manage events</h3>
          </div>
        </div>
        <div className="chip-list">
          {userPermissions
            .filter((permission) => permission === "manage_events" || permission === "view_reports")
            .map((permission) => (
              <span className="chip" key={permission}>
                {formatPermissionLabel(permission)}
              </span>
            ))}
        </div>
      </article>
    </section>
  );
}
