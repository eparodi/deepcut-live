package application

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/deepcut/live/internal/modules/auth/domain"
	"github.com/deepcut/live/internal/shared/errs"
)

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type AuthService struct {
	repo        domain.Repository
	oauthConfig *oauth2.Config
	privateKey  *ecdsa.PrivateKey
	publicKey   *ecdsa.PublicKey
	baseURL     string
	googleCfg   oauth2.Config
	userInfoURL string
}

func NewAuthService(repo domain.Repository, googleClientID, googleClientSecret, baseURL, privateKeyPEM, publicKeyPEM string) *AuthService {
	priv, err := jwt.ParseECPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		panic(fmt.Sprintf("failed to parse private key: %v", err))
	}
	pub, err := jwt.ParseECPublicKeyFromPEM([]byte(publicKeyPEM))
	if err != nil {
		panic(fmt.Sprintf("failed to parse public key: %v", err))
	}
	return &AuthService{
		repo:        repo,
		privateKey:  priv,
		publicKey:   pub,
		baseURL:     baseURL,
		userInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		googleCfg: oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  baseURL + "/api/auth/google/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

// GenerateOAuthURL returns the Google OAuth consent page URL.
func (s *AuthService) GenerateOAuthURL(state string) string {
	return s.googleCfg.AuthCodeURL(state)
}

// ExchangeCodeForToken exchanges an OAuth authorization code for a token.
func (s *AuthService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := s.googleCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return token, nil
}

// GetGoogleUser fetches the authenticated user's profile from Google.
func (s *AuthService) GetGoogleUser(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := s.googleCfg.Client(ctx, token)
	resp, err := client.Get(s.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch google userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("read google response: %w", err)
	}

	var user GoogleUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse google userinfo: %w", err)
	}
	return &user, nil
}

// GenerateJWT creates a signed JWT for the given user ID using ES256.
func (s *AuthService) GenerateJWT(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// ValidateJWT parses and validates a JWT, returning the user ID.
func (s *AuthService) ValidateJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse jwt: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errs.Unauthorized("invalid token")
	}
	sub, err := claims.GetSubject()
	if err != nil {
		return "", errs.Unauthorized("invalid token claims")
	}
	return sub, nil
}

// GenerateStreamKey creates a random stream key and returns the raw value + hash.
func (s *AuthService) GenerateStreamKey() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random: %w", err)
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}

// CreateUser creates a new user with a generated stream key.
func (s *AuthService) CreateUser(ctx context.Context, googleID, email, name, avatarURL string) (*domain.User, error) {
	_, hash, err := s.GenerateStreamKey()
	if err != nil {
		return nil, fmt.Errorf("generate stream key: %w", err)
	}
	user, err := s.repo.CreateUser(ctx, googleID, email, name, avatarURL, hash)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// GetByGoogleID looks up a user by their Google ID.
func (s *AuthService) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	user, err := s.repo.GetByGoogleID(ctx, googleID)
	if err != nil {
		return nil, fmt.Errorf("get by google id: %w", err)
	}
	return user, nil
}

// GetByID looks up a user by their UUID.
func (s *AuthService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}
	return user, nil
}

// GetByStreamKeyHash looks up a user by their hashed stream key.
func (s *AuthService) GetByStreamKeyHash(ctx context.Context, hash string) (*domain.User, error) {
	user, err := s.repo.GetByStreamKeyHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get by stream key hash: %w", err)
	}
	return user, nil
}

// RegenerateStreamKey creates a new stream key for a user.
func (s *AuthService) RegenerateStreamKey(ctx context.Context, userID string) (string, error) {
	raw, hash, err := s.GenerateStreamKey()
	if err != nil {
		return "", fmt.Errorf("generate stream key: %w", err)
	}
	if err := s.repo.UpdateStreamKey(ctx, userID, hash); err != nil {
		return "", fmt.Errorf("update stream key: %w", err)
	}
	return raw, nil
}

// UpdateSettings updates a user's stream title and category.
func (s *AuthService) UpdateSettings(ctx context.Context, userID, title, category string) error {
	if err := s.repo.UpdateSettings(ctx, userID, title, category); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	return nil
}

// SetLiveStatus marks a user as live or offline.
func (s *AuthService) SetLiveStatus(ctx context.Context, userID string, isLive bool) error {
	if err := s.repo.SetLiveStatus(ctx, userID, isLive); err != nil {
		return fmt.Errorf("set live status: %w", err)
	}
	return nil
}

// GetLiveUsers returns all users currently streaming.
func (s *AuthService) GetLiveUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.repo.GetLiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get live users: %w", err)
	}
	return users, nil
}
