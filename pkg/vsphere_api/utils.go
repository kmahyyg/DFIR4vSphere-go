package vsphere_api

import (
	gonanoid "github.com/matoous/go-nanoid"
)

func GetNanoID(length int) (string, error) {
	return gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", length)
}
