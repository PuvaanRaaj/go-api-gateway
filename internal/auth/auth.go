package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"

	"github.com/yourname/api-gateway/internal/store"
)

// Handler deals with authentication endpoints such as /auth/login.
type Handler struct {
	store    *store.Store
	secret   []byte
	tokenTTL time.Duration
	issuer   string
}

// NewHandler builds a login handler.
func NewHandler(st *store.Store, secret []byte, ttl time.Duration) *Handler {
	return &Handler{
		store:    st,
		secret:   secret,
		tokenTTL: ttl,
		issuer:   "api-gateway",
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoginHandler returns an http.Handler for POST /auth/login.
func (h *Handler) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}

		identity, err := h.store.AuthenticateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, store.ErrInvalidCredentials) {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}
			http.Error(w, "authentication failed", http.StatusInternalServerError)
			return
		}

		token, exp, err := h.generateToken(identity)
		if err != nil {
			http.Error(w, "could not issue token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(loginResponse{
			Token:     token,
			ExpiresAt: exp,
		})
	})
}

func (h *Handler) generateToken(identity *store.Identity) (string, time.Time, error) {
	exp := time.Now().Add(h.tokenTTL)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: h.secret}, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	cl := jwt.Claims{
		Subject:   identity.UserID.String(),
		Issuer:    h.issuer,
		Expiry:    jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}

	type customClaims struct {
		Email string `json:"email"`
		jwt.Claims
	}

	tokenBuilder := jwt.Signed(signer).Claims(customClaims{
		Email:  identity.Email,
		Claims: cl,
	})
	out, err := tokenBuilder.Serialize()
	if err != nil {
		return "", time.Time{}, err
	}

	return out, exp, nil
}

// VerifyToken parses and validates the JWT returning the associated identity.
func VerifyToken(token string, secret []byte) (*Identity, error) {
	parsed, err := jwt.ParseSigned(token, nil)
	if err != nil {
		return nil, err
	}

	type customClaims struct {
		Email string `json:"email"`
		jwt.Claims
	}

	var claims customClaims
	if err := parsed.Claims(secret, &claims); err != nil {
		return nil, err
	}

	if err := claims.Validate(jwt.Expected{
		Time: time.Now(),
	}); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, err
	}

	return &Identity{
		UserID: userID,
		Email:  claims.Email,
		Method: "jwt",
	}, nil
}

// IdentityFromStore converts a store identity to an auth identity with the given method label.
func IdentityFromStore(identity *store.Identity, method string) Identity {
	return Identity{
		UserID: identity.UserID,
		Email:  identity.Email,
		Method: method,
	}
}
