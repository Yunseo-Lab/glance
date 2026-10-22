import { useState } from "react";
import { Cloud, Search, Smartphone } from "lucide-react";
import { widgetCatalog } from "../catalog";
import type { WidgetType } from "../types";
export default function WidgetCatalog({
  type,
  onSelect,
}: {
  type: WidgetType;
  onSelect: (type: WidgetType) => void;
}) {
  const [query, setQuery] = useState(""),
    [scope, setScope] = useState("all");
  const filtered = widgetCatalog.filter(
    (widget) =>
      [widget.title, widget.description, widget.type]
        .join(" ")
        .toLocaleLowerCase()
        .includes(query.toLocaleLowerCase()) &&
      (scope === "all" ||
        (scope === "local") === widget.capabilities.includes("local")),
  );
  return (
    <section className="widget-catalog" aria-label="위젯 카탈로그">
      <label className="catalog-search">
        <Search size={15} />
        <span className="sr-only">카탈로그 검색</span>
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="이름이나 용도로 찾아보세요"
        />
      </label>
      <div className="catalog-filters" aria-label="위젯 연결 방식">
        {[
          ["all", "모든 위젯"],
          ["local", "기기 안에서"],
          ["remote", "외부 소식"],
        ].map(([value, title]) => (
          <button
            type="button"
            key={value}
            aria-pressed={scope === value}
            onClick={() => setScope(value)}
          >
            {title}
          </button>
        ))}
      </div>
      <div className="catalog-results">
        {filtered.map((widget) => (
          <button
            type="button"
            key={widget.type}
            aria-pressed={type === widget.type}
            onClick={() => onSelect(widget.type as WidgetType)}
          >
            <strong>
              {widget.title}
              <span className="sr-only"> 선택</span>
            </strong>
            <small>{widget.description}</small>
            <span>
              {widget.capabilities.includes("local") ? (
                <>
                  <Smartphone size={12} /> 인터넷 없이 사용
                </>
              ) : (
                <>
                  <Cloud size={12} />{" "}
                  {["github", "agenda", "json"].includes(widget.type)
                    ? "공개 주소 또는 개인 연결"
                    : "인터넷 연결 · 개인 키 불필요"}
                </>
              )}
            </span>
          </button>
        ))}
      </div>
      {!filtered.length && (
        <p role="status" className="form-help">
          일치하는 위젯이 없습니다. 다른 이름이나 용도로 검색하세요.
        </p>
      )}
    </section>
  );
}
