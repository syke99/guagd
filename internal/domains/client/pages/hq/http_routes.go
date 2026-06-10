package hq

import (
	"log"
	"net/http"
	"strings"

	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
)

func (h *HQClient) HQPage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.PathValue("username"), "@")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	user, err := h.getUserByUsername(r.Context(), username)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sessionRequired := false
	sessionContainer, _ := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})

	isOwner := sessionContainer != nil && sessionContainer.GetUserID() == user.SupertokensID

	data := HQPageData{
		Username: user.Username,
		IsOwner:  isOwner,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := hqTemplate.ExecuteTemplate(w, "hq.html", data); err != nil {
		log.Printf("hqPage: render: %s", err)
	}
}
