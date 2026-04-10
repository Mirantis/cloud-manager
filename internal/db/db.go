package db

import (
	"database/sql"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// APIToken is metadata for a stored API token (secret is never returned after creation).
type APIToken struct {
	ID          int64      `json:"id"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// AppUser is an application user (no password in JSON).
type AppUser struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// DB wraps SQLite access for app users.
type DB struct {
	conn *sql.DB
}

// Open opens the SQLite database and ensures schema exists.
func Open(path string) (*DB, error) {
	suffix := "?_pragma=foreign_keys(1)"
	if strings.Contains(path, "?") {
		suffix = "&_pragma=foreign_keys(1)"
	}
	conn, err := sql.Open("sqlite", path+suffix)
	if err != nil {
		return nil, err
	}
	if pingErr := conn.Ping(); pingErr != nil {
		_ = conn.Close()
		return nil, pingErr
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('admin','editor','viewer')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)
	`)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &DB{conn: conn}, nil
}

// AddUser inserts a new user (passwordHash must be bcrypt).
func (d *DB) AddUser(username, passwordHash, role string) error {
	_, err := d.conn.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role,
	)
	return err
}

// UpsertUser inserts or updates a user (for admin seeding).
func (d *DB) UpsertUser(username, passwordHash, role string) error {
	_, err := d.conn.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)
		 ON CONFLICT(username) DO UPDATE SET password_hash = excluded.password_hash, role = excluded.role`,
		username, passwordHash, role,
	)
	return err
}

func parseCreatedAt(s string) time.Time {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ValidateCredentials returns the user if username/password are valid.
func (d *DB) ValidateCredentials(username, password string) (*AppUser, error) {
	var passwordHash string
	var u AppUser
	var createdAt string
	err := d.conn.QueryRow(
		`SELECT username, password_hash, role, created_at FROM users WHERE username = ?`,
		username,
	).Scan(&u.Username, &passwordHash, &u.Role, &createdAt)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, err
	}
	u.CreatedAt = parseCreatedAt(createdAt)
	return &u, nil
}

// DeleteUser removes a user by username.
func (d *DB) DeleteUser(username string) error {
	res, err := d.conn.Exec(`DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListUsers returns all users (no passwords).
func (d *DB) ListUsers() ([]AppUser, error) {
	rows, err := d.conn.Query(`SELECT username, role, created_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var users []AppUser
	for rows.Next() {
		var u AppUser
		var createdAt string
		if err := rows.Scan(&u.Username, &u.Role, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = parseCreatedAt(createdAt)
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdatePassword sets a new bcrypt hash for the user.
func (d *DB) UpdatePassword(username, passwordHash string) error {
	res, err := d.conn.Exec(`UPDATE users SET password_hash = ? WHERE username = ?`, passwordHash, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.conn.Close()
}

// CreateToken inserts an API token row; tokenHash must be hex-encoded SHA-256 of the raw secret.
func (d *DB) CreateToken(username, tokenHash, description string, expiresAt *time.Time) (int64, error) {
	var exp interface{}
	if expiresAt != nil {
		exp = expiresAt.UTC().Format(time.RFC3339)
	}
	res, err := d.conn.Exec(
		`INSERT INTO api_tokens (username, token_hash, description, expires_at) VALUES (?, ?, ?, ?)`,
		username, tokenHash, description, exp,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ValidateToken looks up a token by hex-encoded SHA-256 hash and returns the app user if valid and not expired.
func (d *DB) ValidateToken(tokenHash string) (*AppUser, error) {
	var u AppUser
	var createdAt string
	var expiresAt sql.NullString
	err := d.conn.QueryRow(`
		SELECT u.username, u.role, u.created_at, t.expires_at
		FROM api_tokens t
		JOIN users u ON u.username = t.username
		WHERE t.token_hash = ?`,
		tokenHash,
	).Scan(&u.Username, &u.Role, &createdAt, &expiresAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid && expiresAt.String != "" {
		exp, parseErr := time.Parse(time.RFC3339, expiresAt.String)
		if parseErr != nil {
			exp = parseCreatedAt(expiresAt.String)
		}
		if !exp.IsZero() && time.Now().UTC().After(exp.UTC()) {
			return nil, sql.ErrNoRows
		}
	}
	u.CreatedAt = parseCreatedAt(createdAt)
	return &u, nil
}

// ListTokens returns non-secret metadata for all API tokens owned by username.
func (d *DB) ListTokens(username string) ([]APIToken, error) {
	rows, err := d.conn.Query(`
		SELECT id, description, created_at, expires_at
		FROM api_tokens WHERE username = ? ORDER BY id DESC`,
		username,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []APIToken
	for rows.Next() {
		var t APIToken
		var createdAt, exp sql.NullString
		if err := rows.Scan(&t.ID, &t.Description, &createdAt, &exp); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			t.CreatedAt = parseCreatedAt(createdAt.String)
		}
		if exp.Valid && exp.String != "" {
			parsed := parseCreatedAt(exp.String)
			if !parsed.IsZero() {
				t.ExpiresAt = &parsed
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteToken removes a token by id if the actor owns it, or if actorRole is admin.
func (d *DB) DeleteToken(id int64, actorUsername, actorRole string) error {
	var res sql.Result
	var err error
	if actorRole == "admin" {
		res, err = d.conn.Exec(`DELETE FROM api_tokens WHERE id = ?`, id)
	} else {
		res, err = d.conn.Exec(`DELETE FROM api_tokens WHERE id = ? AND username = ?`, id, actorUsername)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
