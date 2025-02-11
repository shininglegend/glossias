# stories/database/models.py
from datetime import datetime
from sqlalchemy import (
    Column, Integer, String, ForeignKey, DateTime,
    create_engine, Text, Table, MetaData
)
from sqlalchemy.orm import declarative_base, relationship
from sqlalchemy.schema import ForeignKeyConstraint

Base = declarative_base()

class Story(Base):
    __tablename__ = 'stories'

    story_id = Column(Integer, primary_key=True, autoincrement=True)
    week_number = Column(Integer, nullable=False)
    day_letter = Column(String, nullable=False)
    grammar_point = Column(String)
    last_revision = Column(DateTime, default=datetime.utcnow)
    author_id = Column(String, nullable=False)
    author_name = Column(String, nullable=False)

    # Relationships
    titles = relationship("StoryTitle", back_populates="story", cascade="all, delete-orphan")
    descriptions = relationship("StoryDescription", back_populates="story", cascade="all, delete-orphan")
    lines = relationship("StoryLine", back_populates="story", cascade="all, delete-orphan")

class StoryTitle(Base):
    __tablename__ = 'story_titles'

    story_id = Column(Integer, ForeignKey('stories.story_id', ondelete='CASCADE'), primary_key=True)
    language_code = Column(String, primary_key=True)
    title = Column(String, nullable=False)

    story = relationship("Story", back_populates="titles")

class StoryDescription(Base):
    __tablename__ = 'story_descriptions'

    story_id = Column(Integer, ForeignKey('stories.story_id', ondelete='CASCADE'), primary_key=True)
    language_code = Column(String, primary_key=True)
    description_text = Column(Text, nullable=False)

    story = relationship("Story", back_populates="descriptions")

class StoryLine(Base):
    __tablename__ = 'story_lines'

    story_id = Column(Integer, ForeignKey('stories.story_id', ondelete='CASCADE'), primary_key=True)
    line_number = Column(Integer, primary_key=True)
    text = Column(Text, nullable=False)
    audio_file = Column(String)

    story = relationship("Story", back_populates="lines")
    vocabulary = relationship("VocabularyItem", back_populates="line", cascade="all, delete-orphan")
    grammar = relationship("GrammarItem", back_populates="line", cascade="all, delete-orphan")
    footnotes = relationship("Footnote", back_populates="line", cascade="all, delete-orphan")

class VocabularyItem(Base):
    __tablename__ = 'vocabulary_items'

    id = Column(Integer, primary_key=True, autoincrement=True)
    story_id = Column(Integer)
    line_number = Column(Integer)
    word = Column(String, nullable=False)
    lexical_form = Column(String, nullable=False)
    position_start = Column(Integer, nullable=False)
    position_end = Column(Integer, nullable=False)

    __table_args__ = (
        ForeignKeyConstraint(
            ['story_id', 'line_number'],
            ['story_lines.story_id', 'story_lines.line_number'],
            ondelete='CASCADE'
        ),
    )

    line = relationship("StoryLine", back_populates="vocabulary")

class GrammarItem(Base):
    __tablename__ = 'grammar_items'

    id = Column(Integer, primary_key=True, autoincrement=True)
    story_id = Column(Integer)
    line_number = Column(Integer)
    text = Column(Text, nullable=False)
    position_start = Column(Integer, nullable=False)
    position_end = Column(Integer, nullable=False)

    __table_args__ = (
        ForeignKeyConstraint(
            ['story_id', 'line_number'],
            ['story_lines.story_id', 'story_lines.line_number'],
            ondelete='CASCADE'
        ),
    )

    line = relationship("StoryLine", back_populates="grammar")

class Footnote(Base):
    __tablename__ = 'footnotes'

    id = Column(Integer, primary_key=True, autoincrement=True)
    story_id = Column(Integer)
    line_number = Column(Integer)
    footnote_text = Column(Text, nullable=False)

    __table_args__ = (
        ForeignKeyConstraint(
            ['story_id', 'line_number'],
            ['story_lines.story_id', 'story_lines.line_number'],
            ondelete='CASCADE'
        ),
    )

    line = relationship("StoryLine", back_populates="footnotes")
    references = relationship("FootnoteReference", back_populates="footnote", cascade="all, delete-orphan")

class FootnoteReference(Base):
    __tablename__ = 'footnote_references'

    footnote_id = Column(Integer, ForeignKey('footnotes.id', ondelete='CASCADE'), primary_key=True)
    reference = Column(String, primary_key=True)

    footnote = relationship("Footnote", back_populates="references")
