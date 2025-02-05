
# database.py
import sqlite3
from pathlib import Path
import logging

logger = logging.getLogger(__name__)

class Database:
    def __init__(self):
        self.db_path = Path('../logos-stories/data/stories.db')
        if not self.db_path.exists():
            raise FileNotFoundError(f"Database not found at {self.db_path.absolute()}")

    def get_story(self, story_id):
        """Get story data from database"""
        try:
            logger.debug(f"Connecting to database at {self.db_path.absolute()}")
            with sqlite3.connect(self.db_path) as conn:
                conn.row_factory = sqlite3.Row
                cur = conn.cursor()

                # Get basic story info
                logger.debug(f"Fetching story {story_id}")
                cur.execute("""
                    SELECT s.*, st.title, sd.description_text
                    FROM stories s
                    LEFT JOIN story_titles st ON s.story_id = st.story_id AND st.language_code = 'en'
                    LEFT JOIN story_descriptions sd ON s.story_id = sd.story_id AND sd.language_code = 'en'
                    WHERE s.story_id = ?
                """, (story_id,))
                story_row = cur.fetchone()
                if not story_row:
                    logger.warning(f"No story found with ID {story_id}")
                    return None

                story_data = dict(story_row)
                logger.debug(f"Story data: {story_data}")

                # Get story lines with vocabulary
                cur.execute("""
                    SELECT sl.*,
                           vi.word, vi.lexical_form, vi.position_start, vi.position_end
                    FROM story_lines sl
                    LEFT JOIN vocabulary_items vi ON sl.story_id = vi.story_id
                        AND sl.line_number = vi.line_number
                    WHERE sl.story_id = ?
                    ORDER BY sl.line_number, vi.position_start
                """, (story_id,))

                lines = []
                current_line = None

                for row in cur.fetchall():
                    row_dict = dict(row)
                    logger.debug(f"Processing line: {row_dict}")

                    if not current_line or current_line['line_number'] != row['line_number']:
                        if current_line:
                            lines.append(current_line)
                        current_line = {
                            'text': row['text'],
                            'line_number': row['line_number'],
                            'vocabulary': []
                        }

                    if row['word']:  # If there's vocabulary for this line
                        current_line['vocabulary'].append({
                            'word': row['word'],
                            'lexical_form': row['lexical_form'],
                            'position': [row['position_start'], row['position_end']]
                        })

                if current_line:
                    lines.append(current_line)

                result = {
                    'metadata': {
                        'title': {'en': story_data['title']},
                        'week_number': story_data['week_number'],
                        'day_letter': story_data['day_letter'],
                        'language': 'he'  # Assuming Hebrew
                    },
                    'content': {
                        'lines': lines
                    }
                }
                logger.debug(f"Final result: {result}")
                return result

        except Exception as e:
            logger.error(f"Database error: {e}", exc_info=True)
            raise
