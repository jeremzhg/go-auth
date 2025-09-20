package repository

import (
    "github.com/jmoiron/sqlx"
	"github.com/jeremzhg/go-auth/internal/models"
)

type PolicyRepository interface {
    CreatePolicy(policy models.Policy) (int64, error)
}

type PostgresPolicyRepo struct {
	DB *sqlx.DB
}

func (r *PostgresPolicyRepo) CreatePolicy(policy models.Policy) (error) {
    query := `INSERT INTO policies (ptype, v0, v1, v2) 
              VALUES ('p', $1, $2, $3)`

    _, err := r.DB.Exec(query, policy.Subject, policy.Object, policy.Action)

    return err
}