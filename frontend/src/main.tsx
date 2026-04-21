import { Component, type ErrorInfo, type ReactNode } from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import "./styles.css";

class AppErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("frontend crashed", error, info);
  }

  render() {
    if (this.state.error) {
      return (
        <main className="app-error-boundary">
          <section className="card app-error-card">
            <p className="eyebrow">Frontend runtime error</p>
            <h1>Interface failed after loading data.</h1>
            <p>
              {this.state.error.message || "Unknown render error. Check the browser console for details."}
            </p>
            <button type="button" className="action-button" onClick={() => window.location.reload()}>
              Reload workspace
            </button>
          </section>
        </main>
      );
    }

    return this.props.children;
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <AppErrorBoundary>
    <App />
  </AppErrorBoundary>,
);
