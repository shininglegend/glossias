// [moved from annotator/src/components/Line.tsx]
import React, { useState, useEffect, useCallback } from "react";
import AnnotatedText from "./AnnotatedText";
import AnnotationMenu from "./AnnotationMenu";
import AnnotationModal from "./AnnotationModal";
import Button from "~/components/ui/Button";
import ConfirmDialog from "~/components/ui/ConfirmDialog";
import {
  createAudioUploader,
  createAudioDeleter,
  AudioUploadError,
} from "~/lib/audio";
import { useAuthenticatedFetch } from "../../lib/authFetch";
import type { StoryLine, GrammarPoint } from "../../types/api";

interface Props {
  line: StoryLine;
  completeAudioURL?: string;
  incompleteAudioURL?: string;
  storyGrammarPoints?: GrammarPoint[];
  onSelect: (
    lineNumber: number,
    text: string,
    type: "vocab" | "grammar" | "footnote",
    start: number,
    end: number,
    data?: { text?: string; lexicalForm?: string; grammarPointId?: number },
  ) => void;
}

export default function Line({
  line,
  completeAudioURL,
  incompleteAudioURL,
  storyGrammarPoints = [],
  onSelect,
}: Props) {
  const [menu, setMenu] = useState<{ x: number; y: number } | null>(null);
  const [modal, setModal] = useState<{
    type: "vocab" | "grammar" | "footnote";
    text: string;
  } | null>(null);
  const [selection, setSelection] = useState<{
    start: number;
    end: number;
    text: string;
  } | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const authenticatedFetch = useAuthenticatedFetch();

  const handleClickAway = useCallback((event: MouseEvent) => {
    const target = event.target as HTMLElement;
    if (
      !target.closest(".annotation-menu") &&
      !target.closest(".annotated-text")
    ) {
      setMenu(null);
    }
  }, []);

  useEffect(() => {
    document.addEventListener("mousedown", handleClickAway);
    return () => document.removeEventListener("mousedown", handleClickAway);
  }, [handleClickAway]);

  const handleSelect = (start: number, end: number, text: string) => {
    const sel = window.getSelection();
    if (!sel?.toString()) {
      setMenu(null);
      setSelection(null);
      return;
    }
    const range = sel.getRangeAt(0);
    const rect = range.getBoundingClientRect();
    setMenu({ x: rect.left, y: rect.bottom + 5 });
    setSelection({ start, end, text });
  };

  const handleAnnotate = (type: "vocab" | "grammar" | "footnote") => {
    if (!selection) return;
    setModal({ type, text: selection.text });
    setMenu(null);
  };

  const handleSave = (data: {
    text?: string;
    lexicalForm?: string;
    grammarPointId?: number;
  }) => {
    if (!modal || !selection) return;
    onSelect(
      line.lineNumber,
      selection.text,
      modal.type,
      selection.start,
      selection.end,
      data,
    );
    setModal(null);
    setSelection(null);
    window.getSelection()?.removeAllRanges();
  };

  const [audioUploading, setAudioUploading] = useState<string | null>(null);
  const [audioDeleting, setAudioDeleting] = useState<string | null>(null);
  const [audioPlaying, setAudioPlaying] = useState<string | null>(null);
  const [clearingAnnotations, setClearingAnnotations] = useState(false);
  const [uploadSuccess, setUploadSuccess] = useState<string | null>(null);
  const [localCompleteAudioURL, setLocalCompleteAudioURL] = useState<
    string | undefined
  >(completeAudioURL);
  const uploadAudioFile = createAudioUploader();
  const deleteLineAudio = createAudioDeleter();

  // Check if audio exists based on provided URLs
  const hasAudio = (label: string) => {
    return label === "complete"
      ? !!localCompleteAudioURL
      : !!incompleteAudioURL;
  };

  // Fetch signed URL after successful upload
  const fetchSignedURL = async (label: string) => {
    try {
      const response = await authenticatedFetch(
        `/api/stories/${line.storyId}/audio/signed?label=${label}`,
      );
      if (!response.ok) return null;
      const data = await response.json();
      return data.success ? data.data[line.lineNumber.toString()] : null;
    } catch (e) {
      console.error(`Failed to fetch ${label} audio URL:`, e);
      return null;
    }
  };

  const handleClearAnnotations = async () => {
    if (!line.storyId) {
      alert("Story ID not available. Please refresh the page and try again.");
      setShowClearConfirm(false);
      return;
    }

    setClearingAnnotations(true);
    try {
      const response = await authenticatedFetch(
        `/api/admin/stories/${line.storyId}/annotations?line=${line.lineNumber}`,
        {
          method: "DELETE",
        },
      );

      if (!response.ok) {
        throw new Error(`Failed to clear annotations: ${response.status}`);
      }

      // Refresh the page to reload the story data
      window.location.reload();
    } catch (error) {
      console.error("Error clearing annotations:", error);
      alert("Failed to clear annotations");
    } finally {
      setClearingAnnotations(false);
      setShowClearConfirm(false);
    }
  };

  const handleAudioPlay = async (label: string) => {
    const audioURL =
      label === "complete" ? localCompleteAudioURL : incompleteAudioURL;
    if (!audioURL) return;

    setAudioPlaying(label);
    try {
      const audio = new Audio(audioURL);
      audio.onended = () => setAudioPlaying(null);
      audio.onerror = () => {
        setAudioPlaying(null);
        alert("Failed to play audio");
      };
      await audio.play();
    } catch (error) {
      setAudioPlaying(null);
      alert("Failed to play audio");
    }
  };

  const handleAudioUpload = async (
    event: React.ChangeEvent<HTMLInputElement>,
    label: string,
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setAudioUploading(label);
    setUploadSuccess(null);
    try {
      await uploadAudioFile(file, line.storyId || 0, line.lineNumber, label);
      // Reset input
      event.target.value = "";

      // Show success state
      setUploadSuccess(label);

      // Fetch the new signed URL and update local state
      if (label === "complete") {
        const newURL = await fetchSignedURL(label);
        if (newURL) {
          setLocalCompleteAudioURL(newURL);
        }
      }

      // Clear success state after 3 seconds
      setTimeout(() => setUploadSuccess(null), 3000);
    } catch (error) {
      if (error instanceof AudioUploadError) {
        alert(`Upload failed at ${error.step}: ${error.message}`);
      } else {
        alert("Upload failed: Unknown error");
      }
    } finally {
      setAudioUploading(null);
    }
  };

  const handleAudioDelete = async () => {
    if (!line.storyId) {
      alert("Story ID not available. Please refresh the page and try again.");
      setShowDeleteConfirm(false);
      return;
    }

    setAudioDeleting("all");
    try {
      await deleteLineAudio(line.storyId, line.lineNumber);
    } catch (error) {
      if (error instanceof AudioUploadError) {
        alert(`Delete failed: ${error.message}`);
      } else {
        alert("Delete failed: Unknown error");
      }
    } finally {
      setAudioDeleting(null);
      setShowDeleteConfirm(false);
    }
  };

  const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
  const isRTL = line.languageCode && RTL_LANGUAGES.includes(line.languageCode);

  return (
    <div className={`story-line flex items-start gap-2`}>
      <span className="line-number text-slate-500 mr-1">{line.lineNumber}</span>

      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowClearConfirm(true)}
        disabled={clearingAnnotations}
        className="text-xs"
      >
        {clearingAnnotations ? "Clearing..." : "Clear Annotations"}
      </Button>

      <div
        className={`flex-1 outline-dotted p-1 whitespace-pre ${isRTL ? "text-right" : ""}`}
        dir={isRTL ? "rtl" : "ltr"}
      >
        <AnnotatedText
          text={line.text}
          vocabulary={line.vocabulary}
          grammar={line.grammar}
          grammarPoints={storyGrammarPoints}
          languageCode={line.languageCode}
          onSelect={handleSelect}
        />
      </div>
      <div className="flex items-center gap-4 text-xs w-50">
        {/* Complete Audio Controls */}
        <div className="flex items-center gap-1">
          <span className="text-slate-600">Audio w Vocab:</span>
          {hasAudio("complete") ? (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleAudioPlay("complete")}
                className="text-xs px-1"
                disabled={
                  audioUploading !== null ||
                  audioDeleting !== null ||
                  audioPlaying !== null
                }
              >
                {audioPlaying === "complete" ? "⏸️" : "▶️"}
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={() => setShowDeleteConfirm(true)}
                disabled={audioUploading !== null || audioDeleting !== null}
                className="text-xs px-1"
              >
                {audioDeleting === "all" ? "⏳" : "🗑️"}
              </Button>
            </>
          ) : (
            <>
              <input
                type="file"
                accept="audio/*"
                onChange={(e) => handleAudioUpload(e, "complete")}
                style={{ display: "none" }}
                id={`audio-upload-complete-${line.lineNumber}`}
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() =>
                  document
                    .getElementById(`audio-upload-complete-${line.lineNumber}`)
                    ?.click()
                }
                disabled={audioUploading !== null || audioDeleting !== null}
                className="text-xs px-1"
              >
                {audioUploading === "complete"
                  ? "⏳"
                  : uploadSuccess === "complete"
                    ? "✅"
                    : "📁"}
              </Button>
            </>
          )}
        </div>
      </div>
      {menu && (
        <AnnotationMenu
          x={menu.x}
          y={menu.y}
          onVocab={() => handleAnnotate("vocab")}
          onGrammar={() => handleAnnotate("grammar")}
          onFootnote={() => handleAnnotate("footnote")}
          className="annotation-menu"
        />
      )}
      {modal && (
        <AnnotationModal
          type={modal.type}
          selectedText={modal.text}
          onSave={handleSave}
          onClose={() => setModal(null)}
          storyGrammarPoints={storyGrammarPoints}
        />
      )}
      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleAudioDelete}
        variant="delete"
        title="Delete All Audio"
        message="This will permanently remove all audio files from this line. This action cannot be undone."
        loading={audioDeleting === "all"}
      />
      <ConfirmDialog
        isOpen={showClearConfirm}
        onClose={() => setShowClearConfirm(false)}
        onConfirm={handleClearAnnotations}
        variant="clear"
        title="Clear Line Annotations"
        message={`This will remove all annotations from line ${line.lineNumber}. This action cannot be undone.`}
        loading={clearingAnnotations}
      />
    </div>
  );
}
