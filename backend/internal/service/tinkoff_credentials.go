package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

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

	hint := tokenHint(token)

	cred := domain.UserTinkoffCredential{
		UserID:         userID,
		TokenEncrypted: ciphertext,
		TokenNonce:     nonce,
		TokenHint:      hint,
		IsSandbox:      isSandbox,
	}

	if err := s.repo.Upsert(ctx, cred); err != nil {
		return err
	}

	active, err := s.repo.GetActiveBrokerConnection(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err = s.repo.CreateBrokerConnection(ctx, domain.BrokerConnection{
				UserID:         userID,
				Name:           "Основное подключение",
				TokenEncrypted: ciphertext,
				TokenNonce:     nonce,
				TokenHint:      hint,
				IsSandbox:      isSandbox,
				IsActive:       true,
			})
			return err
		}
		return err
	}
	active.TokenEncrypted = ciphertext
	active.TokenNonce = nonce
	active.TokenHint = hint
	active.IsSandbox = isSandbox
	active.LastSyncAt = nil
	_, err = s.repo.UpdateBrokerConnection(ctx, active)
	return err
}

func (s *TinkoffCredentialService) GetToken(ctx context.Context, userID int64) (string, bool, error) {
	if conn, err := s.repo.GetActiveBrokerConnection(ctx, userID); err == nil {
		token, err := s.decryptBrokerConnection(conn)
		return token, conn.IsSandbox, err
	}

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
	if conn, err := s.repo.GetActiveBrokerConnection(ctx, userID); err == nil {
		return conn.TokenHint, conn.IsSandbox, true, nil
	}

	cred, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", false, false, nil // Not found is okay, just means not connected
	}

	return cred.TokenHint, cred.IsSandbox, true, nil
}

func (s *TinkoffCredentialService) DeleteToken(ctx context.Context, userID int64) error {
	if conn, err := s.repo.GetActiveBrokerConnection(ctx, userID); err == nil {
		if err := s.repo.DeleteBrokerConnection(ctx, userID, conn.ID); err != nil {
			return err
		}
	}
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

func (s *TinkoffCredentialService) CreateBrokerConnection(ctx context.Context, userID int64, name string, token string, isSandbox bool) (domain.BrokerConnection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "T-Invest"
	}
	ciphertext, nonce, err := crypto.Encrypt([]byte(token), s.encryptionKey)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	connections, err := s.repo.ListBrokerConnections(ctx, userID)
	if err != nil {
		return domain.BrokerConnection{}, err
	}
	return s.repo.CreateBrokerConnection(ctx, domain.BrokerConnection{
		UserID:         userID,
		Name:           name,
		TokenEncrypted: ciphertext,
		TokenNonce:     nonce,
		TokenHint:      tokenHint(token),
		IsSandbox:      isSandbox,
		IsActive:       len(connections) == 0,
	})
}

func (s *TinkoffCredentialService) ListBrokerConnections(ctx context.Context, userID int64) ([]domain.BrokerConnection, error) {
	return s.repo.ListBrokerConnections(ctx, userID)
}

func (s *TinkoffCredentialService) SetActiveBrokerConnection(ctx context.Context, userID int64, connectionID int64) error {
	return s.repo.SetActiveBrokerConnection(ctx, userID, connectionID)
}

func (s *TinkoffCredentialService) DeleteBrokerConnection(ctx context.Context, userID int64, connectionID int64) error {
	return s.repo.DeleteBrokerConnection(ctx, userID, connectionID)
}

func (s *TinkoffCredentialService) ListBrokerAccounts(ctx context.Context, userID int64, connectionID int64) ([]domain.BrokerAccount, domain.BrokerConnection, error) {
	conn, token, client, err := s.connectionClient(ctx, userID, connectionID)
	if err != nil {
		return nil, domain.BrokerConnection{}, err
	}
	accounts, err := client.GetAccounts(ctx, token)
	if err != nil {
		return nil, domain.BrokerConnection{}, err
	}
	conn.LastSyncAt = ptrTime(time.Now().UTC())
	_, _ = s.repo.UpdateBrokerConnection(ctx, conn)
	return accounts, conn, nil
}

