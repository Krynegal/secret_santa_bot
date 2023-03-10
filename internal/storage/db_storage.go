package storage

import (
	"database/sql"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"secretSanta/internal/storage/models"
)

type DBStorager interface {
	Storager
}

type DB struct {
	db *sql.DB
}

func NewDatabaseStorage(dsn string) (DBStorager, error) {
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err = database.Ping(); err != nil {
		return nil, err
	}
	db := &DB{db: database}
	return db, nil
}

func (db *DB) AddUser(user models.User) error {
	_, err := db.db.Exec("INSERT INTO users VALUES ($1,$2,$3,$4);",
		user.ID,
		user.Firstname,
		user.Lastname,
		user.Username,
	)
	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code == pgerrcode.UniqueViolation {
			return nil
		}
		return err
	}
	return nil
}

func (db *DB) CreateRoom(roomID int) error {
	_, err := db.db.Exec("INSERT INTO rooms VALUES ($1);", roomID)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) AddWish(roomID, userID int, wish string) error {
	_, err := db.db.Exec("UPDATE id_room_id_user SET wishlist = ($1) WHERE id_user = ($2) AND id_room = ($3);", wish, userID, roomID)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) Wish(roomID, userID int) (string, error) {
	row := db.db.QueryRow(`SELECT wishlist FROM id_room_id_user WHERE id_user = $1 AND id_room = $2;`, userID, roomID)
	var wish string
	err := row.Scan(&wish)
	if err != nil {
		return "", err
	}
	return wish, nil
}

func (db *DB) AssignRoomToUser(roomID, userID int, isOrganizer bool) error {
	row := db.db.QueryRow(`SELECT EXISTS(SELECT * FROM id_room_id_user WHERE id_user = ($1) AND id_room = ($2));`, userID, roomID)
	var isExist bool
	err := row.Scan(&isExist)
	if err != nil {
		if err == sql.ErrNoRows {
			return RoomUserPairExists
		}
		return err
	}
	_, err = db.db.Exec("INSERT INTO id_room_id_user VALUES ($1, $2, $3);", roomID, userID, isOrganizer)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) UsersFromRoom(roomID int) ([]models.User, error) {
	rows, err := db.db.Query(`SELECT users.user_id, users.firstname, users.lastname, users.username, id_room_id_user.organizer FROM users
		JOIN id_room_id_user ON id_room_id_user.id_user = users.user_id
		JOIN rooms ON id_room_id_user.id_room = rooms.room_id
		WHERE room_id = ($1);`, roomID)
	if err != nil {
		return nil, err
	}

	defer func() {
		cerr := rows.Close()
		if cerr != nil {
			err = cerr
		}
	}()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err = rows.Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Username, &user.Organizer); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (db *DB) RoomWhereUserIsOrg(userID int) (int, error) {
	row := db.db.QueryRow(`SELECT room_id FROM rooms
		JOIN id_room_id_user ON id_room_id_user.id_room = rooms.room_id
		JOIN users ON id_room_id_user.id_user = users.user_id
		WHERE user_id = ($1) AND id_room_id_user.organizer = true;`, userID)
	var roomID int
	err := row.Scan(&roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			return -1, UserIsNotOrg
		}
		return -1, err
	}
	return roomID, nil
}
