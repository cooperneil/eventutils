package user

// User represents a user in our system.
type User struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	EmailAddress string `json:"email"`
	Age          int32  `json:"age,omitempty"`
	City         string `json:"city,omitempty"`
	Country      string `json:"country,omitempty"`
}
