import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { WebviewWindow } from "@tauri-apps/api/webviewWindow";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useMemo, useRef, useState } from "react";
import "./App.css";
import { AppErrorBoundary } from "./components/AppErrorBoundary";
import { PageHero } from "./components/PageHero";
import { ShellSidebar } from "./components/ShellSidebar";
import { ShellTopbar } from "./components/ShellTopbar";
import { StartupScreen } from "./components/StartupScreen";
import { STARTUP_MASK_MS, navItems, pageDescriptions } from "./config/app-shell";
import { RenderPageContent } from "./pages/render-page";
import type {
  ActionResponse,
  AdminPresenceResponse,
  DesktopProfileResponse,
  DetachedPanel,
  PageId,
  StatusResponse,
  TelemetryResponse,
  LiveTimingSnapshot,
  TmsScreenConfig,
  TmsUiStatus,
} from "./types/app";
import {
  buildExpandedSections,
  buildLiveBoardRows,
  computeTrackBounds,
  findDefaultPage,
  flagLabel,
  getAllowedPages,
  getDetachedPanel,
  getShellTone,
  groupedNav,
  hasPermission,
  normalizeRole,
  sanitizeProfile,
} from "./utils/app-helpers";
import {
  buildTmsLeaderboard,
  computeTmsPacket,
  encodeTmsPayload,
  loadStoredTmsConfig,
  TMS_STORAGE_KEY,
} from "./utils/tms";


