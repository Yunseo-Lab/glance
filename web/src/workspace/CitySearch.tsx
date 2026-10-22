import { useEffect, useRef, useState } from "react";
import type { components } from "../generated/api";
import { request } from "../types";
type Place = components["schemas"]["LocationResult"];
export default function CitySearch({
  onChoose,
}: {
  onChoose: (place: Place) => void;
}) {
  const [query, setQuery] = useState(""),
    [places, setPlaces] = useState<Place[]>(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);
  async function search() {
    if (query.trim().length < 2) {
      setError("지역 이름을 두 글자 이상 입력하세요.");
      return;
    }
    controller.current?.abort();
    const pending = new AbortController();
    controller.current = pending;
    setBusy(true);
    setError("");
    try {
      setPlaces(
        await request<Place[]>(
          `/api/v1/locations?q=${encodeURIComponent(query.trim())}`,
          { signal: pending.signal },
        ),
      );
    } catch (err) {
      if (!pending.signal.aborted) setError((err as Error).message);
    } finally {
      if (!pending.signal.aborted) setBusy(false);
    }
  }
  return (
    <section className="city-search" aria-label="날씨 지역 검색">
      <label>
        도시 찾기
        <input
          aria-label="도시 찾기"
          value={query}
          maxLength={80}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.nativeEvent.isComposing) {
              event.preventDefault();
              void search();
            }
          }}
        />
      </label>
      <button
        type="button"
        className="button secondary"
        disabled={busy}
        onClick={() => void search()}
      >
        {busy ? "검색 중" : "지역 검색"}
      </button>
      {error && <p role="alert">{error}</p>}
      {places && (
        <div className="city-results">
          {places.map((place, index) => (
            <button
              type="button"
              className="button secondary"
              key={`${place.latitude}:${place.longitude}:${index}`}
              onClick={() => {
                onChoose(place);
                setPlaces(undefined);
              }}
            >
              {[place.name, place.region, place.country]
                .filter(Boolean)
                .join(" · ")}
            </button>
          ))}
          {places.length === 0 && (
            <p role="status">
              검색 결과가 없습니다. 다른 이름을 입력하거나 아래 좌표를 직접
              입력하세요.
            </p>
          )}
        </div>
      )}
    </section>
  );
}
