import React, { useEffect, useMemo, useRef, useState } from "react";
import Modal from "../ui/Modal";
import type { IdentifyTargetWord } from "../../services/api";
import { shuffle } from "../../lib/identifyMachine";

interface IdentifyQuizModalProps {
  isOpen: boolean;
  /** The word being asked about. */
  target: IdentifyTargetWord | undefined;
  /** All of the story's target words; their pictures are the options. */
  options: IdentifyTargetWord[];
  /** Option IDs already clicked and graded wrong for this quiz. */
  wrongPicks: number[];
  /** True while a pick is being graded by the server. */
  checking: boolean;
  /** Set when the last pick could not be saved. */
  error: string | null;
  isRTL: boolean;
  onPick: (targetVocabId: number) => void;
}

/**
 * The Identify picture quiz. Shows the lexical form, auto-plays the word's
 * pronunciation, and lays out the story's five pictures in a random order.
 * The dialog cannot be dismissed by Escape or the backdrop — the only way out
 * is the right picture — so `onClose` is a no-op.
 */
export const IdentifyQuizModal: React.FC<IdentifyQuizModalProps> = ({
  isOpen,
  target,
  options,
  wrongPicks,
  checking,
  error,
  isRTL,
  onPick,
}) => {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const [audioFailed, setAudioFailed] = useState(false);

  // One shuffle per quiz: reshuffling on every render would move the pictures
  // under the student's cursor after a wrong pick.
  const shuffled = useMemo(
    () => shuffle(options),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [options, target?.id],
  );

  const playWord = () => {
    const url = target?.audio_url;
    if (!url) return;
    const audio = audioRef.current ?? new Audio();
    audioRef.current = audio;
    if (audio.src !== url) audio.src = url;
    audio.currentTime = 0;
    setAudioFailed(false);
    audio.play().catch(() => setAudioFailed(true));
  };

  useEffect(() => {
    if (!isOpen || !target) return;
    playWord();
    return () => {
      audioRef.current?.pause();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, target?.id]);

  if (!target) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={() => {}}
      closeDisabled
      closeOnBackdropClick={false}
      title={
        <span className="flex items-center justify-center gap-3">
          <span
            className="text-[5.5rem] font-bold text-amber-700"
            dir={isRTL ? "rtl" : "ltr"}
            lang={isRTL ? "he" : undefined}
          >
            {target.lexical_form}
          </span>
          {target.audio_url && (
            <button
              type="button"
              onClick={playWord}
              className="inline-flex items-center justify-center w-10 h-10 rounded-full bg-primary-500 text-white hover:bg-primary-600"
              aria-label={`Play pronunciation of ${target.lexical_form}`}
            >
              <span className="material-icons" aria-hidden="true">
                volume_up
              </span>
            </button>
          )}
        </span>
      }
      description="Which picture matches this word?"
      className="sm:max-w-2xl"
    >
      {audioFailed && (
        <p className="mt-2 text-sm text-red-600 text-center" role="status">
          Couldn't play the word audio. Use the speaker button to try again.
        </p>
      )}

      <div
        className="mt-6 grid grid-cols-2 sm:grid-cols-3 gap-4"
        role="group"
        aria-label="Picture options"
      >
        {shuffled.map((option) => {
          const wrong = wrongPicks.includes(option.id);
          return (
            <button
              key={option.id}
              type="button"
              disabled={wrong || checking}
              onClick={() => onPick(option.id)}
              data-testid={`identify-option-${option.id}`}
              aria-label={wrong ? "Not this one" : "Picture option"}
              className={`relative aspect-square overflow-hidden rounded-xl border-4 bg-slate-50 transition-transform ${
                wrong
                  ? "border-red-400 opacity-50 cursor-not-allowed"
                  : checking
                    ? "border-slate-200 cursor-wait"
                    : "border-slate-200 hover:border-primary-400 hover:scale-[1.02] cursor-pointer focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-300"
              }`}
            >
              {option.image_url ? (
                <img
                  src={option.image_url}
                  // Intentionally no descriptive alt: the picture *is* the answer.
                  alt=""
                  className="h-full w-full object-cover"
                  draggable={false}
                />
              ) : (
                <span className="flex h-full w-full items-center justify-center text-slate-400 text-sm p-2 text-center">
                  No picture
                </span>
              )}
              {wrong && (
                <span
                  className="absolute inset-0 flex items-center justify-center text-red-600 text-6xl font-bold"
                  aria-hidden="true"
                >
                  ✗
                </span>
              )}
            </button>
          );
        })}
      </div>

      <div className="mt-4 min-h-6 text-center" aria-live="polite">
        {error ? (
          <p className="text-sm text-red-600">{error}</p>
        ) : checking ? (
          <p className="text-sm text-slate-500">Checking…</p>
        ) : wrongPicks.length > 0 ? (
          <p className="text-sm text-red-600">
            Not quite — try another picture.
          </p>
        ) : null}
      </div>
    </Modal>
  );
};
