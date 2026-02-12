package api

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserExists = errors.New("user already exists")
var ErrInvalidCredentials = errors.New("invalid credentials")

type User struct {
	Username string
	PlayerID string
}

type AuthUserRecord struct {
	UsernameNorm string
	Username     string
	PasswordHash string
	PlayerID     string
}

type AuthUserRepository interface {
	Create(usernameNorm, username, passwordHash, playerID string) error
	GetByUsernameNorm(usernameNorm string) (username, passwordHash, playerID string, err error)
}

type AuthStore struct {
	mu            sync.RWMutex
	refreshTokens map[string]string
	repo          AuthUserRepository
	jwtSecret     []byte
	accessTTL     time.Duration
}

// AuthStore 使用 JWT 作为 access_token，refresh 仍为内存 token（MVP）。
func NewAuthStore(secret string, ttl time.Duration) *AuthStore {
	return NewAuthStoreWithRepository(secret, ttl, nil)
}

func NewAuthStoreWithRepository(secret string, ttl time.Duration, repo AuthUserRepository) *AuthStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if repo == nil {
		repo = newInMemoryAuthUserRepo()
	}
	return &AuthStore{
		refreshTokens: make(map[string]string),
		repo:          repo,
		jwtSecret:     []byte(secret),
		accessTTL:     ttl,
	}
}

func (s *AuthStore) Register(username, password string) (User, error) {
	username = strings.TrimSpace(username)
	usernameNorm := normalizeUsername(username)
	if usernameNorm == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	record := AuthUserRecord{
		UsernameNorm: usernameNorm,
		Username:     username,
		PasswordHash: passwordHash,
		PlayerID:     uuid.New().String(),
	}
	if err := s.repo.Create(record.UsernameNorm, record.Username, record.PasswordHash, record.PlayerID); err != nil {
		if isDuplicateUserErr(err) {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return User{
		Username: record.Username,
		PlayerID: record.PlayerID,
	}, nil
}

func (s *AuthStore) Authenticate(username, password string) (User, error) {
	usernameNorm := normalizeUsername(username)
	storedUsername, storedPasswordHash, playerID, err := s.repo.GetByUsernameNorm(usernameNorm)
	if err != nil {
		return User{}, ErrInvalidCredentials
	}
	if err := verifyPassword(storedPasswordHash, password); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return User{
		Username: storedUsername,
		PlayerID: playerID,
	}, nil
}

func (s *AuthStore) IssueTokens(username string) (string, string, error) {
	storedUsername, _, playerID, err := s.repo.GetByUsernameNorm(normalizeUsername(username))
	if err != nil {
		return "", "", ErrInvalidCredentials
	}
	access, err := s.issueJWT(User{Username: storedUsername, PlayerID: playerID})
	if err != nil {
		return "", "", err
	}
	refresh := uuid.New().String()

	s.mu.Lock()
	s.refreshTokens[refresh] = normalizeUsername(storedUsername)
	s.mu.Unlock()

	return access, refresh, nil
}

func (s *AuthStore) Refresh(refresh string) (string, bool) {
	s.mu.RLock()
	usernameNorm, ok := s.refreshTokens[refresh]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	storedUsername, _, playerID, err := s.repo.GetByUsernameNorm(usernameNorm)
	if err != nil {
		return "", false
	}
	access, err := s.issueJWT(User{Username: storedUsername, PlayerID: playerID})
	if err != nil {
		return "", false
	}
	return access, true
}

func (s *AuthStore) Revoke(access string) {
	// No-op for JWT in MVP. Refresh token revocation is handled by caller.
}

func (s *AuthStore) UserByAccessToken(access string) (User, bool) {
	claims, ok := parseJWT(access, s.jwtSecret)
	if !ok {
		return User{}, false
	}
	storedUsername, _, playerID, err := s.repo.GetByUsernameNorm(normalizeUsername(claims.Username))
	if err != nil {
		return User{}, false
	}
	return User{
		Username: storedUsername,
		PlayerID: playerID,
	}, true
}

type AccessClaims struct {
	Username string `json:"usr"`
	jwt.StandardClaims
}

// issueJWT 负责签发 HS256 JWT。
func (s *AuthStore) issueJWT(user User) (string, error) {
	if len(s.jwtSecret) == 0 {
		return "", ErrInvalidCredentials
	}
	claims := AccessClaims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			Subject:   user.PlayerID,
			ExpiresAt: time.Now().Add(s.accessTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// parseJWT 校验 JWT 并返回 claims。
func parseJWT(token string, secret []byte) (AccessClaims, bool) {
	if len(secret) == 0 {
		return AccessClaims{}, false
	}
	parsed, err := jwt.ParseWithClaims(token, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return AccessClaims{}, false
	}
	claims, ok := parsed.Claims.(*AccessClaims)
	if !ok {
		return AccessClaims{}, false
	}
	return *claims, true
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func hashPassword(password string) (string, error) {
	raw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func verifyPassword(passwordHash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
}

func isDuplicateUserErr(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}

type inMemoryAuthUserRepo struct {
	mu   sync.RWMutex
	byID map[string]AuthUserRecord
}

func newInMemoryAuthUserRepo() *inMemoryAuthUserRepo {
	return &inMemoryAuthUserRepo{
		byID: make(map[string]AuthUserRecord),
	}
}

func (r *inMemoryAuthUserRepo) Create(usernameNorm, username, passwordHash, playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[usernameNorm]; ok {
		return ErrUserExists
	}
	r.byID[usernameNorm] = AuthUserRecord{
		UsernameNorm: usernameNorm,
		Username:     username,
		PasswordHash: passwordHash,
		PlayerID:     playerID,
	}
	return nil
}

func (r *inMemoryAuthUserRepo) GetByUsernameNorm(usernameNorm string) (string, string, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.byID[usernameNorm]
	if !ok {
		return "", "", "", gorm.ErrRecordNotFound
	}
	return record.Username, record.PasswordHash, record.PlayerID, nil
}
