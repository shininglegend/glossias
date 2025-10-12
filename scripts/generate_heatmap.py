"""
Generate a heatmap overlay on story text showing grammar point click patterns.
Green intensity = correct answers, Red intensity = incorrect answers.
"""

import os
import sys
import argparse
import psycopg2
from datetime import datetime
from dotenv import load_dotenv

MAX_CLICKS = 10  # Max clicks on a position for intensity normalization


def load_db_config():
    """Load database configuration from .env file."""
    load_dotenv()
    return {
        "user": os.getenv("user"),
        "password": os.getenv("password"),
        "host": os.getenv("host"),
        "port": os.getenv("port"),
        "dbname": os.getenv("dbname"),
    }


def create_db_connection(config):
    """Create database connection using config dict."""
    return psycopg2.connect(
        user=config["user"],
        password=config["password"],
        host=config["host"],
        port=config["port"],
        dbname=config["dbname"],
    )


def get_story_lines(cursor, story_id):
    """Get story text lines in order."""
    cursor.execute(
        """
        SELECT line_number, text
        FROM story_lines
        WHERE story_id = %s
        ORDER BY line_number
        """,
        (story_id,),
    )
    return cursor.fetchall()


def get_grammar_items_with_positions(cursor, story_id):
    """Get grammar items with their text positions."""
    cursor.execute(
        """
        SELECT gi.line_number, gi.grammar_point_id, gi.text,
               gi.position_start, gi.position_end, gp.name
        FROM grammar_items gi
        JOIN grammar_points gp ON gi.grammar_point_id = gp.grammar_point_id
        WHERE gi.story_id = %s
        ORDER BY gi.line_number, gi.position_start
        """,
        (story_id,),
    )
    return cursor.fetchall()


def get_all_click_data(cursor, story_id):
    """Get all clicks (correct and incorrect) with their exact positions."""
    # Get correct clicks - map to grammar item positions
    cursor.execute(
        """
        SELECT gca.line_number, gi.position_start, gi.position_end, 'correct' as click_type
        FROM grammar_correct_answers gca
        JOIN grammar_items gi ON gca.grammar_point_id = gi.grammar_point_id
            AND gca.story_id = gi.story_id
            AND gca.line_number = gi.line_number
        WHERE gca.story_id = %s
        """,
        (story_id,),
    )
    correct_clicks = cursor.fetchall()

    # Get incorrect clicks - use selected_positions
    cursor.execute(
        """
        SELECT line_number, selected_line, selected_positions, 'incorrect' as click_type
        FROM grammar_incorrect_answers
        WHERE story_id = %s
        """,
        (story_id,),
    )
    incorrect_clicks = cursor.fetchall()
    print("Correct clicks:\n", correct_clicks)
    print("\n--\nIncorrect clicks:\n", incorrect_clicks)

    return correct_clicks, incorrect_clicks


def get_grammar_click_data(cursor, story_id):
    """Get grammar click data for summary statistics."""
    cursor.execute(
        """
        SELECT line_number, grammar_point_id, COUNT(*) as count
        FROM grammar_correct_answers
        WHERE story_id = %s
        GROUP BY line_number, grammar_point_id
        """,
        (story_id,),
    )
    correct_data = {(row[0], row[1]): row[2] for row in cursor.fetchall()}

    cursor.execute(
        """
        SELECT line_number, grammar_point_id, selected_positions
        FROM grammar_incorrect_answers
        WHERE story_id = %s
        """,
        (story_id,),
    )
    incorrect_position_data = cursor.fetchall()

    cursor.execute(
        """
        SELECT line_number, grammar_point_id, COUNT(*) as count
        FROM grammar_incorrect_answers
        WHERE story_id = %s
        GROUP BY line_number, grammar_point_id
        """,
        (story_id,),
    )
    incorrect_data = {(row[0], row[1]): row[2] for row in cursor.fetchall()}

    print("correct_data", correct_data)
    print("incorrect_data", incorrect_data)
    print("incorrect position data", incorrect_data)

    return correct_data, incorrect_data, incorrect_position_data


