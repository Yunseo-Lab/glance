import type { components } from "./generated/api";
import { useState } from "react";
import {
  ArrowUpRight,
  ChevronLeft,
  ChevronRight,
  CloudSun,
  ExternalLink,
  LoaderCircle,
  RefreshCw,
} from "lucide-react";
import { RemoteStatus, Updated } from "./workspace/SourcePresentation";
import { safeExternalURL as safeURL } from "./links";
import type { FeedData, Snapshot, WeatherData, Widget } from "./types";

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
  json: ({ widget, snapshot }) => {
    const data = snapshot?.data as
      | components["schemas"]["JSONValueData"]
      | undefined;
    return (
      <>
        <RemoteStatus snapshot={snapshot} />
        {data && (
          <div className="json-value">
            <strong>{data.value}</strong>
            {widget.config.unit && <small>{widget.config.unit}</small>}
          </div>
        )}
        <Updated snapshot={snapshot} sourceID={widget.sourceId} />
      </>
    );
  },
  agenda: ({ widget, snapshot }) => {
    const data = snapshot?.data as
      | components["schemas"]["CalendarData"]
      | undefined;
    return (
      <>
        <RemoteStatus snapshot={snapshot} />
        {data && (
          <div className="agenda-list">
            {data.warnings.map((warning, index) => (
              <p className="source-notice" role="status" key={index}>
                {warning}
              </p>
            ))}
            {data.events.slice(0, widget.config.limit || 5).map((event) => (
              <div className="agenda-event" key={event.id}>
                <time dateTime={event.start}>
                  {new Intl.DateTimeFormat("ko-KR", {
                    timeZone: event.timezone,
                    month: "short",
                    day: "numeric",
                    ...(event.allDay
                      ? {}
                      : { hour: "2-digit", minute: "2-digit" }),
                  }).format(new Date(event.start))}
                  {event.allDay ? " · 종일" : ` · ${event.timezone}`}
                </time>
                <strong>
                  {event.url ? (
                    <a
                      href={safeURL(event.url)}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {event.title}
                    </a>
                  ) : (
                    event.title
                  )}
                </strong>
                {event.location && <small>{event.location}</small>}
              </div>
            ))}
            {data.events.length === 0 && (
              <p className="muted">다가오는 일정이 없습니다.</p>
            )}
          </div>
        )}
        <Updated snapshot={snapshot} sourceID={widget.sourceId} />
      </>
    );
  },
  github: (props) => renderers.rss(props),
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
        ? weather.code == null
          ? "날씨 상태 정보 없음"
          : weather.code === 0
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
                  {Math.round(weather.temperature)}°
                  <span>{weather.unit === "fahrenheit" ? "F" : "C"}</span>
                </strong>
                <p>{description}</p>
              </div>
              <CloudSun size={68} strokeWidth={1.3} />
              <div className="weather-details">
                <span>
                  {weather.feelsLike == null
                    ? "체감 정보 없음"
                    : `체감 ${Math.round(weather.feelsLike)}°`}
                </span>
                <span>
                  {weather.humidity == null
                    ? "습도 정보 없음"
                    : `습도 ${weather.humidity}%`}
                </span>
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
