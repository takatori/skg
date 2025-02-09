package main

import (
	"github.com/labstack/echo/v4"
	"github.com/takatori/skg/internal/server"
)

func main() {

	e := echo.New()
	server.InitServer(e)
	e.Logger.Fatal(e.Start(":8080"))
}
