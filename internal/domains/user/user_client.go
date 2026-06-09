package user

type userClient struct {
	baseRoute string
}

func NewUserClient(baseRoute string) *userClient {
	return &userClient{baseRoute: baseRoute}
}
