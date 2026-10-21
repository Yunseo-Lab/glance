import { useEffect, useState } from "react";
import {
  ArrowUpRight,
  ChevronLeft,
  ChevronRight,
  CloudSun,
  ExternalLink,
  LoaderCircle,
  RefreshCw,
} from "lucide-react";
import { request, ApiError } from "./types";
import type { FeedData, Snapshot, WeatherData, Widget } from "./types";

function safeURL(value: string) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" && !url.username && !url.password
      ? url.href
      : undefined;
  } catch {
    return undefined;
  }
}

function Calendar({ now }: { now: Date }) {
  const [offset, setOffset] = useState(0);
  const month = new Date(now.getFullYear(), now.getMonth() + offset, 1);
  const year = month.getFullYear(),
    index = month.getMonth();
  const leading = (month.getDay() + 6) % 7;
  const days = new Date(year, index + 1, 0).getDate();
  return (
    <div className="calendar-view">
      <div className="calendar-heading">
        <strong>
          {year}년 {index + 1}월
        </strong>
        <div>
          <button
            className="icon-button"
            aria-label="이전 달"
            onClick={() => setOffset(offset - 1)}
          >
            <ChevronLeft size={15} />
          </button>
          <button
            className="icon-button"
            aria-label="다음 달"
            onClick={() => setOffset(offset + 1)}
          >
            <ChevronRight size={15} />
          </button>
        </div>
      </div>
      <div className="calendar-grid">
        {["월", "화", "수", "목", "금", "토", "일"].map((day) => (
          <span className="weekday" key={day}>
            {day}
          </span>
        ))}
        {Array.from({ length: leading }, (_, i) => (
          <span key={`blank-${i}`} />
        ))}
        {Array.from({ length: days }, (_, i) => {
          const day = i + 1,
            today =
              year === now.getFullYear() &&
              index === now.getMonth() &&
              day === now.getDate();
          return (
            <span
              key={day}
              className={today ? "today" : ""}
              aria-current={today ? "date" : undefined}
            >
              {day}
            </span>
          );
        })}
      </div>
    </div>
  );
}

interface WidgetProps {
  widget: Widget;
  snapshot?: Snapshot;
  now: Date;
}
const renderers: Record<
  Widget["type"],
  (props: WidgetProps) => React.ReactNode
> = {
  clock: ({ widget, snapshot, now }) => {
    return (
      <div className="clock-view">
        <span className="eyebrow">
          {widget.config.timezone?.replaceAll("_", " ")}
        </span>
        <div className="clock-time">
          {new Intl.DateTimeFormat("en-GB", {
            timeZone: widget.config.timezone,
            hour: "2-digit",
            minute: "2-digit",
            hourCycle: "h23",
          }).format(now)}
          <span>
            {new Intl.DateTimeFormat("en-GB", {
              timeZone: widget.config.timezone,
              second: "2-digit",
            }).format(now)}
          </span>
        </div>
        <p>
          {new Intl.DateTimeFormat("ko-KR", {
            timeZone: widget.config.timezone,
            month: "long",
            day: "numeric",
            weekday: "long",
          }).format(now)}
        </p>
        <div className="clock-decoration" aria-hidden="true">
          <span />
          <span />
          <span />
          <span />
          <span />
          <span />
          <span />
          <span />
        </div>
      </div>
    );
  },
  calendar: ({ widget, snapshot, now }) => {
    return <Calendar now={now} />;
  },
  note: ({ widget, snapshot, now }) => {
    return (
      <div className="note-view">
        <p>{widget.config.body || "편집에서 첫 메모를 남겨보세요."}</p>
        <span className="note-mark" aria-hidden="true">
          ✳
        </span>
      </div>
    );
  },
  bookmarks: ({ widget, snapshot, now }) => {
    return (
      <div className="bookmark-list">
        {widget.config.links?.length ? (
          widget.config.links.map((link, i) => (
            <a
              key={i}
              href={safeURL(link.url)}
              target="_blank"
              rel="noopener noreferrer"
            >
              <span className="bookmark-letter">{link.title.slice(0, 1)}</span>
              <span>
                <strong>{link.title}</strong>
                <small>
                  {(() => {
                    try {
                      return new URL(link.url).hostname;
                    } catch {
                      return "";
                    }
                  })()}
                </small>
              </span>
              <ArrowUpRight size={16} />
            </a>
          ))
        ) : (
          <p className="muted">편집에서 즐겨찾기를 추가하세요.</p>
        )}
      </div>
    );
  },
  rss: ({ widget, snapshot, now }) => {
    {
      const feed = snapshot?.data as FeedData | undefined;
      return (
        <>
          <RemoteStatus snapshot={snapshot} />
          {feed && (
            <div className="feed-list">
              {feed.items.length ? (
                feed.items.slice(0, widget.config.limit || 5).map((item, i) => (
                  <a
                    key={`${item.url}-${i}`}
                    href={safeURL(item.url)}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <span className="feed-number">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <span>
                      <strong>{item.title}</strong>
                      <small>
                        {feed.title}
                        {item.publishedAt &&
                          ` · ${new Date(item.publishedAt).toLocaleDateString("ko-KR", { month: "short", day: "numeric" })}`}
                      </small>
                    </span>
                    <ExternalLink size={14} />
                  </a>
                ))
              ) : (
                <p className="muted">아직 게시물이 없습니다.</p>
              )}
            </div>
          )}
          <Updated snapshot={snapshot} sourceID={widget.sourceId} />
        </>
      );
    }
  },
  weather: ({ widget, snapshot, now }) => {
    {
      const weather = snapshot?.data as WeatherData | undefined;
      const description = weather
        ? weather.code === 0
          ? "맑음"
          : weather.code <= 3
            ? "구름 조금"
            : weather.code <= 48
              ? "흐림"
              : weather.code <= 67
                ? "비"
                : weather.code <= 77
                  ? "눈"
                  : weather.code <= 86
                    ? "소나기"
                    : "뇌우"
        : "";
      return (
        <>
          <RemoteStatus snapshot={snapshot} />
          {weather && (
            <div className="weather-view">
              <div>
                <span className="eyebrow">{weather.location}</span>
                <strong>
                  {Math.round(weather.temperature)}°<span>C</span>
                </strong>
                <p>{description}</p>
              </div>
              <CloudSun size={68} strokeWidth={1.3} />
              <div className="weather-details">
                <span>체감 {Math.round(weather.feelsLike)}°</span>
                <span>습도 {weather.humidity}%</span>
              </div>
            </div>
          )}
          <Updated snapshot={snapshot} sourceID={widget.sourceId} />
        </>
      );
    }
  },
};
export default function WidgetView(props: WidgetProps) {
  return renderers[props.widget.type](props);
}

