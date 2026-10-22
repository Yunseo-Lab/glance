import { expect, test } from "@playwright/test";
import {
  editorReducer,
  initialEditor,
  type EditorState,
} from "../src/editor/reducer";
import type { Board } from "../src/types";
const fixture: Board = {
  id: "home",
  title: "불변 보드",
  revision: 1,
  theme: "light",
  widgets: [
    {
      id: "one",
      type: "rss",
      title: "소식",
      size: "wide",
      sourceId: "feed",
      config: {},
    },
    {
      id: "two",
      type: "note",
      title: "메모",
      size: "small",
      config: { body: "보존" },
    },
  ],
  sources: [
    { id: "feed", type: "rss", config: { url: "https://example.com/feed" } },
  ],
};
test("editor commands preserve saved data, IDs, references and cancel boundaries", () => {
  const original = JSON.stringify(fixture);
  let state = editorReducer(initialEditor, { type: "load", board: fixture });
  state = editorReducer(state, { type: "begin" });
  state = editorReducer(state, { type: "duplicate", id: "one", newID: "copy" });
  expect(state.draft!.widgets.map((w) => w.id)).toEqual(["one", "copy", "two"]);
  expect(state.draft!.sources).toHaveLength(1);
  state = editorReducer(state, { type: "remove", id: "one" });
  expect(state.draft!.sources).toHaveLength(1);
  state = editorReducer(state, { type: "move", id: "two", to: 0 });
  expect(state.draft!.widgets.map((w) => w.id)).toEqual(["two", "copy"]);
  state = editorReducer(state, { type: "remove", id: "copy" });
  expect(state.draft!.sources).toHaveLength(0);
  expect(state.saved).toEqual(fixture);
  expect(JSON.stringify(fixture)).toBe(original);
  state = editorReducer(state, { type: "cancel" });
  expect(state.draft).toEqual(fixture);
  expect(state.editing).toBe(false);
});
test("invalid moves, missing references, duplicate IDs, cross-board replacements and capacity are rejected", () => {
  const state = editorReducer(initialEditor, { type: "load", board: fixture });
  for (const command of [
    { type: "move", id: "absent", to: 0 },
    { type: "move", id: "one", to: -1 },
    { type: "move", id: "one", to: 0.5 },
    { type: "duplicate", id: "one", newID: "two" },
    { type: "upsert", widget: { ...fixture.widgets[0], sourceId: "absent" } },
    { type: "replace", board: { ...fixture, id: "other" } },
  ] as const)
    expect(editorReducer(state, command)).toBe(state);
  let full: EditorState = state;
  for (let index = 0; index < 30; index++)
    full = editorReducer(full, {
      type: "duplicate",
      id: "two",
      newID: `copy-${index}`,
    });
  expect(full.draft!.widgets).toHaveLength(32);
  expect(
    editorReducer(full, { type: "duplicate", id: "one", newID: "too-many" }),
  ).toBe(full);
  const updated = editorReducer(state, {
    type: "upsert",
    widget: { ...fixture.widgets[1], config: { body: "수정" } },
  });
  expect(updated.saved!.widgets[1].config.body).toBe("보존");
  expect(updated.draft!.widgets[1].config.body).toBe("수정");
});

test("undo and redo restore deleted sources, clear redo on a branch, and stop at save and history limits", () => {
  let state = editorReducer(initialEditor, { type: "load", board: fixture });
  state = editorReducer(state, { type: "begin" });
  state = editorReducer(state, { type: "remove", id: "one" });
  expect(state.draft!.sources).toHaveLength(0);
  state = editorReducer(state, { type: "undo" });
  expect(state.draft).toEqual(fixture);
  state = editorReducer(state, { type: "redo" });
  expect(state.draft!.sources).toHaveLength(0);
  state = editorReducer(state, { type: "undo" });
  state = editorReducer(state, { type: "title", title: "새 분기" });
  expect(state.future).toHaveLength(0);
  expect(editorReducer(state, { type: "redo" })).toBe(state);
  for (let index = 0; index < 80; index++)
    state = editorReducer(state, { type: "title", title: `편집 ${index}` });
  expect(state.past).toHaveLength(50);
  state = editorReducer(state, {
    type: "load",
    board: { ...state.draft!, revision: 2 },
  });
  expect(state.past).toHaveLength(0);
  expect(state.future).toHaveLength(0);
  expect(editorReducer(state, { type: "undo" })).toBe(state);
});
