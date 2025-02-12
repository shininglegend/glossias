// audioPlayer.js
import { normalizeUrl, getUrlFromButton, updateButtonState } from './audioCommon.js';

// Global state
let currentAudio = null;
let allAudioCompleted = false;
let numAudioCompleted = 0;

// Normalize URL to ensure consistent path format
function normalizeUrl(url) {
  // Decode URL-encoded characters (like %20 for spaces)
  const decoded = decodeURIComponent(url);
  
  // Replace backslashes with forward slashes and normalize multiple slashes
  return decoded.replace(/\\/g, "/").replace(/\/+/g, "/");
}

// Extract URL from button's onclick attribute - handles escaped URLs
function getUrlFromButton(button) {
  const onclick = button.getAttribute("onclick");
  const match = onclick?.match(/playAudio\('([^']+)'\)/);
  return match ? match[1] : null;
}

function playAudio(url) {
  const normalizedUrl = normalizeUrl(url);
  console.log("playAudio", normalizedUrl);

  // Handle case when same audio is clicked - normalize both URLs for comparison
  if (currentAudio && normalizeUrl(currentAudio.src).endsWith(normalizedUrl)) {
    togglePlayPause(url);
    return;
  }

  // Stop previous audio if exists
  if (currentAudio) {
    stopCurrentAudio();
  }

  // Start new audio
  startNewAudio(normalizedUrl, url);
}

function togglePlayPause(url) {
  if (!currentAudio) return;
  
  if (currentAudio.paused) {
    currentAudio.play();
    updateButtonState(url, true);
  } else {
    currentAudio.pause();
    updateButtonState(url, false);
  }
}

function stopCurrentAudio() {
  if (!currentAudio) return;
  
  currentAudio.pause();
  currentAudio.currentTime = 0;
  updateButtonState(currentAudio.src, false);
}

function startNewAudio(normalizedUrl, originalUrl) {
  currentAudio = new Audio(normalizedUrl);
  currentAudio.play();
  updateButtonState(originalUrl, true);

  currentAudio.onended = () => {
    numAudioCompleted += 1;
    updateButtonState(originalUrl, false);
    
    const buttons = Array.from(document.querySelectorAll("button"));
    // Normalize both URLs before comparison
    const currentButton = buttons.find(btn => 
      normalizeUrl(getUrlFromButton(btn)) === normalizeUrl(originalUrl)
    );
    const currentIndex = buttons.indexOf(currentButton);

    // Play next audio if available
    if (currentIndex >= 0 && currentIndex < buttons.length - 1) {
      const nextButton = buttons[currentIndex + 1];
      const nextUrl = getUrlFromButton(nextButton);
      if (nextUrl) {
        currentAudio = null; // Reset current audio before playing next
        playAudio(nextUrl);
      }
    } else {
      // Last audio completed
      currentAudio = null;
      allAudioCompleted = true;
      showNextPageButton();
    }
  };
}

function updateButtonState(url, isPlaying) {
  const buttons = document.querySelectorAll("button");
  url = normalizeUrl(url);

  // Use Array.from and explicit null check
  const targetButton = Array.from(buttons).find(button => {
    const buttonUrl = getUrlFromButton(button);
    return buttonUrl && url.endsWith(normalizeUrl(buttonUrl));
  });

  if (!targetButton) {
    console.warn("Button not found for URL:", url);
    return;
  }

  targetButton.setAttribute("data-playing", isPlaying.toString());
  const icon = targetButton.querySelector(".material-icons");
  if (icon) {
    icon.textContent = isPlaying ? "pause" : "play_arrow";
  }
}

function showNextPageButton() {
  const container = document.querySelector(".container");
  const buttons = Array.from(document.querySelectorAll("button"));

  // If the index isn't high enough, they haven't completed all audio. 
  if (!(numAudioCompleted >= buttons.length - 1)) {
    console.log(`Completed only ${numAudioCompleted} out of ${buttons.length - 1}.`);
    // Show a browser warning instead of showing the button, but allow bypassing
    shouldContinue = confirm("It doesn't look like you listened to all audio. Do you want to continue?");
    if (!shouldContinue) {
      return;
    }
  } 
  
  const existingButton = document.getElementById("nextPageButton");
  // If they have completed all
  if (!existingButton && allAudioCompleted) {
    const storyId = document.getElementById("storyID").value;
    const nextButton = document.createElement("div");
    nextButton.id = "nextPageButton";
    nextButton.className = "next-button";
    nextButton.innerHTML = `
      <a href="/stories/${storyId}/page2" class="button-link">
        Continue to Vocab Practice
        <span class="material-icons">arrow_forward</span>
      </a>
    `;
    container.appendChild(nextButton);
  }
}