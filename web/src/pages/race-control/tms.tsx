import { useMemo } from "react";
import { EmptyState } from "../../components/EmptyState";
import type { WorkspacePageProps } from "../../types/app";
import {
  buildFrontDelta,
  buildTmsLeaderboard,
  encodeTmsPayload,
  formatLap,
  resolveLapMetric,
} from "../../utils/tms";

export default function RaceControlTmsPage({
  raceLinks,
  liveTiming,
  tmsConfig,
  tmsState,
  tmsStatus,
  updateTmsState,
  handleSendTmsManual,
}: WorkspacePageProps) {
  const leaderboard = useMemo(() => buildTmsLeaderboard(liveTiming), [liveTiming]);

  const selectedIndex = leaderboard.findIndex((entry) => entry.carNumber === tmsState.selectedCar);
  const selectedEntry = selectedIndex >= 0 ? leaderboard[selectedIndex] : null;
  const aheadEntry = selectedIndex > 0 ? leaderboard[selectedIndex - 1] : null;
  const behindEntry =
    selectedIndex >= 0 && selectedIndex < leaderboard.length - 1 ? leaderboard[selectedIndex + 1] : null;

  const matchedRaceLink = useMemo(() => {
    if (!selectedEntry) return null;
    return raceLinks.find((item) => item.carNumber === Number(selectedEntry.carNumber)) ?? null;
  }, [raceLinks, selectedEntry]);

  const resolvedPrimaryCanId =
    matchedRaceLink?.id ?? (tmsState.primaryCanId ? Number(tmsState.primaryCanId) : tmsConfig?.primaryCanId ?? null);
  const resolvedTestCanId =
    (tmsState.testCanId ? Number(tmsState.testCanId) : null) ?? tmsConfig?.testCanId ?? null;

  const payloadBytes = useMemo(
    () =>
      encodeTmsPayload(
        Number(tmsState.deltaFront) || 0,
        tmsState.frontColor,
        Number(tmsState.deltaRear) || 0,
        tmsState.rearColor,
        Number(tmsState.position) || 0,
        Number(tmsState.lapTime) || 0,
        tmsState.lapColor,
      ),
    [tmsState],
  );

  const testPayloadBytes = useMemo(() => {
    if (!tmsState.useCarOnTestRear) return payloadBytes;
    return encodeTmsPayload(
      Number(tmsState.deltaFront) || 0,
      tmsState.frontColor,
      Number(selectedEntry?.carNumber || 0),
      "yellow",
      Number(tmsState.position) || 0,
      Number(tmsState.lapTime) || 0,
      tmsState.lapColor,
    );
  }, [payloadBytes, selectedEntry?.carNumber, tmsState]);

  const linkedRaceLinks = useMemo(
    () =>
      raceLinks.filter((item) =>
        [resolvedPrimaryCanId, resolvedTestCanId].some((candidate) => candidate && item.id === candidate),
      ),
    [raceLinks, resolvedPrimaryCanId, resolvedTestCanId],
  );

  const currentFront = selectedEntry ? buildFrontDelta(selectedEntry, aheadEntry) : 0;
  const currentRear = behindEntry ? Math.trunc(resolveLapMetric(behindEntry) / 100) : 0;

  return (
    <section className="grid-2">
      <article className="panel panel-large">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Race Control</p>
            <h3>TMS race display</h3>
          </div>
        </div>

        <div className="tms-control-grid">
          <label className="tms-field">
            <span>Voiture</span>
            <select
              value={tmsState.selectedCar}
              onChange={(event) => updateTmsState({ selectedCar: event.target.value })}
            >
              {leaderboard.map((entry) => (
                <option key={entry.carNumber} value={entry.carNumber}>
                  P{entry.position} #{entry.carNumber} {entry.team}
                </option>
              ))}
            </select>
          </label>

          <label className="tms-field">
            <span>CAN ID</span>
            <input
              value={tmsState.canId}
              onChange={(event) => updateTmsState({ canId: event.target.value })}
              placeholder="514"
            />
          </label>

          <div className="tms-actions">
            <button
              className={tmsState.realtimeEnabled ? "primary-button" : "ghost-button"}
              onClick={() => updateTmsState({ realtimeEnabled: !tmsState.realtimeEnabled })}
              disabled={!selectedEntry}
            >
              {tmsState.realtimeEnabled ? "Realtime Car ON" : "Realtime Car OFF"}
            </button>
            <button
              className={tmsState.autoAllEnabled ? "primary-button" : "ghost-button"}
              onClick={() => updateTmsState({ autoAllEnabled: !tmsState.autoAllEnabled })}
              disabled={!leaderboard.length}
            >
              {tmsState.autoAllEnabled ? "Auto All ON" : "Auto All OFF"}
            </button>
          </div>

          <label className="tms-field">
            <span>Racelink secours</span>
            <input
              value={tmsState.primaryCanId}
              onChange={(event) => updateTmsState({ primaryCanId: event.target.value })}
              placeholder="uniquement si voiture non liee"
            />
          </label>

          <label className="tms-field">
            <span>Racelink test</span>
            <input
              value={tmsState.testCanId}
              onChange={(event) => updateTmsState({ testCanId: event.target.value })}
              placeholder="ex: 225609"
            />
          </label>

          <label className="tms-field">
            <span>Delta front</span>
            <input
              value={tmsState.deltaFront}
              onChange={(event) => updateTmsState({ deltaFront: event.target.value })}
            />
          </label>

          <label className="tms-field">
            <span>Couleur front</span>
            <select
              value={tmsState.frontColor}
              onChange={(event) => updateTmsState({ frontColor: event.target.value as typeof tmsState.frontColor })}
            >
              <option value="yellow">Yellow</option>
              <option value="white">White</option>
              <option value="green">Green</option>
              <option value="purple">Purple</option>
            </select>
          </label>

          <label className="tms-field">
            <span>Delta rear</span>
            <input
              value={tmsState.deltaRear}
              onChange={(event) => updateTmsState({ deltaRear: event.target.value })}
            />
          </label>

          <label className="tms-field">
            <span>Couleur rear</span>
            <select
              value={tmsState.rearColor}
              onChange={(event) => updateTmsState({ rearColor: event.target.value as typeof tmsState.rearColor })}
            >
              <option value="yellow">Yellow</option>
              <option value="white">White</option>
              <option value="green">Green</option>
              <option value="purple">Purple</option>
            </select>
          </label>

          <label className="tms-field">
            <span>Position</span>
            <input value={tmsState.position} onChange={(event) => updateTmsState({ position: event.target.value })} />
          </label>

          <label className="tms-field">
            <span>Lap time (ms)</span>
            <input value={tmsState.lapTime} onChange={(event) => updateTmsState({ lapTime: event.target.value })} />
          </label>

          <label className="tms-field">
            <span>Couleur lap</span>
            <select
              value={tmsState.lapColor}
              onChange={(event) => updateTmsState({ lapColor: event.target.value as typeof tmsState.lapColor })}
            >
              <option value="yellow">Yellow</option>
              <option value="white">White</option>
              <option value="green">Green</option>
              <option value="purple">Purple</option>
            </select>
          </label>

          <label className="tms-toggle">
            <input
              type="checkbox"
              checked={tmsState.useCarOnTestRear}
              onChange={(event) => updateTmsState({ useCarOnTestRear: event.target.checked })}
            />
            <span>Sur le Racelink test, mettre le numero de voiture dans le champ rear comme dans l'ancien soft</span>
          </label>

          <div className="tms-actions tms-field-full">
            <button
              className="primary-button"
              onClick={() => void handleSendTmsManual()}
              disabled={tmsStatus.sending || !selectedEntry}
            >
              {tmsStatus.sending ? "Envoi..." : "Envoyer vers les ecrans TMS"}
            </button>
          </div>
        </div>

        {selectedEntry ? (
          <div className="tms-preview-grid">
            <div className="device-card">
              <strong>Source live timing</strong>
              <p>
                P{selectedEntry.position} #{selectedEntry.carNumber} {selectedEntry.team}
              </p>
              <p>
                Front passage calcule: {currentFront} | Rear = dernier tour voiture derriere: {currentRear} | Lap retenu:{" "}
                {formatLap(resolveLapMetric(selectedEntry))}
              </p>
              <p>
                Source voiture: lastLapMs={selectedEntry.lastLapMs || 0} | bestLapMs={selectedEntry.bestLapMs || 0} |
                resolved={resolveLapMetric(selectedEntry)}
              </p>
              <p>
                Source derriere: lastLapMs={behindEntry?.lastLapMs || 0} | bestLapMs={behindEntry?.bestLapMs || 0} |
                resolved={resolveLapMetric(behindEntry)}
              </p>
              <p>
                Realtime car: {tmsState.realtimeEnabled ? "ON" : "OFF"} | Auto all:{" "}
                {tmsState.autoAllEnabled ? "ON" : "OFF"} | Passage: {selectedEntry.passageMs || 0}
              </p>
              <p>
                Cible principale resolue: {resolvedPrimaryCanId ?? "-"} | Racelink voiture: {matchedRaceLink?.id ?? "-"} |
                Secours: {tmsState.primaryCanId || "-"}
              </p>
            </div>

            <div className="device-card">
              <strong>Payload principal</strong>
              <p>[{payloadBytes.join(", ")}]</p>
              <p>
                {JSON.stringify({
                  racelink: { action: "sendcan", id: resolvedPrimaryCanId, canid: Number(tmsState.canId) || 514, data: payloadBytes },
                })}
              </p>
            </div>

            <div className="device-card">
              <strong>Payload test</strong>
              <p>[{testPayloadBytes.join(", ")}]</p>
              <p>
                {JSON.stringify({
                  racelink: { action: "sendcan", id: resolvedTestCanId, canid: Number(tmsState.canId) || 514, data: testPayloadBytes },
                })}
              </p>
            </div>
          </div>
        ) : (
          <EmptyState
            title="Aucune voiture"
            body="Le live timing ne remonte pas encore de classement exploitable pour preparer un message TMS."
          />
        )}

        {tmsStatus.feedback ? (
          <p className={tmsStatus.feedbackTone === "ok" ? "tms-feedback-ok" : "tms-feedback-error"}>
            {tmsStatus.feedback}
          </p>
        ) : null}
        {tmsStatus.sendLog.length ? (
          <div className="tms-log">
            {tmsStatus.sendLog.map((line, index) => (
              <p key={`${index}-${line}`}>{line}</p>
            ))}
          </div>
        ) : null}
      </article>

      <article className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Targets</p>
            <h3>Racelinks resolus</h3>
          </div>
        </div>

        {linkedRaceLinks.length ? (
          <div className="device-grid">
            {linkedRaceLinks.map((item) => (
              <div className="device-card" key={item.id}>
                <strong>RL {item.id}</strong>
                <span>{item.carNumber ? `#${item.carNumber}` : item.status || "active"}</span>
                <p>Flag {item.flag ?? 0} | RSSI {item.rssi || 0} | Batt {item.battery || 0}</p>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState
            title="Targets not visible"
            body="Les identifiants configures ne correspondent pas encore a des Racelinks visibles depuis X2, mais l'envoi CAN peut quand meme partir si les IDs sont bons."
          />
        )}

        <p className="panel-copy">
          Le paquet CAN suit le meme schema que l'ancien outil X2 TMS: <code>racelink -&gt; sendcan -&gt; canid 514 -&gt; data[8]</code>.
          La seule difference voulue ici est que le delta rear est alimente par le dernier tour de la voiture derriere.
        </p>
      </article>
    </section>
  );
}
