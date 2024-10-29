package main

import (
	"log"
	"net/http"

	"github.com/simplegameserver/gameserver/snake"
)

func main() {
    http.HandleFunc("/snake", snake.HandleConnection)

    go snake.GameLoop()

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}