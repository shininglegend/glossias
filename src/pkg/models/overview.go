// This contains information for the implementations in this package.
// It must be kept up to date!
package models

/*
Core Types:
- Story: {Metadata: StoryMetadata, Content: StoryContent}
- StoryMetadata: {StoryID, WeekNumber, DayLetter, Title(map[lang]string), Author, VideoURL, Description, LastRevision}
- StoryLine: {LineNumber, Text, Translations[lang]string, Vocabulary[], Grammar[], AudioFiles[], Footnotes[]}
- LineTranslation: {StoryID, LineNumber, LanguageCode, TranslationText}
- VocabularyItem: {Word, LexicalForm, Position[2]int}
- GrammarItem: {GrammarPointID*, Text, Position[2]int}
- Footnote: {ID, Text, References[]string}
- AudioFile: {ID, FilePath, FileBucket, Label}
- GrammarPoint: {ID, Name, Description}

Database Functions (SQLC-based):
GetStoryData(id int, userID string) (*Story, error) // Full story with all components
GetAllStories(language string, userID string) ([]Story, error) // Basic story list
GetLineAnnotations(storyID, lineNumber int) (*StoryLine, error)
GetStoryAnnotations(storyID int) (map[int]*StoryLine, error)
GetLineText(storyID, lineNumber int) (string, error)

Translation Operations:
GetLineTranslation(storyID, lineNumber int, languageCode string) (string, error)
UpsertLineTranslation(storyID, lineNumber int, languageCode, translationText string) error
GetAllTranslationsForStory(storyID int) ([]LineTranslation, error)
GetTranslationsByLanguage(storyID int, languageCode string) ([]LineTranslation, error)
DeleteLineTranslation(storyID, lineNumber int, languageCode string) error
DeleteStoryTranslations(storyID int) error

Grammar Point Operations:
CreateGrammarPoint(storyID int, name, description string) (*GrammarPoint, error)
GetGrammarPoint(grammarPointID int) (*GrammarPoint, error)
GetGrammarPointByName(name string, storyID int) (*GrammarPoint, error)
ListGrammarPoints() ([]GrammarPoint, error)
GetStoryGrammarPoints(storyID int) ([]GrammarPoint, error)

Audio File Operations:
CreateAudioFile(storyID, lineNumber int, filePath, fileBucket, label string) (*AudioFile, error)
GetLineAudioFiles(storyID, lineNumber int) ([]AudioFile, error)
GetStoryAudioFilesByLabel(storyID int, label string) ([]AudioFile, error)
GetAllStoryAudioFiles(storyID int) ([]AudioFile, error)
DeleteLineAudioFiles(storyID, lineNumber int) error

Save Operations (SQLC-based):
SaveNewStory(*Story) error // Uses CreateStory, UpsertStoryTitle, UpsertStoryDescription, UpsertStoryLine
SaveStoryData(storyID int, story *Story) error // Uses UpdateStory and component upserts

Edit Operations (SQLC-based):
EditStoryText(storyID int, lines []StoryLine) error // Uses DeleteAllStoryLines, UpsertStoryLine
EditStoryMetadata(storyID int, metadata StoryMetadata) error // Uses UpdateStory, DeleteStoryTitles/Descriptions, Upserts
AddLineAnnotations(storyID, lineNumber int, line StoryLine) error // Uses dedup insert functions
UpdateVocabularyAnnotation(storyID, lineNumber int, position [2]int, vocab VocabularyItem) error // Uses UpdateVocabularyByPosition
UpdateVocabularyByWord(storyID, lineNumber int, word string, newLexicalForm string) error // Uses UpdateVocabularyByWord
UpdateGrammarAnnotation(storyID, lineNumber int, position [2]int, grammar GrammarItem) error // Uses UpdateGrammarByPosition
UpdateFootnoteAnnotation(storyID, footnoteID int, footnote Footnote) error // Uses UpdateFootnote, DeleteFootnoteReferences, CreateFootnoteReference
ClearStoryAnnotations(storyID int) error // Uses StoryExists and raw SQL for complex deletes
ClearLineAnnotations(storyID, lineNumber int) error // Uses raw SQL for complex deletes


Delete Operations (SQLC-based):
Delete(storyID int) error // Uses StoryExists, DeleteStory, and component delete functions

Score Operations:
SaveVocabScore(ctx context.Context, userID string, storyID int, lineNumber int, correct bool, incorrectAnswer string) error
SaveGrammarScore(ctx context.Context, userID string, storyID int, lineNumber int, correct bool, selectedLine int, selectedPositions []int) error
SaveGrammarScoresForPoint(ctx context.Context, userID string, storyID int, grammarPointID int, lineScores map[int]bool, incorrectAnswers map[int]struct{SelectedLine int; SelectedPositions []int}) error // Multi-line grammar point scoring
GetUserVocabScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) // Returns map[lineNumber]correct
GetUserGrammarScores(ctx context.Context, userID string, storyID int) (map[int]bool, error) // Returns map[lineNumber]correct

User Operations (SQLC-based):
UpsertUser(userID, email, name string) (*User, error) // Uses UpsertUser
GetUser(userID string) (*User, error) // Uses GetUser
CanUserAccessCourse(userID string, courseID int32) bool // Uses CanUserAccessCourse
IsUserAdmin(userID string) bool // Uses GetUser and IsUserAdminOfAnyCourse
IsUserCourseAdmin(userID string, courseID int32) bool // Uses IsUserCourseAdmin
GetUserCourseAdminRights(userID string) ([]CourseAdminRight, error) // Uses GetUserCourseAdminRights

Course User Operations (SQLC-based):
AddUserToCourseByEmail(email string, courseID int) error // Uses GetUserByEmail, AddUserToCourse
RemoveUserFromCourse(courseID int, userID string) error // Uses RemoveUserFromCourse
DeleteAllUsersFromCourse(courseID int) error // Uses DeleteAllUsersFromCourse
GetCoursesForUser(userID string) ([]UserCourse, error) // Uses GetCoursesForUser
GetUsersForCourse(courseID int) ([]CourseUser, error) // Uses GetUsersForCourse

Error Types:
ErrNotFound, ErrInvalidStoryID, ErrInvalidLineNumber, ErrMissingStoryID,
ErrInvalidWeekNumber, ErrMissingDayLetter, ErrTitleTooShort, ErrMissingAuthorID

Database: Uses SQLC-generated queries with PostgreSQL
- Global variable: queries *db.Queries (initialized via SetDB)
- All functions accept context.Context parameter for proper request lifecycle management
- Type conversions: int ↔ int32, pgtype.* for nullable fields
- Transaction wrapper: withTransaction(func(*sql.Tx) error) error (works with SQLC)
- Complete elimination of raw SQL - all database operations use type-safe SQLC queries
- Import: "glossias/src/pkg/generated/db"
*/
