package main

import (
	"example.com/pz16/internal/db"
	"example.com/pz16/internal/httpapi"
	"example.com/pz16/internal/repo"
	"example.com/pz16/internal/service"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

func main() {
	d, e := db.Open(os.Getenv("DB_DSN"))
	if e != nil {
		log.Fatal(e)
	}
	defer d.Close()
	if e = db.ApplyMigrations(d); e != nil {
		log.Fatal(e)
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	s := service.Service{Notes: repo.NoteRepo{DB: d}}
	httpapi.Router{Svc: &s}.Register(r)
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
