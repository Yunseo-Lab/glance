import React, { useEffect, useRef } from "react";
import { createRoot } from "react-dom/client";
import { GridStack } from "gridstack";
import "gridstack/dist/gridstack.min.css";
function Fixture() {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const grid = GridStack.init(
      { column: 1, cellHeight: 90, margin: 10, disableResize: true },
      ref.current!,
    );
    return () => grid.destroy(false);
  }, []);
  return (
    <div ref={ref} className="grid-stack">
      {["alpha", "beta", "gamma"].map((id, index) => (
        <article
          key={id}
          data-id={id}
          className="grid-stack-item"
          gs-x="0"
          gs-y={index}
          gs-w="1"
          gs-h="1"
        >
          <div
            className="grid-stack-item-content"
            style={{ border: "1px solid", padding: 20 }}
          >
            {id}
          </div>
        </article>
      ))}
    </div>
  );
}
createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Fixture />
  </React.StrictMode>,
);
