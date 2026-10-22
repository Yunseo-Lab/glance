import { useEffect, useState } from "react";
export default function PWAStatus({ dirty }: { dirty: boolean }) {
  const [waiting, setWaiting] = useState<ServiceWorker>();
  const [offline, setOffline] = useState(!navigator.onLine);
  const [updating, setUpdating] = useState(false);
  useEffect(() => {
    const online = () => setOffline(!navigator.onLine);
    addEventListener("online", online);
    addEventListener("offline", online);
    let live = true;
    if ("serviceWorker" in navigator && import.meta.env.PROD) {
      void navigator.serviceWorker
        .register("/sw.js", { updateViaCache: "none" })
        .then((registration) => {
          const check = () => {
            if (live)
              setWaiting(
                registration.active
                  ? registration.waiting || undefined
                  : undefined,
              );
          };
          check();
          registration.addEventListener("updatefound", () =>
            registration.installing?.addEventListener("statechange", check),
          );
        })
        .catch(() => {
          /* Installation is optional; the online application still works. */
        });
    }
    return () => {
      live = false;
      removeEventListener("online", online);
      removeEventListener("offline", online);
    };
  }, []);
  useEffect(() => {
    if (!updating) return;
    const reload = () => location.reload();
    navigator.serviceWorker.addEventListener("controllerchange", reload);
    waiting?.postMessage("ACTIVATE_UPDATE");
    return () =>
      navigator.serviceWorker.removeEventListener("controllerchange", reload);
  }, [updating, waiting]);
  if (!waiting && !offline) return null;
  return (
    <aside
      className="pwa-status"
      aria-label="앱 연결과 업데이트"
      aria-live="polite"
    >
      {offline && (
        <p>
          오프라인입니다. 서버 데이터는 연결이 돌아오면 다시 확인합니다. 개인
          데이터는 기기에 캐시하지 않습니다.
        </p>
      )}
      {waiting && (
        <p>
          새 버전이 준비됐습니다.{" "}
          {dirty
            ? "편집 내용을 저장하거나 취소한 후 업데이트하세요."
            : "업데이트하면 화면을 다시 엽니다."}{" "}
          <button
            className="button secondary"
            disabled={dirty || updating}
            onClick={() => setUpdating(true)}
          >
            {updating ? "업데이트 중…" : "앱 업데이트"}
          </button>
        </p>
      )}
    </aside>
  );
}