func (s *TinkoffCredentialService) SaveBrokerAccountSelection(ctx context.Context, userID int64, connectionID int64, accountID string) error {
	conn, _, _, err := s.connectionClient(ctx, userID, connectionID)
	if err != nil {
		return err
	}
	return s.repo.SaveBrokerAccountSelection(ctx, domain.BrokerAccountSelection{
		UserID:       userID,
		ConnectionID: conn.ID,
		AccountID:    accountID,
	})
}

func (s *TinkoffCredentialService) GetBrokerAccountSelection(ctx context.Context, userID int64) (domain.BrokerAccountSelection, error) {
	return s.repo.GetBrokerAccountSelection(ctx, userID)
}

func (s *TinkoffCredentialService) GetBrokerContext(ctx context.Context, userID int64, connectionID int64, accountID string) (domain.BrokerConnection, string, error) {
	conn, token, client, err := s.connectionClient(ctx, userID, connectionID)
	if err != nil {
		return domain.BrokerConnection{}, "", err
	}
	if accountID != "" {
		return conn, accountID, nil
	}
	if selected, err := s.repo.GetBrokerAccountSelection(ctx, userID); err == nil && selected.ConnectionID == conn.ID && selected.AccountID != "" {
		return conn, selected.AccountID, nil
	}
	accounts, err := client.GetAccounts(ctx, token)
	if err != nil {
		return domain.BrokerConnection{}, "", err
	}
	for _, account := range accounts {
		if account.Status == "ACCOUNT_STATUS_OPEN" {
			return conn, account.ID, nil
		}
	}
	if len(accounts) == 0 {
		return domain.BrokerConnection{}, "", sql.ErrNoRows
	}
	return conn, accounts[0].ID, nil
}

func (s *TinkoffCredentialService) GetPortfolio(ctx context.Context, userID int64, connectionID int64, accountID string) (domain.BrokerConnection, domain.BrokerPortfolio, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, accountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerPortfolio{}, err
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerPortfolio{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerPortfolio{}, err
	}
	portfolio, err := client.GetPortfolio(ctx, token, resolvedAccountID)
	return conn, portfolio, err
}

func (s *TinkoffCredentialService) GetPositions(ctx context.Context, userID int64, connectionID int64, accountID string) (domain.BrokerConnection, domain.BrokerPositions, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, accountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerPositions{}, err
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerPositions{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerPositions{}, err
	}
	positions, err := client.GetPositions(ctx, token, resolvedAccountID)
	return conn, positions, err
}

func (s *TinkoffCredentialService) GetOperations(ctx context.Context, userID int64, connectionID int64, request domain.BrokerOperationsRequest) (domain.BrokerConnection, domain.BrokerOperationsPage, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, request.AccountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOperationsPage{}, err
	}
	request.AccountID = resolvedAccountID
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOperationsPage{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOperationsPage{}, err
	}
	page, err := client.GetOperationsByCursor(ctx, token, request)
	return conn, page, err
}

func (s *TinkoffCredentialService) GetOrders(ctx context.Context, userID int64, connectionID int64, accountID string) (domain.BrokerConnection, []domain.BrokerOrder, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, accountID)
	if err != nil {
		return domain.BrokerConnection{}, nil, err
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, nil, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, nil, err
	}
	orders, err := client.GetOrders(ctx, token, resolvedAccountID)
	return conn, orders, err
}

func (s *TinkoffCredentialService) PostOrder(ctx context.Context, userID int64, connectionID int64, request domain.BrokerPlaceOrderRequest) (domain.BrokerConnection, domain.BrokerOrder, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, request.AccountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOrder{}, err
	}
	request.AccountID = resolvedAccountID
	if request.OrderID == "" {
		request.OrderID = generateOrderRequestID()
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOrder{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOrder{}, err
	}
	order, err := client.PostOrder(ctx, token, request)
	return conn, order, err
}

func (s *TinkoffCredentialService) CancelOrder(ctx context.Context, userID int64, connectionID int64, accountID string, orderID string) (domain.BrokerConnection, domain.BrokerCancelOrderResult, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, accountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerCancelOrderResult{}, err
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerCancelOrderResult{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerCancelOrderResult{}, err
	}
	result, err := client.CancelOrder(ctx, token, resolvedAccountID, orderID)
	return conn, result, err
}

