package usecase

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/brkss/dextrace/internal/domain"
	"github.com/brkss/dextrace/internal/utils"
)

type SibionicUseCase struct {
	authRepo    domain.AuthRepository
	glucoseRepo domain.GlucoseRepository
}

func NewSibionicUseCase(authRepo domain.AuthRepository, glucoseRepo domain.GlucoseRepository) *SibionicUseCase {
	return &SibionicUseCase{
		authRepo:    authRepo,
		glucoseRepo: glucoseRepo,
	}
}

func (uc *SibionicUseCase) GetGlucoseData(user domain.User, userID string) (*[]domain.GetDataResponse, error) {
	token, err := uc.authRepo.Login(user)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	resolvedUserID := userID
	if resolvedUserID == "" {
		resolvedUserID, err = extractUserIDFromToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user ID from token: %w", err)
		}
	}

	data, err := uc.glucoseRepo.GetData(token, resolvedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get glucose data: %w", err)
	}

	var response []domain.GetDataResponse
	for _, glucose := range data.Data.GlucoseInfos {
		response = append(response, domain.GetDataResponse{
			Timestamp: glucose.T,
			Value:     utils.ConvertToMgdl(glucose.V),
		})
	}

	return &response, nil
}

func extractUserIDFromToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token payload: %w", err)
	}

	claims := map[string]interface{}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("invalid token claims: %w", err)
	}

	for _, key := range []string{"userId", "user_id", "uid", "id", "sub"} {
		if rawValue, ok := claims[key]; ok {
			switch value := rawValue.(type) {
			case string:
				if value != "" {
					return value, nil
				}
			case float64:
				return strconv.FormatInt(int64(value), 10), nil
			}
		}
	}

	return "", fmt.Errorf("token does not contain a supported user ID claim")
}
