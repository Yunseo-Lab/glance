import { useState } from "react";
import { request, type Board } from "../types";
import { boardChanges, mergeBoards } from "./merge";
export default function ConflictReview({
  base,
  local,
  endpoint,
  onReapply,
}: {
  base: Board;
  local: Board;
  endpoint: string;
  onReapply: (remote: Board, merged: Board) => void;
}) {
  const [remote, setRemote] = useState<Board>(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function compare() {
    setBusy(true);
    setError("");
    try {
      setRemote(await request<Board>(endpoint));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const result = remote ? mergeBoards(base, local, remote) : undefined;
  return (
    <section className="conflict-review" aria-label="저장 충돌 비교">
      <button
        className="button secondary"
        disabled={busy}
        onClick={() => void compare()}
      >
        {busy ? "비교 중" : "변경 비교"}
      </button>
      {error && <p role="alert">{error}</p>}
      {remote && result && (
        <>
          <h3>내 편집과 저장된 버전 {remote.revision} 비교</h3>
          <div className="conflict-columns">
            <div>
              <h4>내 변경</h4>
              <ul>
                {boardChanges(base, local).map((change, index) => (
                  <li key={index}>{change}</li>
                ))}
              </ul>
            </div>
            <div>
              <h4>다른 탭의 변경</h4>
              <ul>
                {boardChanges(base, remote).map((change, index) => (
                  <li key={index}>{change}</li>
                ))}
              </ul>
            </div>
          </div>
          <p>
            {result.conflicts.length
              ? `같은 항목의 변경 ${result.conflicts.length}개가 겹칩니다. 재적용하면 겹친 항목에 내 편집을 사용합니다.`
              : "다른 탭에서만 바뀐 항목을 보존해 내 편집을 합칩니다."}{" "}
            아직 서버에 저장하지 않습니다.
          </p>
          <button
            className="button secondary"
            onClick={async () => {
              if (
                result.conflicts.length &&
                !window.confirm(
                  "비교한 변경이 겹칩니다. 겹친 항목에 내 편집을 사용해 초안을 다시 만들까요?",
                )
              )
                return;
              setBusy(true);
              try {
                const validated = await request<Board>(`${endpoint}/validate`, {
                  method: "POST",
                  headers: { "Content-Type": "application/json" },
                  body: JSON.stringify(result.board),
                });
                onReapply(remote, validated);
              } catch (err) {
                setError((err as Error).message);
              } finally {
                setBusy(false);
              }
            }}
            disabled={busy}
          >
            내 변경으로 재적용
          </button>
        </>
      )}
    </section>
  );
}
