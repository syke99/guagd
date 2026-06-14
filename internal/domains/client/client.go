package client

import (
	"crypto/rand"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"guagd/internal/domains/client/pages/garage"
	"guagd/internal/domains/client/pages/hq"
	landingpkg "guagd/internal/domains/client/pages/landing"
	"guagd/internal/pkg/db"
	"guagd/internal/pkg/middleware"
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

//go:embed all:pages/landing
var landing embed.FS

//go:embed all:assets
var assets embed.FS

//go:embed all:app
var app embed.FS

type client struct {
	baseRoute string
	publicURL string
	db        db.DB
	sessions  sessions.Getter
	garage    *garage.GarageClient
	hq        *hq.HQClient
	landing   *landingpkg.LandingClient
}

func NewClient(baseRoute, publicURL string, db db.DB, store *storage.Client, heroBuildID, heroGarageID, heroClubID string) *client {
	sg := &sessions.SuperTokensGetter{}
	lc, err := landingpkg.NewLandingClient(db, store, publicURL, heroBuildID, heroGarageID, heroClubID)
	if err != nil {
		log.Printf("landing client init: %s", err)
	}
	return &client{
		baseRoute: baseRoute,
		publicURL: publicURL,
		db:        db,
		sessions:  sg,
		garage:    garage.NewGarageClient(db, store, sg),
		hq:        hq.NewHQClient(db, store, sg),
		landing:   lc,
	}
}

func (c *client) Handlers() map[string]http.HandlerFunc {
	sub, err := fs.Sub(landing, "pages/landing")
	if err != nil {
		log.Printf("error loading landing fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	assetsSub, err := fs.Sub(assets, "assets")
	if err != nil {
		log.Printf("error loading assets fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	appSub, err := fs.Sub(app, "app")
	if err != nil {
		log.Printf("error loading app fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	fileServer := http.FileServer(http.FS(sub))
	assetsServer := http.FileServer(http.FS(assetsSub))
	appServer := http.FileServer(http.FS(appSub))
	landingRoute := prefixRoute(c.baseRoute, "pages/landing/")
	assetsRoute := prefixRoute(c.baseRoute, "assets/")
	appRoute := prefixRoute(c.baseRoute, "app/")

	routes := map[string]http.HandlerFunc{
		"/": func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, app, "app/index.html")
		},
		appRoute: func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(appRoute, appServer).ServeHTTP(w, r)
		},
		landingRoute: func(w http.ResponseWriter, r *http.Request) {
			if _, err := r.Cookie("visitor_id"); err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     "visitor_id",
					Value:    newVisitorID(),
					Path:     "/",
					Expires:  time.Now().Add(365 * 24 * time.Hour),
					HttpOnly: false,
					SameSite: http.SameSiteLaxMode,
				})
			}
			http.StripPrefix(landingRoute, fileServer).ServeHTTP(w, r)
		},
		assetsRoute: func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(assetsRoute, assetsServer).ServeHTTP(w, r)
		},
		"/pages/landing/landing.html": func(w http.ResponseWriter, r *http.Request) {
			if c.landing != nil {
				c.landing.LandingPage(w, r)
				return
			}
			http.ServeFileFS(w, r, landing, "pages/landing/landing.html")
		},
		prefixRoute(c.baseRoute, "waitlist"):         c.waitlist,
		prefixRoute(c.baseRoute, "waitlist/success"): c.waitlistSuccess,
		prefixRoute(c.baseRoute, "waitlist/failure"): c.waitlistFailure,
		prefixRoute(c.baseRoute, "signup"):           c.signupPage,
		prefixRoute(c.baseRoute, "signup/failure"):   c.signupFailure,
		prefixRoute(c.baseRoute, "signin"):           c.signinPage,
		prefixRoute(c.baseRoute, "signin/failure"):   c.signinFailure,
		prefixRoute(c.baseRoute, "track/visit"):      c.trackVisit,
		prefixRoute(c.baseRoute, "access"):           c.accessPage,
		"/garage/{username}": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("HX-Request") != "true" {
				http.ServeFileFS(w, r, app, "app/index.html")
				return
			}
			c.garage.GaragePage(w, r)
		},
		"/fragments/car":                              c.garage.CarModalFragment,
		"/fragments/car-docs":                         c.garage.CarDocsFragment,
		"/api/v1/garage/layout":                      middleware.RequireAuth(c.garage.SaveLayout),
		"/api/v1/garage/theme":                        middleware.RequireAuth(c.garage.SaveTheme),
		"/api/v1/garage/cover":                        middleware.RequireAuth(c.garage.SaveCoverPhoto),
		"DELETE /api/v1/garage/cover":                 middleware.RequireAuth(c.garage.RemoveCoverPhoto),
		"/api/v1/garage/avatar":                       middleware.RequireAuth(c.garage.SaveAvatar),
		"DELETE /api/v1/garage/avatar":                middleware.RequireAuth(c.garage.RemoveAvatar),
		"/api/v1/garage/cars/add":                     middleware.RequireAuth(c.garage.AddCar),
		"/api/v1/garage/cars/update":                  middleware.RequireAuth(c.garage.UpdateCar),
		"/api/v1/garage/cars/remove":                  middleware.RequireAuth(c.garage.RemoveCar),
		"/api/v1/garage/cars/photos":                      c.garage.GetCarPhotos,
		"/api/v1/garage/cars/photos/add":                  middleware.RequireAuth(c.garage.AddCarPhoto),
		"/api/v1/garage/cars/photos/remove":               middleware.RequireAuth(c.garage.RemoveCarPhoto),
		"/api/v1/garage/cars/photos/primary":              middleware.RequireAuth(c.garage.SetCarPhotoPrimary),
		"/api/v1/garage/cars/mods":                            c.garage.GetCarMods,
		"/api/v1/garage/cars/mods/add":                        middleware.RequireAuth(c.garage.AddCarMod),
		"DELETE /api/v1/garage/cars/mods/remove":              middleware.RequireAuth(c.garage.RemoveCarMod),
		"/api/v1/garage/cars/mods/uploads":                         c.garage.GetCarUploads,
		"/api/v1/garage/cars/mods/uploads/add":                     middleware.RequireAuth(c.garage.AddCarUpload),
		"DELETE /api/v1/garage/cars/mods/uploads/remove":           middleware.RequireAuth(c.garage.RemoveCarUpload),
		"/api/v1/garage/cars/maintenance":                           c.garage.GetMaintenance,
		"/api/v1/garage/cars/maintenance/add":                      middleware.RequireAuth(c.garage.AddMaintenance),
		"DELETE /api/v1/garage/cars/maintenance/remove":            middleware.RequireAuth(c.garage.RemoveMaintenance),
		"/api/v1/garage/cars/maintenance/uploads":                   c.garage.GetMaintenanceUploads,
		"/api/v1/garage/cars/maintenance/uploads/add":               middleware.RequireAuth(c.garage.AddMaintenanceUpload),
		"DELETE /api/v1/garage/cars/maintenance/uploads/remove":    middleware.RequireAuth(c.garage.RemoveCarUpload),
		"/api/v1/hq/layout":                           middleware.RequireAuth(c.hq.SaveLayout),
		"/api/v1/hq/theme":                            middleware.RequireAuth(c.hq.SaveTheme),
		"/api/v1/hq/cover":                            middleware.RequireAuth(c.hq.SaveCoverPhoto),
		"DELETE /api/v1/hq/cover":                     middleware.RequireAuth(c.hq.RemoveCoverPhoto),
		"/fragments/hq-member-card":                   c.hq.MemberCardFragment,
		"/api/v1/hq/members":                          c.hq.ListMembers,
		"/api/v1/hq/members/add":                      c.hq.AddMember,
		"/api/v1/hq/members/remove":                   c.hq.RemoveMember,
		"/api/v1/hq/members/search":                   c.hq.SearchMembers,
		"/hq/{username}": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("HX-Request") != "true" {
				http.ServeFileFS(w, r, app, "app/index.html")
				return
			}
			c.hq.HQPage(w, r)
		},
	}

	return routes
}

func newVisitorID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
