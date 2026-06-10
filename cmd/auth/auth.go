package auth

import (
	"log"

	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func Init(coreURL, publicURL, apiKey string) {
	err := supertokens.Init(supertokens.TypeInput{
		Supertokens: &supertokens.ConnectionInfo{
			ConnectionURI: coreURL,
			APIKey:        apiKey,
		},
		AppInfo: supertokens.AppInfo{
			AppName:       "Gaugd",
			APIDomain:     publicURL,
			WebsiteDomain: publicURL,
		},
		RecipeList: []supertokens.Recipe{
			emailpassword.Init(nil),
			session.Init(nil),
		},
	})
	if err != nil {
		log.Fatalf("supertokens init: %s", err)
	}
}
