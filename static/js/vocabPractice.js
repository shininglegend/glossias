// Update DOM elements when selects change
document.addEventListener("DOMContentLoaded", function () {
  const selects = document.querySelectorAll(".vocab-select");
  const checkButton = document.getElementById("checkAnswers");

  selects.forEach((select) => {
    select.addEventListener("change", () => checkAllSelectsFilled());
  });

  function checkAllSelectsFilled() {
    const allFilled = Array.from(selects).every(
      select => select.value !== ""
    );
    checkButton.disabled = !allFilled;
    checkButton.classList.toggle('btn-primary', allFilled);
  }

  checkButton.addEventListener("click", submitAnswers);
});

// Submit answers for checking
// vocabPractice.js
function submitAnswers() {
  const selects = document.querySelectorAll(".vocab-select");

  // Verify all selects are filled before submitting
  if (!Array.from(selects).every(select => select.value !== "")) {
    return;
  }

  const answers = Array.from(selects).map((select) => ({
    lineNumber: parseInt(select.getAttribute("data-line")),
    answers: [select.value]
  }));

  fetch(`/stories/${document.getElementById("storyID").value}/check-vocab`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ answers }),
  })
    .then((response) => response.json())
    .then((results) => showResults(results))
    .catch((error) => console.error("Error:", error));
}

// Show results after checking answers
function showResults(results) {
  const selects = document.querySelectorAll(".vocab-select");
  let correctCount = 0;

  selects.forEach((select, index) => {
    if (results.answers[index].correct) {
      select.classList.add("correct");
      correctCount++;
    } else {
      select.classList.add("incorrect");
    }
    select.disabled = true;
  });

  // Create result container if it doesn't exist
  let resultContainer = document.getElementById("resultContainer");
  if (!resultContainer) {
    resultContainer = document.createElement("div");
    resultContainer.id = "resultContainer";
    document.querySelector(".container").appendChild(resultContainer);
  }

  resultContainer.innerHTML = `
        <div class="score">Score: ${correctCount}/${selects.length}</div>
        <div class="next-button">
            <a href="/stories/${document.getElementById("storyID").value}/page3"
               class="button-link">
                Continue <span class="material-icons">arrow_forward</span>
            </a>
        </div>
    `;
}
