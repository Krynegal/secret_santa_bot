package models

type User struct {
	ID        int
	Firstname string
	Lastname  string
	Username  string
	Organizer bool
}

type Room struct {
	id int
}

type CacheNote struct {
	RoomID      int
	State       string
	IsOrganizer bool
}