function RemoteStatus({ snapshot }: { snapshot?: Snapshot }) {
  if (snapshot?.state === "paused")
    return <p className="source-notice">자동 갱신이 일시정지되었습니다.</p>;
  if (snapshot?.data)
    return snapshot.error ? (
      <p className="source-notice">갱신이 지연되어 마지막 정보를 표시합니다.</p>
    ) : null;
  if (snapshot?.state === "error")
    return (
      <div className="source-empty">
        <CloudSun size={28} />
        <p>{snapshot.error}</p>
        <small>연결 상태를 확인하거나 위젯 설정을 변경하세요.</small>
      </div>
    );
  return (
    <div className="source-empty" role="status">
      <LoaderCircle className="spin" size={24} />
      <p>소식을 가져오고 있어요</p>
      <small>다른 위젯은 바로 사용할 수 있습니다.</small>
    </div>
  );
}
function Updated({
  snapshot,
  sourceID,
}: {
  snapshot?: Snapshot;
  sourceID?: string;
}) {
  return (
    <div className="source-footer">
      {sourceID && (
        <ManualRefresh
          sourceID={sourceID}
          paused={snapshot?.state === "paused"}
        />
      )}
      {snapshot?.updatedAt && (
        <p className="updated">
          <span
            className={
              snapshot.state === "fresh" ? "status-dot" : "status-dot stale"
            }
          />
          {snapshot.state === "fresh" ? "업데이트" : "마지막 업데이트"} ·{" "}
          {new Date(snapshot.updatedAt).toLocaleTimeString("ko-KR", {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      )}
    </div>
  );
}

function ManualRefresh({
  sourceID,
  paused,
}: {
  sourceID: string;
  paused?: boolean;
}) {
  const [busy, setBusy] = useState(false),
    [until, setUntil] = useState(0),
    [tick, setTick] = useState(Date.now()),
    [error, setError] = useState("");
  useEffect(() => {
    if (!until) return;
    const timer = window.setInterval(() => {
      const now = Date.now();
      setTick(now);
      if (now >= until) setUntil(0);
    }, 1000);
    return () => clearInterval(timer);
  }, [until]);
  const seconds = Math.max(0, Math.ceil((until - tick) / 1000));
  async function refresh() {
    setBusy(true);
    setError("");
    try {
      const receipt = await request<{ nextAllowedAt: string }>(
        `/api/v1/sources/${encodeURIComponent(sourceID)}/refresh`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: "{}",
        },
      );
      setTick(Date.now());
      setUntil(Date.parse(receipt.nextAllowedAt));
      window.dispatchEvent(new Event("glance:refresh"));
    } catch (err) {
      const failure = err as ApiError;
      setError(failure.message);
      if (failure.status === 429) {
        setTick(Date.now());
        setUntil(Date.now() + (failure.retryAfter || 30) * 1000);
      }
    } finally {
      setBusy(false);
    }
  }
  return (
    <span className="manual-refresh">
      <button
        type="button"
        className="icon-button"
        aria-label={seconds ? `새로고침 (${seconds}초 대기)` : "새로고침"}
        disabled={paused || busy || seconds > 0}
        onClick={refresh}
      >
        <RefreshCw size={13} className={busy ? "spin" : ""} />
        {seconds > 0 && <small>{seconds}초</small>}
      </button>
      {error && <small role="status">{error}</small>}
    </span>
  );
}
