package handler

import (
	"net/http"

	"github.com/stellar-sponsorship-service/internal/stellar"
)

type InfoHandler struct {
	networkPassphrase string
}

func NewInfoHandler(networkPassphrase string) *InfoHandler {
	return &InfoHandler{networkPassphrase: networkPassphrase}
}

type InfoResponse struct {
	NetworkPassphrase   string   `json:"network_passphrase"`
	BaseReserve         string   `json:"base_reserve"`
	SupportedOperations []string `json:"supported_operations"`
}

func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, InfoResponse{
		NetworkPassphrase:   h.networkPassphrase,
		BaseReserve:         "0.5000000",
		SupportedOperations: stellar.SupportedOperations(),
	})
}
