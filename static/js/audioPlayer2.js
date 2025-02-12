// audioPlayer2.js
import { normalizeUrl, getUrlFromButton, updateButtonState } from './audioCommon.js';

let currentAudio = null;

// Add event listeners
document.addEventListener('DOMContentLoaded', () => {
  const audioButtons = document.querySelectorAll('.audio-button');
  audioButtons.forEach(button => {
    button.addEventListener('click', () => {
      const url = getUrlFromButton(button);
      if (url) playAudio(url);
    });
  });
});

function playAudio(url) {
  const normalizedUrl = normalizeUrl(url);

  if (currentAudio && normalizeUrl(currentAudio.src).endsWith(normalizedUrl)) {
    if (currentAudio.paused) {
      currentAudio.play();
      updateButtonState(url, true);
    } else {
      currentAudio.pause();
      updateButtonState(url, false);
    }
    return;
  }

  if (currentAudio) {
    currentAudio.pause();
    currentAudio.currentTime = 0;
    updateButtonState(currentAudio.src, false);
  }

  currentAudio = new Audio(normalizedUrl);
  currentAudio.play();
  updateButtonState(url, true);

  currentAudio.onended = function () {
    updateButtonState(url, false);
  };
}