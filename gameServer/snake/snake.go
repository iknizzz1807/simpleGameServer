package snake

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
    gridSize   = 20
    initSize   = 3
    canvasSize = 600
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
    ID        string     `json:"id"`
    Body      []Position `json:"body"`
    Direction Position   `json:"direction"`
    Score     int        `json:"score"`
    Conn      *websocket.Conn `json:"-"`
}

type Food struct {
    Position
}

type GameState struct {
    Type    string             `json:"type"`
    Players map[string]*Player `json:"players"`
    Food    []Food             `json:"foods"`
}

type DirectionMessage struct {
    Type      string   `json:"type"`
    Direction Position `json:"direction"`
}

type InitMessage struct {
    Type     string `json:"type"`
    PlayerID string `json:"playerId"`
}

type PlayerJoinedOrLeaveMessages struct {
    Type        string   `json:"type"`
    Message     []string `json:"message"`
    TotalPlayer int      `json:"totalPlayer"`
}

var (
    players = make(map[string]*Player)
    foods   = make([]Food, 0)
    joinOrLeaveMessages = PlayerJoinedOrLeaveMessages{
        Type:        "playerJoinedOrLeave",
        Message:     []string{},
        TotalPlayer: 0,
    }
    mu      sync.Mutex
)

func generateFood() Food {
    return Food{
        Position: Position{
            X: rand.Intn(canvasSize / gridSize),
            Y: rand.Intn(canvasSize / gridSize),
        },
    }
}

func initPlayer(id string) *Player {
    startX := rand.Intn(canvasSize/gridSize - initSize)
    startY := rand.Intn(canvasSize / gridSize)

    body := make([]Position, initSize)
    for i := 0; i < initSize; i++ {
        body[i] = Position{X: startX + i, Y: startY}
    }

    return &Player{
        ID:        id,
        Body:      body,
        Direction: Position{X: -1, Y: 0},
        Score:     0,
    }
}

func pingPlayer(playerID string, conn *websocket.Conn) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
                log.Printf("Ping failed for player %s: %v", playerID, err)
                handlePlayerDisconnect(playerID)
                return
            }
        }
    }
}

func handlePlayerDisconnect(playerID string) {
    mu.Lock()
    if player, exists := players[playerID]; exists {
        player.Conn.Close()
        delete(players, playerID)
        mu.Unlock()
        notifyPlayerJoinedAndLeave(playerID, "leave")
        log.Printf("Player %s disconnected", playerID)
    } else {
        mu.Unlock()
    }
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Websocket error: ", err)
        return
    }
    
    // Thiết lập các parameters cho connection
    conn.SetReadLimit(512) // Giới hạn kích thước message
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    conn.SetPongHandler(func(string) error { 
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    var initMsg InitMessage
    err = conn.ReadJSON(&initMsg)
    if err != nil || initMsg.Type != "init" {
        fmt.Println("Failed to read init message: ", err)
        conn.Close()
        return
    }

    playerID := initMsg.PlayerID

    mu.Lock()
    players[playerID] = initPlayer(playerID)
    players[playerID].Conn = conn
    mu.Unlock()

    notifyPlayerJoinedAndLeave(playerID, "join")
    log.Println("Joined player with id:", playerID)


    // Khởi động goroutine cho ping
    go pingPlayer(playerID, conn)
    // Handle messages
    for {
        var msg DirectionMessage
        err := conn.ReadJSON(&msg)
        if err != nil {
            log.Printf("Player disconnected: %s", playerID)
            handlePlayerDisconnect(playerID)
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
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    }
}

func notifyPlayerJoinedAndLeave(playerId string, joinOrLeave string) {
    // This will send a notification to the client
    joinOrLeaveMessage := fmt.Sprintf("%s %s the game", playerId, joinOrLeave)
    mu.Lock()
    joinOrLeaveMessages.Message = append(joinOrLeaveMessages.Message, joinOrLeaveMessage)
    if joinOrLeave == "join" {
        joinOrLeaveMessages.TotalPlayer++
    } else if joinOrLeave == "leave" {
        joinOrLeaveMessages.TotalPlayer--
    }
    
    messageJSON, err := json.Marshal(joinOrLeaveMessages)
    if err != nil {
        log.Println("JSON Marshal error:", err)
        mu.Unlock()
        return
    }
    mu.Unlock()

    broadcastMessage(messageJSON)
}

func broadcastMessage(message []byte) {
    mu.Lock()
    for playerID, player := range players {
        if err := player.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
            log.Printf("Failed to send message to player %s: %v", playerID, err)
            go handlePlayerDisconnect(playerID)
        }
    }
    mu.Unlock()
}

func GameLoop() {
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
            Players: players,
            Food:    foods,
        }

        stateJSON, _ := json.Marshal(gameState)
        for _, player := range players {
            player.Conn.WriteMessage(websocket.TextMessage, stateJSON)
        }

        mu.Unlock()
    }
}