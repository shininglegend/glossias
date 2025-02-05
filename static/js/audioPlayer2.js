// logos-stories/static/js/audioPlayer2.js
let currentAudio = null;

function playAudio(url) {
  // Normalize URL
  const normalizedUrl = url.replace(/\\/g, "/").replace(/\/+/g, "/");

  // If same audio is playing, handle pause/play
  if (currentAudio && currentAudio.src.endsWith(normalizedUrl)) {
    if (currentAudio.paused) {
      currentAudio.play();
      updateButtonState(url, true);
    } else {
      currentAudio.pause();
      updateButtonState(url, false);
    }
    return;
  }

  // Stop current audio if different
  if (currentAudio) {
    currentAudio.pause();
    currentAudio.currentTime = 0;
    updateButtonState(currentAudio.src, false);
  }

  // Play new audio
  currentAudio = new Audio(normalizedUrl);
  currentAudio.play();
  updateButtonState(url, true);

  currentAudio.onended = function () {
    updateButtonState(url, false);
  };
}

function updateButtonState(url, isPlaying) {
  // Find button directly using URL in onclick attribute
  const targetButton = document.querySelector(
    `.audio-button[onclick*='${url}']`,
  );
  if (targetButton) {
    targetButton.setAttribute("data-playing", isPlaying);
    targetButton.querySelector(".material-icons").textContent = isPlaying
      ? "pause"
      : "play_arrow";
  }
}
