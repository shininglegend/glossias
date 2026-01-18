import React from "react";
import type { GrammarPoint } from "../../types/admin";
import Label from "./Label";
import Input from "./Input";
import Button from "./Button";

type Props = {
  selectedGrammarPoints: GrammarPoint[];
  onChange: (grammarPoints: GrammarPoint[]) => void;
};

export default function GrammarPointSelector({
  selectedGrammarPoints,
  onChange,
}: Props) {
  const [newName, setNewName] = React.useState("");
  const [newDescription, setNewDescription] = React.useState("");

  const removeGrammarPoint = (index: number) => {
    onChange(selectedGrammarPoints.filter((_, i) => i !== index));
  };

  const addGrammarPoint = () => {
    if (!newName.trim()) {
      alert("Grammar point name is required");
      return;
    }

    const newGrammarPoint = {
      id: Date.now(), // Temporary ID for frontend
      name: newName.trim(),
      description: newDescription.trim() || "",
    };

    onChange([...selectedGrammarPoints, newGrammarPoint]);
    setNewName("");
    setNewDescription("");
  };

  return (
    <div className="space-y-3">
      <Label>Grammar Points</Label>

      {/* Selected grammar points display */}
      <div className="flex flex-wrap gap-2">
        {selectedGrammarPoints.map((gp, index) => (
          <div
            key={index}
            className="inline-flex items-center gap-2 px-3 py-1 bg-primary-100 text-primary-800 rounded-full text-sm"
          >
            <span>{gp.name}</span>
            <button
              type="button"
              onClick={() => removeGrammarPoint(index)}
              className="text-primary-600 hover:text-primary-800 font-bold"
            >
              ×
            </button>
          </div>
        ))}
      </div>

      {/* Add new grammar point form */}
      <div className="border rounded p-3 bg-gray-50">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-2 items-end">
          <div>
            <Label>Grammar Point Name</Label>
            <Input
              type="text"
              placeholder="e.g., Present Tense"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
          </div>
          <div>
            <Label>Description (Optional)</Label>
            <Input
              type="text"
              placeholder="Brief description"
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
            />
          </div>
          <div>
            <Button
              type="button"
              onClick={addGrammarPoint}
              className="bg-green-600 hover:bg-green-700"
            >
              Add Grammar Point
            </Button>
          </div>
        </div>
      </div>

      {selectedGrammarPoints.length === 0 && (
        <div className="text-sm text-red-500">
          At least one grammar point is required
        </div>
      )}
    </div>
  );
}
