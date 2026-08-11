package word

import "time"

type State string

const (
	StateNew       State = "new"
	StateLearning  State = "learning"
	StateConfident State = "confident"
	StateMastered  State = "mastered"
)

type Word struct {
	ID           int64
	LanguageID   int64
	NativeWord   string
	LearningWord string
	Article      *string
	State        State
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
