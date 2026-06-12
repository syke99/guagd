package storage

import "testing"

func TestPhotoURLs(t *testing.T) {
	c := &Client{
		carPhotos:     BucketConfig{PublicURL: "https://car.example.com"},
		accountPhotos: BucketConfig{PublicURL: "https://account.example.com"},
	}

	t.Run("CarPhotoURL", func(t *testing.T) {
		got := c.CarPhotoURL("cars/abc/photo.jpg")
		want := "https://car.example.com/cars/abc/photo.jpg"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("AccountPhotoURL", func(t *testing.T) {
		got := c.AccountPhotoURL("accounts/xyz/banner.jpg")
		want := "https://account.example.com/accounts/xyz/banner.jpg"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
