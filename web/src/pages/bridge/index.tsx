import { useEffect, useMemo, useRef, useState } from "react";
import type { WorkspacePageProps } from "../../types/app";
import { flagLabel } from "../../utils/app-helpers";

const BRIDGE_TARGET_URL = "wss://api-veltryx-1.karyuu.be/ws/bridge";
const BRIDGE_SHARED_KEY_B64 = import.meta.env.VITE_BRIDGE_SHARED_KEY_B64 ?? "";
const BRIDGE_MIN_SYNC_INTERVAL_MS = 1;

type BridgeTrackPayload = NonNullable<NonNullable<WorkspacePageProps["telemetry"]>["track"]>;

type BridgePlainPayload = {
  schema: "veltryx.bridge.flags.v1";
  source: "veltryx-tauri";
  user: string;
  sentAt: string;
  connection: {
    connected: boolean;
    state: string;
  };
  globalFlag: {
    code: number;
    label: string;
  };
  sectors: Array<{
    id: number;
    name: string;
    flag: {
      code: number;
      label: string;
    };
  }>;
  track?: BridgeTrackPayload;
  gpsLatest?: Array<{
    racelinkId?: number;
    lat?: number;
    lon?: number;
    speed?: number;
    timestamp?: number;
    vehicle?: string;
  }>;
  raceLinks?: Array<{
    id: number;
    name?: string;
    carNumber?: number;
    flag?: number;
    status?: string;
    baseLink?: number;
  }>;
};

type BridgeEncryptedEnvelope = {
  version: 1;
  agentId: string;
  sentAt: string;
  nonce: string;
  ciphertext: string;
};

type BridgeAck = {
  ok?: boolean;
  error?: string;
  receivedAt?: string;
  schema?: string;
  agentId?: string;
};

type BridgeSyncState = {
  connection: {
    connected: boolean;
    state: string;
  };
  globalFlag: number;
  sectors: Array<{
    id: number;
    name: string;
    flag: number;
  }>;
  track: BridgeTrackPayload | null;
  gpsLatest: Array<{
    racelinkId?: number;
    lat?: number;
    lon?: number;
    speed?: number;
    timestamp?: number;
    vehicle?: string;
  }>;
  raceLinks: Array<{
    id: number;
    name?: string;
    carNumber?: number;
    flag?: number;
    status?: string;
    baseLink?: number;
  }>;
};

function toBase64(buffer: ArrayBuffer | Uint8Array) {
  const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return window.btoa(binary);
}

