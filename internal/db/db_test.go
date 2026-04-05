package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestOpenCreatesSchemaAndUpsertValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	d, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, d.UpsertUser("alice", string(hash), "admin"))

	u, err := d.ValidateCredentials("alice", "secret")
	require.NoError(t, err)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "admin", u.Role)

	_, err = d.ValidateCredentials("alice", "wrong")
	assert.Error(t, err)

	_, err = d.ValidateCredentials("nobody", "secret")
	assert.Error(t, err)
}

func TestAddUserDuplicate(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "d.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	h, _ := bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost)
	require.NoError(t, d.AddUser("u1", string(h), "viewer"))
	err = d.AddUser("u1", string(h), "editor")
	assert.Error(t, err)
}

func TestDeleteUserAndNotFound(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "d.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	h, _ := bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost)
	require.NoError(t, d.AddUser("gone", string(h), "viewer"))
	require.NoError(t, d.DeleteUser("gone"))
	err = d.DeleteUser("gone")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListUsersSorted(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "d.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	h, _ := bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost)
	require.NoError(t, d.AddUser("zebra", string(h), "viewer"))
	require.NoError(t, d.AddUser("alpha", string(h), "editor"))

	list, err := d.ListUsers()
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "alpha", list[0].Username)
	assert.Equal(t, "zebra", list[1].Username)
}

func TestUpdatePassword(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "d.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	h1, _ := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.MinCost)
	require.NoError(t, d.AddUser("bob", string(h1), "editor"))

	h2, _ := bcrypt.GenerateFromPassword([]byte("new"), bcrypt.MinCost)
	require.NoError(t, d.UpdatePassword("bob", string(h2)))

	_, err = d.ValidateCredentials("bob", "old")
	assert.Error(t, err)
	_, err = d.ValidateCredentials("bob", "new")
	assert.NoError(t, err)

	err = d.UpdatePassword("missing", string(h2))
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpsertUpdatesPasswordAndRole(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "d.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })

	h1, _ := bcrypt.GenerateFromPassword([]byte("a"), bcrypt.MinCost)
	require.NoError(t, d.UpsertUser("root", string(h1), "viewer"))

	h2, _ := bcrypt.GenerateFromPassword([]byte("b"), bcrypt.MinCost)
	require.NoError(t, d.UpsertUser("root", string(h2), "admin"))

	u, err := d.ValidateCredentials("root", "b")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Role)
}
