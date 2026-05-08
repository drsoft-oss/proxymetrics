import { useEffect, useReducer } from "react";
import type { AuditEvent, RequestRow, RunStatus } from "@/types/audit";

type State = {
  requests: RequestRow[];
  summary: { completed: number; locMatches: number; typeMatches: number; errors: number };
  finished: boolean;
  status: RunStatus | null;
};

type Action =
  | { type: "request"; row: RequestRow }
  | { type: "summary"; completed: number; locMatches: number; typeMatches: number; errors: number }
  | { type: "finished"; status: RunStatus };

const initial: State = {
  requests: [],
  summary: { completed: 0, locMatches: 0, typeMatches: 0, errors: 0 },
  finished: false,
  status: null,
};

function reducer(state: State, a: Action): State {
  switch (a.type) {
    case "request": return { ...state, requests: [...state.requests, a.row] };
    case "summary": return {
      ...state,
      summary: { completed: a.completed, locMatches: a.locMatches, typeMatches: a.typeMatches, errors: a.errors },
    };
    case "finished": return { ...state, finished: true, status: a.status };
  }
}

export function useAuditEvents(runId: string | null) {
  const [state, dispatch] = useReducer(reducer, initial);

  useEffect(() => {
    if (runId === null) return;
    const es = new EventSource(`/api/v1/audits/${runId}/events`);
    es.onmessage = (ev) => {
      try {
        const e = JSON.parse(ev.data) as AuditEvent;
        if (e.type === "request") {
          const { type: _t, ...row } = e;
          dispatch({ type: "request", row: row as RequestRow });
        } else if (e.type === "summary") {
          dispatch({
            type: "summary",
            completed: e.completed_count,
            locMatches: e.location_match_count,
            typeMatches: e.type_match_count,
            errors: e.error_count,
          });
        } else if (e.type === "finished") {
          dispatch({ type: "finished", status: e.status });
          es.close();
        }
      } catch {
        // ignore malformed events
      }
    };
    es.onerror = () => {
      es.close();
    };
    return () => es.close();
  }, [runId]);

  return state;
}
