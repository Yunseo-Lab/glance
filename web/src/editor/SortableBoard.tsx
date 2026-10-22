import type { ReactNode } from "react";
import {
  DndContext,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  closestCenter,
  type Announcements,
} from "@dnd-kit/core";
import {
  SortableContext,
  useSortable,
  sortableKeyboardCoordinates,
  rectSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical } from "lucide-react";
import type { Widget } from "../types";
export function SortableBoard({
  widgets,
  disabled,
  onMove,
  children,
}: {
  widgets: Widget[];
  disabled: boolean;
  onMove: (id: string, toID: string) => void;
  children: ReactNode;
}) {
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 180, tolerance: 6 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );
  const title = (id: string | number) =>
    widgets.find((widget) => widget.id === id)?.title || "위젯";
  const announcements: Announcements = {
    onDragStart: ({ active }) =>
      `${title(active.id)} 이동 시작. 화살표로 옮기고 스페이스로 완료, Escape로 취소하세요.`,
    onDragOver: ({ active, over }) =>
      over
        ? `${title(active.id)}, ${widgets.findIndex((widget) => widget.id === over.id) + 1}번째 위치`
        : undefined,
    onDragEnd: ({ active, over }) =>
      over
        ? `${title(active.id)}을 ${widgets.findIndex((widget) => widget.id === over.id) + 1}번째로 이동했습니다.`
        : "이동을 취소했습니다.",
    onDragCancel: () => "위젯 이동을 취소했습니다.",
  };
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      accessibility={{
        announcements,
        screenReaderInstructions: {
          draggable:
            "스페이스로 위젯을 집고 화살표로 이동하세요. 스페이스로 놓고 Escape로 취소합니다.",
        },
      }}
      onDragEnd={({ active, over }) => {
        if (!disabled && over && active.id !== over.id)
          onMove(String(active.id), String(over.id));
      }}
    >
      <SortableContext
        items={widgets.map((widget) => widget.id)}
        strategy={rectSortingStrategy}
        disabled={disabled}
      >
        {children}
      </SortableContext>
    </DndContext>
  );
}
export function SortableWidget({
  widget,
  disabled,
  children,
}: {
  widget: Widget;
  disabled: boolean;
  children: (handle: ReactNode) => ReactNode;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: widget.id, disabled });
  const handle = (
    <button
      ref={setActivatorNodeRef}
      className="icon-button sort-handle"
      aria-label={`${widget.title} 순서 바꾸기`}
      {...attributes}
      {...listeners}
    >
      <GripVertical size={16} />
    </button>
  );
  return (
    <article
      ref={setNodeRef}
      id={`widget-${widget.id}`}
      tabIndex={-1}
      data-widget-id={widget.id}
      className={`widget-card ${widget.type}-card size-${widget.size} ${isDragging ? "dragging" : ""}`}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
        zIndex: isDragging ? 2 : undefined,
      }}
    >
      {children(handle)}
    </article>
  );
}
