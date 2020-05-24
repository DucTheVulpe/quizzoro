package storage

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/fatih/structs"
	"github.com/jmoiron/sqlx"
)

type State = string

const (
	StateDefault State = "default"
	StateWaiting State = "waiting"
	StateQuiz    State = "quiz"
)

type UsersStorage interface {
	Create(id int64) error
	Update(id int64, user User) error
	ByID(id int64) (User, error)
	State(id int64) (string, error)
	Privacy(id int64) (bool, error)
	InvertPrivacy(id int64) (bool, error)
	Cache(id int64) (UserCache, error)
	AddPoll(id int64, poll PassedPoll) error
	TopStats() ([]UserStats, error)
	Stats(id int) (UserStats, error)
}

type UsersTable struct {
	*sqlx.DB
}

type User struct {
	Model     `sq:"-"`
	UserCache `sq:",flatten,omitempty"`

	ID      int    `sq:"id,omitempty"`
	State   string `sq:"state,omitempty"`
	Privacy bool   `sq:"privacy,omitempty"`
}

type UserCache struct {
	LastPollID    string `sq:"last_poll_id,omitempty"`
	LastMessageID string `sq:"last_message_id,omitempty"`
	LastCategory  string `sq:"last_category,omitempty"`
}

func (db *UsersTable) Create(id int64) error {
	const q = `INSERT INTO users (id) VALUES (?)`
	_, err := db.Exec(q, id)
	return err
}

func (db *UsersTable) ByID(id int64) (user User, err error) {
	const q = `SELECT * FROM users WHERE id=?`
	return user, db.Get(&user, q, id)
}

func (db *UsersTable) Update(id int64, user User) error {
	q, args, err := sq.
		Update("users").
		SetMap(structs.Map(user)).
		Where("id=?", id).
		ToSql()
	if err != nil {
		return err
	}

	_, err = db.Exec(q, args...)
	return err
}

func (db *UsersTable) State(id int64) (state string, err error) {
	const q = `SELECT state FROM users WHERE id=?`
	return state, db.Get(&state, q, id)
}

func (db *UsersTable) InvertPrivacy(id int64) (privacy bool, _ error) {
	tx, err := db.Beginx()
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(`UPDATE users SET privacy = NOT privacy WHERE id=?`, id)
	if err != nil {
		tx.Rollback()
		return false, err
	}

	err = tx.Get(&privacy, `SELECT privacy FROM users WHERE id=?`, id)
	if err != nil {
		tx.Rollback()
		return false, err
	}

	return privacy, tx.Commit()
}

func (db *UsersTable) Privacy(id int64) (privacy bool, _ error) {
	const q = `SELECT privacy FROM users WHERE id=?`
	return privacy, db.Get(&privacy, q, id)
}

func (db *UsersTable) Cache(id int64) (cache UserCache, err error) {
	const q = `
		SELECT
		   last_poll_id,
		   last_message_id,
		   last_category
		FROM users
		WHERE id=?`

	return cache, db.Get(&cache, q, id)
}

func (db *UsersTable) AddPoll(id int64, poll PassedPoll) error {
	const q = `INSERT INTO passed_polls (user_id, poll_id, correct) VALUES (?, ?, ?)`
	_, err := db.Exec(q, id, poll.PollID, poll.Correct)
	return err
}

func (db *UsersTable) TopStats() (stats []UserStats, err error) {
	const q = `
		SELECT *,
		    (
				SELECT COUNT(*) FROM passed_polls
				WHERE user_id=users.id AND correct=1
			) correct,
		    (
				SELECT COUNT(*) FROM passed_polls
				WHERE user_id=users.id AND correct=0
			) incorrect
		FROM users
		WHERE id > 0
		ORDER BY correct
		LIMIT 3`

	return stats, db.Select(&stats, q)
}

func (db *UsersTable) Stats(id int) (stats UserStats, err error) {
	const q = `
		SELECT *,
		    (
				SELECT COUNT(*) FROM passed_polls
				WHERE user_id=users.id AND correct=1
			) correct,
		    (
				SELECT COUNT(*) FROM passed_polls
				WHERE user_id=users.id AND correct=0
			) incorrect
		FROM users
		WHERE id=?`

	return stats, db.Get(&stats, q, id)
}
