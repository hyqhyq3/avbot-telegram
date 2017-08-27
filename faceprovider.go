package avbot

import (
	"errors"
	"log"
)

type FaceProvider interface {
	GetFace(uid string) ([]byte, string, error)
}

var faceProvider map[string]FaceProvider

func RegisterFaceProvider(name string, provider FaceProvider) {
	if faceProvider == nil {
		faceProvider = make(map[string]FaceProvider)
	}

	if _, found := faceProvider[name]; found {
		log.Fatal("Provider already exists")
	}

	log.Println("register filet type " + name)
	faceProvider[name] = provider
}

func GetFace(name, uid string) (p []byte, filetype string, err error) {
	if provider, found := faceProvider[name]; found {
		return provider.GetFace(uid)
	} else {
		return nil, "", errors.New("faceprovider " + name + " not found")
	}
	return
}
