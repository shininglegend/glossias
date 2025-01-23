// Update the showResults function
function showResults(results) {
  const blanks = document.querySelectorAll(".blank");
  let correctCount = 0;

  blanks.forEach((blank, index) => {
    if (results.answers[index].correct) {
      blank.classList.add("bg-green-100", "border-green-500");
      correctCount++;
    } else {
      blank.classList.add("bg-red-100", "border-red-500");
    }
  });

  // Show score and next page button
  const resultContainer = document.getElementById("resultContainer");
  resultContainer.classList.remove("hidden");

  const scoreDiv = resultContainer.querySelector(".score");
  scoreDiv.textContent = `Score: ${correctCount}/${blanks.length}`;

  const nextButton = resultContainer.querySelector("a");
  nextButton.href = `/stories/${document.getElementById("storyID").value}/page3`;
}

// Update fillBlank function
function fillBlank(blank, word) {
  blank.textContent = word.textContent;
  blank.classList.add("border-solid", "bg-blue-50");
  word.classList.add("opacity-50", "cursor-not-allowed");
  filledBlanks.set(blank, word);
  blank.dataset.answer = word.textContent;
}

// Update clearBlank function
function clearBlank(blank) {
  const word = filledBlanks.get(blank);
  if (word) {
    word.classList.remove("opacity-50", "cursor-not-allowed");
    filledBlanks.delete(blank);
  }
  blank.textContent = "";
  blank.classList.remove(
    "border-solid",
    "bg-blue-50",
    "bg-green-100",
    "bg-red-100",
    "border-green-500",
    "border-red-500",
  );
  delete blank.dataset.answer;

  checkAllBlanksFilled();
}
