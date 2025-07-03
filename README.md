# Logos Stories

A web application for displaying stories with synchronized text and audio.

## Installation & Setup
1. Run steps 1-3 of `Go`  

### Part 1: Go
1. Install Go (1.21 or later) from [golang.org](https://golang.org)
2. Clone this repository:
   ```bash
   git clone https://github.com/shininglegend/glossias
   cd glossias
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. ```bash
   go run main.go
   ```

### To stop:
1. Ctrl-c


## Adding Content

### Stories
Add stories via the admin interface at `/admin`.

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
- Code written by Titus unless otherwise noted.
- AI assistance provided by claude.ai, GitHub Copilot, and Ollama using multiple models. Documentation is in AiUsage.md.

### Academic Context
This project was developed under the oversight of Dr. Derrick Tate for academic credit at Sattler College.
