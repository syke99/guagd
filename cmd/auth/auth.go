package auth

import "github.com/supertokens/supertokens-golang/supertokens"

func Connect(connectionURI, key string) {
	supertokens.Init(supertokens.TypeInput{
		Supertokens: &supertokens.ConnectionInfo{
			ConnectionURI: connectionURI,
			APIKey:        key,
		},
	})
}
