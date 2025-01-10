let currentAudio = null;

function playAudio(url) {
  console.log("playAudio", url);
  if (currentAudio && currentAudio.src.endsWith(url)) {
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

  currentAudio = new Audio(url);
  currentAudio.play();
  updateButtonState(url, true);

  currentAudio.onended = function () {
    updateButtonState(url, false);
    const buttons = document.querySelectorAll("button");
    const currentButton = Array.from(buttons).find((button) =>
      button.getAttribute("onclick").includes(url),
    );
    const currentIndex = Array.from(buttons).indexOf(currentButton);

    if (currentIndex < buttons.length - 1) {
      const nextButton = buttons[currentIndex + 1];
      const nextUrl = nextButton.getAttribute("onclick").match(/'([^']+)'/)[1];
      console.log("nextUrl", nextUrl);
      playAudio(nextUrl);
    }
  };
}

function updateButtonState(url, isPlaying) {
  const buttons = document.querySelectorAll("button");
  const targetButton = Array.from(buttons).find((button) =>
    button.getAttribute("onclick").includes(url),
  );
  if (targetButton) {
    targetButton.setAttribute("data-playing", isPlaying.toString());
    const icon = targetButton.querySelector(".material-icons");
    icon.textContent = isPlaying ? "pause" : "play_arrow";
  }
}
