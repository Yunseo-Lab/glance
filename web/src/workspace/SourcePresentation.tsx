import { useEffect, useState } from "react";
import { CloudSun, LoaderCircle, RefreshCw } from "lucide-react";
import { ApiError, request, type Snapshot } from "../types";
const failureMessages: Record<string, string> = {
  authentication: "연결 인증을 다시 확인하세요.",
  forbidden: "연결 권한을 확인하세요.",
  rate_limited: "정보 제공자의 요청 한도에 도달했습니다.",
  timeout: "정보 제공자의 응답이 늦어지고 있습니다.",
  invalid_response: "받은 정보의 형식을 확인할 수 없습니다.",
  not_found: "정보 주소를 찾을 수 없습니다.",
};
export function RemoteStatus({ snapshot }: { snapshot?: Snapshot }) {
  if (snapshot?.state === "paused")
    return (
      <p className="source-notice" role="status">
        자동 갱신이 일시정지되었습니다.
      </p>
    );
  const message = snapshot?.errorCode
    ? failureMessages[snapshot.errorCode] || "연결이 지연되고 있습니다."
    : "";
  if (snapshot?.data) {
    return snapshot.state === "stale" || snapshot.error ? (
      <div className="source-notice" role="status">
        <p>
          {snapshot.error
            ? "갱신이 지연되어 마지막 정보를 표시합니다."
            : "마지막 정보를 표시하며 새 소식을 확인하고 있습니다."}
        </p>
        {message && <small>{message}</small>}
        {snapshot.retryAt && (
          <small>
            다음 시도 {new Date(snapshot.retryAt).toLocaleString("ko-KR")}
          </small>
        )}
      </div>
    ) : null;
  }
  if (snapshot?.state === "error")
    return (
      <div className="source-empty" role="status">
        <CloudSun size={28} />
        <p>{message || snapshot.error || "정보를 가져오지 못했습니다."}</p>
        <small>
          {snapshot.retryAt
            ? `다음 시도 ${new Date(snapshot.retryAt).toLocaleString("ko-KR")}`
            : "연결 상태를 확인하거나 위젯 설정을 변경하세요."}
        </small>
      </div>
    );
  return (
    <div className="source-empty" role="status">
      <LoaderCircle className="spin" size={24} />
      <p>소식을 가져오고 있어요</p>
      <small>다른 위젯은 바로 사용할 수 있습니다.</small>
    </div>
  );
}
export function Updated({
  snapshot,
  sourceID,
}: {
  snapshot?: Snapshot;
  sourceID?: string;
}) {
  const date = snapshot?.updatedAt || snapshot?.attemptedAt;
  return (
    <div className="source-footer">
      {sourceID && (
        <ManualRefresh
          sourceID={sourceID}
          paused={snapshot?.state === "paused"}
          nextAllowedAt={snapshot?.manualRefreshAfter}
        />
      )}
      {date && (
        <p className="updated">
          <span
            className={
              snapshot?.state === "fresh" ? "status-dot" : "status-dot stale"
            }
          />
          {snapshot?.updatedAt
            ? snapshot.state === "fresh"
              ? "업데이트"
              : "마지막 업데이트"
            : "마지막 시도"}{" "}
          ·{" "}
          {new Date(date).toLocaleString("ko-KR", {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      )}
    </div>
  );
}
function ManualRefresh({
  sourceID,
  paused,
  nextAllowedAt,
}: {
  sourceID: string;
  paused?: boolean;
  nextAllowedAt?: string;
}) {
  const [busy, setBusy] = useState(false),
    [until, setUntil] = useState(0),
    [tick, setTick] = useState(Date.now()),
    [error, setError] = useState("");
  useEffect(() => {
    if (nextAllowedAt) {
      const next = Date.parse(nextAllowedAt);
      if (next > Date.now()) {
        setTick(Date.now());
        setUntil((value) => Math.max(value, next));
      }
    }
  }, [nextAllowedAt]);
  useEffect(() => {
    if (!until) return;
    const timer = window.setInterval(() => {
      const now = Date.now();
      setTick(now);
      if (now >= until) setUntil(0);
    }, 1000);
    return () => clearInterval(timer);
  }, [until]);
  const seconds = Math.max(0, Math.ceil((until - tick) / 1000));
  async function refresh() {
    setBusy(true);
    setError("");
    try {
      const receipt = await request<{ nextAllowedAt: string }>(
        `/api/v1/sources/${encodeURIComponent(sourceID)}/refresh`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: "{}",
        },
      );
      setTick(Date.now());
      setUntil(Date.parse(receipt.nextAllowedAt));
      window.dispatchEvent(new Event("glance:refresh"));
    } catch (err) {
      const failure = err as ApiError;
      setError(failure.message);
      if (failure.status === 429) {
        setTick(Date.now());
        setUntil(Date.now() + (failure.retryAfter || 30) * 1000);
      }
    } finally {
      setBusy(false);
    }
  }
  return (
    <span className="manual-refresh">
      <button
        type="button"
        className="icon-button"
        aria-label={
          paused
            ? "새로고침 (일시정지)"
            : seconds
              ? `새로고침 (${seconds}초 대기)`
              : "새로고침"
        }
        disabled={paused || busy || seconds > 0}
        onClick={refresh}
      >
        <RefreshCw size={13} className={busy ? "spin" : ""} />
        {seconds > 0 && <small>{seconds}초</small>}
      </button>
      {error && <small role="status">{error}</small>}
    </span>
  );
}
