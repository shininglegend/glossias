// audioCommon.js
export function normalizeUrl(url) {
    const decoded = decodeURIComponent(url);
    return decoded.replace(/\\/g, "/").replace(/\/+/g, "/");
}
  
export function getUrlFromButton(button) {
    if (button.dataset.url) {
        return button.dataset.url;
    };
    
    const onclick = button.getAttribute("onclick");
    const match = onclick?.match(/playAudio\('([^']+)'\)/);
    return match ? match[1] : null;
}

export function updateButtonState(url, isPlaying) {
    const buttons = document.querySelectorAll("button");
    url = normalizeUrl(url);

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