def calculate_text_positions(lines, char_width=0.6, line_height=1.2):
    """Calculate character positions for each line of text."""
    positions = {}
    max_line_length = max(len(line[1]) for line in lines) if lines else 50

    for line_num, text in lines:
        # Check if text contains Hebrew characters
        has_hebrew = any(ord(char) >= 0x590 and ord(char) <= 0x5FF for char in text)

        positions[line_num] = {
            "text": text,
            "y_pos": -line_num * line_height,  # Lines go downward
            "char_positions": [],
            "is_rtl": has_hebrew,
        }

        # Calculate x position for each character
        for i, char in enumerate(text):
            if has_hebrew:
                # RTL: start from right and go left
                x_pos = (max_line_length - i - 1) * char_width
            else:
                # LTR: normal left-to-right
                x_pos = i * char_width
            positions[line_num]["char_positions"].append(
                {"char": char, "x": x_pos, "y": positions[line_num]["y_pos"]}
            )

    return positions


def create_text_heatmap_output(
    lines, grammar_items, correct_data, incorrect_data, incorrect_position_data
):
    """Create text-based heatmap showing click counts by character position."""
    text_heatmap = []

    for line_num, text in lines:
        # Initialize click count array for this line
        click_counts = [0] * len(text)

        # Add correct clicks from grammar items
        for (
            item_line_num,
            gp_id,
            item_text,
            pos_start,
            pos_end,
            gp_name,
        ) in grammar_items:
            if item_line_num != line_num:
                continue

            # Get correct click count for this grammar item
            correct_count = correct_data.get((line_num, gp_id), 0)

            # Apply correct clicks to each position in the grammar item range
            for pos in range(pos_start, min(pos_end, len(text))):
                click_counts[pos] += correct_count

        # Add incorrect clicks from selected_positions
        for inc_line_num, inc_gp_id, selected_positions in incorrect_position_data:
            if inc_line_num != line_num:
                continue

            # selected_positions is an array of clicked positions
            if selected_positions:
                for pos in selected_positions:
                    if 0 <= pos < len(text):
                        click_counts[pos] += 1

        # Convert to string representation
        click_string = "".join(
            str(min(count, 9)) for count in click_counts
        )  # Cap at 9 for readability

        text_heatmap.append(
            {"line_number": line_num, "text": text, "clicks": click_string}
        )

    return text_heatmap


def print_text_heatmap(text_heatmap):
    """Print the text-based heatmap."""
    print(f"\nText-based Click Heatmap:")
    print("=" * 50)
    print("Format: Line text followed by click counts (0-9, 9=9+)")
    print("=" * 50)

    for line_data in text_heatmap:
        print(f"Line {line_data['line_number']}:")
        print(f"Text:   {line_data['text']}")
        print(f"Clicks: {line_data['clicks']}")
        print()


def generate_output_filename(story_id, custom_output):
    """Generate output filename with timestamp if not provided."""
    if custom_output:
        return custom_output
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    return f"grammar_heatmap_{story_id}_{timestamp}.html"


def main():
    parser = argparse.ArgumentParser(
        description="Generate grammar click heatmap on story text"
    )
    parser.add_argument("story_id", type=int, help="Story ID to generate heatmap for")
    parser.add_argument(
        "-o",
        "--output",
        help="Output file path (default: grammar_heatmap_STORYID_TIMESTAMP.html)",
    )

    args = parser.parse_args()
    # Use this output filename for HTML
    output_file = generate_output_filename(args.story_id, args.output)

    try:
        # Database connection
        config = load_db_config()
        connection = create_db_connection(config)
        cursor = connection.cursor()

        # Get story data
        lines = get_story_lines(cursor, args.story_id)
        if not lines:
            print(f"No story lines found for story {args.story_id}")
            sys.exit(1)

        # print("lines", lines)

        grammar_items = get_grammar_items_with_positions(cursor, args.story_id)
        if not grammar_items:
            print(f"No grammar items found for story {args.story_id}")
            sys.exit(1)

        # print("grammar items", grammar_items)

        # Get summary click data for text heatmap
        correct_data, incorrect_data, incorrect_position_data = get_grammar_click_data(
            cursor, args.story_id
        )

        # TODO: Create html-based visualization

        # Create and print text-based heatmap
        text_heatmap = create_text_heatmap_output(
            lines, grammar_items, correct_data, incorrect_data, incorrect_position_data
        )
        print_text_heatmap(text_heatmap)

        # TODO: Print summary

        # Cleanup
        cursor.close()
        connection.close()

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
