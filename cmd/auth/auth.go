package auth

import (
	"log"
	"net/http"

	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
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
			session.Init(&sessmodels.TypeInput{
				GetTokenTransferMethod: func(req *http.Request, forCreateNewSession bool, userContext supertokens.UserContext) sessmodels.TokenTransferMethod {
					return sessmodels.CookieTransferMethod
				},
			}),
		},
	})
	if err != nil {
		log.Fatalf("supertokens init: %s", err)
	}
}
