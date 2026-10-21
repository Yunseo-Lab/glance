import {
  Database,
  ExternalLink,
  Inbox,
  LoaderCircle,
  Pause,
  Play,
  RefreshCw,
  Settings,
} from "lucide-react";
import { useState } from "react";
import type { components } from "../generated/api";
import { ApiError, request, type Board, type FeedData, type Snapshot } from "../types";
import {safeExternalURL} from "../links";
import ConnectionManager from "./ConnectionManager";
import { useResource } from "./useResource";
type SourceStatus = components["schemas"]["SourceStatus"];
function PanelHeading({
  title,
  description,
  Icon,
}: {
  title: string;
  description: string;
  Icon: typeof Database;
}) {
  return (
    <section className="page-heading">
      <div>
        <span className="eyebrow">
          <Icon size={17} />
        </span>
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
    </section>
  );
}
export function SourcePanel({ boards,onNavigate,onAddWidget }: { boards: Board[];onNavigate:(id:string)=>void;onAddWidget:()=>void }) {
  const { data, error, reload } = useResource<SourceStatus[]>(
    "/api/v1/sources",
    15000,
  );
  const [actionError, setActionError] = useState(""),
    [busy, setBusy] = useState("");
  const [cooldowns,setCooldowns]=useState<Record<string,number>>({});
  async function refresh(status:SourceStatus){
    setBusy(status.source.id);setActionError("");
    try{const result=await request<{nextAllowedAt:string}>(`/api/v1/sources/${status.source.id}/refresh`,{method:"POST",headers:{"Content-Type":"application/json"},body:"{}"});setCooldowns(current=>({...current,[status.source.id]:Date.parse(result.nextAllowedAt)}));reload()}catch(err){if(err instanceof ApiError&&err.status===429)setCooldowns(current=>({...current,[status.source.id]:Date.now()+(err.retryAfter||30)*1000}));setActionError((err as Error).message)}finally{setBusy("")}
  }
  async function pause(status: SourceStatus) {
    setBusy(status.source.id);
    setActionError("");
    try {
      await request(`/api/v1/sources/${status.source.id}/pause`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ paused: !status.paused }),
      });
      reload();
    } catch (err) {
      setActionError((err as Error).message);
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <PanelHeading
        title="데이터 소스"
        description="정보를 가져오는 연결과 갱신 상태를 확인하세요."
        Icon={Database}
      />
      {(error || actionError) && (
        <div className="banner error" role="alert">
          {error || actionError}
          <button className="button secondary" onClick={reload}>
            다시 확인
          </button>
        </div>
      )}
      {!data && !error && (
        <p role="status">
          <LoaderCircle className="spin" size={16} /> 연결 상태를 확인하고
          있어요.
        </p>
      )}
      <ConnectionManager onChanged={reload}/>
      <div className="section-heading"><h2>정보를 가져오는 소스</h2><button className="button secondary" onClick={onAddWidget}>소식 위젯 추가</button></div>
      <div className="resource-list">
        {data?.map((status) => {
          const source = status.source;
          let name = source.config.location || source.type;
          try {
            if (source.config.url) name = new URL(source.config.url).hostname;
          } catch {}
          return (
            <article className="resource-card" key={source.id}>
              <div>
                <span
                  className={`status-dot ${status.snapshot.errorCode ? "stale" : ""}`}
                />
                <h2>{name}</h2>
                <p>
                  {status.paused
                    ? "일시정지"
                    : status.running
                      ? "가져오는 중"
                      : status.snapshot.errorCode
                        ? "갱신 지연"
                        : status.snapshot.state === "fresh"
                          ? "최신 정보"
                          : "갱신 대기"}
                </p>
                <div className="source-board-links">{status.boardIds.map(id=><button className="text-button" key={id} onClick={()=>onNavigate(id)}>{boards.find(board=>board.id===id)?.title||id} 보드 열기</button>)}</div>
                <small>{status.boardIds.length}개 보드에서 사용 중 · 제거하려면 해당 보드에서 위젯을 삭제하세요.</small>
                {status.snapshot.updatedAt && (
                  <small>
                    마지막 성공{" "}
                    {new Date(status.snapshot.updatedAt).toLocaleString(
                      "ko-KR",
                    )}
                  </small>
                )}
                {status.snapshot.retryAt && (
                  <small>
                    다음 시도{" "}
                    {new Date(status.snapshot.retryAt).toLocaleString("ko-KR")}
                  </small>
                )}
              </div>
              <div className="resource-actions"><button className="button secondary" aria-label={`${name} 지금 갱신`} disabled={busy===source.id||status.paused||Date.now()<Math.max(cooldowns[source.id]||0,Date.parse(status.snapshot.manualRefreshAfter||"")||0,Date.parse(status.snapshot.retryAt||"")||0)} onClick={()=>void refresh(status)}><RefreshCw size={14}/> 지금 갱신</button>
              <button
                className="button secondary"
                disabled={busy === source.id}
                aria-label={`${name} ${status.paused ? "계속 갱신" : "일시정지"}`}
                onClick={() => void pause(status)}
              >
                {status.paused ? <Play size={14} /> : <Pause size={14} />}{" "}
                {status.paused ? "계속 갱신" : "일시정지"}
              </button></div>
            </article>
          );
        })}
      </div>
      {data?.length === 0 && (
        <div className="empty-board">
          <Database size={30} />
          <h2>연결된 소스가 없습니다</h2>
          <p>
            대시보드에 소식이나 날씨 위젯을 추가하면 이곳에서 상태를 볼 수
            있습니다.
          </p>
        </div>
      )}
    </>
  );
}
export function ReadingPanel({
  board,
  snapshots,
}: {
  board: Board;
  snapshots: Record<string, Snapshot>;
}) {
  const items = new Map<
    string,
    { title: string; url: string; source: string; publishedAt?: string }
  >();
  for (const source of board.sources.filter(
    (source) => source.type === "rss",
  )) {
    const data = snapshots[source.id]?.data as FeedData | undefined;
    for (const item of Array.isArray(data?.items) ? data.items : [])
      if (!items.has(item.url))
        items.set(item.url, { ...item, source: data!.title });
  }
  return (
    <>
      <PanelHeading
        title="모아 읽기"
        description="이 보드의 소식 위젯에서 도착한 최신 항목을 모았습니다."
        Icon={Inbox}
      />
      <div className="reading-list">
        {[...items.values()].map((item) => (
          <a
            className="resource-card"
            key={item.url}
            href={safeExternalURL(item.url)}
            target="_blank"
            rel="noopener noreferrer"
          >
            <div>
              <small>{item.source}</small>
              <h2>{item.title}</h2>
              {item.publishedAt && (
                <small>
                  {new Date(item.publishedAt).toLocaleString("ko-KR")}
                </small>
              )}
            </div>
            <ExternalLink size={16} />
          </a>
        ))}
      </div>
      {items.size === 0 && (
        <div className="empty-board">
          <Inbox size={30} />
          <h2>아직 도착한 소식이 없습니다</h2>
          <p>소식 위젯이 갱신되면 원문을 여기서 모아볼 수 있습니다.</p>
        </div>
      )}
    </>
  );
}
export function SettingsPanel() {
  const { data, error, reload } = useResource<{
    activeSources: number;
    runningSources: number;
    failedSources: number;
    limits: Record<string, number>;
  }>("/api/v1/diagnostics", 15000);
  return (
    <>
      <PanelHeading
        title="설정"
        description="내 공간의 실행 상태와 데이터 보관 정책을 확인하세요."
        Icon={Settings}
      />
      {error && (
        <div className="banner error" role="alert">
          {error}
          <button onClick={reload}>다시 확인</button>
        </div>
      )}
      <div className="resource-list">
        <section className="resource-card">
          <div>
            <h2>실행 상태</h2>
            {data ? (
              <p>
                사용 중인 소스 {data.activeSources}개 · 갱신 중{" "}
                {data.runningSources}개 · 갱신 지연 {data.failedSources}개
              </p>
            ) : (
              <p>상태를 확인하고 있어요.</p>
            )}
          </div>
          <button
            className="icon-button"
            aria-label="실행 상태 다시 확인"
            onClick={reload}
          >
            <RefreshCw size={16} />
          </button>
        </section>
        <section className="resource-card">
          <div>
            <h2>내 데이터 보관</h2>
            <p>
              저장하지 않은 항목은 90일 뒤 정리합니다. 저장한 항목과
              메모·북마크는 자동 정리로 지우지 않습니다.
            </p>
            <small>
              개인 연결의 암호화 키는 데이터 백업과 별도로 안전하게 보관하세요.
            </small>
          </div>
        </section>
        {data && (
          <section className="resource-card">
            <div>
              <h2>이 공간의 한도</h2>
              <p>
                보드 {data.limits.boards}개 · 보드당 위젯{" "}
                {data.limits.widgetsPerBoard}개 · 소스{" "}
                {data.limits.activeSources}개
              </p>
            </div>
          </section>
        )}
      </div>
    </>
  );
}
