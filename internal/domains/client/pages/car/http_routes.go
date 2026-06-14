package car

import (
	"log"
	"net/http"

	"guagd/internal/pkg/models"
)

func (c *CarPageClient) CarPage(w http.ResponseWriter, r *http.Request) {
	shortID := r.URL.Query().Get("c")
	if len(shortID) < 8 {
		http.NotFound(w, r)
		return
	}
	shortID = shortID[:8]

	ctx := r.Context()

	car, err := c.getCarByShortID(ctx, shortID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	owner, err := c.getOwner(ctx, car.ID)
	if err != nil {
		log.Printf("carPage: get owner: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	photos, err := c.getPhotos(ctx, car.ID)
	if err != nil {
		log.Printf("carPage: get photos: %s", err)
		photos = []models.CarPhoto{}
	}

	mods, err := c.getMods(ctx, car.ID)
	if err != nil {
		log.Printf("carPage: get mods: %s", err)
		mods = []models.Mod{}
	}

	maintenance, err := c.getMaintenance(ctx, car.ID)
	if err != nil {
		log.Printf("carPage: get maintenance: %s", err)
		maintenance = []models.Maintenance{}
	}

	docs, err := c.getDocs(ctx, car.ID)
	if err != nil {
		log.Printf("carPage: get docs: %s", err)
		docs = []models.CarPageDoc{}
	}

	sessionContainer, _ := c.sessions.GetOptionalSession(r, w)

	data := models.CarPageData{
		Car:             car,
		Owner:           owner,
		Photos:          photos,
		Mods:            mods,
		Maintenance:     maintenance,
		Docs:            docs,
		AvatarURL:       owner.AvatarURL,
		IsAuthenticated: sessionContainer != nil,
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	if err := carPageTemplate.ExecuteTemplate(w, "car.html", data); err != nil {
		log.Printf("carPage: render: %s", err)
	}
}
