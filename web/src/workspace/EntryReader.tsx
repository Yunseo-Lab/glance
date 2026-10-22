import { useModalDialog } from "./useModalDialog";
import { useEffect, useRef } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Bookmark,
  Check,
  ExternalLink,
  X,
} from "lucide-react";
import type { components } from "../generated/api";
import { safeExternalURL } from "../links";
type Entry = components["schemas"]["Entry"];
export default function EntryReader({
  item,
  busy,
  onClose,
  onPrevious,
  onNext,
  onRead,
  onSave,
}: {
  item: Entry;
  busy: boolean;
  onClose: () => void;
  onPrevious?: () => void;
  onNext?: () => void;
  onRead: () => void;
  onSave: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  useModalDialog(dialog, '[aria-label="읽기 창 닫기"]');
  useEffect(() => {
    function shortcut(event: KeyboardEvent) {
      const target = event.target as HTMLElement;
      if (
        event.defaultPrevented ||
        event.isComposing ||
        event.keyCode === 229 ||
        event.ctrlKey ||
        event.metaKey ||
        event.altKey ||
        !dialog.current?.contains(target) ||
        target.closest("input,textarea,select,[contenteditable=true]")
      )
        return;
      switch (event.key.toLowerCase()) {
        case "j":
          if (onNext) {
            event.preventDefault();
            onNext();
          }
          break;
        case "k":
          if (onPrevious) {
            event.preventDefault();
            onPrevious();
          }
          break;
        case "m":
          if (!busy) {
            event.preventDefault();
            onRead();
          }
          break;
        case "s":
          if (!busy) {
            event.preventDefault();
            onSave();
          }
          break;
      }
    }
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, [item, busy, onNext, onPrevious, onRead, onSave]);
  const address = safeExternalURL(item.url);
  let origin = "";
  if (address) origin = new URL(address).hostname;
  return (
    <dialog
      className="entry-reader"
      ref={dialog}
      aria-labelledby="reader-title"
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <div className="dialog-header">
        <span>모아 읽기</span>
        <button
          className="icon-button"
          aria-label="읽기 창 닫기"
          onClick={onClose}
        >
          <X size={20} />
        </button>
      </div>
      <div className="reader-meta">
        <span>{item.sourceTitle || "수집한 소식"}</span>
        <span>{origin}</span>
        <time dateTime={item.publishedAt || item.collectedAt}>
          {new Date(item.publishedAt || item.collectedAt).toLocaleString(
            "ko-KR",
          )}
        </time>
      </div>
      <h1 id="reader-title">{item.title}</h1>
      <p className="reader-privacy">
        서버에 보관한 발췌입니다. 원문과 외부 이미지는 자동으로 불러오지
        않습니다.
      </p>
      <div className="reader-excerpt">
        {item.excerpt || "보관된 발췌가 없습니다. 원문에서 확인하세요."}
      </div>
      <div className="entry-tags">
        {item.tags.map((tag) => (
          <span className="tag-chip" key={tag}>
            {tag}
          </span>
        ))}
      </div>
      <div className="entry-actions">
        <button
          className="button secondary"
          disabled={busy}
          aria-label="읽기 항목 읽음 전환"
          aria-pressed={item.isRead}
          onClick={onRead}
        >
          <Check size={15} />
          읽음
        </button>
        <button
          className="button secondary"
          disabled={busy}
          aria-label="읽기 항목 저장 전환"
          aria-pressed={item.isSaved}
          onClick={onSave}
        >
          <Bookmark size={15} />
          저장
        </button>
        {address && (
          <a
            className="button primary"
            href={address}
            target="_blank"
            rel="noopener noreferrer"
          >
            <ExternalLink size={15} /> 원문 열기
          </a>
        )}
      </div>
      <div className="reader-navigation">
        <button
          className="button secondary"
          disabled={!onPrevious}
          onClick={onPrevious}
        >
          <ArrowLeft size={15} />
          이전 소식
        </button>
        <button
          className="button secondary"
          disabled={!onNext}
          onClick={onNext}
        >
          다음 소식
          <ArrowRight size={15} />
        </button>
      </div>
      <p className="form-help">
        j/k 소식 이동 · m 읽음 전환 · s 저장 전환 · Escape 닫기
      </p>
    </dialog>
  );
}
