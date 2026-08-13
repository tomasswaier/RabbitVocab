package wordform

import "time"

type WordForm struct {
	ID           int64
	WordID       int64
	Subject      string
	Form         string
	Tense        *string
	NativeWord   string
	LearningWord string
	CreatedAt    time.Time
}
