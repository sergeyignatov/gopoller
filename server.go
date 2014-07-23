package main

import (
	"github.com/sergeyignatov/gopoller/models"

	"github.com/go-martini/martini"
)

func main() {
	m := martini.Classic()
	m.Get("/", func() string {
		m := models.NewTask("ya.ru", 30)
		go m.Start()
		m.Stop()
		return "Hello world!"

	})
	m.Run()

}