func (s *TinkoffCredentialService) GetOrderState(ctx context.Context, userID int64, connectionID int64, accountID string, orderID string) (domain.BrokerConnection, domain.BrokerOrder, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, accountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOrder{}, err
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOrder{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerOrder{}, err
	}
	order, err := client.GetOrderState(ctx, token, resolvedAccountID, orderID)
	return conn, order, err
}

func (s *TinkoffCredentialService) OpenSandboxAccount(ctx context.Context, userID int64, connectionID int64) (domain.BrokerConnection, string, error) {
	conn, token, client, err := s.connectionClient(ctx, userID, connectionID)
	if err != nil {
		return domain.BrokerConnection{}, "", err
	}
	if !conn.IsSandbox {
		return domain.BrokerConnection{}, "", fmt.Errorf("active connection is not sandbox")
	}
	accountID, err := client.OpenSandboxAccount(ctx, token)
	if err != nil {
		return domain.BrokerConnection{}, "", err
	}
	_ = s.repo.SaveBrokerAccountSelection(ctx, domain.BrokerAccountSelection{
		UserID:       userID,
		ConnectionID: conn.ID,
		AccountID:    accountID,
	})
	conn.LastSyncAt = ptrTime(time.Now().UTC())
	updated, updateErr := s.repo.UpdateBrokerConnection(ctx, conn)
	if updateErr == nil {
		conn = updated
	}
	return conn, accountID, nil
}

func (s *TinkoffCredentialService) SandboxPayIn(ctx context.Context, userID int64, connectionID int64, accountID string, amount domain.MoneyValue) (domain.BrokerConnection, domain.BrokerSandboxPayInResult, error) {
	conn, resolvedAccountID, err := s.GetBrokerContext(ctx, userID, connectionID, accountID)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerSandboxPayInResult{}, err
	}
	if !conn.IsSandbox {
		return domain.BrokerConnection{}, domain.BrokerSandboxPayInResult{}, fmt.Errorf("active connection is not sandbox")
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerSandboxPayInResult{}, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, domain.BrokerSandboxPayInResult{}, err
	}
	result, err := client.SandboxPayIn(ctx, token, resolvedAccountID, amount)
	return conn, result, err
}

func (s *TinkoffCredentialService) connectionClient(ctx context.Context, userID int64, connectionID int64) (domain.BrokerConnection, string, BrokerClient, error) {
	var conn domain.BrokerConnection
	var err error
	if connectionID > 0 {
		conn, err = s.repo.GetBrokerConnection(ctx, userID, connectionID)
	} else {
		conn, err = s.repo.GetActiveBrokerConnection(ctx, userID)
	}
	if err != nil {
		return domain.BrokerConnection{}, "", nil, err
	}
	token, err := s.decryptBrokerConnection(conn)
	if err != nil {
		return domain.BrokerConnection{}, "", nil, err
	}
	client, err := brokerClientForConnection(s.instruments, conn.IsSandbox)
	if err != nil {
		return domain.BrokerConnection{}, "", nil, err
	}
	return conn, token, client, nil
}

func (s *TinkoffCredentialService) decryptBrokerConnection(conn domain.BrokerConnection) (string, error) {
	plaintext, err := crypto.Decrypt(conn.TokenEncrypted, conn.TokenNonce, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}
	return string(plaintext), nil
}

func brokerClientForConnection(instruments InstrumentService, isSandbox bool) (BrokerClient, error) {
	targeted := InstrumentServiceForSandbox(instruments, isSandbox)
	client, ok := targeted.(BrokerClient)
	if !ok || client == nil {
		return nil, ErrTinkoffUnavailable
	}
	return client, nil
}

func tokenHint(token string) string {
	token = strings.TrimSpace(token)
	if len(token) > 8 {
		return "****" + token[len(token)-4:]
	}
	if token == "" {
		return ""
	}
	return "****"
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func generateOrderRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("order-%d", time.Now().UnixNano())
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}
