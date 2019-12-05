package main

import (
	"bufio"
	"os"

	"github.com/byuoitav/common"
	"github.com/byuoitav/maeservision/helpers"
	"github.com/labstack/echo"
)

func main() {

	go start()
	port := ":5275"
	router := common.NewRouter()

	// websocket
	router.GET("/websocket", func(context echo.Context) error {
		helpers.ServeWebsocket(context.Response().Writer, context.Request())
		return nil
	})

	router.Static("/", "index.html")
	router.File("/style.css", "style.css")

	router.Start(port)
}

func start() {
	reader := bufio.NewReader(os.Stdin)
	for {
		reader.ReadString('\n')
		go helpers.StartRekognition()
	}
}
