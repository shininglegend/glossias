
# logos-stories/scripts/convertToJSON.py
import re
import json
from pathlib import Path
from datetime import datetime, UTC
import frontmatter  # python-frontmatter package

class StoryConverter:
    def __init__(self, input_path: Path):
        self.input_path = input_path
        self.output_dir = Path("../static/stories/stories_text")

    def parse_vocab(self, line: str) -> tuple[str, list]:
        # Parse {v:word|lexical} format
        vocab_items = []
        clean_text = line

        for match in re.finditer(r'{v:([^|]+)\|([^}]+)}', line):
            word, lexical = match.group(1), match.group(2)
            start, end = match.span()
            vocab_items.append({
                "word": word,
                "lexicalForm": lexical,
                "position": [start, end]
            })
            clean_text = clean_text.replace(match.group(0), word)

        return clean_text, vocab_items

    def parse_grammar(self, line: str) -> tuple[str, list]:
        # Parse <g>text</g> format
        grammar_items = []
        clean_text = line

        for match in re.finditer(r'<g>([^<]+)</g>', line):
            text = match.group(1)
            start, end = match.span()
            grammar_items.append({
                "text": text,
                "position": [start, end]
            })
            clean_text = clean_text.replace(match.group(0), text)

        return clean_text, grammar_items

    def convert(self):
        with open(self.input_path) as f:
            post = frontmatter.load(f)

        # Extract story ID from filename if not in metadata
        default_story_id = self.input_path.stem

        # Calculate week and day from story ID if not provided
        default_week = int(default_story_id[:-1]) if default_story_id[:-1].isdigit() else 1
        default_day = default_story_id[-1] if len(default_story_id) > 0 else 'a'

        metadata = {
            "storyId": post.get('storyId', default_story_id),
            "weekNumber": post.get('week', default_week),
            "dayLetter": post.get('day', default_day),
            "title": post.get('title', {"en": "Untitled"}),
            "author": {
                "id": post.get('authorId', "unknown"),
                "name": post.get('author', "Anonymous")
            },
            "lastRevision": datetime.now(UTC).isoformat() + "Z"
        }
        lines = []
        current_line = 1

        # Parse content lines
        content_lines = post.content.split('\n')
        for line in content_lines:
            if not line.strip():
                continue

            # Remove line number prefix if present
            line = re.sub(r'^\d+\.\s*', '', line.strip())

            clean_text, vocab = self.parse_vocab(line)
            clean_text, grammar = self.parse_grammar(clean_text)

            lines.append({
                "lineNumber": current_line,
                "text": clean_text,
                "vocabulary": vocab,
                "grammar": grammar,
                "footnotes": []  # Footnotes handled separately
            })
            current_line += 1

        story = {
            "metadata": metadata,
            "content": {"lines": lines}
        }

        # Write JSON output
        output_path = self.output_dir / f"{metadata['storyId']}.json"
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(story, f, ensure_ascii=False, indent=2)

# Usage
if __name__ == "__main__":
    import sys
    if len(sys.argv) < 2:
        print("Usage: python convertToJSON.py <input_file>")
        sys.exit(1)

    converter = StoryConverter(Path(sys.argv[1]))
    converter.convert()
