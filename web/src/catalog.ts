import catalog from "./generated/catalog.json";
import type { WidgetType } from "./types";
export const widgetCatalog = catalog.widgets;
export const providerCatalog = catalog.providers;
export const labels = Object.fromEntries(widgetCatalog.map(widget => [widget.type, widget.title])) as Record<WidgetType, string>;
