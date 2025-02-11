# stories/database/setup.py
from .models import Base
from .connection import DatabaseConnection

def init_database(db_path: str = "data/stories.db"):
    """Initialize the database with all tables"""
    db = DatabaseConnection(db_path)
    Base.metadata.create_all(db.engine)
    return db
