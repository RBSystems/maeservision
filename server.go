package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/byuoitav/common"
	"github.com/byuoitav/maeservision/helpers"
	"github.com/labstack/echo"
)

func main() {

	go start()
	port := ":5275"
	router := common.NewRouter()
	started := false

	// websocket
	router.GET("/websocket", func(context echo.Context) error {
		helpers.ServeWebsocket(context.Response().Writer, context.Request())
		return nil
	})
	router.GET("/start", func(context echo.Context) error {
		fmt.Println("in start")
		if !started {
			started = true
			fmt.Println("Starting")
			helpers.StartRekognition()
			started = false
			fmt.Println("Finished")
		}
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
		fmt.Println("Starting")
		helpers.StartRekognition()
	}
}
