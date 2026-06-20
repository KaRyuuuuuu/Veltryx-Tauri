import { Component } from "react";
import type { ErrorBoundaryProps, ErrorBoundaryState } from "../types/app";

export class AppErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { error: "" };
  }

  static getDerivedStateFromError(error: unknown) {
    return { error: error instanceof Error ? error.message : String(error) };
  }

  override render() {
    if (this.state.error) {
      return (
        <div className="login-shell tone-admin">
          <section className="login-card">
            <p className="eyebrow">Veltryx Desktop</p>
            <h1>Runtime error</h1>
            <p className="login-text">
              The application crashed while rendering the workspace.
            </p>
            <p className="status-error">{this.state.error}</p>
          </section>
        </div>
      );
    }

    return this.props.children;
  }
}
