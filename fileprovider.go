package avbot

import (
	"errors"
	"log"
)

type FileProvider interface {
	GetFile(fileid string) ([]byte, string, error)
}

var fileProvider map[string]FileProvider

func RegisterFileProvider(filetype string, provider FileProvider) {
	if fileProvider == nil {
		fileProvider = make(map[string]FileProvider)
	}

	if _, found := fileProvider[filetype]; found {
		log.Fatal("Provider already exists")
	}

	log.Println("register filet type " + filetype)
	fileProvider[filetype] = provider
}

func GetFile(filetype, fileid string) ([]byte, string, error) {
	if provider, found := fileProvider[filetype]; found {
		return provider.GetFile(fileid)
	} else {
		return nil, "", errors.New("fileprovider " + filetype + " not found")
	}
}
