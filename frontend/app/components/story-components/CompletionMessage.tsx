import React from "react";

interface CompletionMessageProps {
  currentStepName: string;
  nextStepName: string;
  onContinue: () => void;
}

export const CompletionMessage: React.FC<CompletionMessageProps> = ({
  currentStepName,
  nextStepName,
  onContinue,
}) => {
  return (
    <div className="text-center m-10 p-8 bg-green-50 rounded-xl border-2 border-green-500">
      <div className="mb-5">
        <h3 className="text-green-700 m-0 text-2xl">
          Great job! That's the end of the {currentStepName} exercise.
        </h3>
      </div>
      <div className="mt-5">
        <button
          onClick={onContinue}
          className="next-button inline-flex items-center gap-2 px-8 py-4 bg-green-500 text-white rounded-lg text-lg font-semibold transition-all duration-200 shadow-lg hover:bg-green-600"
        >
          <span>Continue to {nextStepName}</span>
          <span className="material-icons">arrow_forward</span>
        </button>
      </div>
    </div>
  );
};
