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

func (repo *PostgresPolicyRepo) CreatePolicy(policy models.Policy) (int64, error) {
    var newID int64
    
    query := `INSERT INTO policies (subject, object, action) 
              VALUES ($1, $2, $3) 
              RETURNING id`

    err := repo.DB.QueryRow(query, policy.Subject, policy.Object, policy.Action).Scan(&newID)
    if err != nil {
        return 0, err
    }

    return newID, nil
}