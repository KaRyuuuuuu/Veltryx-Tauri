import { getLandingTitle } from "../utils/app-helpers";
import type { DesktopUser } from "../types/app";

type ShellTopbarProps = {
  user: DesktopUser;
  showDetachLive: boolean;
  showDetachMap: boolean;
  canConnect: boolean;
  busy: boolean;
  connected: boolean;
  onDetachLive: () => void;
  onDetachMap: () => void;
  onConnectToggle: () => void;
  onLogout: () => void;
};

export function ShellTopbar({
  user,
  showDetachLive,
  showDetachMap,
  canConnect,
  busy,
  connected,
  onDetachLive,
  onDetachMap,
  onConnectToggle,
  onLogout,
}: ShellTopbarProps) {
  return (
    <header className="app-topbar">
      <div className="topbar-brand">
        <img className="topbar-mark-image" src="/branding/veltryx_icon.png" alt="Veltryx icon" />
        <div>
          <div className="topbar-subtitle">{getLandingTitle(user)}</div>
        </div>
      </div>
      <div className="topbar-actions">
        {showDetachLive ? (
          <button className="ghost-button" onClick={onDetachLive}>
            Detach Live
          </button>
        ) : null}
        {showDetachMap ? (
          <button className="ghost-button" onClick={onDetachMap}>
            Detach Map
          </button>
        ) : null}
        {canConnect ? (
          <button className="primary-button" onClick={onConnectToggle} disabled={busy}>
            {busy ? "Working..." : connected ? "Disconnect X2" : "Connect X2"}
          </button>
        ) : null}
        <button className="ghost-button" onClick={onLogout}>
          Logout
        </button>
      </div>
    </header>
  );
}
