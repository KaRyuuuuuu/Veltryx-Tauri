import type { WorkspacePageProps } from "../../types/app";
import { formatDateTime } from "../../utils/app-helpers";

export default function ViewerLivePage({ telemetry, liveTitle, liveTopInfo, liveBottomMessage, liveBoardRows }: WorkspacePageProps) {
  const leaders = liveBoardRows.slice(0, 3);
  const liveCount = liveBoardRows.length;
  const latestTick = formatDateTime(telemetry?.lastAt);

  return (
    <section className="grid-1">
      <article className="panel live-board-panel veltryx-live-panel">
        <div className="veltryx-live-hero">
          <div className="veltryx-live-heading">
            <span className="veltryx-live-kicker">Veltryx live control</span>
            <div>
              <p className="eyebrow">Live timing</p>
              <h3>{liveTitle}</h3>
            </div>
            <p className="veltryx-live-summary">{liveTopInfo}</p>
          </div>
          <div className="veltryx-live-meta">
            <div className="veltryx-live-meta-card">
              <span>Last sync</span>
              <strong>{latestTick}</strong>
            </div>
            <div className="veltryx-live-meta-card">
              <span>Cars classified</span>
              <strong>{liveCount}</strong>
            </div>
          </div>
        </div>

        {leaders.length ? (
          <div className="veltryx-live-podium">
            {leaders.map((row, index) => (
              <div className={`veltryx-live-podium-card rank-${index + 1}`} key={`${row.pos}-${row.team}-${row.num}`}>
                <div className="veltryx-live-rank">{row.pos}</div>
                <div>
                  <strong>{row.team}</strong>
                  <p>
                    #{row.num} • {row.car} • {row.className}
                  </p>
                </div>
                <span>{row.bestLap || row.lastLap || "-"}</span>
              </div>
            ))}
          </div>
        ) : null}

        <div className="live-board-topbar live-board-topbar-veltryx">
          <div>
            <p className="eyebrow">Live timing</p>
            <h3>Race feed</h3>
          </div>
          <div className="live-board-clock">{latestTick}</div>
        </div>

        <div className="live-board-info veltryx-live-info">{liveTopInfo}</div>
        <div className="live-board-message">{liveBottomMessage}</div>

        <div className="leaderboard-shell veltryx-leaderboard-shell">
          <div className="leaderboard-header">
            <span>POS</span>
            <span>NOW</span>
            <span>NUM</span>
            <span>TEAM</span>
            <span>CAR</span>
            <span>CLASS</span>
            <span>LAPS</span>
            <span>LAST LAP</span>
            <span>BEST LAP</span>
            <span>S1</span>
            <span>S2</span>
            <span>S3</span>
            <span>GAP</span>
            <span>INFO</span>
          </div>

          {liveBoardRows.length ? (
            <div className="leaderboard-body">
              {liveBoardRows.map((row) => (
                <div className="leaderboard-row" key={`${row.pos}-${row.team}-${row.info}`}>
                  <span className="leaderboard-pos">{row.pos}</span>
                  <span className="leaderboard-now">{row.now}</span>
                  <span className="leaderboard-num">{row.num}</span>
                  <span className="leaderboard-team">{row.team}</span>
                  <span>{row.car}</span>
                  <span>{row.className}</span>
                  <span>{row.laps}</span>
                  <span>{row.lastLap}</span>
                  <span className="leaderboard-best">{row.bestLap}</span>
                  <span>{row.s1}</span>
                  <span>{row.s2}</span>
                  <span>{row.s3}</span>
                  <span>{row.gap}</span>
                  <span className="leaderboard-info">{row.info}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="leaderboard-empty">Waiting for classification data</div>
          )}
        </div>
      </article>
    </section>
  );
}
