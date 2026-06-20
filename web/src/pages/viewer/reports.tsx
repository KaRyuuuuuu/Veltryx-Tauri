import type { WorkspacePageProps } from "../../types/app";
import { hasPermission, formatPermissionLabel } from "../../utils/app-helpers";

export default function ViewerReportsPage({ profile, telemetry, user, viewerPermissions }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Reports</p>
            <h3>Session reports</h3>
          </div>
        </div>
        <div className="report-list">
          {[
            {
              name: "Session export",
              type: "PDF",
              state: hasPermission(profile, "export_reports") ? "Ready" : "Read only",
            },
            {
              name: "Flags summary",
              type: "CSV",
              state: telemetry?.totalMsgs ? "Fresh data" : "No data",
            },
            {
              name: "Race incidents",
              type: "JSON",
              state: hasPermission(profile, "view_reports") ? "Visible" : "Hidden",
            },
          ].map((report) => (
            <div className="report-card" key={report.name}>
              <div>
                <strong>{report.name}</strong>
                <p>{report.type}</p>
              </div>
              <span className="badge">{report.state}</span>
            </div>
          ))}
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Access</p>
            <h3>Available actions</h3>
          </div>
        </div>
        {user.role === "viewer" ? (
          <ul className="plain-list">
            <li>Consultation des rapports</li>
            <li>Acces simple aux documents de session</li>
            <li>Pas d outils techniques affiches</li>
          </ul>
        ) : (
          <div className="chip-list">
            {viewerPermissions.map((permission) => (
              <span className="chip" key={permission}>
                {formatPermissionLabel(permission)}
              </span>
            ))}
          </div>
        )}
      </article>
    </section>
  );
}
