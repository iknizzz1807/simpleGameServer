package main

import (
	"log"
	"net/http"

	"github.com/simplegameserver/gameserver/caro"
	"github.com/simplegameserver/gameserver/graph"
	"github.com/simplegameserver/gameserver/snake"
)

func main() {
	http.HandleFunc("/snake", snake.HandleConnection)
	http.HandleFunc("/graph", graph.HandleConnection)
	http.HandleFunc("/caro", caro.HandleConnection)

	go snake.GameLoop()
	go graph.GameLoop()

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
