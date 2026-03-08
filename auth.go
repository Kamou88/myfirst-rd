package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	authTokenTTL         = 7 * 24 * time.Hour
	initialAdminUsername = "hemng"
	initialAdminPassword = "He193452323"
)

type authContextKey string

const currentUserKey authContextKey = "current_user"

type authUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	IsActive bool   `json:"isActive"`
}

type authLoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authLoginResponse struct {
	Token   string   `json:"token"`
	Expires int64    `json:"expires"`
	User    authUser `json:"user"`
}

type userPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsActive *bool  `json:"isActive"`
}

func (a *app) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := validateSession(a.db, token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), currentUserKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUserFromRequest(r *http.Request) (authUser, bool) {
	user, ok := r.Context().Value(currentUserKey).(authUser)
	return user, ok
}

func (a *app) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload authLoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	user, err := authenticateUser(a.db, strings.TrimSpace(payload.Username), payload.Password)
	if err != nil {
		http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}
	token, expires, err := createSession(a.db, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(authLoginResponse{
		Token:   token,
		Expires: expires,
		User:    user,
	})
}

func (a *app) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, ok := currentUserFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (a *app) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token != "" {
		_, _ = a.db.Exec(`DELETE FROM user_sessions WHERE token = ?`, token)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := listUsers(a.db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	case http.MethodPost:
		var payload userPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		user, err := createUser(a.db, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(user)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) handleUserByID(w http.ResponseWriter, r *http.Request) {
	idText := strings.TrimPrefix(r.URL.Path, "/api/users/")
	id, err := strconv.Atoi(idText)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	current, _ := currentUserFromRequest(r)
	switch r.Method {
	case http.MethodPut:
		var payload userPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		user, ok, err := updateUser(a.db, id, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(user)
	case http.MethodDelete:
		if id == current.ID {
			http.Error(w, "不能删除当前登录用户", http.StatusBadRequest)
			return
		}
		ok, err := deleteUser(a.db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func ensureInitialAdminUser(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	salt, err := randomHex(16)
	if err != nil {
		return err
	}
	hash := hashPassword(initialAdminPassword, salt)
	_, err = db.Exec(
		`INSERT INTO users (username, password_hash, password_salt, is_active) VALUES (?, ?, ?, 1)`,
		initialAdminUsername, hash, salt,
	)
	return err
}

func authenticateUser(db *sql.DB, username string, password string) (authUser, error) {
	var (
		user authUser
		hash string
		salt string
	)
	var active int
	err := db.QueryRow(
		`SELECT id, username, password_hash, password_salt, is_active FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &hash, &salt, &active)
	if err != nil {
		return authUser{}, err
	}
	if active == 0 {
		return authUser{}, errors.New("user disabled")
	}
	if hashPassword(password, salt) != hash {
		return authUser{}, errors.New("bad password")
	}
	user.IsActive = true
	return user, nil
}

func createSession(db *sql.DB, userID int) (string, int64, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", 0, err
	}
	expires := time.Now().Add(authTokenTTL).Unix()
	if _, err := db.Exec(
		`INSERT INTO user_sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expires,
	); err != nil {
		return "", 0, err
	}
	return token, expires, nil
}

func validateSession(db *sql.DB, token string) (authUser, error) {
	var user authUser
	var active int
	var expires int64
	err := db.QueryRow(`
SELECT u.id, u.username, u.is_active, s.expires_at
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = ?
`, token).Scan(&user.ID, &user.Username, &active, &expires)
	if err != nil {
		return authUser{}, err
	}
	if expires < time.Now().Unix() {
		_, _ = db.Exec(`DELETE FROM user_sessions WHERE token = ?`, token)
		return authUser{}, errors.New("session expired")
	}
	if active == 0 {
		return authUser{}, errors.New("user disabled")
	}
	user.IsActive = true
	return user, nil
}

func listUsers(db *sql.DB) ([]authUser, error) {
	rows, err := db.Query(`SELECT id, username, is_active FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]authUser, 0)
	for rows.Next() {
		var item authUser
		var active int
		if err := rows.Scan(&item.ID, &item.Username, &active); err != nil {
			return nil, err
		}
		item.IsActive = active != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func createUser(db *sql.DB, payload userPayload) (authUser, error) {
	username := strings.TrimSpace(payload.Username)
	if username == "" {
		return authUser{}, errText("username is required")
	}
	if strings.TrimSpace(payload.Password) == "" {
		return authUser{}, errText("password is required")
	}
	salt, err := randomHex(16)
	if err != nil {
		return authUser{}, err
	}
	active := 1
	if payload.IsActive != nil && !*payload.IsActive {
		active = 0
	}
	hash := hashPassword(payload.Password, salt)
	res, err := db.Exec(
		`INSERT INTO users (username, password_hash, password_salt, is_active) VALUES (?, ?, ?, ?)`,
		username, hash, salt, active,
	)
	if err != nil {
		return authUser{}, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return authUser{}, err
	}
	return authUser{ID: int(id64), Username: username, IsActive: active != 0}, nil
}

func updateUser(db *sql.DB, id int, payload userPayload) (authUser, bool, error) {
	var (
		old    authUser
		active int
	)
	if err := db.QueryRow(`SELECT id, username, is_active FROM users WHERE id = ?`, id).Scan(&old.ID, &old.Username, &active); err != nil {
		if err == sql.ErrNoRows {
			return authUser{}, false, nil
		}
		return authUser{}, false, err
	}
	old.IsActive = active != 0

	username := strings.TrimSpace(payload.Username)
	if username == "" {
		username = old.Username
	}
	newActive := old.IsActive
	if payload.IsActive != nil {
		newActive = *payload.IsActive
	}
	if _, err := db.Exec(`UPDATE users SET username = ?, is_active = ? WHERE id = ?`, username, boolToInt(newActive), id); err != nil {
		return authUser{}, false, err
	}
	if strings.TrimSpace(payload.Password) != "" {
		salt, err := randomHex(16)
		if err != nil {
			return authUser{}, false, err
		}
		hash := hashPassword(payload.Password, salt)
		if _, err := db.Exec(`UPDATE users SET password_hash = ?, password_salt = ? WHERE id = ?`, hash, salt, id); err != nil {
			return authUser{}, false, err
		}
	}
	if !newActive {
		_, _ = db.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, id)
	}
	return authUser{ID: id, Username: username, IsActive: newActive}, true, nil
}

func deleteUser(db *sql.DB, id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func hashPassword(password string, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
