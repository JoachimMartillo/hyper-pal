package modelsHyper

type Auth struct {
	Email			string				`json:"email"`
	Password		string				`json:"password"`
}

func CreateAuth(email, password string) *Auth {
	return &Auth{
		Email: email,
		Password: password,
	}
}
