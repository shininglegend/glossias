# stories/database/connection.py
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from contextlib import contextmanager
from stories.database.models import Story, StoryLine

class DatabaseConnection:
    def __init__(self, db_path: str = "data/stories.db"):
        self.engine = create_engine(f"sqlite:///{db_path}", echo=False)
        self.SessionLocal = sessionmaker(bind=self.engine)

    @contextmanager
    def get_session(self):
        session = self.SessionLocal()
        try:
            yield session
            session.commit()
        except:
            session.rollback()
            raise
        finally:
            session.close()

    def get_story_data(self, story_id: int) -> Story:
        """Fetches complete story data including all relationships"""
        with self.get_session() as session:
            story = session.query(Story)\
                .join(Story.titles)\
                .join(Story.lines)\
                .outerjoin(StoryLine.vocabulary)\
                .filter(Story.story_id == story_id)\
                .first()

            if story:
                # Ensure relationships are loaded
                session.refresh(story)

            return story
