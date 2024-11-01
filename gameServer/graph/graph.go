package graph

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
    Id   string          `json:"id"`
    Name string          `json:"name"`
    Conn *websocket.Conn `json:"-"`
}

type Position struct {
    X int `json:"x"`
    Y int `json:"y"`
}

type Monster struct {
    X        int    `json:"x"`
    Y        int    `json:"y"`
    OfPlayer Player `json:"ofPlayer"`
}

type GameState struct {
    Type        string    `json:"type"`
    Monsters    []Monster `json:"monsters"`
    Players     []Player  `json:"players"`
    CurrentTurn string    `json:"turn"` // This uses an id of the player to indicate
}

type AddMonsterMessage struct {
    Type    string  `json:"type"`
    Monster Monster `json:"monster"`
}

type InitMessage struct {
    Type   string `json:"type"`
    Player Player `json:"player"`
}

type PlayerJoinedOrLeaveMessages struct {
    Type        string   `json:"type"`
    Message     []string `json:"message"`
    TotalPlayer int      `json:"totalPlayer"`
}

var (
    players = make(map[string]*Player)
    joinOrLeaveMessages = PlayerJoinedOrLeaveMessages{
        Type:        "playerJoinedOrLeave",
        Message:     []string{},
        TotalPlayer: 0,
    }
    mu       sync.Mutex
    upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }
)

func initPlayer(player Player) *Player {
    return &Player{
        Id:   player.Id,
        Name: player.Name,
    }
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket error:", err)
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
        log.Println("Failed to read init message: ", err)
        conn.Close()
        return
    }

    playerID := initMsg.Player.Id

    mu.Lock()
    players[playerID] = initPlayer(initMsg.Player)
    players[playerID].Conn = conn
    mu.Unlock()

    notifyPlayerJoinedAndLeave(playerID, "join")
    log.Println("Joined player with id:", playerID)

    // Khởi động goroutine cho ping
    go pingPlayer(playerID, conn)

    // Main message loop
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("Unexpected close error for player %s: %v", playerID, err)
            }
            handlePlayerDisconnect(playerID)
            break
        }
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    }
}

// When a player disconnects, the server will try to ping to the user to see any remaining chance of connecting
// before being 100% sure that disconnection is expected
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

// func broadcastGameState() {
//     mu.Lock()
//     defer mu.Unlock()

//     gameState := GameState{
//         Type:     "gameState",
//         Monsters: monsters,
//     }
//     stateJSON, err := json.Marshal(gameState)
//     if err != nil {
//         log.Println("JSON Marshal error:", err)
//         return
//     }

//     for _, player := range players {
//         err := player.Conn.WriteMessage(websocket.TextMessage, stateJSON)
//         if err != nil {
//             log.Println("Write error:", err)
//             player.Conn.Close()
//             delete(players, player.Id)
//         }
//     }
// }

func GameLoop() { // Now just skip this function dont care about it
    // ticker := time.NewTicker(100 * time.Millisecond)
    // defer ticker.Stop()

    // for range ticker.C {
    //     broadcastGameState()
    // }
}

        // var msg AddMonsterMessage
        // err := conn.ReadJSON(&msg)
        // if err != nil {
        //     log.Println("Read error:", err)
        //     mu.Lock()
        //     delete(players, playerID)
        //     notifyPlayerJoinedAndLeave(playerID, "leave")
        //     mu.Unlock()
        //     break
        // }

        // if msg.Type == "addMonster" {
        //     mu.Lock()
        //     monsters = append(monsters, msg.Monster)
        //     mu.Unlock()
        // }