import type { PhaseReadiness } from "../../types/admin";

interface ReadinessPanelProps {
  readiness: PhaseReadiness | null;
  /** What the phase needs, shown while there is nothing to fix. */
  requirement: string;
}

/**
 * Shows whether a phase's content is complete, and lists what is missing when it
 * is not. The rules reported here are the ones the schema cannot enforce — five
 * target words each seen at least twice, two produce segments plus an
 * explanation, five ordered recall sentences — so this panel is the author's only
 * view of them.
 */
export default function ReadinessPanel({
  readiness,
  requirement,
}: ReadinessPanelProps) {
  if (!readiness) return null;

  if (readiness.ready) {
    return (
      <div className="bg-green-50 border-l-4 border-green-400 p-4 mb-4 rounded">
        <div className="flex items-start">
          <span className="material-icons text-green-600 mr-2">
            check_circle
          </span>
          <p className="text-green-800 text-sm">
            This phase is fully authored. {requirement}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4 mb-4 rounded">
      <div className="flex items-start">
        <span className="material-icons text-yellow-600 mr-2">warning</span>
        <div>
          <p className="text-yellow-900 text-sm font-medium mb-1">
            {readiness.issues.length}{" "}
            {readiness.issues.length === 1 ? "item" : "items"} left before
            students can use this phase
          </p>
          <p className="text-yellow-800 text-xs mb-2">{requirement}</p>
          <ul className="list-disc list-inside space-y-1">
            {readiness.issues.map((issue, index) => (
              <li
                key={`${issue.field ?? ""}-${index}`}
                className="text-yellow-900 text-sm"
              >
                {issue.message}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
