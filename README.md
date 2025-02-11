# Logos Stories

A web application for displaying stories with synchronized text and audio.

## Installation & Setup
1. Run steps 1-3 of `Go`  
2. Run steps 1-3 of `Python`
3. Run steps 1-2 of `nginx`
#### Future runs
1. Start Nginx `./start-dev.sh`
2. Start VS code debugging by selecting `Go + Python`

### Part 1: Go
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

### Part 2: Python
1. Install Python (3.11 or later)
2. Create and activate a venv in the root project directory
3. Install dependancies
```bash
pip install -r requirements.txt
```

### Part 3: Nginx
1. Install nginx
2. Make the dev executable
```bash
chmod +x start-dev.sh
```

### To stop:
1. Stop nginx
```bash
nginx -s stop
```


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
