import { useEffect, useRef, useState } from "react";
import { Plus, Trash2, X } from "lucide-react";
import type { Link, Source, Widget, WidgetType } from "./types";
import { validateWidget, validateWidgetDetail } from "./validation";

import WidgetPreview from "./workspace/WidgetPreview";
import WidgetCatalog from "./workspace/WidgetCatalog";
import { labels } from "./catalog";

export default function WidgetEditor({
  widget,
  source,
  onClose,
  onSave,
}: {
  widget?: Widget;
  source?: Source;
  onClose: () => void;
  onSave: (widget: Widget, source?: Source) => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [type, setType] = useState<WidgetType>(widget?.type || "rss");
  const [title, setTitle] = useState(widget?.title || labels.rss);
  const [size, setSize] = useState<Widget["size"]>(widget?.size || "small");
  const [timezone, setTimezone] = useState(
    widget?.config.timezone || "Asia/Seoul",
  );
  const [body, setBody] = useState(widget?.config.body || "");
  const [links, setLinks] = useState<Link[]>(
    widget?.config.links || [{ title: "", url: "" }],
  );
  const [limit, setLimit] = useState(widget?.config.limit || 5);
  const [url, setUrl] = useState(source?.config.url || "");
  const [location, setLocation] = useState(source?.config.location || "서울");
  const [latitude, setLatitude] = useState(
    String(source?.config.latitude ?? 37.5665),
  );
  const [longitude, setLongitude] = useState(
    String(source?.config.longitude ?? 126.978),
  );
  const [error, setError] = useState("");
  const [errorField, setErrorField] = useState("");
  const fieldProps = (field: string) => ({
    "data-field": field,
    "aria-invalid": errorField === field || undefined,
    "aria-describedby": errorField === field ? "editor-error" : undefined,
  });

  useEffect(() => {
    dialog.current?.showModal();
 dialog.current?.querySelector<HTMLInputElement>("input")?.focus();
  }, []);
  function buildDraft(commit = false) {
    const next: Widget = {
      configVersion: 1,
      id: widget?.id || (commit ? crypto.randomUUID() : "preview"),
      type,
      title: title.trim(),
      size,
      config: {},
    };
    if (type === "clock") next.config = { timezone };
    if (type === "note") next.config = { body };
    if (type === "bookmarks")
      next.config = {
        links: links.map((link) => ({
          title: link.title.trim(),
          url: link.url.trim(),
        })),
      };
    if (type === "rss") next.config = { limit };
    let nextSource: Source | undefined;
    if (type === "rss" || type === "weather") {
      const config =
        type === "rss"
          ? { url: url.trim() }
          : {
              latitude: latitude.trim() === "" ? NaN : Number(latitude),
              longitude: longitude.trim() === "" ? NaN : Number(longitude),
              location: location.trim(),
            };
      // Changing a shared source creates a new source; other widgets stay intact.
      const unchanged =
        source?.type === type &&
        JSON.stringify(source.config) === JSON.stringify(config);
      nextSource = {
        configVersion: 1,
        id: unchanged
          ? source.id
          : commit
            ? crypto.randomUUID()
            : "preview-source",
        type,
        config,
      };
      next.sourceId = nextSource.id;
    }
    return { widget: next, source: nextSource };
  }
  const preview = buildDraft();
  function submit(event: React.SubmitEvent) {
    event.preventDefault();
    setError("");
    const { widget: next, source: nextSource } = buildDraft(true);
    const validationError = validateWidgetDetail(next, nextSource);
    if (validationError) {
      setError(validationError.message);
      setErrorField(validationError.field);
      requestAnimationFrame(() =>
        dialog.current
          ?.querySelector<HTMLElement>(
            `[data-field="${CSS.escape(validationError.field)}"]`,
          )
          ?.focus(),
      );
      return;
    }
    onSave(next, nextSource);
  }
  return (
    <dialog
      ref={dialog}
      onCancel={onClose}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      aria-labelledby="editor-title"
    >
      <form
        noValidate
        onSubmit={submit}
        onChange={() => {
          setError("");
          setErrorField("");
        }}
      >
        <div className="dialog-header">
          <div>
            <span className="eyebrow">MAKE IT YOURS</span>
            <h2 id="editor-title">{widget ? "위젯 설정" : "새 위젯 추가"}</h2>
          </div>
          <button
            type="button"
            className="icon-button"
            aria-label="닫기"
            onClick={onClose}
          >
            <X size={21} />
          </button>
        </div>
        {!widget && (
          <WidgetCatalog
            type={type}
            onSelect={(next) => {
              setType(next);
              setTitle(labels[next]);
            }}
          />
        )}
        {!widget && (
          <label>
            위젯 종류
            <select
              aria-label="위젯 종류"
              value={type}
              onChange={(event) => {
                const t = event.target.value as WidgetType;
                setType(t);
                setTitle(labels[t]);
              }}
            >
              {Object.entries(labels).map(([value, label]) => (
                <option value={value} key={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
        )}
        <div className="form-row">
          <label>
            제목
            <input
              {...fieldProps("title")}
              aria-label="제목"
              required
              maxLength={30}
              value={title}
              onChange={(event) => setTitle(event.target.value)}
            />
          </label>
          <label>
            크기
            <select
              value={size}
              onChange={(event) =>
                setSize(event.target.value as Widget["size"])
              }
            >
              <option value="small">기본</option>
              <option value="wide">넓게</option>
            </select>
          </label>
        </div>
        {type === "clock" && (
          <label>
            시간대
            <input
              {...fieldProps("timezone")}
              aria-label="시간대"
              required
              value={timezone}
              onChange={(event) => setTimezone(event.target.value)}
              placeholder="Asia/Seoul"
            />
            <small>예: Asia/Seoul, America/New_York, Europe/London</small>
          </label>
        )}
        {type === "rss" && (
          <>
            <label>
              RSS / Atom 주소
              <input
                required
                type="url"
                pattern="https://.*"
                maxLength={2048}
                {...fieldProps("url")}
                aria-label="RSS / Atom 주소"
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                placeholder="https://example.com/feed.xml"
              />
              <small>공개 HTTPS 피드를 연결할 수 있습니다.</small>
            </label>
            <label>
              표시할 소식 수
              <input
                required
                type="number"
                min={1}
                max={20}
                {...fieldProps("limit")}
                aria-label="표시할 소식 수"
                value={limit}
                onChange={(event) => setLimit(Number(event.target.value))}
              />
            </label>
          </>
        )}
        {type === "weather" && (
          <>
            <label>
              지역 이름
              <input
                required
                maxLength={30}
                {...fieldProps("location")}
                aria-label="지역 이름"
                value={location}
                onChange={(event) => setLocation(event.target.value)}
              />
            </label>
            <div className="form-row">
              <label>
                위도
                <input
                  required
                  type="number"
                  step="any"
                  min={-90}
                  max={90}
                  {...fieldProps("latitude")}
                  aria-label="위도"
                  value={latitude}
                  onChange={(event) => setLatitude(event.target.value)}
                />
              </label>
              <label>
                경도
                <input
                  required
                  type="number"
                  step="any"
                  min={-180}
                  max={180}
                  {...fieldProps("longitude")}
                  aria-label="경도"
                  value={longitude}
                  onChange={(event) => setLongitude(event.target.value)}
                />
              </label>
            </div>
            <p className="form-help">
              Open-Meteo의 현재 날씨를 표시합니다. 기본 좌표는 서울입니다.
            </p>
          </>
        )}
        {type === "note" && (
          <div className="form-field">
            <label htmlFor="note-body">내용</label>
            <textarea
              {...fieldProps("body")}
              aria-label="내용"
              id="note-body"
              rows={7}
              maxLength={4000}
              value={body}
              onChange={(event) => setBody(event.target.value)}
              placeholder="오늘 기억하고 싶은 것을 적어보세요."
            />
          </div>
        )}
        {type === "bookmarks" && (
          <fieldset {...fieldProps("links")} tabIndex={-1}>
            <legend>즐겨찾기</legend>
            {links.map((link, i) => (
              <div className="link-fields" key={i}>
                <label className="sr-only" htmlFor={`link-title-${i}`}>
                  즐겨찾기 {i + 1} 이름
                </label>
                <input
                  {...fieldProps(`links.${i}.title`)}
                  id={`link-title-${i}`}
                  required
                  maxLength={30}
                  placeholder="이름"
                  value={link.title}
                  onChange={(event) =>
                    setLinks(
                      links.map((l, j) =>
                        j === i ? { ...l, title: event.target.value } : l,
                      ),
                    )
                  }
                />
                <label className="sr-only" htmlFor={`link-url-${i}`}>
                  즐겨찾기 {i + 1} 주소
                </label>
                <input
                  {...fieldProps(`links.${i}.url`)}
                  id={`link-url-${i}`}
                  required
                  type="url"
                  pattern="https://.*"
                  placeholder="https://"
                  value={link.url}
                  onChange={(event) =>
                    setLinks(
                      links.map((l, j) =>
                        j === i ? { ...l, url: event.target.value } : l,
                      ),
                    )
                  }
                />
                <button
                  type="button"
                  className="icon-button"
                  aria-label={`즐겨찾기 ${i + 1} 삭제`}
                  onClick={() => setLinks(links.filter((_, j) => j !== i))}
                >
                  <Trash2 size={15} />
                </button>
              </div>
            ))}
            <button
              type="button"
              className="text-button"
              disabled={links.length >= 20}
              onClick={() => setLinks([...links, { title: "", url: "" }])}
            >
              <Plus size={15} /> 링크 추가
            </button>
          </fieldset>
        )}
        {type === "calendar" && (
          <p className="form-help">
            이번 달의 날짜를 표시합니다. 화살표로 이전 달과 다음 달을 볼 수
            있습니다.
          </p>
        )}
        <WidgetPreview
          widget={preview.widget}
          invalid={Boolean(validateWidget(preview.widget, preview.source))}
        />
        {error && (
          <p id="editor-error" className="error-text" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-footer">
          <button className="button secondary" type="button" onClick={onClose}>
            취소
          </button>
          <button className="button primary" type="submit">
            {widget ? "설정 적용" : "위젯 추가"}
          </button>
        </div>
      </form>
    </dialog>
  );
}
