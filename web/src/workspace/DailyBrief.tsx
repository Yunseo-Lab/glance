import { useState } from "react";
import { CalendarDays, ExternalLink, RefreshCw } from "lucide-react";
import type { components } from "../generated/api";
import { safeExternalURL } from "../links";
import { useResource } from "./useResource";
function today(zone: string) {
  const parts = new Intl.DateTimeFormat("en", {
    timeZone: zone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  const value = (kind: string) =>
    parts.find((part) => part.type === kind)!.value;
  return `${value("year")}-${value("month")}-${value("day")}`;
}
export default function DailyBrief({ onRead }: { onRead: () => void }) {
  const [zone, setZone] = useState(
      () => Intl.DateTimeFormat().resolvedOptions().timeZone,
    ),
    [date, setDate] = useState(() =>
      today(Intl.DateTimeFormat().resolvedOptions().timeZone),
    ),
    [query, setQuery] = useState(() => ({
      date: today(Intl.DateTimeFormat().resolvedOptions().timeZone),
      zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    })),
    [formError, setFormError] = useState("");
  const { data, error, reload } = useResource<
    components["schemas"]["DailyBrief"]
  >(
    `/api/v1/brief?date=${encodeURIComponent(query.date)}&timezone=${encodeURIComponent(query.zone)}`,
  );
  return (
    <>
      <section className="page-heading">
        <div>
          <span className="eyebrow">
            <CalendarDays size={17} />
          </span>
          <h1>하루 브리핑</h1>
          <p>
            내 서버에 모인 소식을 날짜별로 정리하고 원문 출처를 함께 보여줍니다.
          </p>
        </div>
        <button
          className="icon-button"
          aria-label="브리핑 다시 확인"
          onClick={reload}
        >
          <RefreshCw size={17} />
        </button>
      </section>
      <form
        className="brief-controls"
        onSubmit={(event) => {
          event.preventDefault();
          try {
            new Intl.DateTimeFormat("ko-KR", { timeZone: zone }).format();
            setFormError("");
            setQuery({ date, zone });
          } catch {
            setFormError("올바른 시간대를 입력하세요. 예: Asia/Seoul");
          }
        }}
      >
        <label>
          브리핑 날짜
          <input
            aria-label="브리핑 날짜"
            type="date"
            min="1900-01-01"
            max="2100-12-31"
            required
            value={date}
            onChange={(event) => setDate(event.target.value)}
          />
        </label>
        <label>
          브리핑 시간대
          <input
            aria-label="브리핑 시간대"
            value={zone}
            maxLength={100}
            required
            onChange={(event) => setZone(event.target.value)}
          />
        </label>
        <button className="button secondary" type="submit">
          브리핑 보기
        </button>
      </form>
      {(formError || error) && (
        <p role="alert" className="banner error">
          {formError || error}
        </p>
      )}
      {data && (
        <>
          <div className="brief-counts">
            <div>
              <small>모인 소식</small>
              <strong>{data.total}</strong>
            </div>
            <div>
              <small>안 읽음</small>
              <strong>{data.unread}</strong>
            </div>
            <div>
              <small>저장함</small>
              <strong>{data.saved}</strong>
            </div>
          </div>
          <p className="form-help">
            {data.date} · {data.timezone}
            {data.asOf &&
              ` · 마지막 수집 ${new Date(data.asOf).toLocaleString("ko-KR", { timeZone: data.timezone })}`}
          </p>
          <div className="brief-items">
            {data.items.map((item) => (
              <article className="resource-card" key={item.id}>
                <div>
                  <small>{item.sourceTitle || "수집한 소식"}</small>
                  <h2>
                    <a
                      href={safeExternalURL(item.url)}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {item.title} <ExternalLink size={13} />
                    </a>
                  </h2>
                  {item.excerpt && (
                    <p>
                      {Array.from(item.excerpt).slice(0, 300).join("")}
                      {Array.from(item.excerpt).length > 300 ? "…" : ""}
                    </p>
                  )}
                  <small>
                    출처:{" "}
                    {(() => {
                      try {
                        return new URL(item.url).hostname;
                      } catch {
                        return "원문 주소 없음";
                      }
                    })()}
                  </small>
                </div>
              </article>
            ))}
          </div>
          {data.truncated && (
            <p role="status">
              최근 소식 20개를 표시합니다. 전체 소식은 모아 읽기에서 확인하세요.
            </p>
          )}
          {data.items.length === 0 && (
            <div className="empty-board">
              <CalendarDays size={30} />
              <h2>이 날짜에 모인 소식이 없습니다</h2>
              <p>다른 날짜를 선택하거나 소식 위젯의 갱신 상태를 확인하세요.</p>
            </div>
          )}
          <button className="button secondary" onClick={onRead}>
            모아 읽기로 이동
          </button>
        </>
      )}
      {!data && !error && <p role="status">브리핑을 만들고 있어요.</p>}
    </>
  );
}
