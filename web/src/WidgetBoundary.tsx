import { Component, type ReactNode } from "react";

// Keep the card's editing controls outside this boundary so a broken widget
// remains configurable. Its key changes when the widget settings are changed.
export default class WidgetBoundary extends Component<
  { children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  render() {
    if (this.state.failed)
      return (
        <div className="source-empty" role="alert">
          <strong>이 위젯을 표시할 수 없습니다.</strong>
          <p>다른 위젯은 정상적으로 사용할 수 있습니다.</p>
          <small>편집에서 이 위젯의 설정을 확인해 주세요.</small>
        </div>
      );
    return this.props.children;
  }
}
