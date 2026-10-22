import {useModalDialog} from "./useModalDialog";
import { useEffect, useRef, useState } from "react";
import { Copy, Plus, Trash2, X } from "lucide-react";
import { request, type Board } from "../types";
export default function BoardManager({
  boards,
  currentID,
  dirty,
  onClose,
  onNavigate,
}: {
  boards: Board[];
  currentID: string;
  dirty: boolean;
  onClose: () => void;
  onNavigate: (id: string) => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [title, setTitle] = useState(""),
    [template, setTemplate] = useState(""),
    [names, setNames] = useState<Record<string, string>>({}),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useModalDialog(dialog,'input');
  async function act(work: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await work();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function create(clone?: Board) {
    const result = await request<Board>("/api/v1/boards", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(
        clone
          ? { title: clone.title + " 복사", cloneFrom: clone.id }
          : { title: title.trim(), template: template || undefined },
      ),
    });
    onNavigate(result.id);
    onClose();
  }
  return (
    <dialog
      ref={dialog}
      aria-labelledby="boards-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <div className="dialog-header">
        <h2 id="boards-title">보드 관리</h2>
        <button
          className="icon-button"
          disabled={busy}
          aria-label="보드 관리 닫기"
          onClick={onClose}
        >
          <X size={18} />
        </button>
      </div>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void act(() => create());
        }}
      >
        <label>
          새 보드 이름
          <input
            required
            maxLength={100}
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            disabled={busy || boards.length >= 10}
          />
        </label>
        <label>
          시작할 구성
          <select
            aria-label="시작할 구성"
            value={template}
            onChange={(event) => setTemplate(event.target.value)}
            disabled={busy}
          >
            <option value="">빈 보드</option>
            <option value="daily">일상</option>
            <option value="developer">개발자</option>
            <option value="news">뉴스</option>
          </select>
        </label>
        <button
          className="button primary"
          disabled={busy || boards.length >= 10}
        >
          <Plus size={16} /> 보드 만들기
        </button>
      </form>
      <p className="form-help">
        {boards.length}/10개 사용 중. 복제하면 위젯 ID는 새로 만들고 소스는 함께
        사용합니다.
      </p>
      {dirty && (
        <p className="form-help">
          편집 중인 보드는 먼저 저장하거나 취소한 뒤 이름을 바꾸거나 삭제하세요.
        </p>
      )}
      {error && (
        <p role="alert" className="error-text">
          {error}
        </p>
      )}
      <div className="board-manager-list">
        {boards.map((board) => (
          <section className="board-manager-row" key={board.id}>
            <label>
              {board.title} 보드 이름
              <input
                value={names[board.id] ?? board.title}
                disabled={busy || (dirty && board.id === currentID)}
                maxLength={100}
                onChange={(event) =>
                  setNames((current) => ({
                    ...current,
                    [board.id]: event.target.value,
                  }))
                }
              />
            </label>
            <div className="resource-actions">
              <button
                className="button secondary"
                disabled={
                  busy ||
                  !names[board.id]?.trim() ||
                  names[board.id] === board.title ||
                  (dirty && board.id === currentID)
                }
                aria-label={`${board.title} 이름 저장`}
                onClick={() =>
                  void act(async () => {
                    await request(`/api/v1/boards/${board.id}`, {
                      method: "PUT",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({
                        ...board,
                        title: names[board.id].trim(),
                      }),
                    });
                    if (board.id === currentID) onNavigate(board.id);
                    setNames((current) => {
                      const next = { ...current };
                      delete next[board.id];
                      return next;
                    });
                  })
                }
              >
                이름 저장
              </button>
              <button
                className="icon-button"
                disabled={busy || boards.length >= 10}
                aria-label={`${board.title} 보드 복제`}
                onClick={() => void act(() => create(board))}
              >
                <Copy size={16} />
              </button>
              <button
                className="icon-button danger"
                disabled={
                  busy ||
                  board.id === "home" ||
                  (dirty && board.id === currentID)
                }
                aria-label={`${board.title} 보드 삭제`}
                title={
                  board.id === "home"
                    ? "기본 보드는 삭제할 수 없습니다."
                    : undefined
                }
                onClick={() => {
                  if (
                    window.confirm(
                      `${board.title} 보드와 배치를 삭제할까요? 저장한 읽기 항목과 다른 보드에서 사용하는 소스는 보존합니다.`,
                    )
                  )
                    void act(async () => {
                      await request(`/api/v1/boards/${board.id}`, {
                        method: "DELETE",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ revision: board.revision }),
                      });
                      if (board.id === currentID) {
                        onNavigate("home");
                        onClose();
                      }
                    });
                }}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </section>
        ))}
      </div>
    </dialog>
  );
}
