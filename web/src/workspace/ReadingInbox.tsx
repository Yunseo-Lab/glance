import { useState } from "react";
import {
  Bookmark,
  BookmarkCheck,
  Check,
  ExternalLink,
  Inbox,
  RefreshCw,
} from "lucide-react";
import type { components } from "../generated/api";
import { request, type Board, type Snapshot } from "../types";
import { safeExternalURL } from "../links";
import { queryClient } from "./queries";
import { useResource } from "./useResource";
type Entry = components["schemas"]["Entry"];
type Page = components["schemas"]["EntryPage"];
type State = components["schemas"]["EntryState"];
type Input = components["schemas"]["EntryStateInput"];
const isEntryQuery = (query: { queryKey: readonly unknown[] }) =>
  String(query.queryKey[1]).startsWith("/api/v1/entries?");
export default function ReadingInbox(_: {
  board: Board;
  snapshots: Record<string, Snapshot>;
}) {
  const [status, setStatus] = useState("all"),
    [cursor, setCursor] = useState(""),
    [history, setHistory] = useState<string[]>([]),
    [busy, setBusy] = useState<Set<string>>(new Set()),
    [failure, setFailure] = useState("");
  const path = `/api/v1/entries?status=${status}&limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`;
  const { data, error, reload } = useResource<Page>(path, 15000);
  function update(id: string, changes: Partial<Entry>) {
    queryClient.setQueriesData<Page>({ predicate: isEntryQuery }, (current) =>
      current
        ? {
            ...current,
            items: current.items.map((item) =>
              item.id === id ? { ...item, ...changes } : item,
            ),
          }
        : current,
    );
  }
  async function mutate(item: Entry, input: Input) {
    if (busy.has(item.id)) return;
    setBusy((current) => new Set(current).add(item.id));
    setFailure("");
    await queryClient.cancelQueries({ predicate: isEntryQuery });
    const previous = {
      isRead: item.isRead,
      isSaved: item.isSaved,
      tags: item.tags,
    };
    update(item.id, {
      ...(input.read === undefined ? {} : { isRead: input.read }),
      ...(input.saved === undefined ? {} : { isSaved: input.saved }),
    });
    try {
      const result = await request<State>(`/api/v1/entries/${item.id}/state`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      });
      update(item.id, result);
    } catch (err) {
      update(item.id, previous);
      setFailure((err as Error).message);
    } finally {
      setBusy((current) => {
        const next = new Set(current);
        next.delete(item.id);
        return next;
      });
      void queryClient.invalidateQueries({ predicate: isEntryQuery });
    }
  }
  const items =
    data?.items.filter((item) =>
      status === "unread"
        ? !item.isRead
        : status === "saved"
          ? item.isSaved
          : true,
    ) || [];
  return (
    <>
      <section className="page-heading">
        <div>
          <span className="eyebrow">
            <Inbox size={17} />
          </span>
          <h1>모아 읽기</h1>
          <p>
            모든 보드에서 수집한 소식과 읽음·저장 기록을 내 서버에 보관합니다.
          </p>
        </div>
        <button
          className="icon-button"
          aria-label="읽기 목록 다시 확인"
          onClick={reload}
        >
          <RefreshCw size={17} />
        </button>
      </section>
      <div className="reading-toolbar" role="group" aria-label="읽기 상태 필터">
        {[
          ["all", "전체"],
          ["unread", "안 읽음"],
          ["saved", "저장함"],
        ].map(([value, label]) => (
          <button
            className="button secondary"
            key={value}
            aria-pressed={status === value}
            onClick={() => {
              setStatus(value);
              setCursor("");
              setHistory([]);
            }}
          >
            {label}
          </button>
        ))}
        <small>
          {data ? `${data.total.toLocaleString()}개 항목` : "불러오는 중"}
        </small>
      </div>
      {data?.capacityReached && (
        <p className="banner" role="status">
          보관 한도 10,000개에 도달해 새 항목 수집을 보류합니다. 저장한 항목은
          자동으로 지우지 않습니다.
        </p>
      )}
      {(error || failure) && (
        <p className="banner error" role="alert">
          {failure || error}
          <button className="button secondary" onClick={reload}>
            다시 확인
          </button>
        </p>
      )}
      <div className="reading-list">
        {items.map((item) => (
          <article
            className={`reading-entry resource-card ${item.isRead ? "is-read" : ""}`}
            key={item.id}
            data-entry-id={item.id}
          >
            <div>
              <small>
                {item.sourceTitle || "수집한 소식"} ·{" "}
                {new Date(item.publishedAt || item.collectedAt).toLocaleString(
                  "ko-KR",
                )}
              </small>
              <h2>
                <a
                  href={safeExternalURL(item.url)}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {item.title} <ExternalLink size={13} />
                </a>
              </h2>
              {item.excerpt && <p>{item.excerpt}</p>}
              <div className="entry-actions">
                <button
                  className="button secondary"
                  disabled={busy.has(item.id)}
                  aria-label={`${item.title} ${item.isRead ? "안 읽음으로" : "읽음으로"}`}
                  aria-pressed={item.isRead}
                  onClick={() => void mutate(item, { read: !item.isRead })}
                >
                  <Check size={14} />
                  {item.isRead ? "읽음" : "안 읽음"}
                </button>
                <button
                  className="button secondary"
                  disabled={busy.has(item.id)}
                  aria-label={`${item.title} ${item.isSaved ? "저장 해제" : "저장"}`}
                  aria-pressed={item.isSaved}
                  onClick={() => void mutate(item, { saved: !item.isSaved })}
                >
                  {item.isSaved ? (
                    <BookmarkCheck size={14} />
                  ) : (
                    <Bookmark size={14} />
                  )}{" "}
                  {item.isSaved ? "저장됨" : "저장"}
                </button>
              </div>
            </div>
          </article>
        ))}
      </div>
      {data && items.length === 0 && (
        <div className="empty-board">
          <Inbox size={30} />
          <h2>
            {status === "saved"
              ? "저장한 소식이 없습니다"
              : status === "unread"
                ? "안 읽은 소식이 없습니다"
                : "아직 도착한 소식이 없습니다"}
          </h2>
          <p>RSS나 GitHub 위젯이 갱신되면 항목을 모아볼 수 있습니다.</p>
        </div>
      )}
      <div className="reading-pagination">
        <button
          className="button secondary"
          disabled={history.length === 0}
          onClick={() => {
            setCursor(history[history.length - 1]);
            setHistory((current) => current.slice(0, -1));
          }}
        >
          이전 항목
        </button>
        <button
          className="button secondary"
          disabled={!data?.nextCursor}
          onClick={() => {
            setHistory((current) => [...current, cursor]);
            setCursor(data!.nextCursor!);
          }}
        >
          다음 항목
        </button>
      </div>
    </>
  );
}
