package models

type User struct {
	ID int `json:"id"`
	UserMap
}

type UserMap struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
}

func MapToUser(id int, user *UserMap) User {
	return User{
		ID:      id,
		UserMap: *user,
	}
}
