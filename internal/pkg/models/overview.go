// This contains information for the implenetations in this package.
// It must be kept up to date!
package models

/*
Core Types:
- Story: {Metadata: StoryMetadata, Content: StoryContent}
- StoryMetadata: {StoryID, WeekNumber, DayLetter, Title(map[lang]string), Author, GrammarPoint, Description, LastRevision}
- StoryLine: {LineNumber, Text, Vocabulary[], Grammar[], AudioFile*, Footnotes[]}
- VocabularyItem: {Word, LexicalForm, Position[2]int}
- GrammarItem: {Text, Position[2]int}
- Footnote: {ID, Text, References[]string}

Database Functions:
GetStoryData(id int) (*Story, error) // Full story with all components
GetAllStories(language string) ([]Story, error) // Basic story list
GetLineAnnotations(storyID, lineNumber int) (*StoryLine, error)
GetStoryAnnotations(storyID int) (map[int]*StoryLine, error)
GetLineText(storyID, lineNumber int) (string, error)

Save Operations:
SaveNewStory(*Story) error
SaveStoryData(storyID int, story *Story) error

Edit Operations:
EditStoryText(storyID int, lines []StoryLine) error
EditStoryMetadata(storyID int, metadata StoryMetadata) error
AddLineAnnotations(storyID, lineNumber int, line StoryLine) error
UpdateVocabularyAnnotation(storyID, lineNumber int, position [2]int, vocab VocabularyItem) error
UpdateVocabularyByWord(storyID, lineNumber int, word string, newLexicalForm string) error
UpdateGrammarAnnotation(storyID, lineNumber int, position [2]int, grammar GrammarItem) error
UpdateFootnoteAnnotation(storyID, footnoteID int, footnote Footnote) error
ClearStoryAnnotations(storyID int) error
ClearLineAnnotations(storyID, lineNumber int) error

Delete Operations:
Delete(storyID int) error // Removes story and all associated data

Error Types:
ErrNotFound, ErrInvalidStoryID, ErrInvalidLineNumber, ErrMissingStoryID,
ErrInvalidWeekNumber, ErrMissingDayLetter, ErrTitleTooShort, ErrMissingAuthorID

Database: Uses PostgreSQL, or a mock thereof
Transaction wrapper: withTransaction(func(*sql.Tx) error) error
*/
