package delivery

import (
	"net/http"

	"github.com/brkss/dextrace/internal/domain"
	"github.com/brkss/dextrace/internal/usecase"
	"github.com/gin-gonic/gin"
)

type GlucoseHandler struct {
	nighscoutUsecase *usecase.NightscoutUsecase
	glucoseUseCase   *usecase.SibionicUseCase
	userID           string
	user             domain.User
}

func NewGlucoseHandler(glucoseUseCase *usecase.SibionicUseCase, nighscoutUseCase *usecase.NightscoutUsecase, userID string, user domain.User) *GlucoseHandler {
	return &GlucoseHandler{
		glucoseUseCase:   glucoseUseCase,
		nighscoutUsecase: nighscoutUseCase,
		userID:           userID,
		user:             user,
	}
}

func (h *GlucoseHandler) GetGlucoseData(c *gin.Context) {
	data, err := h.glucoseUseCase.GetGlucoseData(h.user, h.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *GlucoseHandler) PushToNightscout(c *gin.Context) {
	data, err := h.glucoseUseCase.GetGlucoseData(h.user, h.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = h.nighscoutUsecase.PushData(*data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}
