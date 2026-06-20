import type { NavItem, PageId } from "../types/app";

export const STARTUP_MASK_MS = 2200;

export const navItems: NavItem[] = [
  { id: "account-profile", label: "My Account", section: "Account" },
  { id: "viewer-home", label: "Viewer Home", section: "Viewer", permission: "view_live_timing" },
  { id: "viewer-live", label: "Live Timing", section: "Viewer", permission: "view_live_timing" },
  { id: "viewer-map", label: "Map", section: "Viewer", permission: "view_map" },
  { id: "viewer-sectors", label: "Sectors", section: "Viewer", permission: "view_sectors" },
  { id: "viewer-flags", label: "Flags", section: "Viewer", permission: "view_flags" },
  { id: "viewer-reports", label: "Reports", section: "Viewer", permission: "view_reports" },
  { id: "rc-overview", label: "Overview", section: "Race Control", permission: "control_flags" },
  { id: "rc-flags", label: "Flags", section: "Race Control", permission: "control_flags" },
  { id: "rc-racelinks", label: "Racelinks", section: "Race Control", permission: "x2_connection" },
  { id: "rc-tms", label: "TMS Screens", section: "Race Control", permission: "x2_connection" },
  { id: "rc-baselinks", label: "Baselinks", section: "Race Control", permission: "x2_connection" },
  { id: "bridge", label: "Bridge", section: "Bridge", permission: "x2_connection" },
  { id: "bridge-health", label: "Health", section: "Bridge", permission: "x2_connection" },
  { id: "logs", label: "Logs", section: "Bridge", permission: "view_logs" },
  { id: "admin-presence", label: "Connected", section: "Admin", permission: "manage_users" },
  { id: "admin-accounts", label: "Accounts", section: "Admin", permission: "manage_users" },
  { id: "admin-users", label: "Users", section: "Admin", permission: "manage_users" },
  { id: "admin-events", label: "Events", section: "Admin", permission: "manage_events" },
  { id: "admin-settings", label: "Settings", section: "Admin", permission: "system_settings" },
];

export const pageDescriptions: Record<PageId, string> = {
  "account-profile": "Personal account page with your role, plan, tenant and current access.",
  "viewer-home": "Desktop alternative to the website, centered on live timing, map access, and read-only race context.",
  "viewer-live": "Live timing page for following the current session in real time.",
  "viewer-map": "Track map and on-circuit context for consultation roles.",
  "viewer-sectors": "Sector page for following split status and sector colors.",
  "viewer-flags": "Flags page for checking the current race status at a glance.",
  "viewer-reports": "Reports page for reading or exporting session documents depending on role.",
  "rc-overview": "Operational overview inspired by RIStoMylaps for DC workflows.",
  "rc-flags": "Race control actions inspired by RIStoMylaps flag operations.",
  "rc-racelinks": "Operational surface for racelink supervision and quick control.",
  "rc-tms": "Manual TMS screen switching and CAN dispatch towards the configured racelinks.",
  "rc-baselinks": "Operational surface for baselink visibility and control.",
  bridge: "Bridge and local agent module inspired by local_pc for X2 connectivity.",
  "bridge-health": "Bridge health and machine diagnostics module inspired by local_pc.",
  logs: "Technical logs and bridge diagnostics.",
  "admin-presence": "Administrative supervision of connected desktop operators.",
  "admin-accounts": "Administrative account management view for supervision and forced disconnects.",
  "admin-users": "Administrative user and permission overview.",
  "admin-events": "Event and championship administration workspace.",
  "admin-settings": "System-level administration and desktop configuration.",
};
