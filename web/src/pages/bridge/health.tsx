import type { WorkspacePageProps } from "../../types/app";
import { formatDateTime, formatRoleLabel } from "../../utils/app-helpers";

export default function BridgeHealthPage({ telemetry, status, user }: WorkspacePageProps) {
  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Health</p>
            <h3>Bridge and machine status</h3>
          </div>
        </div>
        <div className="stats-stack">
          <div className="stack-row">
            <span>Session heartbeat</span>
            <strong>{formatDateTime(telemetry?.lastAt)}</strong>
          </div>
          <div className="stack-row">
            <span>Desktop auth</span>
            <strong>Active</strong>
          </div>
          <div className="stack-row">
            <span>Transport mode</span>
            <strong>{status.state}</strong>
          </div>
          <div className="stack-row">
            <span>Client role</span>
            <strong>{formatRoleLabel(user.role)}</strong>
          </div>
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Expected local_pc features</p>
            <h3>Bridge diagnostics scope</h3>
          </div>
        </div>
        <ul className="plain-list">
          <li>Bridge process health</li>
          <li>WebSocket forwarding status</li>
          <li>X2 handshake age</li>
          <li>Machine identity and diagnostics</li>
        </ul>
      </article>
    </section>
  );
}