function AppShell() {
  const detachedPanel = getDetachedPanel();
  const isDetachedWindow = detachedPanel !== null;
  const [status, setStatus] = useState<StatusResponse>({ connected: false, state: "disconnected" });
  const [telemetry, setTelemetry] = useState<TelemetryResponse | null>(null);
  const [liveTiming, setLiveTiming] = useState<LiveTimingSnapshot | null>(null);
  const [profile, setProfile] = useState<DesktopProfileResponse | null>(null);
  const [booting, setBooting] = useState(true);
  const [busy, setBusy] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [page, setPage] = useState<PageId>(detachedPanel ?? "viewer-home");
  const [adminPresence, setAdminPresence] = useState<AdminPresenceResponse | null>(null);
  const [tmsConfig, setTmsConfig] = useState<TmsScreenConfig | null>(null);
  const [tmsState, setTmsState] = useState(() => loadStoredTmsConfig());
  const [tmsStatus, setTmsStatus] = useState<TmsUiStatus>({
    sending: false,
    feedback: "",
    feedbackTone: "ok",
    sendLog: [],
  });
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});

  async function bootstrapProfile() {
    const startedAt = Date.now();
    try {
      const data = await invoke<DesktopProfileResponse>("bootstrap_desktop_session");
      setProfile(sanitizeProfile(data));
    } catch (err) {
      setProfile({ authenticated: false, user: null });
      setError(String(err));
    } finally {
      const remaining = Math.max(0, STARTUP_MASK_MS - (Date.now() - startedAt));
      if (remaining > 0) {
        await new Promise((resolve) => window.setTimeout(resolve, remaining));
      }
      setBooting(false);
    }
  }

  async function refreshTelemetry() {
    if (refreshing) return;
    setRefreshing(true);
    try {
      const data = await invoke<TelemetryResponse>("get_x2_telemetry");
      setTelemetry(data);
      setStatus({ connected: data.connected, state: data.state });
      setError("");
    } catch (err) {
      setError(String(err));
    } finally {
      setRefreshing(false);
    }
  }

  async function refreshLiveTiming() {
    if (!hasPermission(profile, "view_live_timing") && !hasPermission(profile, "x2_connection")) {
      setLiveTiming(null);
      return;
    }
    try {
      const data = await invoke<LiveTimingSnapshot>("get_live_timing_snapshot");
      setLiveTiming(data);
    } catch {
      setLiveTiming(null);
    }
  }

  async function refreshAdminPresence() {
    if (!hasPermission(profile, "manage_users")) {
      setAdminPresence(null);
      return;
    }
    try {
      const data = await invoke<AdminPresenceResponse>("get_admin_desktop_presence");
      setAdminPresence(data);
    } catch {
      setAdminPresence({ sessions: [], total: 0 });
    }
  }

  async function refreshTmsConfig() {
    if (!hasPermission(profile, "x2_connection")) {
      setTmsConfig(null);
      return;
    }
    try {
      const data = await invoke<TmsScreenConfig>("get_tms_screen_config");
      setTmsConfig(data);
    } catch {
      setTmsConfig(null);
    }
  }

  async function handleDetachWindow(target: DetachedPanel) {
    if (!target) return;

    const label = `${target}-${Date.now()}`;
    const panel = target === "viewer-live" ? "live" : "map";
    const title = target === "viewer-live" ? "Veltryx Live Timing" : "Veltryx Map";

    try {
      const windowRef = new WebviewWindow(label, {
        title,
        width: target === "viewer-live" ? 1440 : 1280,
        height: target === "viewer-live" ? 920 : 860,
        minWidth: 980,
        minHeight: 680,
        resizable: true,
        url: `index.html?panel=${panel}`,
      });

      windowRef.once("tauri://error", (event: { payload: unknown }) => {
        setError(String(event.payload));
      });
    } catch (err) {
      setError(String(err));
    }
  }

  useEffect(() => {
    void bootstrapProfile();
  }, []);

  useEffect(() => {
    void getCurrentWindow().setIcon("/branding/veltryx_icon.png").catch(() => {
      // Ignore icon failures in browsers or restricted dev contexts.
    });
  }, []);

  useEffect(() => {
    if (!profile?.authenticated) {
      return;
    }

    void refreshTelemetry();
    void refreshLiveTiming();
    void refreshAdminPresence();
    void refreshTmsConfig();

    const intervalMs = 3000;

    const interval = window.setInterval(() => {
      // Fallback safety poll in case the realtime stream drops.
      void refreshTelemetry();
      void refreshLiveTiming();
      void refreshAdminPresence();
      void refreshTmsConfig();
    }, intervalMs);

    return () => window.clearInterval(interval);
  }, [page, profile?.authenticated, profile?.user?.role, profile?.user?.permissions?.join(",")]);

  useEffect(() => {
    if (!profile?.authenticated) {
      return;
    }

    let cancelled = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | null = null;

    const connect = () => {
      if (cancelled) return;
      socket = new WebSocket("ws://127.0.0.1:8080/api/x2/stream");

      socket.onmessage = (event) => {
        try {
          const next = JSON.parse(event.data) as TelemetryResponse;
          setTelemetry(next);
          setStatus({ connected: next.connected, state: next.state });
          setError("");
        } catch {
          // Ignore malformed frames, fallback polling remains active.
        }
      };

      socket.onclose = () => {
        if (cancelled) return;
        reconnectTimer = window.setTimeout(connect, 900);
      };

      socket.onerror = () => {
        socket?.close();
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
      }
      socket?.close();
    };
  }, [profile?.authenticated]);

  useEffect(() => {
    const allowed = getAllowedPages(profile, navItems);
    if (detachedPanel) {
      setPage(detachedPanel);
      return;
    }
    if (!allowed.some((item) => item.id === page)) {
      setPage(findDefaultPage(profile, navItems));
    }
    void refreshAdminPresence();
  }, [detachedPanel, page, profile?.authenticated, profile?.user?.role]);

  async function handleDisconnectDesktopSession(sessionId: string) {
    setBusy(true);
    try {
      await invoke("disconnect_desktop_session", { sessionId });
      await refreshAdminPresence();
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleConnectToggle() {
    if (!hasPermission(profile, "x2_connection")) {
      setError("Permission x2_connection required");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const command = status.connected ? "disconnect_x2" : "connect_x2";
      const data = await invoke<ActionResponse>(command);
      setStatus({ connected: data.connected, state: data.state });
      if (data.error) setError(data.error);
      await refreshTelemetry();
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleSetGlobalFlag(flag: number) {
    setBusy(true);
    try {
      await invoke("set_global_flag", { flag });
      await refreshTelemetry();
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleSetSectorFlag(sectorId: number, flag: number) {
    setBusy(true);
    try {
      await invoke("set_sector_flag", { sectorId, flag });
      await refreshTelemetry();
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleSendTmsCan(
    data: number[],
    primaryCanId?: number | null,
    testCanId?: number | null,
    canId?: number | null,
  ) {
    const response = await invoke<{ ok: boolean; canId: number; data: number[]; targets: number[] }>("send_tms_can", {
      data,
      primaryCanId: primaryCanId ?? null,
      testCanId: testCanId ?? null,
      canId: canId ?? null,
    });
    setError("");
    return response;
  }

  function updateTmsState(patch: Partial<typeof tmsState>) {
    setTmsState((current) => ({ ...current, ...patch }));
  }

  const user = profile?.user ?? null;
  const previousTmsSnapshotRef = useRef<Record<string, ReturnType<typeof buildTmsLeaderboard>[number]>>({});
  const bestFrontByCarRef = useRef<Record<string, number>>({});
  const bestRearByCarRef = useRef<Record<string, number>>({});
  const bestLapByCarRef = useRef<Record<string, number>>({});
  const bestFrontSessionRef = useRef(Number.POSITIVE_INFINITY);
  const bestRearSessionRef = useRef(Number.POSITIVE_INFINITY);
  const bestLapSessionRef = useRef(Number.POSITIVE_INFINITY);
  const lastAutoSendKeyRef = useRef("");
  const lastAutoSendByCarRef = useRef<Record<string, string>>({});
  const userPermissions = Array.isArray(user?.permissions) ? user.permissions : [];
  const normalizedUserRole = normalizeRole(user?.role);
  const allowedPages = useMemo(() => getAllowedPages(profile, navItems), [profile]);
  const navGroups = useMemo(() => groupedNav(allowedPages), [allowedPages]);
  const shellTone = getShellTone(normalizedUserRole);
  const pageTitle = allowedPages.find((item) => item.id === page)?.label ?? "Workspace";
  const pageDescription = pageDescriptions[page];
  const events = telemetry?.events ?? [];
  const canConnect = hasPermission(profile, "x2_connection");
  const canSeeLogs = hasPermission(profile, "view_logs");
  const viewerPermissions =
    userPermissions.filter(
      (permission) =>
        permission.startsWith("view_") || permission === "view_reports" || permission === "export_reports",
    );
  const controlPermissions =
    userPermissions.filter(
      (permission) =>
        permission.startsWith("control_") ||
        permission === "clear_flags" ||
        permission === "streamdeck_access" ||
        permission === "x2_connection",
    );
  const adminPermissions =
    userPermissions.filter(
      (permission) =>
        permission.startsWith("manage_") || permission === "admin_panel" || permission === "system_settings",
    );
  const tmsLeaderboard = useMemo(() => buildTmsLeaderboard(liveTiming), [liveTiming]);
  const liveBoardRows = buildLiveBoardRows(liveTiming);
  const liveTitle = liveTiming?.session?.name || "LiveTiming";
  const liveSeries = liveTiming?.session?.series || "Live";
  const liveSessionLabel = liveTiming?.session?.sessionType || "Waiting";
  const liveCircuit = liveTiming?.session?.circuit || "Circuit unavailable";
  const liveTopInfo = `Race: ${liveSeries} | Session: ${liveSessionLabel} | Circuit: ${liveCircuit}`;
  const liveBottomMessage = liveTiming?.lastMessage?.text
    ? `[${liveTiming.lastMessage.code || liveTiming.lastMessage.tag || "MSG"}] ${liveTiming.lastMessage.text}`
    : "No message";
  const trackBounds = computeTrackBounds(telemetry?.track);
  const trackName = telemetry?.track?.name || liveCircuit || "Track";
  const trackPathCount = telemetry?.track?.paths?.length ?? 0;
  const trackGpsPoints = (telemetry?.gpsLatest ?? []).filter(
    (point) =>
      typeof point.lat === "number" &&
      Number.isFinite(point.lat) &&
      typeof point.lon === "number" &&
      Number.isFinite(point.lon),
  );
  const raceLinks = [...(telemetry?.raceLinks ?? [])].sort((a, b) => a.id - b.id);
  const baseLinks = telemetry?.baseLinks ?? [];
  const sectorStates = [...(telemetry?.sectors ?? [])].sort((a, b) => a.id - b.id);
  const globalFlagLabel = flagLabel(telemetry?.globalFlag);
  const showDetachLive = !isDetachedWindow && page === "viewer-live";
  const showDetachMap = !isDetachedWindow && page === "viewer-map";

  useEffect(() => {
    setExpandedSections((current) => {
      const defaults = buildExpandedSections(navGroups, page);
      const merged: Record<string, boolean> = {};
      for (const section of Object.keys(navGroups)) {
        merged[section] = current[section] ?? defaults[section] ?? false;
      }
      return merged;
    });
  }, [navGroups, page]);

  useEffect(() => {
    window.localStorage.setItem(TMS_STORAGE_KEY, JSON.stringify(tmsState));
  }, [tmsState]);

  useEffect(() => {
    if (!tmsState.primaryCanId && tmsConfig?.primaryCanId) {
      setTmsState((current) => ({ ...current, primaryCanId: String(tmsConfig.primaryCanId) }));
    }
    if (!tmsState.testCanId && tmsConfig?.testCanId) {
      setTmsState((current) => ({ ...current, testCanId: String(tmsConfig.testCanId) }));
    }
  }, [tmsConfig?.primaryCanId, tmsConfig?.testCanId]);

  useEffect(() => {
    if (!tmsState.selectedCar && tmsLeaderboard[0]?.carNumber) {
      setTmsState((current) => ({ ...current, selectedCar: tmsLeaderboard[0].carNumber }));
    }
  }, [tmsLeaderboard, tmsState.selectedCar]);

  useEffect(() => {
    if (!tmsLeaderboard.length) {
      previousTmsSnapshotRef.current = {};
      return;
    }

    for (const entry of tmsLeaderboard) {
      const packet = computeTmsPacket(
        entry,
        tmsLeaderboard,
        previousTmsSnapshotRef.current,
        bestFrontByCarRef.current,
        bestRearByCarRef.current,
        bestLapByCarRef.current,
        bestFrontSessionRef.current,
        bestRearSessionRef.current,
        bestLapSessionRef.current,
        tmsState.useCarOnTestRear,
      );

      if (packet.front > 0) {
        const currentBest = bestFrontByCarRef.current[entry.carNumber];
        if (typeof currentBest !== "number" || packet.front < currentBest) {
          bestFrontByCarRef.current[entry.carNumber] = packet.front;
        }
        if (packet.front < bestFrontSessionRef.current) {
          bestFrontSessionRef.current = packet.front;
        }
      }

      if (packet.rear > 0) {
        const currentBest = bestRearByCarRef.current[entry.carNumber];
        if (typeof currentBest !== "number" || packet.rear < currentBest) {
          bestRearByCarRef.current[entry.carNumber] = packet.rear;
        }
        if (packet.rear < bestRearSessionRef.current) {
          bestRearSessionRef.current = packet.rear;
        }
      }

      if (packet.lap > 0) {
        const currentBest = bestLapByCarRef.current[entry.carNumber];
        if (typeof currentBest !== "number" || packet.lap < currentBest) {
          bestLapByCarRef.current[entry.carNumber] = packet.lap;
        }
        if (packet.lap < bestLapSessionRef.current) {
          bestLapSessionRef.current = packet.lap;
        }
      }
    }

    previousTmsSnapshotRef.current = Object.fromEntries(
      tmsLeaderboard.map((entry) => [entry.carNumber, entry]),
    );
  }, [tmsLeaderboard, tmsState.useCarOnTestRear]);

  useEffect(() => {
    const selectedEntry = tmsLeaderboard.find((entry) => entry.carNumber === tmsState.selectedCar);
    if (!selectedEntry) return;

    const packet = computeTmsPacket(
      selectedEntry,
      tmsLeaderboard,
      previousTmsSnapshotRef.current,
      bestFrontByCarRef.current,
      bestRearByCarRef.current,
      bestLapByCarRef.current,
      bestFrontSessionRef.current,
      bestRearSessionRef.current,
      bestLapSessionRef.current,
      tmsState.useCarOnTestRear,
    );

    setTmsState((current) => {
      const next = {
        ...current,
        deltaFront: String(packet.front),
        deltaRear: String(packet.rear),
        lapTime: String(packet.lap),
        position: String(packet.position),
        frontColor: packet.frontColor,
        rearColor: packet.rearColor,
        lapColor: packet.lapColor,
      };
      return JSON.stringify(next) === JSON.stringify(current) ? current : next;
    });
  }, [tmsLeaderboard, tmsState.selectedCar, tmsState.useCarOnTestRear]);

  useEffect(() => {
    lastAutoSendKeyRef.current = "";
  }, [tmsState.selectedCar]);

  async function dispatchTmsPayload(
    primaryTarget: number | null | undefined,
    primaryPayload: number[],
    customTestTarget?: number | null,
    customTestPayload?: number[],
  ) {
    const sentTargets: number[] = [];
    const selectedCanId = Number(tmsState.canId) || 514;

    if (primaryTarget) {
      setTmsStatus((current) => ({
        ...current,
        sendLog: [...current.sendLog, `Envoi vers Racelink principal ${primaryTarget}...`],
      }));
      const result = await handleSendTmsCan(primaryPayload, primaryTarget, null, selectedCanId);
      sentTargets.push(...result.targets);
      setTmsStatus((current) => ({
        ...current,
        sendLog: [...current.sendLog, `OK principal ${primaryTarget} -> CAN ${result.canId}`],
      }));
    }

    const testTarget = customTestTarget ?? (tmsState.testCanId ? Number(tmsState.testCanId) : tmsConfig?.testCanId ?? null);
    if (testTarget) {
      setTmsStatus((current) => ({
        ...current,
        sendLog: [...current.sendLog, `Envoi vers Racelink test ${testTarget}...`],
      }));
      const result = await handleSendTmsCan(customTestPayload ?? primaryPayload, testTarget, null, selectedCanId);
      sentTargets.push(...result.targets);
      setTmsStatus((current) => ({
        ...current,
        sendLog: [...current.sendLog, `OK test ${testTarget} -> CAN ${result.canId}`],
      }));
    }

    return sentTargets;
  }

  async function handleSendTmsManual() {
    const selectedEntry = tmsLeaderboard.find((entry) => entry.carNumber === tmsState.selectedCar);
    const matchedRaceLink = raceLinks.find((item) => item.carNumber === Number(selectedEntry?.carNumber || 0)) ?? null;
    const primaryTarget = matchedRaceLink?.id ?? (tmsState.primaryCanId ? Number(tmsState.primaryCanId) : null);
    const payload = encodeTmsPayload(
      Number(tmsState.deltaFront) || 0,
      tmsState.frontColor,
      Number(tmsState.deltaRear) || 0,
      tmsState.rearColor,
      Number(tmsState.position) || 0,
      Number(tmsState.lapTime) || 0,
      tmsState.lapColor,
    );
    const testPayload = tmsState.useCarOnTestRear
      ? encodeTmsPayload(
          Number(tmsState.deltaFront) || 0,
          tmsState.frontColor,
          Number(selectedEntry?.carNumber || 0),
          "yellow",
          Number(tmsState.position) || 0,
          Number(tmsState.lapTime) || 0,
          tmsState.lapColor,
        )
      : payload;

    if (!payload.length) {
      setTmsStatus({ sending: false, feedback: "Le payload CAN est vide.", feedbackTone: "error", sendLog: ["Erreur: payload CAN vide."] });
      return;
    }

    setTmsStatus((current) => ({
      ...current,
      sending: true,
      sendLog: [
        `Preparation message TMS pour voiture #${selectedEntry?.carNumber || "?"}`,
        `Payload principal: [${payload.join(", ")}]`,
        `Payload test: [${testPayload.join(", ")}]`,
      ],
    }));

    try {
      const sentTargets = await dispatchTmsPayload(primaryTarget, payload, undefined, testPayload);
      if (!sentTargets.length) {
        throw new Error("Aucun Racelink cible n'est configure.");
      }
      setTmsStatus((current) => ({
        ...current,
        sending: false,
        feedback: `Envoye sur ${sentTargets.join(", ")} avec CAN ${Number(tmsState.canId) || 514}.`,
        feedbackTone: "ok",
        sendLog: [...current.sendLog, `Succes final: ${sentTargets.join(", ")}`],
      }));
    } catch (error) {
      setTmsStatus((current) => ({
        ...current,
        sending: false,
        feedback: String(error),
        feedbackTone: "error",
        sendLog: [...current.sendLog, `Erreur finale: ${String(error)}`],
      }));
    }
  }

  useEffect(() => {
    const selectedEntry = tmsLeaderboard.find((entry) => entry.carNumber === tmsState.selectedCar);
    if (!tmsState.realtimeEnabled || !selectedEntry || tmsStatus.sending) {
      return;
    }

    const autoKey = `${selectedEntry.carNumber}:${selectedEntry.passageMs}:${selectedEntry.position}:${selectedEntry.lastLapMs}:${selectedEntry.bestLapMs}`;
    if (selectedEntry.passageMs <= 0 || autoKey === lastAutoSendKeyRef.current) {
      return;
    }

    const matchedRaceLink = raceLinks.find((item) => item.carNumber === Number(selectedEntry.carNumber)) ?? null;
    const primaryTarget = matchedRaceLink?.id ?? (tmsState.primaryCanId ? Number(tmsState.primaryCanId) : null);
    const packet = computeTmsPacket(
      selectedEntry,
      tmsLeaderboard,
      previousTmsSnapshotRef.current,
      bestFrontByCarRef.current,
      bestRearByCarRef.current,
      bestLapByCarRef.current,
      bestFrontSessionRef.current,
      bestRearSessionRef.current,
      bestLapSessionRef.current,
      tmsState.useCarOnTestRear,
    );

    lastAutoSendKeyRef.current = autoKey;
    void (async () => {
      setTmsStatus((current) => ({
        ...current,
        sending: true,
        sendLog: [
          ...current.sendLog.slice(-10),
          `Auto-send TMS pour voiture #${selectedEntry.carNumber}`,
          `Payload principal: [${packet.payload.join(", ")}]`,
          `Payload test: [${packet.testPayload.join(", ")}]`,
        ],
      }));
      try {
        const sentTargets = await dispatchTmsPayload(primaryTarget, packet.payload, undefined, packet.testPayload);
        setTmsStatus((current) => ({
          ...current,
          sending: false,
          feedback: `Auto-envoye sur ${sentTargets.join(", ")} avec CAN ${Number(tmsState.canId) || 514}.`,
          feedbackTone: "ok",
          sendLog: [...current.sendLog, `Succes final: ${sentTargets.join(", ")}`],
        }));
      } catch (error) {
        setTmsStatus((current) => ({
          ...current,
          sending: false,
          feedback: String(error),
          feedbackTone: "error",
          sendLog: [...current.sendLog, `Erreur finale: ${String(error)}`],
        }));
      }
    })();
  }, [raceLinks, tmsConfig?.testCanId, tmsLeaderboard, tmsState, tmsStatus.sending]);

  useEffect(() => {
    if (!tmsState.autoAllEnabled || tmsStatus.sending || !tmsLeaderboard.length) {
      return;
    }

    const pendingEntries = tmsLeaderboard.filter((entry) => {
      const autoKey = `${entry.carNumber}:${entry.passageMs}:${entry.position}:${entry.lastLapMs}:${entry.bestLapMs}`;
      return entry.passageMs > 0 && lastAutoSendByCarRef.current[entry.carNumber] !== autoKey;
    });

    if (!pendingEntries.length) return;

    void (async () => {
      setTmsStatus((current) => ({ ...current, sending: true }));
      try {
        for (const entry of pendingEntries) {
          const racelink = raceLinks.find((item) => item.carNumber === Number(entry.carNumber)) ?? null;
          const autoKey = `${entry.carNumber}:${entry.passageMs}:${entry.position}:${entry.lastLapMs}:${entry.bestLapMs}`;
          const packet = computeTmsPacket(
            entry,
            tmsLeaderboard,
            previousTmsSnapshotRef.current,
            bestFrontByCarRef.current,
            bestRearByCarRef.current,
            bestLapByCarRef.current,
            bestFrontSessionRef.current,
            bestRearSessionRef.current,
            bestLapSessionRef.current,
            tmsState.useCarOnTestRear,
          );
          lastAutoSendByCarRef.current[entry.carNumber] = autoKey;
          setTmsStatus((current) => ({
            ...current,
            sendLog: [
              ...current.sendLog.slice(-14),
              `Auto-all TMS voiture #${entry.carNumber}`,
              `Payload principal: [${packet.payload.join(", ")}]`,
              `Payload test: [${packet.testPayload.join(", ")}]`,
            ],
          }));
          const sentTargets = await dispatchTmsPayload(
            racelink?.id ?? (tmsState.primaryCanId ? Number(tmsState.primaryCanId) : null),
            packet.payload,
            undefined,
            packet.testPayload,
          );
          setTmsStatus((current) => ({
            ...current,
            feedback: `Auto-all envoye pour #${entry.carNumber} sur ${sentTargets.join(", ")}.`,
            feedbackTone: "ok",
            sendLog: [...current.sendLog, `Succes auto-all #${entry.carNumber}: ${sentTargets.join(", ")}`],
          }));
        }
      } catch (error) {
        setTmsStatus((current) => ({
          ...current,
          feedback: String(error),
          feedbackTone: "error",
          sendLog: [...current.sendLog, `Erreur auto-all: ${String(error)}`],
        }));
      } finally {
        setTmsStatus((current) => ({ ...current, sending: false }));
      }
    })();
  }, [raceLinks, tmsConfig?.testCanId, tmsLeaderboard, tmsState, tmsStatus.sending]);

  const stage = booting ? "boot" : "workspace";

  const workspaceContent =
    user ? (
      <motion.div
        key="workspace"
        className={`app-shell ${shellTone} ${isDetachedWindow ? "detached-shell" : ""}`}
        initial={{ opacity: 0, y: 14 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -10 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
      >
      {!isDetachedWindow ? (
        <ShellTopbar
          user={user}
          showDetachLive={showDetachLive}
          showDetachMap={showDetachMap}
          canConnect={canConnect}
          busy={busy}
          connected={status.connected}
          onDetachLive={() => void handleDetachWindow("viewer-live")}
          onDetachMap={() => void handleDetachWindow("viewer-map")}
          onConnectToggle={handleConnectToggle}
        />
      ) : null}

      <div className={`workspace-shell ${isDetachedWindow ? "workspace-shell-detached" : ""}`}>
        {!isDetachedWindow ? (
          <ShellSidebar
            name={user.name}
            username={user.username}
            role={user.role}
            navGroups={navGroups}
            expandedSections={expandedSections}
            currentPage={page}
            onToggleSection={(section) =>
              setExpandedSections((current) => ({
                ...current,
                [section]: !current[section],
              }))
            }
            onSelectPage={setPage}
          />
        ) : null}

        <main className={`workspace-content ${isDetachedWindow ? "workspace-content-detached" : ""}`}>
          <PageHero title={pageTitle} description={pageDescription} error={error} />
          <div className="page-transition-layer">
            <AnimatePresence mode="wait" initial={false}>
              <motion.div
                key={page}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.18, ease: "easeOut" }}
              >
                <RenderPageContent
                  page={page}
                  user={user}
                  profile={profile}
                  status={status}
                  telemetry={telemetry}
                  liveTiming={liveTiming}
                  events={events}
                  userPermissions={userPermissions}
                  viewerPermissions={viewerPermissions}
                  controlPermissions={controlPermissions}
                  adminPermissions={adminPermissions}
                  liveBoardRows={liveBoardRows}
                  liveTitle={liveTitle}
                  liveTopInfo={liveTopInfo}
                  liveBottomMessage={liveBottomMessage}
                  trackBounds={trackBounds}
                  trackName={trackName}
                  trackPathCount={trackPathCount}
                  trackGpsPoints={trackGpsPoints}
                  raceLinks={raceLinks}
                  baseLinks={baseLinks}
                  sectorStates={sectorStates}
                  globalFlagLabel={globalFlagLabel}
                  adminPresence={adminPresence}
                  tmsConfig={tmsConfig}
                  tmsState={tmsState}
                  tmsStatus={tmsStatus}
                  busy={busy}
                  canSeeLogs={canSeeLogs}
                  isDetachedWindow={isDetachedWindow}
                  handleDetachWindow={handleDetachWindow}
                  handleSetGlobalFlag={handleSetGlobalFlag}
                  handleSetSectorFlag={handleSetSectorFlag}
                  handleSendTmsCan={handleSendTmsCan}
                  updateTmsState={updateTmsState}
                  handleSendTmsManual={handleSendTmsManual}
                  handleDisconnectDesktopSession={handleDisconnectDesktopSession}
                />
              </motion.div>
            </AnimatePresence>
          </div>
        </main>
      </div>
      </motion.div>
    ) : null;

  return (
    <AnimatePresence mode="wait">
      {stage === "boot" ? <StartupScreen key="boot" shellTone={shellTone} /> : null}
      {stage === "workspace" ? workspaceContent : null}
    </AnimatePresence>
  );
}

export default function App() {
  return (
    <AppErrorBoundary>
      <AppShell />
    </AppErrorBoundary>
  );
}
