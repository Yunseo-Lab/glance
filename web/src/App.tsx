import {editorReducer,initialEditor,pruneBoard as prune} from "./editor/reducer";
import {queryClient,queryOptions,clearPrivateQueries} from "./workspace/queries";
import {useResource} from "./workspace/useResource";
import { useEffect, useRef, useState, useReducer } from "react";
import {
  ArrowDown,
  ArrowUp,
  CalendarDays,
  Check,
  CheckCheck,
  ChevronRight,
  Clock3,
  Download,
  Copy,
  LogOut,
  Upload,
  CloudSun,
  GripVertical,
  LayoutDashboard,
  LoaderCircle,
  Moon,
  NotebookPen,
  Pencil,
  Plus,
  Rss,
  Search,
  Settings2,
  Sun,
  Trash2,
  X,
  Bookmark,
} from "lucide-react";
import CommandPalette from "./workspace/CommandPalette";
import StarterTemplates from "./workspace/StarterTemplates";
import WorkspaceNav from "./workspace/WorkspaceNav";
import { useWorkspaceNavigation } from "./workspace/navigation";
import {
  SourcePanel,
  ReadingPanel,
  SettingsPanel,
} from "./workspace/WorkspacePanels";
import WidgetEditor from "./WidgetEditor";
import Login from "./Login";
import WidgetView from "./WidgetView";
import WidgetBoundary from "./WidgetBoundary";
import { useSnapshots } from "./useSnapshots";
import { downloadBoard, parseDashboardFile } from "./transfer";
import {
  request,
  ApiError,
  type Board,
  type Source,
  type Widget,
  type WidgetType,
} from "./types";

const icons = {
  clock: Clock3,
  calendar: CalendarDays,
  rss: Rss,
  weather: CloudSun,
  bookmarks: Bookmark,
  note: NotebookPen,
};
type Category = "all" | "news" | "daily";

