import { invoke } from "@tauri-apps/api/core";
import { useState } from "react";
import type { WorkspacePageProps } from "../../types/app";
import { formatPermissionLabel } from "../../utils/app-helpers";

export default function AdminSettingsPage({ adminPermissions }: WorkspacePageProps) {
  const [updaterBusy, setUpdaterBusy] = useState(false);
  const [updaterState, setUpdaterState] = useState("Idle");

  async function handleCheckUpdate() {
    setUpdaterBusy(true);
    try {
      const result = await invoke<{ available?: boolean; version?: string; body?: string }>("updater_check");
      if (result.available) {
        setUpdaterState(`Update available: ${result.version || "unknown"}`);
      } else {
        setUpdaterState("No update available");
      }
    } catch (err) {
      setUpdaterState(`Check failed: ${String(err)}`);
    } finally {
      setUpdaterBusy(false);
    }
  }

  async function handleInstallUpdate() {
    setUpdaterBusy(true);
    try {
      const result = await invoke<{ updated?: boolean; version?: string; reason?: string }>("updater_install");
      if (result.updated) {
        setUpdaterState(`Update installed (${result.version || "unknown"}). Restart required.`);
      } else {
        setUpdaterState(`Install skipped: ${result.reason || "no_update"}`);
      }
    } catch (err) {
      setUpdaterState(`Install failed: ${String(err)}`);
    } finally {
      setUpdaterBusy(false);
    }
  }

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Settings</p>
            <h3>System and desktop settings</h3>
          </div>
        </div>
        <div className="stats-stack">
          <div className="stack-row">
            <span>Desktop mode</span>
            <strong>Unified</strong>
          </div>
          <div className="stack-row">
            <span>Backend URL</span>
            <strong>127.0.0.1:8080</strong>
          </div>
          <div className="stack-row">
            <span>Website URL</span>
            <strong>veltryx.karyuu.be</strong>
          </div>
          <div className="stack-row">
            <span>X2 / RIS configuration</span>
            <strong>%APPDATA%\veltryx\veltryx.env</strong>
          </div>
          <div className="stack-row">
            <span>Updater state</span>
            <strong>{updaterState}</strong>
          </div>
        </div>
        <div className="button-row">
          <button className="cta-button" onClick={() => void handleCheckUpdate()} disabled={updaterBusy}>
            {updaterBusy ? "Working..." : "Check update"}
          </button>
          <button className="ghost-button" onClick={() => void handleInstallUpdate()} disabled={updaterBusy}>
            Install update
          </button>
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Admin policy</p>
            <h3>Settings scope</h3>
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
    </section>
  );
}
