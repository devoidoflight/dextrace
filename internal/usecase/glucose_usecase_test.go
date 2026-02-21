package usecase

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/brkss/dextrace/internal/domain"
)

type mockAuthRepository struct {
	token string
	err   error
}

func (m mockAuthRepository) Login(user domain.User) (string, error) {
	return m.token, m.err
}

type mockGlucoseRepository struct {
	capturedUserID string
	response       *domain.GlucoseDataResponse
	err            error
}

func (m *mockGlucoseRepository) GetData(token string, userID string) (*domain.GlucoseDataResponse, error) {
	m.capturedUserID = userID
	return m.response, m.err
}

func TestGetGlucoseData_ResolvesUserIDFromToken(t *testing.T) {
	claims := map[string]interface{}{"userId": "abc-123"}
	payload, _ := json.Marshal(claims)
	token := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"

	glucoseRepo := &mockGlucoseRepository{response: &domain.GlucoseDataResponse{Success: true}}
	uc := NewSibionicUseCase(mockAuthRepository{token: token}, glucoseRepo)

	_, err := uc.GetGlucoseData(domain.User{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if glucoseRepo.capturedUserID != "abc-123" {
		t.Fatalf("expected resolved user id 'abc-123', got %q", glucoseRepo.capturedUserID)
	}
}

func TestExtractUserIDFromToken_MissingClaim(t *testing.T) {
	claims := map[string]interface{}{"role": "user"}
	payload, _ := json.Marshal(claims)
	token := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"

	_, err := extractUserIDFromToken(token)
	if err == nil {
		t.Fatal("expected error for missing user ID claim")
	}
}
