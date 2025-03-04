# Logos Stories

A web application for displaying stories with synchronized text and audio.

## Installation & Setup
1. Run steps 1-3 of `Go`

### Installing Go & Postgres
*Note: Steps 2-3 may be skipped/delayed for testing, but will result in many broken features.*
1. Install Go (1.21 or later) from [golang.org](https://golang.org)
2. Install and start up postgresql from [postgresql.org](https://www.postgresql.org/download/)
3. Add a `DATABASE_URL` to your environment with your [postgres username, password, port, etc](https://www.prisma.io/docs/orm/overview/databases/postgresql#connection-url)
4. Clone this repository:
   ```bash
   git clone https://github.com/shininglegend/glossias
   cd glossias
   ```
5. Install dependencies:
   ```bash
   go mod tidy
   ```
6. ```bash
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
- Code written by Titus Murphy unless otherwise noted.
- AI assistance provided by claude.ai, GitHub Copilot, and Ollama using multiple models. Documentation is in AiUsage.md.

### Academic Context
This project was developed under the oversight of Dr. Derrick Tate for academic credit at Sattler College.
