import React, { useState } from "react";
import { createRoot } from "react-dom/client";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  closestCenter,
} from "@dnd-kit/core";
import {
  SortableContext,
  useSortable,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  arrayMove,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
function Card({ id }: { id: string }) {
  const { attributes, listeners, setNodeRef, transform, transition } =
    useSortable({ id });
  return (
    <article
      ref={setNodeRef}
      data-id={id}
      style={{
        padding: 20,
        margin: 10,
        border: "1px solid",
        transform: CSS.Transform.toString(transform),
        transition,
      }}
    >
      <button {...attributes} {...listeners} style={{ touchAction: "none" }}>
        {id} move
      </button>
    </article>
  );
}
function Fixture() {
  const [items, setItems] = useState(["alpha", "beta", "gamma"]);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragEnd={({ active, over }) => {
        if (over)
          setItems((items) =>
            arrayMove(
              items,
              items.indexOf(String(active.id)),
              items.indexOf(String(over.id)),
            ),
          );
      }}
    >
      <SortableContext items={items} strategy={verticalListSortingStrategy}>
        {items.map((id) => (
          <Card key={id} id={id} />
        ))}
      </SortableContext>
      <output>{JSON.stringify(items)}</output>
    </DndContext>
  );
}
createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Fixture />
  </React.StrictMode>,
);
