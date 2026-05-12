package service

import (
	"context"
	"fmt"

	"invest/backend/internal/crypto"
	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type TinkoffCredentialService struct {
	repo          repository.TinkoffCredentialRepository
	encryptionKey []byte
	instruments   InstrumentService
}

func NewTinkoffCredentialService(repo repository.TinkoffCredentialRepository, encryptionKey string, instruments InstrumentService) *TinkoffCredentialService {
	return &TinkoffCredentialService{
		repo:          repo,
		encryptionKey: []byte(encryptionKey),
		instruments:   instruments,
	}
}

func (s *TinkoffCredentialService) SaveToken(ctx context.Context, userID int64, token string, isSandbox bool) error {
	ciphertext, nonce, err := crypto.Encrypt([]byte(token), s.encryptionKey)
	if err != nil {
		return err
	}

	hint := ""
	if len(token) > 8 {
		hint = "****" + token[len(token)-4:]
	}

	cred := domain.UserTinkoffCredential{
		UserID:         userID,
		TokenEncrypted: ciphertext,
		TokenNonce:     nonce,
		TokenHint:      hint,
		IsSandbox:      isSandbox,
	}

	return s.repo.Upsert(ctx, cred)
}

func (s *TinkoffCredentialService) GetToken(ctx context.Context, userID int64) (string, bool, error) {
	cred, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", false, err
	}

	plaintext, err := crypto.Decrypt(cred.TokenEncrypted, cred.TokenNonce, s.encryptionKey)
	if err != nil {
		return "", false, fmt.Errorf("failed to decrypt token: %w", err)
	}

	return string(plaintext), cred.IsSandbox, nil
}

func (s *TinkoffCredentialService) GetHint(ctx context.Context, userID int64) (string, bool, bool, error) {
	cred, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", false, false, nil // Not found is okay, just means not connected
	}

	return cred.TokenHint, cred.IsSandbox, true, nil
}

func (s *TinkoffCredentialService) DeleteToken(ctx context.Context, userID int64) error {
	return s.repo.Delete(ctx, userID)
}

func (s *TinkoffCredentialService) TestToken(ctx context.Context, token string, isSandbox bool) error {
	instruments := InstrumentServiceForSandbox(s.instruments, isSandbox)
	if instruments == nil {
		return ErrTinkoffUnavailable
	}
	_, err := instruments.FindInstrument(ctx, token, "SBER")
	return err
}
