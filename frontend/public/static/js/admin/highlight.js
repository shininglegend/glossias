// glossias/static/js/admin/highlight.js
class TextHighlighter {
  constructor() {
    this.tooltipElement = this.createTooltip();
    document.body.appendChild(this.tooltipElement);
  }

  createTooltip() {
    const tooltip = document.createElement("div");
    tooltip.className = "tooltip";
    return tooltip;
  }

  highlightLine(lineElement, annotations) {
    const text = lineElement.textContent;

    /* Replace character-by-character tracking with boundary markers */
    const boundaries = this.createBoundaries(text, annotations);
    const html = this.generateHTML(text, boundaries);

    lineElement.innerHTML = html;
    this.attachTooltipListeners(lineElement);
  }

  createBoundaries(text, annotations) {
    const boundaries = new Map(); // Using Map for O(1) lookups

    // Process vocabulary annotations
    (annotations.vocabulary || []).forEach((vocab) => {
      const [start, end] = vocab.position;

      if (!boundaries.has(start)) boundaries.set(start, []);
      if (!boundaries.has(end)) boundaries.set(end, []);

      boundaries.get(start).push({
        type: "start",
        class: "vocab-highlight",
        data: `data-lexical="${encodeURIComponent(vocab.lexicalForm)}"`,
        priority: 1,
      });

      boundaries.get(end).push({
        type: "end",
        class: "vocab-highlight",
        priority: -1,
      });
    });

    // Process grammar annotations
    (annotations.grammar || []).forEach((grammar) => {
      const [start, end] = grammar.position;

      if (!boundaries.has(start)) boundaries.set(start, []);
      if (!boundaries.has(end)) boundaries.set(end, []);

      boundaries.get(start).push({
        type: "start",
        class: "grammar-highlight",
        data: `data-grammar="${encodeURIComponent(grammar.text)}"`,
        priority: 2,
      });

      boundaries.get(end).push({
        type: "end",
        class: "grammar-highlight",
        priority: -2,
      });
    });

    return new Map([...boundaries.entries()].sort((a, b) => a[0] - b[0]));
  }

  attachTooltipListeners(lineElement) {
    const highlightedElements = lineElement.querySelectorAll(
      ".vocab-highlight, .grammar-highlight",
    );

    highlightedElements.forEach((element) => {
      element.addEventListener("mouseenter", (e) => {
        const text = e.target.hasAttribute("data-lexical")
          ? `Lexical form: ${decodeURIComponent(e.target.dataset.lexical)}`
          : `Grammar note: ${decodeURIComponent(e.target.dataset.grammar)}`;

        this.tooltipElement.textContent = text;
        this.tooltipElement.style.display = "block";

        const rect = e.target.getBoundingClientRect();
        this.tooltipElement.style.left = `${rect.left}px`;
        this.tooltipElement.style.top = `${rect.bottom + 5}px`;
      });

      element.addEventListener("mouseleave", () => {
        this.tooltipElement.style.display = "none";
      });
    });
  }

  generateHTML(text, boundaries) {
    let html = "";
    let lastPos = 0;
    const openTags = [];

    for (const [pos, markers] of boundaries) {
      // Add text before current position
      html += text.substring(lastPos, pos);

      // Sort markers to handle nested tags properly
      markers.sort((a, b) => {
        if (a.type === b.type) return b.priority - a.priority;
        return a.type === "end" ? 1 : -1;
      });

      // Process markers
      markers.forEach((marker) => {
        if (marker.type === "start") {
          html += `<span class="${marker.class}" ${marker.data || ""}>`;
          openTags.push(marker);
        } else {
          // Find matching opening tag
          const matchIndex = openTags.findIndex(
            (t) => t.class === marker.class,
          );
          if (matchIndex !== -1) {
            // Close all tags up to match and reopen others
            const toReopen = openTags.splice(matchIndex + 1);
            for (let i = matchIndex; i >= 0; i--) {
              html += "</span>";
            }
            openTags.splice(matchIndex, 1);
            // Reopen tags that were closed
            toReopen.forEach((tag) => {
              html += `<span class="${tag.class}" ${tag.data || ""}>`;
              openTags.push(tag);
            });
          }
        }
      });

      lastPos = pos;
    }

    // Add remaining text
    html += text.substring(lastPos);

    // Close any remaining tags
    while (openTags.length) {
      html += "</span>";
      openTags.pop();
    }

    return html;
  }
}

// Initialize when DOM is loaded
document.addEventListener("DOMContentLoaded", () => {
  const highlighter = new TextHighlighter();

  // Process each line
  document.querySelectorAll(".story-line").forEach((line) => {
    const textElement = line.querySelector(".story-line-text");
    if (textElement) {
      // Get annotations from the data attributes
      const annotations = {
        vocabulary: JSON.parse(line.getAttribute("data-vocabulary") || "[]"),
        grammar: JSON.parse(line.getAttribute("data-grammar") || "[]"),
      };

      highlighter.highlightLine(textElement, annotations);
    }
  });
});
