import { Component, type ErrorInfo, type ReactNode } from 'react';

interface State { error: Error | null }

export default class ErrorBoundary extends Component<{ children: ReactNode }, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('HEEWS UI error:', error, info);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="state-banner failure" style={{ padding: 24, textAlign: 'left' }}>
          <strong>UI component failed to render.</strong>
          <p style={{ margin: '8px 0 0', fontSize: 12 }}>{String(this.state.error.message ?? this.state.error)}</p>
          <button onClick={() => this.setState({ error: null })} style={{ marginTop: 12 }}>
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}