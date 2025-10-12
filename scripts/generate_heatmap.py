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

    return correct_data, incorrect_data, incorrect_position_data


def get_all_grammar_points(grammar_items):
    """Extract unique grammar point IDs and names."""
    gp_map = {}
    for _, gp_id, _, _, _, gp_name in grammar_items:
        if gp_id not in gp_map:
            gp_map[gp_id] = gp_name
    return gp_map


def build_heatmap_data(lines, grammar_items, correct_data, incorrect_position_data, grammar_point_id):
    """Build character-level heatmap for a specific grammar point."""
    # Initialize heatmap structure: {line_num: {char_pos: {'correct': count, 'incorrect': count}}}
    heatmap = {}
    
    for line_num, text in lines:
        heatmap[line_num] = {i: {'correct': 0, 'incorrect': 0} for i in range(len(text))}
    
    # Add correct clicks (spread across grammar item span)
    for item_line_num, gp_id, item_text, pos_start, pos_end, gp_name in grammar_items:
        if gp_id != grammar_point_id:
            continue
        
        correct_count = correct_data.get((item_line_num, gp_id), 0)
        
        if correct_count > 0 and item_line_num in heatmap:
            for pos in range(pos_start, min(pos_end, len(heatmap[item_line_num]))):
                heatmap[item_line_num][pos]['correct'] += correct_count
    
    # Add incorrect clicks (precise positions)
    for inc_line_num, inc_gp_id, selected_positions in incorrect_position_data:
        if inc_gp_id != grammar_point_id:
            continue
        
        if selected_positions and inc_line_num in heatmap:
            for pos in selected_positions:
                if 0 <= pos < len(heatmap[inc_line_num]):
                    heatmap[inc_line_num][pos]['incorrect'] += 1
    
    return heatmap


def generate_html_heatmap(lines, heatmap_data, grammar_point_name, story_id):
    """Generate HTML with color-coded text overlay."""
    
    def get_background_color(correct_count, incorrect_count):
        """Calculate background color based on click counts."""
        if correct_count == 0 and incorrect_count == 0:
            return 'transparent'
        
        # Normalize intensities
        correct_intensity = min(correct_count / MAX_CLICKS, 1.0)
        incorrect_intensity = min(incorrect_count / MAX_CLICKS, 1.0)
        
        # Blend colors: green for correct, red for incorrect
        if incorrect_count > correct_count:
            # More incorrect: red with intensity
            opacity = incorrect_intensity * 0.7
            return f'rgba(255, 0, 0, {opacity})'
        elif correct_count > incorrect_count:
            # More correct: green with intensity
            opacity = correct_intensity * 0.7
            return f'rgba(0, 200, 0, {opacity})'
        else:
            # Equal: yellow
            opacity = max(correct_intensity, incorrect_intensity) * 0.7
            return f'rgba(255, 200, 0, {opacity})'
    
    html_lines = []
    
    for line_num, text in lines:
        if line_num not in heatmap_data:
            html_lines.append(f'<div class="story-line">{text}</div>')
            continue
        
        # Check if text contains Hebrew
        has_hebrew = any(ord(char) >= 0x590 and ord(char) <= 0x5FF for char in text)
        direction = 'rtl' if has_hebrew else 'ltr'
        
        # Build character-by-character spans
        char_spans = []
        for i, char in enumerate(text):
            click_data = heatmap_data[line_num][i]
            bg_color = get_background_color(click_data['correct'], click_data['incorrect'])
            
            title = f"Correct: {click_data['correct']}, Incorrect: {click_data['incorrect']}"
            char_spans.append(
                f'<span style="background-color: {bg_color};" title="{title}">{char}</span>'
            )
        
        html_lines.append(
            f'<div class="story-line" style="direction: {direction};">{"".join(char_spans)}</div>'
        )
    
    # Build full HTML
    html = f"""<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Grammar Heatmap: {grammar_point_name} (Story {story_id})</title>
    <style>
        body {{
            font-family: 'Arial', sans-serif;
            max-width: 900px;
            margin: 40px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }}
        .header {{
            background-color: #fff;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        h1 {{
            margin: 0 0 10px 0;
            color: #333;
            font-size: 24px;
        }}
        .subtitle {{
            color: #666;
            font-size: 14px;
            margin: 5px 0;
        }}
        .legend {{
            background-color: #fff;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .legend-title {{
            font-weight: bold;
            margin-bottom: 10px;
            color: #333;
        }}
        .legend-item {{
            display: inline-block;
            margin-right: 20px;
            margin-bottom: 5px;
        }}
        .legend-color {{
            display: inline-block;
            width: 20px;
            height: 20px;
            vertical-align: middle;
            margin-right: 5px;
            border: 1px solid #ccc;
        }}
        .story-container {{
            background-color: #fff;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .story-line {{
            font-size: 20px;
            line-height: 2;
            margin-bottom: 5px;
            white-space: pre-wrap;
        }}
        .story-line span {{
            padding: 2px 0;
            transition: background-color 0.2s;
        }}
        .story-line span:hover {{
            outline: 2px solid #333;
        }}
    </style>
</head>
<body>
    <div class="header">
        <h1>Grammar Click Heatmap</h1>
        <div class="subtitle">Story ID: {story_id}</div>
        <div class="subtitle">Grammar Point: {grammar_point_name}</div>
        <div class="subtitle">Generated: {datetime.now().strftime("%Y-%m-%d %H:%M:%S")}</div>
    </div>
    
    <div class="legend">
        <div class="legend-title">Legend:</div>
        <div class="legend-item">
            <span class="legend-color" style="background-color: rgba(0, 200, 0, 0.7);"></span>
            <span>Correct Clicks</span>
        </div>
        <div class="legend-item">
            <span class="legend-color" style="background-color: rgba(255, 0, 0, 0.7);"></span>
            <span>Incorrect Clicks</span>
        </div>
        <div class="legend-item">
            <span class="legend-color" style="background-color: rgba(255, 200, 0, 0.7);"></span>
            <span>Mixed (Equal Correct/Incorrect)</span>
        </div>
        <div style="margin-top: 10px; font-size: 13px; color: #666;">
            Hover over characters to see click counts. Color intensity indicates frequency.
        </div>
    </div>
    
    <div class="story-container">
        {"".join(html_lines)}
    </div>
</body>
</html>"""
    
    return html


