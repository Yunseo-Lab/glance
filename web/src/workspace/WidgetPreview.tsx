import { Cloud, Eye } from "lucide-react";
import WidgetView from "../WidgetView";
import WidgetBoundary from "../WidgetBoundary";
import type { Widget } from "../types";
export default function WidgetPreview({
  widget,
  invalid,
}: {
  widget: Widget;
  invalid?: boolean;
}) {
  const remote = Boolean(widget.sourceId);
  return (
    <section className="widget-preview" aria-label="위젯 미리보기">
      <div className="preview-heading">
        <Eye size={14} />
        <strong>미리보기</strong>
        <small>저장 전에는 새 정보를 가져오지 않습니다.</small>
      </div>
      <div className="preview-card">
        <h3>{widget.title || "위젯 제목"}</h3>
        {remote ? (
          <div className="source-empty">
            <Cloud size={24} />
            <p>
              {widget.type === "weather"
                ? "연결한 지역의 현재 날씨가 표시됩니다."
                : "연결한 소스의 최신 소식이 표시됩니다."}
            </p>
            <small>실제 정보는 대시보드를 저장한 뒤 불러옵니다.</small>
          </div>
        ) : invalid ? (
          <p className="form-help">
            설정을 입력하면 표시 모양을 확인할 수 있습니다.
          </p>
        ) : (
          <div inert>
            <WidgetBoundary key={JSON.stringify(widget)}>
              <WidgetView widget={widget} now={new Date()} />
            </WidgetBoundary>
          </div>
        )}
      </div>
    </section>
  );
}
