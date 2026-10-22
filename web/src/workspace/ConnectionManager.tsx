import { useEffect, useRef, useState } from "react";
import { KeyRound, Plus, ShieldCheck, Trash2, Unplug, X } from "lucide-react";
import type { components } from "../generated/api";
import { request } from "../types";
import { useResource } from "./useResource";
type Connection = components["schemas"]["Connection"];
type Input = components["schemas"]["ConnectionInput"];
export default function ConnectionManager({
  onChanged,
}: {
  onChanged: () => void;
}) {
  const { data, error, reload } = useResource<Connection[]>(
    "/api/v1/connections",
  );
  const [create, setCreate] = useState<Connection | true | false>(false),
    [busy, setBusy] = useState<Set<string>>(new Set()),
    [failure, setFailure] = useState(""),
    [report, setReport] = useState<Record<string, string>>({});
  async function act(
    connection: Connection,
    action: "test" | "revoke" | "delete",
  ) {
    if (
      action === "revoke" &&
      !window.confirm(
        "연결 비밀정보를 지우고 사용하는 소스의 갱신을 중단할까요?",
      )
    )
      return;
    if (
      action === "delete" &&
      !window.confirm(
        "이 연결을 삭제할까요? 사용하는 소스가 있으면 먼저 제거해야 합니다.",
      )
    )
      return;
    setBusy((current) => new Set(current).add(connection.id));
    setFailure("");
    try {
      if (action === "test") {
        const result = await request<
          components["schemas"]["ConnectionTestReport"]
        >(`/api/v1/connections/${connection.id}/test`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: "{}",
        });
        setReport((current) => ({
          ...current,
          [connection.id]: result.connected
            ? `연결 확인${result.identity ? `: ${result.identity}` : ""} · 확인한 범위: ${result.capabilities.join(", ")}`
            : result.errorCode === "authentication"
              ? "인증 정보를 확인해 주세요. 연결을 다시 만들 수 있습니다."
              : "연결하지 못했습니다. 주소·권한·연결 상태를 확인해 주세요.",
        }));
      } else
        await request(
          `/api/v1/connections/${connection.id}${action === "revoke" ? "/revoke" : ""}`,
          {
            method: action === "delete" ? "DELETE" : "POST",
            headers: { "Content-Type": "application/json" },
            body: "{}",
          },
        );
      reload();
      onChanged();
    } catch (err) {
      setFailure((err as Error).message);
    } finally {
      setBusy((current) => {
        const next = new Set(current);
        next.delete(connection.id);
        return next;
      });
    }
  }
  return (
    <section className="connection-manager" aria-labelledby="connections-title">
      <div className="section-heading">
        <div>
          <h2 id="connections-title">
            <KeyRound size={18} /> 개인 연결
          </h2>
          <p>토큰과 개인 주소는 목록이나 설정 내보내기에 포함하지 않습니다.</p>
        </div>
        <button className="button secondary" onClick={() => setCreate(true)}>
          <Plus size={15} /> 연결 추가
        </button>
      </div>
      {(error || failure) && (
        <p className="banner error" role="alert">
          {failure || error}
        </p>
      )}
      <div className="resource-list">
        {data?.map((connection) => (
          <article className="resource-card" key={connection.id}>
            <div>
              <h3>{connection.name}</h3>
              <small>
                {connection.kind} · {connection.endpoint}
              </small>
              <p>
                {connection.status === "revoked"
                  ? "철회됨"
                  : connection.status === "connected"
                    ? "최근 연결 검사 성공"
                    : "연결 검사 필요"}{" "}
                ·{" "}
                {connection.hasSecret
                  ? "비밀정보 보관됨"
                  : "저장된 비밀정보 없음"}
              </p>
              {connection.lastTestedAt && (
                <small>
                  마지막 검사{" "}
                  {new Date(connection.lastTestedAt).toLocaleString("ko-KR")}
                </small>
              )}
              {report[connection.id] && (
                <p role="status">{report[connection.id]}</p>
              )}
            </div>
            <div className="resource-actions">
              {!connection.hasSecret && (
                <button
                  className="button secondary"
                  onClick={() => setCreate(connection)}
                >
                  다시 연결
                </button>
              )}
              <button
                className="button secondary"
                disabled={busy.has(connection.id) || !connection.hasSecret}
                aria-label={`${connection.name} 연결 테스트`}
                onClick={() => void act(connection, "test")}
              >
                <ShieldCheck size={14} /> 검사
              </button>
              <button
                className="icon-button"
                disabled={busy.has(connection.id) || !connection.hasSecret}
                aria-label={`${connection.name} 연결 철회`}
                onClick={() => void act(connection, "revoke")}
              >
                <Unplug size={16} />
              </button>
              <button
                className="icon-button danger"
                disabled={busy.has(connection.id)}
                aria-label={`${connection.name} 연결 삭제`}
                onClick={() => void act(connection, "delete")}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </article>
        ))}
      </div>
      {data?.length === 0 && (
        <p className="form-help">
          아직 개인 연결이 없습니다. 공개 RSS와 날씨는 개인 키 없이 사용할 수
          있습니다.
        </p>
      )}
      {create && (
        <CreateConnection
          initial={typeof create === "object" ? create : undefined}
          onClose={() => setCreate(false)}
          onSaved={() => {
            setCreate(false);
            reload();
            onChanged();
          }}
        />
      )}
    </section>
  );
}
function CreateConnection({
  initial,
  onClose,
  onSaved,
}: {
  initial?: Connection;
  onClose: () => void;
  onSaved: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [name, setName] = useState(initial?.name || ""),
    [kind, setKind] = useState<Input["kind"]>(initial?.kind || "github"),
    [endpoint, setEndpoint] = useState(""),
    [token, setToken] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    dialog.current?.showModal();
    dialog.current?.querySelector("input")?.focus();
  }, []);
  async function submit(event: React.SubmitEvent) {
    event.preventDefault();
    setError("");
    if (!name.trim() || new TextEncoder().encode(name.trim()).length > 100) {
      setError("연결 이름을 입력하거나 조금 더 짧게 줄여주세요.");
      return;
    }
    setBusy(true);
    try {
      await request<Connection>("/api/v1/connections", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          kind,
          endpoint: kind === "github" ? "" : endpoint.trim(),
          token: token.trim(),
        }),
      });
      setToken("");
      setEndpoint("");
      onSaved();
    } catch (err) {
      setToken("");
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <dialog
      ref={dialog}
      aria-labelledby="connection-form-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <form onSubmit={submit}>
        <div className="dialog-header">
          <h2 id="connection-form-title">새 연결 추가</h2>
          <button
            type="button"
            className="icon-button"
            aria-label="연결 창 닫기"
            disabled={busy}
            onClick={onClose}
          >
            <X size={18} />
          </button>
        </div>
        {initial && (
          <p className="form-help">
            새 비밀정보로 대체 연결을 만듭니다. 기존 연결은 철회 상태로 남고,
            사용하는 위젯에서 새 연결을 선택해야 합니다.
          </p>
        )}
        <label>
          연결 이름
          <input
            required
            autoComplete="off"
            value={name}
            disabled={busy}
            onChange={(event) => setName(event.target.value)}
          />
        </label>
        <label>
          연결 종류
          <select
            value={kind}
            disabled={busy}
            onChange={(event) => {
              setKind(event.target.value as Input["kind"]);
              setToken("");
              setEndpoint("");
              setError("");
            }}
          >
            <option value="github">GitHub</option>
            <option value="calendar">개인 일정 주소</option>
            <option value="json">개인 JSON API</option>
          </select>
        </label>
        {kind !== "github" && (
          <label>
            정보 주소
            <input
              aria-label="정보 주소"
              required
              type="url"
              pattern="https://.*"
              maxLength={2048}
              autoComplete="off"
              spellCheck={false}
              value={endpoint}
              disabled={busy}
              onChange={(event) => setEndpoint(event.target.value)}
            />
            <small>
              개인 주소 전체를 보호하고 목록에는 사이트 주소만 표시합니다.
            </small>
          </label>
        )}
        {kind !== "calendar" && (
          <label>
            접근 토큰
            <input
              aria-label="접근 토큰"
              aria-describedby="connection-token-help"
              type="password"
              required={kind === "github"}
              minLength={kind === "github" ? 16 : undefined}
              maxLength={4096}
              autoComplete="off"
              spellCheck={false}
              value={token}
              disabled={busy}
              onChange={(event) => setToken(event.target.value)}
            />
            <small id="connection-token-help">
              필요한 읽기 권한만 허용한 토큰을 사용하세요. 쓰기 권한은 필요하지
              않습니다.
            </small>
          </label>
        )}
        <p className="form-help">
          연결 키 파일도 데이터 백업과 별도로 보관해야 복원할 수 있습니다. 저장
          후 검사를 눌러 연결 범위를 확인하세요.
        </p>
        {error && (
          <p className="error-text" role="alert">
            {error}
          </p>
        )}
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
            {busy ? "저장 중" : "연결 저장"}
          </button>
        </div>
      </form>
    </dialog>
  );
}
