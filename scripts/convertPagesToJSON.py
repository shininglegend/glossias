# logos-stories/scripts/convertPagesToJSON.py
import re
import json
from pathlib import Path
from datetime import datetime, UTC
from typing import Dict, List, Tuple, Any

class StoryConverter:
    def __init__(self, input_path: Path):
        self.input_path = input_path
        # Ensure output directory exists
        self.output_dir = Path("../static/stories/stories_text")
        self.output_dir.mkdir(parents=True, exist_ok=True)

    def parse_metadata(self, metadata_text: str) -> Dict[str, Any]:
        # Initialize with default values
        metadata = {
            "storyId": "",
            "weekNumber": 1,
            "dayLetter": "a",
            "title": {"en": "", "he": ""},
            "author": {"id": "", "name": ""},
            "grammarPoint": "",
            "lastRevision": datetime.now(UTC).isoformat() + "Z"
        }

        # Parse YAML-like structure
        current_dict = None
        for line in metadata_text.strip().split('\n'):
            if not line.strip(): continue

            if ':' in line and not line.startswith(' '):
                key, value = [x.strip() for x in line.split(':', 1)]
                if value:
                    if key == 'week':
                        metadata['weekNumber'] = int(value)
                    elif key == 'day':
                        metadata['dayLetter'] = value.strip('"')
                    elif key == 'storyId':
                        metadata['storyId'] = value.strip('"')
                    elif key == 'authorId':
                        metadata['author']['id'] = value.strip('"')
                    elif key == 'author':
                        metadata['author']['name'] = value.strip('"')
                    elif key == 'grammarPoint':
                        metadata['grammarPoint'] = value.strip('"')
                else:
                    current_dict = key
            elif current_dict == 'title' and line.strip().startswith(('en:', 'he:')):
                lang, text = [x.strip() for x in line.strip().split(':', 1)]
                metadata['title'][lang] = text.strip('"')

        return metadata

    def parse_footnotes(self, content: str) -> Tuple[str, List[Dict[str, Any]]]:
        # Extract footnotes and clean content
        footnotes = []
        content_lines = []

        for line in content.split('\n'):
            if line.strip().startswith('[') and ']:' in line:
                # Parse footnote
                footnote_id = int(re.search(r'\[(\d+)\]', line).group(1))
                footnote_text = line.split(':', 1)[1].strip()
                footnotes.append({
                    "id": footnote_id,
                    "text": footnote_text,
                    "references": []
                })
            else:
                content_lines.append(line)

        return '\n'.join(content_lines), footnotes

    def parse_line(self, line: str, footnotes: List[Dict[str, Any]]) -> Dict[str, Any]:
        # Initialize line data
        clean_text = line
        vocabulary = []
        grammar = []
        line_footnotes = []

        # Parse vocabulary items {v:word|lexical}
        for match in re.finditer(r'\{v:([^|]+)\|([^}]+)\}', line):
            word, lexical = match.group(1), match.group(2)
            start, end = match.span()
            vocabulary.append({
                "word": word,
                "lexicalForm": lexical,
                "position": [start, end]
            })
            clean_text = clean_text[:start] + word + clean_text[end:]

        # Parse grammar points <g>text</g>
        for match in re.finditer(r'<g>([^<]+)</g>', clean_text):
            text = match.group(1)
            start, end = match.span()
            grammar.append({
                "text": text,
                "position": [start, end]
            })
            clean_text = clean_text[:start] + text + clean_text[end:]

        # Check for footnote references and add them
        for footnote in footnotes:
            if f'[{footnote["id"]}]' in clean_text:
                line_footnotes.append(footnote)
                clean_text = clean_text.replace(f'[{footnote["id"]}]', '')

        return {
            "text": clean_text.strip(),
            "vocabulary": vocabulary,
            "grammar": grammar,
            "footnotes": line_footnotes
        }

    def convert(self):
        # Read input file
        content = self.input_path.read_text(encoding='utf-8')

        # Split into metadata and content sections
        try:
            metadata_text, content_text = content.split('## Content', 1)
        except ValueError:
            raise ValueError("File must contain '## Content' section")

        # Parse metadata
        metadata = self.parse_metadata(metadata_text.replace('## Metadata\n', ''))

        # Parse content and footnotes
        content_text, footnotes = self.parse_footnotes(content_text.strip())

        # Process lines
        lines = []
        for line_num, line in enumerate(content_text.split('\n'), 1):
            if not line.strip():
                continue

            line_data = self.parse_line(line, footnotes)
            line_data['lineNumber'] = line_num
            lines.append(line_data)

        # Construct final JSON
        story_data = {
            "metadata": metadata,
            "content": {
                "lines": lines
            }
        }

        # Write output to a JSON file named the same in the output directory
        output_path = self.output_dir / f"{self.input_path.stem}.json"
        with output_path.open('w', encoding='utf-8') as f:
            json.dump(story_data, f, ensure_ascii=False, indent=2)

def main():
    import sys
    if len(sys.argv) != 2:
        print("Usage: python convertPagesToJSON.py <input_file>")
        return

    converter = StoryConverter(Path(sys.argv[1]))
    converter.convert()

if __name__ == "__main__":
    main()
