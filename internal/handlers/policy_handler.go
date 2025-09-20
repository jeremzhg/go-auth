package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/jeremzhg/go-auth/internal/models"
)

type PolicyHandler struct {
		Enforcer *casbin.Enforcer
}

type response struct{
	Allowed bool `json:"allowed"`
}

func (h *PolicyHandler) CreatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	policy := models.Policy{}
  if err := json.NewDecoder(r.Body).Decode(&policy); err != nil{
		http.Error(w, "failed to decode policy", http.StatusBadRequest)
		return
	}

	_, err := h.Enforcer.AddPolicy(policy.Subject, policy.Object, policy.Action)
	if err != nil{
		log.Printf("ERROR: failed to insert policy: %v", err)
		http.Error(w, "failed to insert policy into db", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(policy)
}

func (h *PolicyHandler) Check(w http.ResponseWriter, r *http.Request) {
	request := models.Request{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil{
		http.Error(w, "failed to read request", http.StatusBadRequest)
		return
	}
	allowed, err := h.Enforcer.Enforce(request.Subject, request.Object, request.Action)
	if err != nil{
		http.Error(w, "failed to check policy", http.StatusInternalServerError)
		return
	}

	resp := response{Allowed: allowed}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}