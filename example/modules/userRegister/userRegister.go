package userRegister

type User struct {
	ID    int
	Name  string
	Email string
}

func (u *User) Create(data ...any) (any, error) {
	created := make([]*User, 0, len(data))
	for _, item := range data {
		user := item.(*User)
		user.ID = 123
		created = append(created, user)
	}
	return created, nil
}

func (u *User) Read(data ...any) (any, error) {
	results := make([]*User, 0, len(data))
	for _, item := range data {
		user := item.(*User)
		results = append(results, &User{ID: user.ID, Name: "Found " + user.Name, Email: user.Email})
	}
	return results, nil
}
