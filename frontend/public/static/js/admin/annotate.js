// glossias/static/js/admin/annotate.js
const AnnotationManager = {
  selectedText: "",
  selectionStart: 0,
  selectionEnd: 0,
  currentLine: null,

  init() {
    this.bindEvents();
  },

  bindEvents() {
    // Handle text selection events
    document.querySelectorAll(".story-line-text").forEach((line) => {
      line.addEventListener("mouseup", this.handleTextSelection.bind(this));
    });

    // Bind annotation button handlers
    document
      .getElementById("addVocabBtn")
      .addEventListener("click", this.addVocabularyItem.bind(this));
    document
      .getElementById("addGrammarBtn")
      .addEventListener("click", this.addGrammarItem.bind(this));
    document
      .getElementById("addFootnoteBtn")
      .addEventListener("click", this.addFootnote.bind(this));
  },

  handleTextSelection(event) {
    const selection = window.getSelection();
    if (selection.toString().length > 0) {
      this.selectedText = selection.toString();
      const range = selection.getRangeAt(0);
      const lineElement = event.target.closest(".story-line");

      this.currentLine = lineElement;
      this.selectionStart = range.startOffset;
      this.selectionEnd = range.endOffset;

      // Show annotation options
      this.showAnnotationOptions(event.pageX, event.pageY);
    }
  },

  showAnnotationOptions(x, y) {
    const menu = document.getElementById("annotationMenu");
    menu.style.display = "block";
    menu.style.left = `${x}px`;
    menu.style.top = `${y}px`;
  },

  // Update the addVocabularyItem method
  async addVocabularyItem() {
    if (!this.selectedText || !this.currentLine) return;

    const vocabItem = {
      word: this.selectedText,
      lexicalForm: "",
      position: [this.selectionStart, this.selectionEnd],
    };

    const lexicalForm = await this.showVocabModal(vocabItem);
    if (lexicalForm) {
      vocabItem.lexicalForm = lexicalForm;
      const response = await this.saveAnnotation("vocabulary", vocabItem);
      if (response.success) {
        // Parse current vocabulary
        const currentVocab = JSON.parse(
          this.currentLine.getAttribute("data-vocabulary") || "[]",
        );
        currentVocab.push(vocabItem);
        this.currentLine.setAttribute(
          "data-vocabulary",
          JSON.stringify(currentVocab),
        );

        // Hide annotation menu
        document.getElementById("annotationMenu").style.display = "none";

        // Immediately refresh highlights
        const highlighter = new TextHighlighter();
        const textElement = this.currentLine.querySelector(".story-line-text");
        if (textElement) {
          highlighter.highlightLine(textElement, {
            vocabulary: currentVocab,
            grammar: JSON.parse(
              this.currentLine.getAttribute("data-grammar") || "[]",
            ),
          });
        }
      }
    }
  },

  async addGrammarItem() {
    if (!this.selectedText || !this.currentLine) return;

    const grammarItem = {
      text: this.selectedText,
      position: [this.selectionStart, this.selectionEnd],
    };

    const response = await this.saveAnnotation("grammar", grammarItem);
    if (response.success) {
      const currentGrammar = JSON.parse(
        this.currentLine.dataset.grammar || "[]",
      );
      currentGrammar.push(grammarItem);
      this.currentLine.dataset.grammar = JSON.stringify(currentGrammar);

      // Hide annotation menu
      document.getElementById("annotationMenu").style.display = "none";

      // Immediately refresh highlights
      const highlighter = new TextHighlighter();
      const textElement = this.currentLine.querySelector(".story-line-text");
      highlighter.highlightLine(textElement, {
        vocabulary: JSON.parse(this.currentLine.dataset.vocabulary || "[]"),
        grammar: currentGrammar,
      });
    }
  },

  async addFootnote() {
    if (!this.selectedText) return;

    const footnote = {
      text: "",
      references: [this.selectedText],
    };

    // Show modal for footnote text
    const footnoteText = await this.showFootnoteModal();
    if (footnoteText) {
      footnote.text = footnoteText;
      const response = await this.saveAnnotation("footnote", footnote);
      if (response.success) {
        // Refresh the page to show new footnote
        window.location.reload();
      }
    }
  },

  async saveAnnotation(type, data) {
    const lineNumber = parseInt(this.currentLine.dataset.line);
    const storyId = document.getElementById("storyId").value;

    try {
      const response = await fetch(`/admin/stories/${storyId}/annotate`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          lineNumber,
          [type]: data,
        }),
      });

      if (!response.ok) throw new Error("Failed to save annotation");
      return await response.json();
    } catch (error) {
      console.error("Error saving annotation:", error);
      alert("Failed to save annotation. Please try again.");
      return { success: false };
    }
  },

  // Modal helpers
  showVocabModal(item) {
    return new Promise((resolve) => {
      const lexicalForm = prompt("Enter lexical form:", "");
      resolve(lexicalForm);
    });
  },

  showFootnoteModal() {
    return new Promise((resolve) => {
      const text = prompt("Enter footnote text:", "");
      resolve(text);
    });
  },
};

// Close annotation menu when clicking outside
document.addEventListener("click", function (e) {
  const menu = document.getElementById("annotationMenu");
  if (!menu.contains(e.target) && !e.target.closest(".story-line-text")) {
    menu.style.display = "none";
  }
});

// Initialize when DOM is loaded
document.addEventListener("DOMContentLoaded", () => {
  AnnotationManager.init();
});
