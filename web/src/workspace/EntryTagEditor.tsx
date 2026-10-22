import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
export default function EntryTagEditor({
  initial,
  count,
  onClose,
  onSave,
}: {
  initial: string[];
  count: number;
  onClose: () => void;
  onSave: (tags: string[]) => Promise<boolean>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [text, setText] = useState(initial.join(", ")),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    dialog.current?.showModal();
    dialog.current?.querySelector("input")?.focus();
  }, []);
  return (
    <dialog
      ref={dialog}
      aria-labelledby="entry-tags-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <form
        onSubmit={async (event) => {
          event.preventDefault();
          const tags = text.trim()
            ? text.split(",").map((tag) => tag.trim().normalize("NFC"))
            : [];
          if (
            tags.length > 10 ||
            tags.some(
              (tag) => !tag || new TextEncoder().encode(tag).length > 40,
            )
          ) {
            setError("태그는 10개까지, 각 태그는 40바이트 이내로 입력하세요.");
            return;
          }
          setBusy(true);
          setError("");
          try {
            if (await onSave(tags)) onClose();
            else setError("태그를 저장하지 못했습니다. 다시 시도해 주세요.");
          } finally {
            setBusy(false);
          }
        }}
      >
        <div className="dialog-header">
          <h2 id="entry-tags-title">{count}개 항목 태그 편집</h2>
          <button
            type="button"
            className="icon-button"
            aria-label="태그 편집 닫기"
            disabled={busy}
            onClick={onClose}
          >
            <X size={18} />
          </button>
        </div>
        <label>
          태그
          <input
            aria-label="태그"
            value={text}
            maxLength={410}
            disabled={busy}
            onChange={(event) => setText(event.target.value)}
          />
          <small>
            쉼표로 구분하세요. 비우면 선택한 항목의 태그를 지웁니다. 일괄 편집은
            기존 태그를 이 목록으로 바꿉니다.
          </small>
        </label>
        {error && <p role="alert">{error}</p>}
        <div className="dialog-footer">
          <button
            type="button"
            className="button secondary"
            disabled={busy}
            onClick={onClose}
          >
            취소
          </button>
          <button type="submit" className="button primary" disabled={busy}>
            태그 저장
          </button>
        </div>
      </form>
    </dialog>
  );
}