export default function App() {
  const [{saved,draft,editing},dispatchEditor]=useReducer(editorReducer,initialEditor);
  const [editor, setEditor] = useState<Widget | "new" | null>(null);
  const [authRequired, setAuthRequired] = useState(false);
  const [authEnabled, setAuthEnabled] = useState(false);
  const [category, setCategory] = useState<Category>("all");
  const [query, setQuery] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const [dragged, setDragged] = useState<string>();
  const [now, setNow] = useState(new Date());
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [focusWidgetId, setFocusWidgetId] = useState<string>();
  const importInput = useRef<HTMLInputElement>(null);
 const [commandOpen,setCommandOpen]=useState(false);

  const dirty = JSON.stringify(saved) !== JSON.stringify(draft);
  const { route, navigate } = useWorkspaceNavigation(() => {
    if (saving) return false;
    return (
      !dirty ||
      window.confirm("저장하지 않은 변경 사항을 버리고 다른 보드로 이동할까요?")
    );
  });
  const boardID = route.boardID;
  const boardEndpoint =
    boardID === "home" ? "/api/v1/dashboard" : `/api/v1/boards/${boardID}`;
  const {data:boards=[]}=useResource<Board[]>("/api/v1/boards",0,!authRequired);
  const {
    snapshots,
    error: connectionError,
    retry: retryConnection,
  } = useSnapshots(
    authRequired || !["board", "inbox"].includes(route.page)
      ? undefined
      : saved?.revision,
    boardID,
  );

  useEffect(() => {
    const expired = () => {
      setAuthRequired(true);
      setAuthEnabled(true);
    };
    window.addEventListener("glance:unauthorized", expired);
    const controller = new AbortController();
    request<{ enabled: boolean; authenticated: boolean }>(
      "/api/v1/auth/session",
      { signal: controller.signal },
    )
      .then((session) => {
        setAuthEnabled(session.enabled);
        if (!session.authenticated) setAuthRequired(true);
      })
      .catch(() => {});
    return () => {
      controller.abort();
      window.removeEventListener("glance:unauthorized", expired);
    };
  }, []);

  useEffect(()=>{
   function shortcut(event:KeyboardEvent){
    if(event.isComposing||event.keyCode===229||event.altKey||event.defaultPrevented||document.querySelector("dialog[open]"))return;
    if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==="k"){event.preventDefault();setCommandOpen(true);return}
    const target=event.target as HTMLElement|null;
    if(event.key==="/"&&!event.ctrlKey&&!event.metaKey&&!target?.closest("input,textarea,select,[contenteditable=true]")&&route.page==="board"){event.preventDefault();document.querySelector<HTMLInputElement>('[aria-label="위젯 검색"]')?.focus()}
   }
   window.addEventListener("keydown",shortcut);return()=>window.removeEventListener("keydown",shortcut)
  },[route.page]);
  async function importFile(file: File) {
    if (!saved || saving) return;
    setSaving(true);
    setError("");
    setNotice("");
    try {
      if (file.size > 1_048_576)
        throw new Error("설정 파일은 1MB 이하여야 합니다.");
      const candidate = parseDashboardFile(await file.text());
      const validated = await request<Board>(`${boardEndpoint}/validate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...candidate,
          id: saved.id,
          revision: saved.revision,
        }),
      });
      dispatchEditor({type:"replace",board:validated});
      dispatchEditor({type:"begin"});
      setCategory("all");
      setQuery("");
      setNotice("설정을 불러왔습니다. 확인한 뒤 대시보드 저장을 눌러주세요.");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    if (!focusWidgetId) return;
    const frame = requestAnimationFrame(() => {
      const element = document.getElementById(`widget-${focusWidgetId}`);
      element?.scrollIntoView({ block: "nearest", behavior: "instant" });
      element?.focus({ preventScroll: true });
    });
    return () => cancelAnimationFrame(frame);
  }, [focusWidgetId]);

  useEffect(() => {
    const controller = new AbortController();
    if (saved?.id !== boardID) {
      dispatchEditor({type:"clear"});
      setEditor(null);
      setConflict(false);
      setCategory("all");
      setQuery("");
    }
    queryClient.fetchQuery({...queryOptions<Board>(boardEndpoint),staleTime:0,queryFn:({signal})=>request<Board>(boardEndpoint,{signal:AbortSignal.any([signal,controller.signal])})})
      .then((board) => {
        dispatchEditor({type:"load",board});
        setError("");
      })
      .catch((err) => {
        if (!controller.signal.aborted) setError(String(err.message));
      });
    return () => controller.abort();
  }, [loadAttempt, boardID]);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);
  useEffect(() => {
    function warn(event: BeforeUnloadEvent) {
      if (dirty) {
        event.preventDefault();
        event.returnValue = "";
      }
    }
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  useEffect(() => {
    document.documentElement.dataset.theme = draft?.theme || "light";
  }, [draft?.theme]);

  async function persist(board: Board) {
    if (!board.title.trim()) {
      setError("대시보드 이름을 입력하세요.");
      return;
    }
    setSaving(true);
    setError("");
    setConflict(false);
    setNotice("");
    try {
      const result = await request<Board>(boardEndpoint, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(prune({ ...board, title: board.title.trim() })),
      });
      dispatchEditor({type:"load",board:result});
      setNotice("변경 사항을 저장했습니다.");
    } catch (err) {
      setError((err as Error).message);
      setConflict(err instanceof ApiError && err.status === 409);
    } finally {
      setSaving(false);
    }
  }
  function updateWidget(widget: Widget, source?: Source) {
    if (!draft) return;
    if(!draft.widgets.some(item=>item.id===widget.id)){setCategory("all");setQuery("");setFocusWidgetId(widget.id)}
    dispatchEditor({type:"upsert",widget,source});
    setEditor(null);
    setNotice("대시보드 저장을 눌러 변경 사항을 보관하세요.");
  }
  function move(id:string,to:number){dispatchEditor({type:"move",id,to})}
  function duplicateWidget(id:string){
    if(!draft||saving||draft.widgets.length>=32)return;
    const newID=crypto.randomUUID();dispatchEditor({type:"duplicate",id,newID});setCategory("all");setQuery("");setFocusWidgetId(newID);setNotice("위젯을 복제했습니다. 대시보드 저장을 눌러 보관하세요.");
  }
  function beginAdd() {
    navigate("board");
    if (!draft || draft.widgets.length >= 32) return;
    dispatchEditor({type:"begin"});
    setEditor("new");
  }
  function discardDraft(): boolean {
    if (dirty && !window.confirm("저장하지 않은 변경 사항을 버릴까요?"))
      return false;
    dispatchEditor({type:"cancel"});
    setError("");
    setNotice("변경 사항을 취소했습니다.");
    return true;
  }
  async function reloadLatest() {
    if (
      dirty &&
      !window.confirm(
        "최신 설정을 불러오면 저장하지 않은 편집은 사라집니다. 먼저 백업할 수 있습니다. 계속할까요?",
      )
    )
      return;
    setSaving(true);
    try {
      const board = await request<Board>(boardEndpoint);
      dispatchEditor({type:"load",board});
      setError("");
      setConflict(false);
      setNotice("최신 설정을 불러왔습니다.");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  if (authRequired)
    return (
      <Login
        onSuccess={() => {
          setAuthRequired(false);
          setError("");
          if (!saved) setLoadAttempt((attempt) => attempt + 1);
        }}
      />
    );

  if (!draft || !saved)
    return (
      <div className="boot-screen">
        <div className="brand-mark">
          g<span>·</span>
        </div>
        {error ? (
          <>
            <p role="alert">{error}</p>
            <button
              className="button primary"
              onClick={() => setLoadAttempt(loadAttempt + 1)}
            >
              다시 연결
            </button>
            <button
              className="button secondary"
              onClick={() => navigate("board", "home")}
            >
              기본 보드로 이동
            </button>
          </>
        ) : (
          <>
            <LoaderCircle className="spin" size={24} />
            <p>나의 공간을 준비하고 있어요</p>
          </>
        )}
      </div>
    );
  const visible = draft.widgets.filter(
    (widget) =>
      (category === "all" ||
        (category === "news"
          ? widget.type === "rss"
          : widget.type !== "rss")) &&
      widget.title.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
  );
  const categories: {
    id: Category;
    label: string;
    icon: typeof LayoutDashboard;
  }[] = [
    { id: "all", label: "전체 보기", icon: LayoutDashboard },
    { id: "news", label: "관심 소식", icon: Rss },
    { id: "daily", label: "나의 일상", icon: Sun },
  ];
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main">
        본문으로 이동
      </a>
      <aside className="sidebar">
        <a
          className="brand"
          href="/"
          aria-label="Glance 홈"
          onClick={(event) => {
            event.preventDefault();
            if (dirty && !discardDraft()) return;
            setCategory("all");
            setQuery("");
            navigate("board", "home");
          }}
        >
          <div className="brand-mark">
            g<span>·</span>
          </div>
          <span>
            glance<span className="brand-period">.</span>
          </span>
        </a>
        <div className="workspace">
          <span className="workspace-avatar">나</span>
          <div>
            <strong>나의 공간</strong>
            <small>하루를 여는 작은 습관</small>
          </div>
        </div>
        <WorkspaceNav route={route} navigate={navigate} boards={boards} />
        <p className="nav-caption">이 보드</p>
        <nav aria-label="대시보드 분류">
          {categories.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              className={`nav-item ${category === id ? "active" : ""}`}
              aria-current={route.page === "board" && category === id ? "page" : undefined}
              onClick={() => {
                setCategory(id);
                navigate("board");
              }}
            >
              <Icon size={18} />
              <span>{label}</span>
              {id === "all" && <small>{draft.widgets.length}</small>}
            </button>
          ))}
        </nav>
        <button
          className="sidebar-add"
          onClick={beginAdd}
          disabled={saving || draft.widgets.length >= 32}
        >
          <Plus size={17} /> 새 위젯 추가
        </button>
        <div className="sidebar-bottom">
          <div className="mini-orbit" aria-hidden="true">
            <span>✳</span>
          </div>
          <strong>
            덜 찾아보고,
            <br />더 여유롭게.
          </strong>
          <p>
            필요한 정보가 모이는
            <br />
            나만의 시작 페이지.
          </p>
          <span className="version">GLANCE / PERSONAL DASHBOARD</span>
        </div>
      </aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="breadcrumbs">
            <LayoutDashboard size={16} />
            <span>나의 공간</span>
            <ChevronRight size={13} />
            <strong>
              {route.page === "board"
                ? draft.title
                : (
                    {
                      sources: "데이터 소스",
                      inbox: "모아 읽기",
                      settings: "설정",
                      "not-found": "찾을 수 없음",
                    } as const
                  )[route.page]}
            </strong>
          </div>
          <div className="topbar-actions">
            <button className="icon-button" aria-label="명령 팔레트 열기" title="Ctrl / ⌘ K" onClick={()=>setCommandOpen(true)}><Search size={18}/></button>
            <label className="search" hidden={route.page !== "board"}>
              <Search size={16} />
              <input
                aria-label="위젯 검색"
                placeholder="위젯 검색"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
              {query && (
                <button
                  className="icon-button"
                  aria-label="검색 지우기"
                  onClick={() => setQuery("")}
                >
                  <X size={14} />
                </button>
              )}
            </label>
            <button
              className="icon-button theme-toggle"
              disabled={saving}
              aria-label={draft.theme === "light" ? "어두운 테마" : "밝은 테마"}
              onClick={() => {
                const next: Board = {
                  ...draft,
                  theme: draft.theme === "light" ? "dark" : "light",
                };
                if (editing) dispatchEditor({type:"theme",theme:next.theme});
                else void persist(next);
              }}
            >
              {draft.theme === "light" ? <Moon size={19} /> : <Sun size={19} />}
            </button>
            {authEnabled && (
              <button
                className="icon-button"
                aria-label="로그아웃"
                disabled={saving}
                onClick={async () => {
                  if (
                    dirty &&
                    !window.confirm(
                      "저장하지 않은 편집이 있습니다. 로그아웃할까요?",
                    )
                  )
                    return;
                  try {
                    await request("/api/v1/auth/logout", { method: "POST" });
                    setAuthRequired(true);
                    clearPrivateQueries();
                    window.dispatchEvent(new Event("glance:logout"));
                    dispatchEditor({type:"clear"});
                    setEditor(null);
                    setError("");
                    setNotice("");
                  } catch (err) {
                    setError((err as Error).message);
                  }
                }}
              >
                <LogOut size={18} />
              </button>
            )}
            <span className="user-avatar" aria-label="개인 공간">
              나
            </span>
          </div>
        </header>
        <WorkspaceNav
          mobile
          route={route}
          navigate={navigate}
          boards={boards}
        />
        <main id="main" tabIndex={-1}>
          {route.page === "sources" && <SourcePanel boards={boards} onNavigate={id=>navigate("board",id)} onAddWidget={beginAdd}/>}
          {route.page === "inbox" && (
            <ReadingPanel board={saved} snapshots={snapshots} />
          )}
          {route.page === "settings" && <SettingsPanel />}
          {route.page === "not-found" && (
            <div className="empty-board">
              <h1>이 페이지를 찾을 수 없습니다</h1>
              <button
                className="button primary"
                onClick={() => navigate("board", "home")}
              >
                내 대시보드로 이동
              </button>
            </div>
          )}
          <div hidden={route.page !== "board"}>
            <section className="page-heading">
              <div>
                <div className="date-line">
                  <span className="date-dot" />
                  {now.toLocaleDateString("ko-KR", {
                    year: "numeric",
                    month: "long",
                    day: "numeric",
                    weekday: "long",
                  })}
                </div>
                <h1>
                  오늘도, 한눈에<span>.</span>
                </h1>
                <p>내게 필요한 소식을 한곳에 모아보세요.</p>
              </div>
              <div className="heading-actions">
                <input
                  ref={importInput}
                  type="file"
                  accept=".json,application/json"
                  aria-label="설정 JSON 파일"
                  hidden
                  onChange={(event) => {
                    const file = event.currentTarget.files?.[0];
                    event.currentTarget.value = "";
                    if (file) void importFile(file);
                  }}
                />
                <button
                  className="button secondary"
                  disabled={saving}
                  onClick={() => {
                    if (
                      dirty &&
                      !window.confirm(
                        "현재 편집 대신 가져온 설정을 확인할까요? 저장 전에는 기존 설정이 유지됩니다.",
                      )
                    )
                      return;
                    importInput.current?.click();
                  }}
                >
                  <Upload size={15} /> 설정 가져오기
                </button>
                <button
                  className="button secondary"
                  disabled={saving}
                  title="마지막으로 저장된 설정을 내려받습니다"
                  onClick={() => {
                    downloadBoard(saved);
                    setNotice(
                      dirty
                        ? "저장된 설정을 내보냈습니다. 현재 편집은 파일에 포함되지 않습니다."
                        : "설정 파일을 내보냈습니다.",
                    );
                  }}
                >
                  <Download size={15} /> 설정 내보내기
                </button>
                {editing ? (
                  <>
                    <button
                      className="button secondary"
                      disabled={saving}
                      onClick={() => discardDraft()}
                    >
                      취소
                    </button>
                    <button
                      className="button primary"
                      disabled={saving}
                      onClick={() => void persist(draft)}
                    >
                      {saving ? (
                        <LoaderCircle size={16} className="spin" />
                      ) : (
                        <Check size={16} />
                      )}{" "}
                      대시보드 저장
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      className="button secondary"
                      disabled={saving}
                      onClick={() => {
                        dispatchEditor({type:"begin"});
                        setNotice("");
                      }}
                    >
                      <Pencil size={15} /> 편집
                    </button>
                    <button
                      className="button primary"
                      disabled={saving || draft.widgets.length >= 32}
                      onClick={beginAdd}
                    >
                      <Plus size={17} /> 위젯 추가
                    </button>
                  </>
                )}
              </div>
            </section>
            {connectionError && (
              <div className="banner connection-banner" role="status">
                {connectionError}
                <button className="button secondary" onClick={retryConnection}>
                  다시 연결
                </button>
              </div>
            )}
            {error && (
              <div className="banner error" role="alert">
                <span>{error}</span>
                {conflict && (
                  <div className="conflict-actions">
                    <button
                      className="button secondary"
                      disabled={saving}
                      onClick={() => {
                        downloadBoard(draft);
                        setNotice("현재 편집을 파일로 백업했습니다.");
                      }}
                    >
                      내 편집 백업
                    </button>
                    <button
                      className="button secondary"
                      disabled={saving}
                      onClick={() => void reloadLatest()}
                    >
                      최신 상태 불러오기
                    </button>
                  </div>
                )}
                <button
                  className="icon-button"
                  aria-label="오류 알림 닫기"
                  onClick={() => setError("")}
                >
                  <X size={16} />
                </button>
              </div>
            )}
            {notice && (
              <div className="banner" role="status">
                <CheckCheck size={16} />
                {notice}
                <button
                  className="icon-button"
                  aria-label="알림 닫기"
                  onClick={() => setNotice("")}
                >
                  <X size={16} />
                </button>
              </div>
            )}
            {editing && (
              <div className="board-settings">
                <label htmlFor="board-title">대시보드 이름</label>
                <input
                  id="board-title"
                  value={draft.title}
                  maxLength={30}
                  disabled={saving}
                  onChange={(event) =>
                    dispatchEditor({type:"title",title:event.target.value})
                  }
                />
                <span className="draft-status" aria-live="polite">
                  {dirty ? "저장하지 않은 변경 사항" : "저장된 상태"}
                </span>
              </div>
            )}
            <div className="board-toolbar">
              <div className="board-tabs" aria-label="위젯 필터">
                {categories.map(({ id, label }) => (
                  <button
                    key={id}
                    className={category === id ? "selected" : ""}
                    aria-pressed={category === id}
                    onClick={() => {
                      setCategory(id);
                      navigate("board");
                    }}
                  >
                    {label}
                  </button>
                ))}
              </div>
              <span className="board-status">
                {editing ? (
                  <>
                    <GripVertical size={14} /> 드래그하거나 화살표로 순서를
                    바꾸세요
                  </>
                ) : (
                  <>
                    <span className="status-dot" />
                    나만의 대시보드 · {draft.widgets.length}개 위젯
                  </>
                )}
              </span>
            </div>
            {saved.revision===1&&!editing&&<StarterTemplates onPreview={template=>{dispatchEditor({type:"replace",board:{...template,id:saved.id,title:saved.title,revision:saved.revision}});dispatchEditor({type:"begin"});setCategory("all");setQuery("");setNotice("템플릿을 불러왔습니다. 확인 후 대시보드 저장을 눌러주세요.")}}/>}
<div
              className={`widget-grid ${editing ? "is-editing" : ""}`}
              inert={saving}
            >
              {visible.map((widget) => {
                const Icon = icons[widget.type as WidgetType] || NotebookPen;
                const index = draft.widgets.findIndex(
                  (w) => w.id === widget.id,
                );
                return (
                  <article
                    key={widget.id}
                    id={`widget-${widget.id}`}
                    tabIndex={-1}
                    data-widget-id={widget.id}
                    className={`widget-card ${widget.type}-card size-${widget.size} ${dragged === widget.id ? "dragging" : ""}`}
                    draggable={editing}
                    onDragStart={(event) => {
                      setDragged(widget.id);
                      event.dataTransfer.setData("text/plain", widget.id);
                      event.dataTransfer.effectAllowed = "move";
                    }}
                    onDragEnd={() => setDragged(undefined)}
                    onDragOver={(event) => {
                      if (editing) event.preventDefault();
                    }}
                    onDrop={(event) => {
                      if (!editing) return;
                      event.preventDefault();
                      move(event.dataTransfer.getData("text/plain"), index);
                      setDragged(undefined);
                    }}
                  >
                    <div className="widget-header">
                      <div className="widget-title">
                        <span className={`widget-icon ${widget.type}`}>
                          <Icon size={16} />
                        </span>
                        <h2>{widget.title}</h2>
                      </div>
                      {editing && (
                        <div className="widget-controls">
                          <button
                            className="icon-button"
                            disabled={saving || draft.widgets.length >= 32}
                            aria-label={`${widget.title} 복제`}
                            onClick={() => duplicateWidget(widget.id)}
                          >
                            <Copy size={14} />
                          </button>
                          <button
                            className="icon-button"
                            disabled={index === 0}
                            aria-label={`${widget.title} 앞으로 이동`}
                            onClick={() => move(widget.id, index - 1)}
                          >
                            <ArrowUp size={14} />
                          </button>
                          <button
                            className="icon-button"
                            disabled={index === draft.widgets.length - 1}
                            aria-label={`${widget.title} 뒤로 이동`}
                            onClick={() => move(widget.id, index + 1)}
                          >
                            <ArrowDown size={14} />
                          </button>
                          <button
                            className="icon-button"
                            aria-label={`${widget.title} 설정`}
                            onClick={() => setEditor(widget)}
                          >
                            <Settings2 size={14} />
                          </button>
                          <button
                            className="icon-button danger"
                            aria-label={`${widget.title} 삭제`}
                            onClick={() =>
                              dispatchEditor({type:"remove",id:widget.id})
                            }
                          >
                            <Trash2 size={14} />
                          </button>
                        </div>
                      )}
                    </div>
                    <WidgetBoundary key={JSON.stringify(widget)}>
                      <WidgetView
                        widget={widget}
                        snapshot={
                          widget.sourceId
                            ? snapshots[widget.sourceId]
                            : undefined
                        }
                        now={now}
                      />
                    </WidgetBoundary>
                  </article>
                );
              })}
              {editing && (
                <button
                  className="add-widget-card"
                  disabled={draft.widgets.length >= 32}
                  onClick={beginAdd}
                >
                  <span>
                    <Plus size={25} />
                  </span>
                  <strong>나의 대시보드 채우기</strong>
                  <small>필요한 위젯을 추가해 보세요</small>
                </button>
              )}
            </div>
            {!visible.length && !editing && (
              <div className="empty-board">
                <Search size={32} />
                <h2>
                  {query ? "검색 결과가 없습니다" : "아직 위젯이 없습니다"}
                </h2>
                <p>
                  {query
                    ? "다른 이름으로 검색해 보세요."
                    : "첫 위젯을 추가해 나만의 공간을 만들어보세요."}
                </p>
                <button
                  className="button secondary"
                  onClick={() => {
                    if (query) setQuery("");
                    else beginAdd();
                  }}
                >
                  {query ? "검색 지우기" : "위젯 추가"}
                </button>
              </div>
            )}
            <footer className="page-footer">
              <span>필요한 만큼만, 나답게.</span>
              <span>
                glance<span className="brand-period">.</span>
              </span>
            </footer>
          </div>
        </main>
      </div>
      {commandOpen&&<CommandPalette boards={boards} onNavigate={navigate} onAdd={beginAdd} onClose={()=>setCommandOpen(false)}/>}
      {editor && (
        <WidgetEditor
          key={editor === "new" ? "new" : editor.id}
          widget={editor === "new" ? undefined : editor}
          source={
            editor === "new"
              ? undefined
              : draft.sources.find((source) => source.id === editor.sourceId)
          }
          onClose={() => setEditor(null)}
          onSave={updateWidget}
        />
      )}
    </div>
  );
}
