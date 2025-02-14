// logos-stories/static/js/admin/delete.js
let currentStoryId = null;

function showDeleteModal(storyId, storyTitle) {
  currentStoryId = storyId;
  const modal = document.getElementById("deleteModal");
  const titleSpan = document.getElementById("deleteStoryTitle");

  titleSpan.textContent = storyTitle;
  modal.classList.remove("hidden");
}

function closeDeleteModal() {
  const modal = document.getElementById("deleteModal");
  modal.classList.add("hidden");
  currentStoryId = null;
}

async function confirmDelete() {
  if (!currentStoryId) return;

  try {
    const response = await fetch(`/admin/stories/delete/${currentStoryId}`, {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const result = await response.json();

    if (result.success) {
      // Remove the story item from the DOM
      const storyElement = document.querySelector(
        `[data-story-id="${currentStoryId}"]`,
      );
      if (storyElement) {
        storyElement.remove();
      }

      // Show success notification
      showNotification("Story deleted successfully", "success");
    } else {
      throw new Error("Failed to delete story");
    }
  } catch (error) {
    console.error("Error:", error);
    showNotification("Failed to delete story", "error");
  } finally {
    closeDeleteModal();
  }
}

function showNotification(message, type = "success") {
  const notification = document.createElement("div");
  notification.className = `fixed top-4 right-4 p-4 rounded-lg ${
    type === "success" ? "bg-green-500" : "bg-red-500"
  } text-white shadow-lg transition-opacity duration-500`;
  notification.textContent = message;

  document.body.appendChild(notification);

  // Remove notification after 3 seconds
  setTimeout(() => {
    notification.style.opacity = "0";
    setTimeout(() => {
      notification.remove();
    }, 500);
  }, 3000);
}

// Close modal if clicking outsides
document.getElementById("deleteModal").addEventListener("click", (e) => {
  if (e.target.id === "deleteModal") {
    closeDeleteModal();
  }
});

// Allow closing modal with Escape key
document.addEventListener("keydown", (e) => {
  if (
    e.key === "Escape" &&
    !document.getElementById("deleteModal").classList.contains("hidden")
  ) {
    closeDeleteModal();
  }
});