function fromBase64(value: string) {
  const binary = window.atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

async function importBridgeKey() {
  if (!BRIDGE_SHARED_KEY_B64) {
    throw new Error("VITE_BRIDGE_SHARED_KEY_B64 manquant");
  }

  const rawKey = fromBase64(BRIDGE_SHARED_KEY_B64);
  return window.crypto.subtle.importKey("raw", rawKey, "AES-GCM", false, ["encrypt"]);
}

async function encryptBridgePayload(payload: BridgePlainPayload, agentId: string): Promise<BridgeEncryptedEnvelope> {
  const key = await importBridgeKey();
  const nonce = window.crypto.getRandomValues(new Uint8Array(12));
  const plaintext = new TextEncoder().encode(JSON.stringify(payload));
  const ciphertext = await window.crypto.subtle.encrypt({ name: "AES-GCM", iv: nonce }, key, plaintext);

  return {
    version: 1,
    agentId,
    sentAt: payload.sentAt,
    nonce: toBase64(nonce),
    ciphertext: toBase64(ciphertext),
  };
}

export default function BridgePage({ status, telemetry, sectorStates, trackGpsPoints, raceLinks, user }: WorkspacePageProps) {
  const socketRef = useRef<WebSocket | null>(null);
  const retryTimerRef = useRef<number | null>(null);
  const syncTimerRef = useRef<number | null>(null);
  const lastSentSignatureRef = useRef("");
  const lastDispatchAtRef = useRef(0);
  const [socketState, setSocketState] = useState<"connecting" | "open" | "disconnected">("connecting");
  const [syncState, setSyncState] = useState<"pending" | "synced" | "error">("pending");
  const [lastSentAt, setLastSentAt] = useState("");
  const [lastAckAt, setLastAckAt] = useState("");
  const [lastError, setLastError] = useState("");
  const [lastPayload, setLastPayload] = useState("");
  const [sending, setSending] = useState(false);

  const agentId = `veltryx-tauri-${user.username}`;

  const syncStatePayload = useMemo<BridgeSyncState>(() => {
    const normalizedSectors = [...sectorStates]
      .map((sector) => ({
        id: sector.id,
        name: sector.name || `S ${sector.id}`,
        flag: sector.flag ?? 0,
      }))
      .sort((left, right) => left.id - right.id);

    return {
      connection: {
        connected: status.connected,
        state: status.state,
      },
      globalFlag: telemetry?.globalFlag ?? 0,
      sectors: normalizedSectors,
      track: telemetry?.track ?? null,
      gpsLatest: [...trackGpsPoints]
        .filter((point) => typeof point.lat === "number" && typeof point.lon === "number")
        .map((point) => ({
          racelinkId: point.racelinkId,
          lat: point.lat,
          lon: point.lon,
          speed: point.speed,
          timestamp: point.timestamp,
          vehicle: point.vehicle,
        }))
        .sort((left, right) => (left.racelinkId ?? 0) - (right.racelinkId ?? 0)),
      raceLinks: [...raceLinks]
        .map((item) => ({
          id: item.id,
          name: item.name,
          carNumber: item.carNumber,
          flag: item.flag,
          status: item.status,
          baseLink: item.baseLink,
        }))
        .sort((left, right) => left.id - right.id),
    };
  }, [sectorStates, status.connected, status.state, telemetry?.globalFlag, telemetry?.track, trackGpsPoints, raceLinks]);

  const syncSignature = useMemo(() => JSON.stringify(syncStatePayload), [syncStatePayload]);

  const plainPayload = useMemo<BridgePlainPayload>(() => {
    const sentAt = new Date().toISOString();
    return {
      schema: "veltryx.bridge.flags.v1",
      source: "veltryx-tauri",
      user: user.username,
      sentAt,
      connection: syncStatePayload.connection,
      globalFlag: {
        code: syncStatePayload.globalFlag,
        label: flagLabel(syncStatePayload.globalFlag),
      },
      sectors: syncStatePayload.sectors.map((sector) => ({
        id: sector.id,
        name: sector.name,
        flag: {
          code: sector.flag,
          label: flagLabel(sector.flag),
        },
      })),
      track: syncStatePayload.track ?? undefined,
      gpsLatest: syncStatePayload.gpsLatest,
      raceLinks: syncStatePayload.raceLinks,
    };
  }, [syncStatePayload, user.username]);

  useEffect(() => {
    let cancelled = false;

    const clearRetry = () => {
      if (retryTimerRef.current !== null) {
        window.clearTimeout(retryTimerRef.current);
        retryTimerRef.current = null;
      }
      if (syncTimerRef.current !== null) {
        window.clearTimeout(syncTimerRef.current);
        syncTimerRef.current = null;
      }
    };

    const connect = () => {
      if (cancelled) {
        return;
      }

      clearRetry();
      setSocketState("connecting");

      const socket = new WebSocket(BRIDGE_TARGET_URL);
      socketRef.current = socket;

      socket.onopen = () => {
        if (cancelled) {
          socket.close();
          return;
        }
        setSocketState("open");
        setSyncState("pending");
        setLastError("");
      };

      socket.onmessage = (event) => {
        try {
          const ack = JSON.parse(event.data) as BridgeAck;
          if (ack.ok) {
            setLastAckAt(ack.receivedAt ?? new Date().toISOString());
            setSyncState("synced");
            setLastError("");
            return;
          }
          if (ack.error) {
            setSyncState("error");
            setLastError(ack.error);
          }
        } catch {
          setSyncState("error");
          setLastError("Ack bridge illisible");
        }
      };

      socket.onerror = () => {
        setSyncState("error");
        setLastError("Connexion bridge impossible");
      };

      socket.onclose = () => {
        if (cancelled) {
          return;
        }
        setSocketState("disconnected");
        setSyncState("error");
        retryTimerRef.current = window.setTimeout(connect, 1500);
      };
    };

    connect();

    return () => {
      cancelled = true;
      clearRetry();
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, []);

  async function sendCurrentPayload(force = false) {
    const socket = socketRef.current;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error("Le bridge VPS n'est pas connecte");
    }
    if (!force && lastSentSignatureRef.current === syncSignature) {
      return;
    }

    setSyncState("pending");
    const encrypted = await encryptBridgePayload(plainPayload, agentId);
    const serialized = JSON.stringify(encrypted, null, 2);
    socket.send(serialized);
    lastDispatchAtRef.current = Date.now();
    lastSentSignatureRef.current = syncSignature;
    setLastPayload(serialized);
    setLastSentAt(plainPayload.sentAt);
  }

  function queueSync(force = false) {
    if (syncTimerRef.current !== null) {
      window.clearTimeout(syncTimerRef.current);
      syncTimerRef.current = null;
    }

    const elapsed = Date.now() - lastDispatchAtRef.current;
    const delay = force || elapsed >= BRIDGE_MIN_SYNC_INTERVAL_MS ? 0 : BRIDGE_MIN_SYNC_INTERVAL_MS - elapsed;

    syncTimerRef.current = window.setTimeout(() => {
      syncTimerRef.current = null;
      void sendCurrentPayload(force).catch((error) => {
        setSyncState("error");
        setLastError(String(error));
      });
    }, delay);
  }

  useEffect(() => {
    if (socketState !== "open") {
      return;
    }

    queueSync();
  }, [socketState, syncSignature, plainPayload]);

  async function handleManualSend() {
    setSending(true);
    try {
      await sendCurrentPayload(true);
      setLastError("");
    } catch (error) {
      setSyncState("error");
      setLastError(String(error));
    } finally {
      setSending(false);
    }
  }

  return (
    <section className="grid-3">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Bridge VPS</p>
            <h3>Flags and sectors sync</h3>
            <p className="panel-copy">
              La page bridge pousse uniquement les drapeaux globaux et les etats de secteurs, deja formates pour le VPS
              et chiffrés en AES-GCM.
            </p>
          </div>
          <button className="primary-button" onClick={() => void handleManualSend()} disabled={sending || socketState !== "open"}>
            {sending ? "Envoi..." : "Envoyer maintenant"}
          </button>
        </div>
        <div className="stats-grid">
          <div className="stat-card">
            <span>Bridge socket</span>
            <strong>{socketState}</strong>
          </div>
          <div className="stat-card">
            <span>Bridge sync</span>
            <strong>{syncState}</strong>
          </div>
          <div className="stat-card">
            <span>Sectors</span>
            <strong>{plainPayload.sectors.length}</strong>
          </div>
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Bridge target</p>
            <h3>Remote endpoint</h3>
          </div>
        </div>
        <div className="bridge-meta-list">
          <div className="stack-row">
            <span>WebSocket</span>
            <strong>{BRIDGE_TARGET_URL}</strong>
          </div>
          <div className="stack-row">
            <span>Agent ID</span>
            <strong>{agentId}</strong>
          </div>
          <div className="stack-row">
            <span>Last send</span>
            <strong>{lastSentAt || "Aucun envoi"}</strong>
          </div>
          <div className="stack-row">
            <span>Last ack</span>
            <strong>{lastAckAt || "Aucun ack"}</strong>
          </div>
        </div>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Security</p>
            <h3>Transport state</h3>
          </div>
        </div>
        <ul className="plain-list">
          <li>Transport: WSS</li>
          <li>Payload: AES-GCM</li>
          <li>Key source: Vite env `VITE_BRIDGE_SHARED_KEY_B64`</li>
          <li>Server ack: {lastAckAt || "Aucun"}</li>
          <li>Error: {lastError || "Aucune"}</li>
        </ul>
      </article>
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Encrypted envelope</p>
            <h3>Last pushed payload</h3>
          </div>
        </div>
        <pre className="raw-block">{lastPayload || "Aucun payload chiffre envoye pour le moment."}</pre>
      </article>
      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Formatted preview</p>
            <h3>Decrypted business data</h3>
          </div>
        </div>
        <pre className="raw-block">{JSON.stringify(plainPayload, null, 2)}</pre>
      </article>
    </section>
  );
}
