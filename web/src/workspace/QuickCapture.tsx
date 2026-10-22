import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import type { Capture } from "../editor/quickCapture";
export default function QuickCapture({
  staged,
  onClose,
  onSave,
}: {
  staged: boolean;
  onClose: () => void;
  onSave: (input: Capture) => Promise<boolean>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [kind, setKind] = useState<Capture["kind"]>("note"),
    [title, setTitle] = useState(""),
    [body, setBody] = useState(""),
    [url, setURL] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    dialog.current?.showModal();
    dialog.current?.querySelector("textarea")?.focus();
  }, []);
  return (
    <dialog
      ref={dialog}
      aria-labelledby="quick-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <form
        onSubmit={async (event) => {
          event.preventDefault();
          setBusy(true);
          setError("");
          try {
            if (await onSave({ kind, title, body, url })) onClose();
            else setError("저장하지 못했습니다. 입력한 내용은 유지했습니다.");
          } catch (err) {
            setError((err as Error).message);
          } finally {
            setBusy(false);
          }
        }}
      >
        <div className="dialog-header">
          <h2 id="quick-title">빠른 기록</h2>
          <button
            type="button"
            className="icon-button"
            aria-label="빠른 기록 닫기"
            disabled={busy}
            onClick={onClose}
          >
            <X size={18} />
          </button>
        </div>
        <div className="reading-toolbar" role="group" aria-label="기록 종류">
          <button
            type="button"
            className="button secondary"
            disabled={busy}
            aria-pressed={kind === "note"}
            onClick={() => setKind("note")}
          >
            메모
          </button>
          <button
            type="button"
            className="button secondary"
            disabled={busy}
            aria-pressed={kind === "bookmark"}
            onClick={() => setKind("bookmark")}
          >
            즐겨찾기
          </button>
        </div>
        <label>
          기록 제목
          <input
            aria-label="기록 제목"
            value={title}
            maxLength={100}
            disabled={busy}
            onChange={(event) => setTitle(event.target.value)}
            placeholder={kind === "note" ? "빠른 메모" : "사이트 이름"}
          />
        </label>
        {kind === "note" ? (
          <label>
            메모 내용
            <textarea
              aria-label="메모 내용"
              required
              value={body}
              maxLength={12000}
              rows={7}
              disabled={busy}
              onChange={(event) => setBody(event.target.value)}
              placeholder="Markdown 제목·목록·체크리스트·강조를 사용할 수 있어요."
            />
          </label>
        ) : (
          <label>
            즐겨찾기 주소
            <input
              aria-label="즐겨찾기 주소"
              type="url"
              required
              value={url}
              maxLength={2048}
              disabled={busy}
              onChange={(event) => setURL(event.target.value)}
              placeholder="https://example.com"
            />
          </label>
        )}
        <p className="form-help">
          {staged
            ? "현재 보드의 편집 초안에 추가합니다. 나중에 대시보드 저장을 눌러주세요."
            : "현재 보드에 바로 저장합니다."}{" "}
          메모의 HTML과 외부 이미지는 실행하거나 불러오지 않습니다.
        </p>
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
            {busy ? "보관 중" : staged ? "초안에 추가" : "기록 저장"}
          </button>
        </div>
      </form>
    </dialog>
  );
}
