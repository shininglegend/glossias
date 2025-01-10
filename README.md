# Logos Stories

A web application for displaying stories with synchronized text and audio.

## Installation & Setup

1. Install Go (1.21 or later) from [golang.org](https://golang.org)
2. Clone this repository:
   ```bash
   git clone https://github.com/yourusername/logos-stories
   cd logos-stories
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Run the application:
   ```bash
   go run .
   ```
5. Access the application at `http://localhost:8080`

## Adding Content

### Text Files
Place text files in `static/stories/stories_text/` with the following format:
- Filename: `[language_code]_[number][letter].txt` (e.g., `gr_0a.txt`, `hb_9b.txt`)
- First line: Story title
- Subsequent lines: Story text
- Use one `|` before and after to mark the vocab words. (ex: `γινωσω και |εγω|`).
- Use  `||` before and after to mark the grammar points. (ex: `||γινωσω εγω|| και`)
- Words will be split at a space character.
- Empty lines are preserved

### Audio Files
Place audio files in `static/stories/stories_audio/[story_id]/` where:
- `[story_id]` matches the text file name without extension
- Each audio file corresponds to one non-empty line in the text file
- Audio files should be numbered sequentially (e.g., `1.mp3`, `2.mp3`)

## Credits

### Content
- Most story text and audio files are by Dr. Jesse Scheumann, all rights reserved, used with permission
- All other story text and audio files were created by Titus Murphy, all rights reserved.
- If you would like to reuse this code in accordance with the license, please remove the story text and audio files and replace them with your own content.


### Development
- Code written by Titus Murphy unless otherwise noted.
- AI assistance provided by Claude.ai and GitHub Copilot and documented in AiUsage.md.

### Academic Context
This project was developed under the oversight of Dr. Derrick Tate for academic credit at Sattler College.
