// internal/repository/user_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/souvik03-136/Go-Store/internal/models"
)

type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) CreateUser(user *models.User) error {
	_, err := r.db.Exec(`INSERT INTO users(id,username,email,password_hash,role,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}
func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	return r.scanUser(r.db.QueryRow(`SELECT id,username,email,password_hash,role,created_at,updated_at,auth_version FROM users WHERE id=$1`, id))
}
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	return r.scanUser(r.db.QueryRow(`SELECT id,username,email,password_hash,role,created_at,updated_at,auth_version FROM users WHERE email=$1`, email))
}
func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	return r.scanUser(r.db.QueryRow(`SELECT id,username,email,password_hash,role,created_at,updated_at,auth_version FROM users WHERE username=$1`, username))
}

func (r *UserRepository) UpdateUser(user *models.User, passwordChanged bool) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("beginning user update: %w", err)
	}
	defer tx.Rollback()
	var version int64
	err = tx.QueryRow(`UPDATE users SET username=$1,email=$2,password_hash=$3,updated_at=$4,auth_version=auth_version+CASE WHEN $5 THEN 1 ELSE 0 END WHERE id=$6 RETURNING auth_version`, user.Username, user.Email, user.PasswordHash, time.Now(), passwordChanged, user.ID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("user not found")
	}
	if err != nil {
		return fmt.Errorf("updating user %s: %w", user.ID, err)
	}
	action := "profile_update"
	if passwordChanged {
		action = "credential_change"
	}
	if err := insertAuditEvent(context.Background(), tx, AuditEvent{ActorID: user.ID, ActorSubject: user.ID, OwnerID: user.ID, ResourceType: "user", ResourceID: user.ID, Action: action, Outcome: "success", Details: map[string]any{"password_changed": passwordChanged}}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing user update: %w", err)
	}
	user.AuthVersion = version
	return nil
}
func (r *UserRepository) DeleteUser(id string) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("beginning user deletion: %w", err)
	}
	defer tx.Rollback()
	var ownsData bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM files WHERE owner_id=$1) OR EXISTS(SELECT 1 FROM upload_reservations WHERE owner_id=$1)`, id).Scan(&ownsData); err != nil {
		return fmt.Errorf("checking user storage cleanup: %w", err)
	}
	if ownsData {
		return errors.New("user still owns files or uploads; delete files and wait for cleanup before deleting account")
	}
	if err := insertAuditEvent(context.Background(), tx, AuditEvent{ActorID: id, ActorSubject: id, OwnerID: id, ResourceType: "user", ResourceID: id, Action: "account_delete", Outcome: "success", Details: map[string]any{}}); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("deleting user %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("user not found")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing account deletion: %w", err)
	}
	return nil
}
func (r *UserRepository) GetUserRole(id string) (string, error) {
	var role string
	err := r.db.QueryRow(`SELECT role FROM users WHERE id=$1`, id).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("user not found")
	}
	if err != nil {
		return "", fmt.Errorf("reading user role: %w", err)
	}
	return role, nil
}
func (r *UserRepository) GetAuthVersion(id string) (int64, error) {
	var version int64
	err := r.db.QueryRow(`SELECT auth_version FROM users WHERE id=$1`, id).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("user not found")
	}
	if err != nil {
		return 0, fmt.Errorf("reading user session version: %w", err)
	}
	return version, nil
}
func (r *UserRepository) ListUsers() ([]*models.User, error) {
	rows, err := r.db.Query(`SELECT id,username,email,role,created_at,updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()
	users := make([]*models.User, 0)
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating user rows: %w", err)
	}
	return users, nil
}
func (r *UserRepository) scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.AuthVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	return &u, nil
}
