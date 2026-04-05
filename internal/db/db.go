package db

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

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
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
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
		conn.Close()
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
	defer rows.Close()
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
