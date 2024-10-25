package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	gridSize = 20
	initSize = 3
	canvasSize = 400
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"` 
}

type Player struct {
	ID string `json:"id"`
	Body []Position `json:"body"`
	Direction Position `json:"direction"`
	Score int `json:"score"`
	Conn *websocket.Conn `json:"-"`
}

type Food struct {
	Position
}

type GameState struct {
	Type string `json:"type"`
	Players map[string]*Player `json:"players"`
	Food []Food `json:"foods"`
}

type DirectionMessage struct {
	Type string `json:"type"`
	Direction Position `json:"direction"`
}

var (
	players = make(map[string]*Player) // Global variable as list of pointers
	foods = make([]Food, 0) // List of food to eat
	mu sync.Mutex
)

func generateFood() Food {
	return Food {
		Position: Position {
			X: rand.Intn(canvasSize/gridSize),
            Y: rand.Intn(canvasSize/gridSize),
		},
	}
}

func initPlayer(id string) *Player {
	startX := rand.Intn(canvasSize/gridSize - initSize)
    startY := rand.Intn(canvasSize/gridSize)

	body := make([]Position, initSize) // List of all parts of the body, with the size = initSize
	for i := 0; i < initSize; i++ {
		body[i] = Position{X: startX + i, Y: startY}
	}

	return &Player{
		ID: id,
		Body: body,
		Direction: Position{X: -1, Y: 0},
		Score: 0,
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request) { // This is used to make http 3 way handshake
	// for the future websocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)

	if(err != nil) {
		fmt.Println("Websocket error: ", err)
		return
	}

	defer conn.Close()

	playerID := fmt.Sprintf("player-%d", rand.Intn(1000)) // Random an id for the playerId

	mu.Lock()
	players[playerID] = initPlayer(playerID)
	players[playerID].Conn = conn
	mu.Unlock() // Lock and unlock the resource

	// Send initial game state
    initMessage, _ := json.Marshal(map[string]string{
        "type":     "init",
        "playerId": playerID,
    })

	conn.WriteMessage(websocket.TextMessage, initMessage)

	log.Printf("New player connected: %s", playerID)

	// Handle messages
	for { 
		// Infinit loop without break condition
		var msg DirectionMessage
		err := conn.ReadJSON(&msg) // Read the JSON message and pass its value to the msg variable
		// Message from the client:
		// JSON.stringify({
		// 	type: "direction",
		// 	direction: direction,
		// })


		if err != nil {
			log.Printf("Player disconnected: %s", playerID)
            mu.Lock()
            delete(players, playerID)
            mu.Unlock()
			// Once again use lock and unlock to lock the resource when work with id
            break
		}

		if msg.Type == "direction" {
			mu.Lock()
			if player, exists := players[playerID]; exists {
				// Prevent 180-degree turns
				if !(player.Direction.X == -msg.Direction.X && player.Direction.Y == -msg.Direction.Y) {
					player.Direction = msg.Direction
				}
			}
			mu.Unlock()
		}
	}

}

// The loop to loop the game, use the list of players provided above to calculate the game logic
func gameLoop() {
    // Initialize foods
    for i := 0; i < 5; i++ {
        foods = append(foods, generateFood())
    }
    
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for range ticker.C {
        mu.Lock()
        
        // Update each player
        for _, player := range players {
            // Create new head position
            newHead := Position{
                X: player.Body[0].X + player.Direction.X,
                Y: player.Body[0].Y + player.Direction.Y,
            }
            
            // Check wall collision
            if newHead.X < 0 || newHead.X >= canvasSize/gridSize ||
                newHead.Y < 0 || newHead.Y >= canvasSize/gridSize {
                // Reset player
                newPlayer := initPlayer(player.ID)
                player.Body = newPlayer.Body
                player.Direction = newPlayer.Direction
                player.Score = 0
                continue
            }
            
            // Check food collision
            ateFood := false
            for i, food := range foods {
                if newHead.X == food.X && newHead.Y == food.Y {
                    foods[i] = generateFood()
                    ateFood = true
                    player.Score++
                    break
                }
            }
            
            // Move snake
            newBody := make([]Position, len(player.Body))
            copy(newBody[1:], player.Body[:len(player.Body)-1])
            newBody[0] = newHead
            
            if ateFood {
                player.Body = append(newBody, player.Body[len(player.Body)-1])
            } else {
                player.Body = newBody
            }
        }
        
        // Send game state to all players
        gameState := GameState{
            Type:    "gameState",
            Players: players, // Each player has his own score
            Food:   foods,
        }
        
        stateJSON, _ := json.Marshal(gameState)
        for _, player := range players {
            player.Conn.WriteMessage(websocket.TextMessage, stateJSON)
        }
        
        mu.Unlock()
    }
}

func main() {
    // rand.Seed(time.Now().UnixNano())
    
    // WebSocket endpoint
    http.HandleFunc("/ws", handleConnection) // This has an infinite loop
	// This is used to receive message from the client
    
    // Start game loop
    go gameLoop()
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}