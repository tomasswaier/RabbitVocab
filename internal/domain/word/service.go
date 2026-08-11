package word

import (
	"context"
	"errors"

	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
)

var (
	ErrLanguageIDRequired = errors.New("languageId is required: user has multiple languages")
	ErrNoLanguages        = errors.New("user has no languages configured")
)

type Service struct {
	words     Repository
	languages language.Repository
}

func NewService(words Repository, languages language.Repository) *Service {
	return &Service{words: words, languages: languages}
}

// resolveLanguageID returns languageID if provided, otherwise resolves it
// from the user's languages, requiring exactly one to exist.
func (s *Service) resolveLanguageID(ctx context.Context, userID int64, languageID *int64) (int64, error) {
	if languageID != nil {
		return *languageID, nil
	}

	langs, err := s.languages.ListByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	switch len(langs) {
	case 0:
		return 0, ErrNoLanguages
	case 1:
		return langs[0].ID, nil
	default:
		return 0, ErrLanguageIDRequired
	}
}

func (s *Service) InsertWord(ctx context.Context, userID int64, languageID *int64, nativeWord, learningWord string, article *string) (*Word, error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	return s.words.Insert(ctx, resolvedID, nativeWord, learningWord, article)
}

func (s *Service) GetRandomWords(ctx context.Context, userID int64, languageID *int64, count int32) ([]*Word, error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	return s.words.GetRandom(ctx, resolvedID, count)
}

func (s *Service) UpdateWordState(ctx context.Context, id int64, state State) (*Word, error) {
	return s.words.UpdateState(ctx, id, state)
}
