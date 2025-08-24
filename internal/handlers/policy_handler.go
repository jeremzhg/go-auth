package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jeremzhg/go-auth/internal/models"
	"github.com/jeremzhg/go-auth/internal/repository"
)

type PolicyHandler struct {
    Repo repository.PolicyRepository
}

func (h *PolicyHandler) CreatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	policy := models.Policy{}
  if err := json.NewDecoder(r.Body).Decode(&policy); err != nil{
		http.Error(w, "failed to decode policy", http.StatusBadRequest)
		return
	}

	id, err := h.Repo.CreatePolicy(policy)
	if err != nil{
		http.Error(w, "failed to insert policy into db", http.StatusInternalServerError)
		return
	}

	policy.Id = int(id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(policy)
}