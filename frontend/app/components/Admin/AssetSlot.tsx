import React from "react";
import Button from "~/components/ui/Button";
import Label from "~/components/ui/Label";

interface AssetSlotProps {
  label: string;
  /** File input accept attribute, e.g. "image/*". */
  accept: string;
  uploading: boolean;
  hasAsset: boolean;
  onSelect: (file: File) => void;
  onClear: () => void;
  /** Rendered when a signed read URL is available. */
  preview: React.ReactNode;
}

/**
 * One upload slot for a phase asset: pick a file to replace what is there, or
 * clear it. Used for a target word's pronunciation and picture and for a recall
 * sentence's picture, all of which are stored as a path on the owning row.
 */
export default function AssetSlot({
  label,
  accept,
  uploading,
  hasAsset,
  onSelect,
  onClear,
  preview,
}: AssetSlotProps) {
  const inputRef = React.useRef<HTMLInputElement>(null);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) onSelect(file);
    // Reset so re-picking the same file fires another change event.
    event.target.value = "";
  };

  return (
    <div>
      <Label className="mb-1">{label}</Label>

      <div className="min-h-[3rem] mb-2">
        {preview ??
          (hasAsset ? (
            <p className="text-xs text-slate-500">
              Uploaded — preview unavailable
            </p>
          ) : (
            <p className="text-xs text-slate-400">Nothing uploaded yet</p>
          ))}
      </div>

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        onChange={handleChange}
        className="hidden"
      />

      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={uploading}
          onClick={() => inputRef.current?.click()}
        >
          {uploading ? "Uploading..." : hasAsset ? "Replace" : "Upload"}
        </Button>

        {hasAsset && (
          <Button
            variant="ghost"
            size="sm"
            disabled={uploading}
            onClick={onClear}
          >
            Clear
          </Button>
        )}
      </div>
    </div>
  );
}
