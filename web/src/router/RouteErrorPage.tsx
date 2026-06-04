import { useRouteError, isRouteErrorResponse } from 'react-router-dom';

function getErrorMessage(error: unknown): string {
  if (isRouteErrorResponse(error)) {
    return `${error.status} ${error.statusText}`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === 'string') {
    return error;
  }
  return 'Unknown application error';
}

export function RouteErrorPage() {
  const error = useRouteError();
  const message = getErrorMessage(error);

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="mx-auto flex min-h-screen max-w-3xl flex-col justify-center gap-4 px-6 py-12">
        <div className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          Management UI error
        </div>
        <h1 className="text-3xl font-semibold tracking-tight">The management page crashed</h1>
        <p className="text-sm leading-6 text-muted-foreground">
          Check the browser console and the server logs, then reload after fixing the underlying
          state or API issue.
        </p>
        <pre className="overflow-x-auto rounded-md border border-border bg-card p-4 text-sm leading-6">
          {message}
        </pre>
      </div>
    </div>
  );
}
