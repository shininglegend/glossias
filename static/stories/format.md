
# Story File Format Specification

## Overview
Stories are stored in JSON format with a structured schema for metadata, text content, vocabulary items, grammar points, and footnotes.

## File Structure
Each story file contains two main sections:
1. Metadata
2. Content

### Metadata Section
Contains essential information about the story:
- `storyId`: Unique identifier (e.g., 1)
- `weekNumber`: Numerical week (e.g., 9)
- `dayLetter`: Single letter for day (e.g., "b")
- `title`: Multi-language map of story titles
- `author`: Author information (id and name)
- `grammarPoint`: What the grammar point is. Will be displayed as "This story's grammar point is: [grammar point], find x examples in the text."
- `description`: Optional description of the story. Contains a language code and the text of the description.
- `lastRevision`: ISO-8601 timestamp of last update

### Content Section
Contains an array of lines, each with:
- `lineNumber`: 1-based line number
- `text`: Main text content
- `vocabulary`: Array of vocabulary items
- `grammar`: Array of grammar points
- `audioFile`: Optional audio filename
- `footnotes`: Array of footnotes

#### Vocabulary Items
Each vocabulary item contains:
- `word`: The word as it appears in text
- `lexicalForm`: The dictionary/lexical form
- `position`: Array of [start, end] character positions in the line

#### Grammar Points
Each grammar point contains:
- `text`: The grammatical construction
- `position`: Array of [start, end] character positions in the line

#### Footnotes
Each footnote contains:
- `id`: Numerical identifier
- `text`: Footnote content
- `references`: Optional array of reference strings

## Example
```json
{
  "metadata": {
    "storyId": "9b",
    "weekNumber": 9,
    "dayLetter": "b",
    "title": {
      "en": "Covenant with Ashdod",
      "he": "ברית עם אשדוד"
    },
    "author": {
      "id": "auth123",
      "name": "John Smith"
    },
    "grammarPoint": "Veqatal verbs. See 9a in the workbook.",
    "lastRevision": "2024-01-17T12:00:00Z"
  },
  "content": {
    "lines": [
      {
        "lineNumber": 1,
        "text": "הָיְתָה בְרִית בֵּין יִשְׂרָאֵל וּבֵין אַשְׁדּוֹד לֵאמֹר",
        "vocabulary": [
          {
            "word": "בְרִית",
            "lexicalForm": "ברית",
            "position": [7, 12]
          }
        ],
        "grammar": [],
        "audioFile": "9b_l1.mp3",
        "footnotes": []
      },
      {
        "lineNumber": 2,
        "text": "אִם תַּעֲשׂוּ עִמָּנוּ טוֹב",
        "vocabulary": [],
        "grammar": [],
        "audioFile": "9b_l2.mp3",
        "footnotes": []
      },
      {
        "lineNumber": 3,
        "text": "וְעָשִׂינוּ עִמָּכֶם טוֹב",
        "vocabulary": [],
        "grammar": [
          {
            "text": "וְעָשִׂינוּ",
            "position": [0, 7]
          },
          {
            "text": "עִמָּכֶם טוֹב",
            "position": [8, 16]
          }
        ],
        "audioFile": "9b_l3.mp3",
        "footnotes": []
      }
    ]
  }
}
```

## Notes
- All character positions are zero-based
- Audio files are optional. It is encouraged to use this pattern for naming: `[story_id]-[line_number].mp3`
  - Line numbers should be zero-padded to a consistent length, usually 2 digits (e.g., `-01.mp3`, `-12.mp3`)
- Multiple vocabulary words and grammar points can appear in a single line
- Lines are numbered sequentially starting from 1
