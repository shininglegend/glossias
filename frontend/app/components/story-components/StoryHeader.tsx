import React from "react";

interface StoryHeaderProps {
  storyTitle: string;
  isPlaying: boolean;
  onPlayStoryAudio: () => void;
}

export const StoryHeader: React.FC<StoryHeaderProps> = ({
  storyTitle,
  isPlaying,
  onPlayStoryAudio,
}) => {
  return (
    <header>
      <h1>{storyTitle}</h1>
      <h2>Step 1: Vocabulary Practice</h2>

      <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
        <div className="flex items-start justify-center">
          <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
          <div>
            <p className="text-gray-700 mb-2">
              Listen to the audio and fill in the blanks with the correct
              vocabulary words.
            </p>
            <p className="text-gray-700">
              Click the play button first, then select answers for the
              highlighted vocabulary gaps.
            </p>
          </div>
        </div>
      </div>
      <button
        onClick={onPlayStoryAudio}
        className={`inline-flex items-center gap-2 px-5 py-3 my-5 text-white border-none rounded-lg text-base cursor-pointer transition-colors duration-200 ${
          isPlaying
            ? "bg-red-500 hover:bg-red-600"
            : "bg-blue-500 hover:bg-blue-600"
        }`}
        type="button"
      >
        <span className="material-icons">
          {isPlaying ? "pause" : "play_arrow"}
        </span>
        {isPlaying ? "Pause Audio" : "Play Audio"}
      </button>
    </header>
  );
};
