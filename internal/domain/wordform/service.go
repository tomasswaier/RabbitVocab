package wordform

import (
	"context"
	"errors"

	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
)

var (
	ErrLanguageIDRequired  = errors.New("languageId is required: user has multiple languages")
	ErrNoLanguages         = errors.New("user has no languages configured")
	ErrNotFoundOrForbidden = errors.New("word form not found or not owned by user")
)

type Service struct {
	wordForms Repository
	languages language.Repository
}

func NewService(wordForms Repository, languages language.Repository) *Service {
	return &Service{wordForms: wordForms, languages: languages}
}

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

func (s *Service) InsertWordForm(ctx context.Context, wordID int64, subject, form string, tense *string) (*WordForm, error) {
	return s.wordForms.Insert(ctx, wordID, subject, form, tense)
}

func (s *Service) GetRandomWordForms(ctx context.Context, userID int64, languageID *int64, count int32) ([]*WordForm, error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	return s.wordForms.GetRandom(ctx, resolvedID, count)
}

func (s *Service) DeleteWordForm(ctx context.Context, userID, wordFormID int64) error {
	deleted, err := s.wordForms.Delete(ctx, wordFormID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFoundOrForbidden
	}
	return nil
}
