import EntryReader from "./EntryReader";
import { useEffect, useState } from "react";
import {
  Bookmark,
  BookmarkCheck,
  Check,
  ExternalLink,
  Inbox,
  RefreshCw,
  Tag,
} from "lucide-react";
import type { components } from "../generated/api";
import { request, type Board, type Snapshot } from "../types";
import { safeExternalURL } from "../links";
import { queryClient } from "./queries";
import { useResource } from "./useResource";
import EntryTagEditor from "./EntryTagEditor";
type Entry = components["schemas"]["Entry"];
type Page = components["schemas"]["EntryPage"];
type State = components["schemas"]["EntryState"];
type Input = components["schemas"]["EntryStateInput"];
type SourceStatus = components["schemas"]["SourceStatus"];
const isEntryQuery = (query: { queryKey: readonly unknown[] }) =>
  String(query.queryKey[1]).startsWith("/api/v1/entries?");
export default function ReadingInbox({
  board,
}: {
  board: Board;
  snapshots: Record<string, Snapshot>;
}) {
  const [reader, setReader] = useState<Entry>(),
    [activeID, setActiveID] = useState("");
  const [searchInput, setSearchInput] = useState(""),
    [searchQuery, setSearchQuery] = useState(""),
    [composing, setComposing] = useState(false);
  const [status, setStatus] = useState("all"),
    [scope, setScope] = useState(""),
    [source, setSource] = useState(""),
    [tag, setTag] = useState(""),
    [cursor, setCursor] = useState(""),
    [history, setHistory] = useState<string[]>([]),
    [busy, setBusy] = useState<Set<string>>(new Set()),
    [failure, setFailure] = useState(""),
    [selected, setSelected] = useState<Set<string>>(new Set()),
    [tagEditor, setTagEditor] = useState<Entry[]>();
  const parameters = new URLSearchParams({ status, limit: "30" });
  if (searchQuery) parameters.set("q", searchQuery);
  if (scope) parameters.set("board", scope);
  if (source) parameters.set("source", source);
  if (tag) parameters.set("tag", tag);
  if (cursor) parameters.set("cursor", cursor);
  const path = "/api/v1/entries?" + parameters;
  const { data, error, reload } = useResource<Page>(path, 15000),
    { data: tags = [] } =
      useResource<components["schemas"]["EntryTag"][]>("/api/v1/tags"),
    { data: sources = [] } = useResource<SourceStatus[]>(
      "/api/v1/sources",
      15000,
    );
  useEffect(() => {
    setSelected(new Set());
  }, [path]);
  useEffect(() => {
    if (composing) return;
    const timer = window.setTimeout(() => {
      setSearchQuery(searchInput.trim());
      setCursor("");
      setHistory([]);
    }, 350);
    return () => clearTimeout(timer);
  }, [searchInput, composing]);
  function resetPage() {
    setCursor("");
    setHistory([]);
  }
  function update(id: string, changes: Partial<Entry>) {
    setReader((current) =>
      current?.id === id ? { ...current, ...changes } : current,
    );
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
  async function mutateMany(items: Entry[], input: Input): Promise<boolean> {
    if (items.length === 0 || items.some((item) => busy.has(item.id)))
      return false;
    setBusy(
      (current) => new Set([...current, ...items.map((item) => item.id)]),
    );
    setFailure("");
    await queryClient.cancelQueries({ predicate: isEntryQuery });
    const previous = new Map(
      items.map((item) => [
        item.id,
        { isRead: item.isRead, isSaved: item.isSaved, tags: item.tags },
      ]),
    );
    for (const item of items)
      update(item.id, {
        ...(input.read === undefined ? {} : { isRead: input.read }),
        ...(input.saved === undefined ? {} : { isSaved: input.saved }),
        ...(input.tags === undefined ? {} : { tags: input.tags }),
      });
    try {
      if (items.length === 1) {
        const result = await request<State>(
          `/api/v1/entries/${items[0].id}/state`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(input),
          },
        );
        update(items[0].id, result);
      } else
        await request("/api/v1/entries/bulk/state", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ ids: items.map((item) => item.id), ...input }),
        });
      return true;
    } catch (err) {
      for (const [id, state] of previous) update(id, state);
      setFailure((err as Error).message);
      return false;
    } finally {
      setBusy((current) => {
        const next = new Set(current);
        for (const item of items) next.delete(item.id);
        return next;
      });
      void queryClient.invalidateQueries({ predicate: isEntryQuery });
    }
  }
  const items =
    data?.items.filter(
      (item) =>
        (status === "unread"
          ? !item.isRead
          : status === "saved"
            ? item.isSaved
            : true) &&
        (!tag || item.tags.includes(tag)),
    ) || [];
  const selectedItems = items.filter((item) => selected.has(item.id));
  const readerIndex = items.findIndex((item) => item.id === reader?.id);
  function closeReader() {
    setReader(undefined);
    requestAnimationFrame(() => {
      const target = document.querySelector<HTMLButtonElement>(
        `[data-entry-id="${activeID}"] .entry-title-button`,
      );
      if (target) target.focus();
      else
        document
          .querySelector<HTMLInputElement>('[aria-label="수집한 소식 검색"]')
          ?.focus();
    });
  }
  useEffect(() => {
    function shortcut(event: KeyboardEvent) {
      const target = event.target as HTMLElement;
      if (
        reader ||
        event.defaultPrevented ||
        event.isComposing ||
        event.keyCode === 229 ||
        event.ctrlKey ||
        event.metaKey ||
        event.altKey ||
        document.querySelector("dialog[open]") ||
        target.closest("input,textarea,select,[contenteditable=true]")
      )
        return;
      if (event.key === "/") {
        event.preventDefault();
        document
          .querySelector<HTMLInputElement>('[aria-label="수집한 소식 검색"]')
          ?.focus();
        return;
      }
      const index = items.findIndex((item) => item.id === activeID);
      if (event.key === "j" || event.key === "k") {
        event.preventDefault();
        const next =
          event.key === "j"
            ? Math.min(index + 1, items.length - 1)
            : Math.max(index - 1, 0);
        if (items[next]) {
          setActiveID(items[next].id);
          document
            .querySelector<HTMLButtonElement>(
              `[data-entry-id="${items[next].id}"] .entry-title-button`,
            )
            ?.focus();
        }
      }
      if (event.key === "o" && items[index]) {
        event.preventDefault();
        setReader(items[index]);
      }
    }
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, [reader, items, activeID]);
  return (
    <>
      <section className="page-heading">
        <div>
          <span className="eyebrow">
            <Inbox size={17} />
          </span>
          <h1>모아 읽기</h1>
          <p>수집한 소식과 읽음·저장·태그 기록을 내 서버에 보관합니다.</p>
        </div>
        <button
          className="icon-button"
          aria-label="읽기 목록 다시 확인"
          onClick={reload}
        >
          <RefreshCw size={17} />
        </button>
      </section>
      <label className="reading-search">
        수집한 소식 검색
        <input
          aria-label="수집한 소식 검색"
          value={searchInput}
          maxLength={200}
          placeholder="제목·본문·소스에서 검색"
          onChange={(event) => setSearchInput(event.target.value)}
          onCompositionStart={() => setComposing(true)}
          onCompositionEnd={() => setComposing(false)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.nativeEvent.isComposing) {
              setSearchQuery(searchInput.trim());
              resetPage();
            }
          }}
        />
      </label>
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
              resetPage();
            }}
          >
            {label}
          </button>
        ))}
        <small>
          {data ? `${data.total.toLocaleString()}개 항목` : "불러오는 중"}
        </small>
      </div>
      <div className="reading-filters">
        <label>
          보드 범위
          <select
            aria-label="보드 범위"
            value={scope}
            onChange={(event) => {
              setScope(event.target.value);
              resetPage();
            }}
          >
            <option value="">모든 보드</option>
            <option value={board.id}>{board.title}</option>
          </select>
        </label>
        <label>
          소스별 보기
          <select
            aria-label="소스별 보기"
            value={source}
            onChange={(event) => {
              setSource(event.target.value);
              resetPage();
            }}
          >
            <option value="">모든 소스</option>
            {sources
              .filter((item) => ["rss", "github"].includes(item.source.type))
              .map((item) => (
                <option key={item.source.id} value={item.source.id}>
                  {item.source.config.repository ||
                    (() => {
                      try {
                        return new URL(item.source.config.url || "").hostname;
                      } catch {
                        return item.source.type;
                      }
                    })()}
                </option>
              ))}
          </select>
        </label>
        <label>
          태그별 보기
          <select
            aria-label="태그별 보기"
            value={tag}
            onChange={(event) => {
              setTag(event.target.value);
              resetPage();
            }}
          >
            <option value="">모든 태그</option>
            {tags.map((item) => (
              <option key={item.tag} value={item.tag}>
                {item.tag} ({item.count})
              </option>
            ))}
          </select>
        </label>
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
      {items.length > 0 && (
        <div className="reading-bulk">
          <label>
            <input
              aria-label="이 페이지 모두 선택"
              type="checkbox"
              checked={items.every((item) => selected.has(item.id))}
              onChange={(event) =>
                setSelected(
                  event.target.checked
                    ? new Set(items.map((item) => item.id))
                    : new Set(),
                )
              }
            />{" "}
            이 페이지 모두 선택
          </label>
          {selectedItems.length > 0 && (
            <>
              <span>{selectedItems.length}개 선택</span>
              <button
                className="button secondary"
                disabled={busy.size > 0}
                onClick={() => void mutateMany(selectedItems, { read: true })}
              >
                선택 항목 읽음
              </button>
              <button
                className="button secondary"
                disabled={busy.size > 0}
                onClick={() => void mutateMany(selectedItems, { saved: true })}
              >
                선택 항목 저장
              </button>
              <button
                className="button secondary"
                disabled={busy.size > 0}
                onClick={() => setTagEditor(selectedItems)}
              >
                선택 태그 지정
              </button>
            </>
          )}
        </div>
      )}
      <div className="reading-list">
        {items.map((item) => (
          <article
            className={`reading-entry resource-card ${item.isRead ? "is-read" : ""}`}
            key={item.id}
            data-entry-id={item.id}
          >
            <div>
              <label className="entry-select">
                <input
                  type="checkbox"
                  aria-label={`${item.title} 선택`}
                  disabled={busy.has(item.id)}
                  checked={selected.has(item.id)}
                  onChange={(event) =>
                    setSelected((current) => {
                      const next = new Set(current);
                      if (event.target.checked) next.add(item.id);
                      else next.delete(item.id);
                      return next;
                    })
                  }
                />{" "}
                선택
              </label>
              <small>
                {item.sourceTitle || "수집한 소식"} ·{" "}
                {new Date(item.publishedAt || item.collectedAt).toLocaleString(
                  "ko-KR",
                )}
              </small>
              <h2>
                <button
                  className="entry-title-button"
                  aria-label={`${item.title} 읽기`}
                  onFocus={() => setActiveID(item.id)}
                  onClick={() => {
                    setActiveID(item.id);
                    setReader(item);
                  }}
                >
                  {item.title}
                </button>
                <a
                  className="entry-original-link"
                  href={safeExternalURL(item.url)}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={`${item.title} 원문 열기`}
                >
                  <ExternalLink size={14} />
                </a>
              </h2>
              {item.excerpt && <p>{item.excerpt}</p>}
              <div className="entry-tags">
                {item.tags.map((value) => (
                  <button
                    className="tag-chip"
                    key={value}
                    onClick={() => {
                      setTag(value);
                      resetPage();
                    }}
                  >
                    {value}
                  </button>
                ))}
              </div>
              <div className="entry-actions">
                <button
                  className="button secondary"
                  disabled={busy.has(item.id)}
                  aria-label={`${item.title} ${item.isRead ? "안 읽음으로" : "읽음으로"}`}
                  aria-pressed={item.isRead}
                  onClick={() =>
                    void mutateMany([item], { read: !item.isRead })
                  }
                >
                  <Check size={14} />
                  {item.isRead ? "읽음" : "안 읽음"}
                </button>
                <button
                  className="button secondary"
                  disabled={busy.has(item.id)}
                  aria-label={`${item.title} ${item.isSaved ? "저장 해제" : "저장"}`}
                  aria-pressed={item.isSaved}
                  onClick={() =>
                    void mutateMany([item], { saved: !item.isSaved })
                  }
                >
                  {item.isSaved ? (
                    <BookmarkCheck size={14} />
                  ) : (
                    <Bookmark size={14} />
                  )}{" "}
                  {item.isSaved ? "저장됨" : "저장"}
                </button>
                <button
                  className="button secondary"
                  disabled={busy.has(item.id)}
                  aria-label={`${item.title} 태그 편집`}
                  onClick={() => setTagEditor([item])}
                >
                  <Tag size={14} />
                  태그
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
                : "해당하는 소식이 없습니다"}
          </h2>
          <p>필터를 바꾸거나 소식 위젯이 갱신된 뒤 확인해 보세요.</p>
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
      {reader && (
        <EntryReader
          item={reader}
          busy={busy.has(reader.id)}
          onClose={closeReader}
          onPrevious={
            readerIndex > 0
              ? () => {
                  setActiveID(items[readerIndex - 1].id);
                  setReader(items[readerIndex - 1]);
                }
              : undefined
          }
          onNext={
            readerIndex < items.length - 1
              ? () => {
                  const item = items[Math.max(0, readerIndex + 1)];
                  setActiveID(item.id);
                  setReader(item);
                }
              : undefined
          }
          onRead={() => void mutateMany([reader], { read: !reader.isRead })}
          onSave={() => void mutateMany([reader], { saved: !reader.isSaved })}
        />
      )}
      {tagEditor && (
        <EntryTagEditor
          initial={tagEditor.length === 1 ? tagEditor[0].tags : []}
          count={tagEditor.length}
          onClose={() => setTagEditor(undefined)}
          onSave={(tags) => mutateMany(tagEditor, { tags })}
        />
      )}
    </>
  );
}
