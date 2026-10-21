import { useState } from "react";
import { Sparkles } from "lucide-react";
import type { components } from "../generated/api";
import { request, type Board } from "../types";
import { useResource } from "./useResource";
export default function StarterTemplates({
  onPreview,
}: {
  onPreview: (board: Board) => void;
}) {
  const { data, error } =
    useResource<components["schemas"]["Template"][]>("/api/v1/templates");
  const [busy, setBusy] = useState(""),
    [failure, setFailure] = useState("");
  async function choose(id: string) {
    setBusy(id);
    setFailure("");
    try {
      onPreview(await request<Board>(`/api/v1/templates/${id}`));
    } catch (err) {
      setFailure((err as Error).message);
    } finally {
      setBusy("");
    }
  }
  if (error)
    return (
      <p className="form-help">
        템플릿을 불러오지 못했습니다. 위젯 추가로 직접 시작할 수 있습니다.
      </p>
    );
  return (
    <section className="starter-templates" aria-labelledby="starter-title">
      <div>
        <Sparkles size={18} />
        <h2 id="starter-title">어떤 하루를 만들까요?</h2>
        <p>
          템플릿을 미리 보고 저장하세요. 개인 API 키 없이 시작할 수 있습니다.
        </p>
      </div>
      <div className="template-options">
        {data?.map((template) => (
          <button
            type="button"
            key={template.id}
            disabled={Boolean(busy)}
            onClick={() => void choose(template.id)}
          >
            <strong>{template.title}</strong>
            <small>{template.description}</small>
            <span>
              {busy === template.id
                ? "준비 중"
                : template.requiresNetwork
                  ? "인터넷 연결 필요"
                  : "인터넷 없이 사용"}
            </span>
          </button>
        ))}
      </div>
      {failure && (
        <p className="error-text" role="alert">
          {failure}
        </p>
      )}
    </section>
  );
}
