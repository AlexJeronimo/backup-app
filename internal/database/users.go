package database

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int          `json:"id" db:"id"`
	Username     string       `json:"username" db:"username"`
	PasswordHash string       `json:"-" db:"password_hash"`
	CreatedAt    sql.NullTime `json:"created_at" db:"created_at"`
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(username, password string) (*User, error) {
	//Password hashing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("password hashing error: %w", err)
	}

	query := `INSERT INTO users (username, password_hash) VALUES (?, ?);`
	result, err := r.db.Exec(query, username, string(hashedPassword))
	if err != nil {
		return nil, fmt.Errorf("user insert error '%s': %w", username, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("new user ID getting error: %w", err)
	}

	return &User{
		ID:           int(id),
		Username:     username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	}, nil
}

func (r *UserRepo) GetUserByUsername(username string) (*User, error) {
	var user User
	var createdAtStr string
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username=?;`
	row := r.db.QueryRow(query, username)

	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user '%s' not found", username)
		}
		return nil, fmt.Errorf("user getting error '%s': %w", username, err)
	}

	parsedTime, err := time.Parse("2006-01-02 15:04:05.000", createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing created_at for user '%s': %w", username, err)
	}
	user.CreatedAt = sql.NullTime{Time: parsedTime, Valid: true}

	return &user, nil
}

func (r *UserRepo) AuthenticateUser(username, password string) (*User, error) {
	user, err := r.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("wrong password for user '%s'", username)
	}

	return user, nil
}

// Update User password //TODO: add update for another fields
func (r *UserRepo) UpdateUser(user *User, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("new password hashing error: %w", err)
	}

	query := `UPDATE users SET password_hash = ? WHERE id = ?;`
	_, err = r.db.Exec(query, string(hashedPassword), user.ID)
	if err != nil {
		return fmt.Errorf("user update error '%s': %w", user.Username, err)
	}

	user.PasswordHash = string(hashedPassword)
	return nil
}

func (r *UserRepo) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = ?;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delelte user with ID '%d' error: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting number of deleted rows  error: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %d not found for deleting", id)
	}
	return nil
}

func (r *UserRepo) GetAllUsers() ([]User, error) {
	query := `SELECT id, username, password_hash, created_at FROM users;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("getting all users error: %w", err)
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User
		var createdAtStr string
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &createdAtStr); err != nil {
			return nil, fmt.Errorf("user row scanning error: %w", err)
		}

		parsedTime, err := time.Parse("2006-01-02 15:04:05.000", createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing created_at for user '%s': %w", user.Username, err)
		}
		user.CreatedAt = sql.NullTime{Time: parsedTime, Valid: true}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}
	return users, nil
}
