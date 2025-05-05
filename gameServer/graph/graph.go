package graph

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Score int             `json:"score"`
	Conn  *websocket.Conn `json:"-"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Monster struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	OfPlayer string  `json:"ofPlayer"` // Player ID
}

type GameState struct {
	Type     string    `json:"type"`
	Monsters []Monster `json:"monsters"`
	Players  []Player  `json:"players"`
}

type AddMonsterMessage struct {
	Type    string  `json:"type"`
	Monster Monster `json:"monster"`
}

type GraphMessage struct {
	Type       string     `json:"type"`
	Expression string     `json:"expression"`
	Points     []Position `json:"points"`
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
	players             = make(map[string]*Player)
	monsters            []Monster
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
		ID:    player.ID,
		Name:  player.Name,
		Score: 0,
	}
}

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket error:", err)
		return
	}

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	var initMsg InitMessage
	err = conn.ReadJSON(&initMsg)
	if err != nil || initMsg.Type != "init" {
		log.Println("Failed to read init message:", err)
		conn.Close()
		return
	}

	playerID := initMsg.Player.ID

	mu.Lock()
	players[playerID] = initPlayer(initMsg.Player)
	players[playerID].Conn = conn
	mu.Unlock()

	notifyPlayerJoinedAndLeave(playerID, "join")
	log.Println("Joined player with id:", playerID)

	go pingPlayer(playerID, conn)

	for {
		var msg interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error for player %s: %v", playerID, err)
			handlePlayerDisconnect(playerID)
			break
		}

		data, _ := json.Marshal(msg)
		var msgType struct {
			Type string `json:"type"`
		}
		json.Unmarshal(data, &msgType)

		if msgType.Type == "addMonster" {
			var addMsg AddMonsterMessage
			json.Unmarshal(data, &addMsg)
			mu.Lock()
			monsters = append(monsters, addMsg.Monster)
			broadcastGameState()
			mu.Unlock()
		} else if msgType.Type == "graph" {
			var graphMsg GraphMessage
			json.Unmarshal(data, &graphMsg)
			mu.Lock()
			processGraph(playerID, graphMsg.Points)
			broadcastGameState()
			mu.Unlock()
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}

func pingPlayer(playerID string, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
			log.Printf("Ping failed for player %s: %v", playerID, err)
			handlePlayerDisconnect(playerID)
			return
		}
	}
}

func handlePlayerDisconnect(playerID string) {
	mu.Lock()
	if player, exists := players[playerID]; exists {
		player.Conn.Close()
		delete(players, playerID)
		// Remove monsters associated with this player
		newMonsters := []Monster{}
		for _, m := range monsters {
			if m.OfPlayer != playerID {
				newMonsters = append(newMonsters, m)
			}
		}
		monsters = newMonsters
		mu.Unlock()
		notifyPlayerJoinedAndLeave(playerID, "leave")
		log.Printf("Player %s disconnected", playerID)
	} else {
		mu.Unlock()
	}
}

func notifyPlayerJoinedAndLeave(playerID string, joinOrLeave string) {
	joinOrLeaveMessage := fmt.Sprintf("%s %s the game", playerID, joinOrLeave)
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

func broadcastGameState() {
	playerList := make([]Player, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, Player{ID: p.ID, Name: p.Name, Score: p.Score})
	}

	gameState := GameState{
		Type:     "gameState",
		Monsters: monsters,
		Players:  playerList,
	}
	stateJSON, err := json.Marshal(gameState)
	if err != nil {
		log.Println("JSON Marshal error:", err)
		return
	}

	broadcastMessage(stateJSON)
}

func processGraph(playerID string, points []Position) {
	const HIT_DISTANCE = 0.5 // Distance threshold for monster hit
	newMonsters := []Monster{}
	for _, m := range monsters {
		hit := false
		for _, p := range points {
			distance := math.Sqrt(math.Pow(p.X-m.X, 2) + math.Pow(p.Y-m.Y, 2))
			if distance < HIT_DISTANCE {
				hit = true
				// Award points to the player who plotted the graph
				if player, exists := players[playerID]; exists {
					player.Score++
				}
				// Award points to the monster's owner if different
				if m.OfPlayer != playerID {
					if owner, exists := players[m.OfPlayer]; exists {
						owner.Score++
					}
				}
				break
			}
		}
		if !hit {
			newMonsters = append(newMonsters, m)
		}
	}
	monsters = newMonsters
}

func GameLoop() {
	// Placeholder for future real-time updates if needed
}
