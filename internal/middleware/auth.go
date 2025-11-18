package middleware

import (
    "net/http"
    "strings"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")

            if token == "" || !strings.HasPrefix(token, "Bearer ") {
                http.Error(w, "missing or invalid token", http.StatusUnauthorized)
                return
            }

            reqAPIKey := strings.TrimPrefix(token, "Bearer ")
            if reqAPIKey != apiKey {
                http.Error(w, "invalid API key", http.StatusUnauthorized)
                return 
            }

            next.ServeHTTP(w, r)
        })
    }
}