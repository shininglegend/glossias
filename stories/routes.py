# routes.py
from flask import render_template, jsonify, request, current_app
from .database import Database
import traceback, os
from pathlib import Path

def configure_routes(app):
    db = Database()

    @app.route('/stories/<int:story_id>/page2')
    def page2(story_id):
        try:
            current_app.logger.info(f"Fetching story {story_id}")

            # Get story data
            story = db.get_story(story_id)
            if not story:
                current_app.logger.warning(f"Story {story_id} not found")
                return "Story not found", 404

            # Process lines and vocabulary
            vocab_bank = []
            processed_lines = []

            # Prepare audio directory path
            audio_dir = Path(f"./static/stories/stories_audio/{story['metadata']['language']}_{story['metadata']['week_number']}{story['metadata']['day_letter']}")

            # Get list of audio files if directory exists
            try:
                print(os.getcwd())
                audio_files = sorted(list(audio_dir.glob('*.mp3')))
                current_app.logger.debug(f"Found audio files: {audio_files}\nAt: {audio_dir}")
            except Exception as e:
                current_app.logger.warning(f"Could not read audio directory: {e}")
                audio_files = []

            for i, line in enumerate(story['content']['lines']):
                try:
                    processed_line, line_vocab = process_story_line(line, audio_files, i, current_app)
                    processed_lines.append(processed_line)
                    vocab_bank.extend(line_vocab)
                except Exception as line_error:
                    current_app.logger.error(f"Error processing line: {line_error}")
                    current_app.logger.error(f"Line data: {line}")
                    raise

            # Remove duplicates and sort vocab bank
            vocab_bank = sorted(set(vocab_bank))

            data = {
                'story_id': story_id,
                'story_title': story['metadata']['title']['en'],
                'lines': processed_lines,
                'vocab_bank': vocab_bank
            }

            current_app.logger.debug(f"Rendered data: {data}")
            return render_template('page2_py.html', **data)

        except Exception as e:
            current_app.logger.error(f"Error serving page2: {str(e)}")
            current_app.logger.error(traceback.format_exc())
            raise


    @app.route('/api/check-vocab', methods=['POST'])
    def check_vocab():
        try:
            data = request.get_json()
            current_app.logger.debug(f"Received vocab check request: {data}")

            answers = data.get('answers', [])
            response = {
                'answers': [
                    {
                        'correct': answer['word'] == answer['answer'],
                        'word': answer['word']
                    }
                    for answer in answers
                ]
            }

            return jsonify(response)

        except Exception as e:
            current_app.logger.error(f"Error checking vocab: {str(e)}")
            current_app.logger.error(traceback.format_exc())
            return jsonify({'error': str(e)}), 500

def process_story_line(line, audio_files, i, current_app):
    """Process a single story line with vocabulary and audio."""
    series = []
    last_end = 0
    text = line['text']
    vocab = sorted(line['vocabulary'], key=lambda x: x['position'][0])
    vocab_words = [v['lexical_form'] for v in vocab]

    # Create pairs of text and whether a vocab word follows
    for v in vocab:
        start = v['position'][0]
        if start >= last_end:
            series.append({'text': text[last_end:start], 'needs_vocab': True})
        last_end = v['position'][1]

    # Add remaining text with no following vocab
    if last_end < len(text):
        series.append({'text': text[last_end:], 'needs_vocab': False})

    audio_url = None
    if i < len(audio_files):
        rel_path = audio_files[i].relative_to('./static')
        audio_url = f"/static/{rel_path}"
        current_app.logger.debug(f"Audio URL for line {i}: {audio_url}")

    return {
        'text': series,
        'audio_url': audio_url,
        'has_vocab': bool(vocab)
    }, vocab_words