def generate_output_filename(story_id, grammar_point_id, grammar_point_name, custom_output):
    """Generate output filename with timestamp if not provided."""
    if custom_output:
        return custom_output
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    safe_name = grammar_point_name.replace(' ', '_').replace('/', '_')
    return f"grammar_heatmap_{story_id}_gp{grammar_point_id}_{safe_name}_{timestamp}.html"


def main():
    parser = argparse.ArgumentParser(
        description="Generate grammar click heatmap on story text"
    )
    parser.add_argument("story_id", type=int, help="Story ID to generate heatmap for")
    parser.add_argument(
        "-o",
        "--output",
        help="Output file path (default: grammar_heatmap_STORYID_GPID_NAME_TIMESTAMP.html)",
    )

    args = parser.parse_args()

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

        grammar_items = get_grammar_items_with_positions(cursor, args.story_id)
        if not grammar_items:
            print(f"No grammar items found for story {args.story_id}")
            sys.exit(1)

        correct_data, incorrect_data, incorrect_position_data = get_grammar_click_data(
            cursor, args.story_id
        )

        # Get all unique grammar points
        grammar_points = get_all_grammar_points(grammar_items)
        
        print(f"\nGenerating heatmaps for {len(grammar_points)} grammar point(s)...")
        
        # Generate one heatmap per grammar point
        for gp_id, gp_name in grammar_points.items():
            print(f"\nProcessing: {gp_name} (ID: {gp_id})")
            
            # Build heatmap data for this grammar point
            heatmap_data = build_heatmap_data(
                lines, grammar_items, correct_data, incorrect_position_data, gp_id
            )
            
            # Generate HTML
            html_content = generate_html_heatmap(lines, heatmap_data, gp_name, args.story_id)
            
            # Save to file
            output_file = generate_output_filename(args.story_id, gp_id, gp_name, args.output)
            with open(output_file, 'w', encoding='utf-8') as f:
                f.write(html_content)
            
            print(f"Saved: {output_file}")
            
            # Print summary stats
            total_correct = sum(correct_data.get((ln, gp_id), 0) for ln, _ in lines)
            total_incorrect = sum(
                1 for ln, g_id, pos_list in incorrect_position_data 
                if g_id == gp_id and pos_list
                for _ in pos_list
            )
            print(f"  Correct clicks: {total_correct}")
            print(f"  Incorrect clicks: {total_incorrect}")

        print(f"\nComplete. Generated {len(grammar_points)} heatmap file(s).")

        cursor.close()
        connection.close()

    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()