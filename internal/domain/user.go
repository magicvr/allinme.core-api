package domain

// User is the authenticated principal (minimal fields for $context.user).
type User struct {
	ID           string
	Username     string
	Name         string
	PasswordHash string
	Roles        []string
}

// PublicUser is the API-safe user payload (no password hash).
type PublicUser struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
}

// ToPublic maps User to PublicUser.
func (u User) ToPublic() PublicUser {
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	return PublicUser{
		ID:       u.ID,
		Username: u.Username,
		Name:     u.Name,
		Roles:    roles,
	}
}

// HasRole reports whether the user has role.
func (u User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether the user has any of the roles.
func (u User) HasAnyRole(roles ...string) bool {
	for _, want := range roles {
		if u.HasRole(want) {
			return true
		}
	}
	return false
}
