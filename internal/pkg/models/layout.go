package models

type LayoutItem struct {
	Component string `json:"component"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	W         int    `json:"w"`
	H         int    `json:"h"`
}
