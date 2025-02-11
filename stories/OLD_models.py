
# stories/models.py
from dataclasses import dataclass
from typing import List, Dict, Optional
from datetime import datetime

@dataclass
class Author:
    id: str
    name: str

@dataclass
class Description:
    language: str
    text: str

@dataclass
class StoryMetadata:
    story_id: int
    week_number: int
    day_letter: str
    title: Dict[str, str]
    author: Author
    grammar_point: str
    description: Description
    last_revision: datetime

@dataclass
class VocabularyItem:
    word: str
    lexical_form: str
    position: List[int]  # [start, end]

@dataclass
class GrammarItem:
    text: str
    position: List[int]  # [start, end]

@dataclass
class Footnote:
    id: int
    text: str
    references: List[str]

@dataclass
class StoryLine:
    line_number: int
    text: str
    vocabulary: List[VocabularyItem]
    grammar: List[GrammarItem]
    audio_file: Optional[str]
    footnotes: List[Footnote]

@dataclass
class StoryContent:
    lines: List[StoryLine]

@dataclass
class Story:
    metadata: StoryMetadata
    content: StoryContent
    description: Description

@dataclass
class Page2Data:
    story: Story
    vocab_bank: List[str]
    audio_folder: str
    audio_path: str