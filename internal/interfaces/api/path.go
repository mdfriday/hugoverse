package api

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed admin/static/*
var staticFiles embed.FS

func adminStaticDir() http.FileSystem {
	staticDir := os.Getenv("HUGOVERSE_ADMIN_STATIC_DIR")
	if staticDir == "" {
		fsys, err := fs.Sub(staticFiles, "admin/static")
		if err != nil {
			log.Fatal(err)
		}
		return http.FS(fsys)
	}
	return http.Dir(staticDir)
}

func getWd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Couldn't find working directory", err)
	}
	return wd
}
