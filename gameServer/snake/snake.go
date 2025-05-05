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
	// Định nghĩa số ô mong muốn
	numCells   = 30
	canvasSize = 600

	initSize = 3
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
	ID        string          `json:"id"`
	Body      []Position      `json:"body"`
	Direction Position        `json:"direction"`
	Score     int             `json:"score"`
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

// Global variables
var (
	players             = make(map[string]*Player)
	foods               = make([]Food, 0)
	joinOrLeaveMessages = PlayerJoinedOrLeaveMessages{
		Type:        "playerJoinedOrLeave",
		Message:     []string{},
		TotalPlayer: 0,
	}
	mu sync.Mutex
)

func generateFood() Food {
	return Food{
		Position: Position{
			X: rand.Intn(numCells),
			Y: rand.Intn(numCells),
		},
	}
}

func initPlayer(id string) *Player {
	// Đặt vị trí bắt đầu trong phạm vi số ô mới
	startX := rand.Intn(numCells-initSize*2) + initSize // Cách lề trái ít nhất initSize ô
	startY := rand.Intn(numCells)

	body := make([]Position, initSize)
	for i := 0; i < initSize; i++ {
		body[i] = Position{X: startX + initSize - 1 - i, Y: startY}
	}

	return &Player{
		ID:        id,
		Body:      body,
		Direction: Position{X: 1, Y: 0}, // Hướng sang phải ban đầu
		Score:     0,
		// Conn sẽ được gán sau khi kết nối
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
	joinOrLeaveText := fmt.Sprintf("Player %s %s the game", playerId, joinOrLeave) // Đổi tên biến

	mu.Lock()
	// Cập nhật trạng thái global trước
	joinOrLeaveMessages.Message = append(joinOrLeaveMessages.Message, joinOrLeaveText)
	newTotalPlayers := joinOrLeaveMessages.TotalPlayer
	if joinOrLeave == "join" {
		newTotalPlayers++
	} else if joinOrLeave == "leave" {
		newTotalPlayers--
		if newTotalPlayers < 0 {
			newTotalPlayers = 0
		} // Đảm bảo không âm
	}
	joinOrLeaveMessages.TotalPlayer = newTotalPlayers

	// Chuẩn bị message để gửi (chỉ chứa tin nhắn mới nhất)
	msgToSend := PlayerJoinedOrLeaveMessages{
		Type:        "playerJoinedOrLeave",
		Message:     []string{joinOrLeaveText}, // Chỉ gửi tin nhắn mới nhất
		TotalPlayer: newTotalPlayers,
	}
	mu.Unlock() // Mở khóa trước khi marshal và broadcast

	messageJSON, err := json.Marshal(msgToSend)
	if err != nil {
		log.Println("JSON Marshal error in notify:", err)
		return
	}

	// Sử dụng cơ chế broadcast không khóa lâu
	broadcastNotification(messageJSON)
}

func GameLoop() {
	// Initialize foods
	mu.Lock()            // Khóa để khởi tạo food an toàn
	if len(foods) == 0 { // Chỉ khởi tạo nếu chưa có food
		for range 5 {
			foods = append(foods, generateFood())
		}
	}
	mu.Unlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock() // Khóa toàn bộ quá trình cập nhật game state

		playersToReset := []string{}              // Danh sách ID người chơi cần reset
		playerUpdates := make(map[string]*Player) // Lưu trạng thái mới của player (nếu không reset)

		// --- Vòng 1: Tính toán di chuyển và kiểm tra va chạm ---
		for playerID, player := range players {
			if player == nil || len(player.Body) == 0 { // Bỏ qua nếu player không hợp lệ
				continue
			}

			// Tạo vị trí đầu mới
			newHead := Position{
				X: player.Body[0].X + player.Direction.X,
				Y: player.Body[0].Y + player.Direction.Y,
			}

			// 1. Kiểm tra va chạm tường
			if newHead.X < 0 || newHead.X >= numCells ||
				newHead.Y < 0 || newHead.Y >= numCells {
				log.Printf("Player %s hit the wall.", playerID)
				playersToReset = append(playersToReset, playerID)
				continue // Chuyển sang người chơi tiếp theo
			}

			// 2. Kiểm tra va chạm với người chơi khác (bao gồm cả thân của họ)
			collisionWithOther := false
			for otherID, otherPlayer := range players {
				if playerID == otherID {
					continue
				} // Bỏ qua chính mình
				if otherPlayer == nil || len(otherPlayer.Body) == 0 {
					continue
				} // Bỏ qua player khác không hợp lệ

				for _, segment := range otherPlayer.Body {
					if newHead.X == segment.X && newHead.Y == segment.Y {
						log.Printf("Player %s hit player %s.", playerID, otherID)
						collisionWithOther = true
						break
					}
				}
				if collisionWithOther {
					break
				}
			}
			if collisionWithOther {
				playersToReset = append(playersToReset, playerID)
				continue // Chuyển sang người chơi tiếp theo
			}

			// --- Nếu không va chạm tường hoặc người khác, tiếp tục xử lý ---

			// 3. Kiểm tra va chạm thức ăn
			ateFood := false
			foodIndexToRemove := -1
			for i, food := range foods {
				if newHead.X == food.X && newHead.Y == food.Y {
					foodIndexToRemove = i
					ateFood = true
					player.Score++ // Tăng điểm trực tiếp trên player hiện tại
					break
				}
			}
			// Xóa food đã ăn và tạo food mới (nếu có)
			if foodIndexToRemove != -1 {
				foods = append(foods[:foodIndexToRemove], foods[foodIndexToRemove+1:]...)
				foods = append(foods, generateFood())
			}

			// 4. Cập nhật thân rắn
			var newBody []Position
			if ateFood {
				// Thêm đầu mới vào đầu slice, giữ nguyên đuôi
				newBody = append([]Position{newHead}, player.Body...)
			} else {
				// Thêm đầu mới, bỏ đuôi cũ
				newBody = append([]Position{newHead}, player.Body[:len(player.Body)-1]...)
			}
			player.Body = newBody // Cập nhật body

			// Lưu trạng thái player đã cập nhật để áp dụng sau
			playerUpdates[playerID] = player
		} // Kết thúc vòng lặp kiểm tra va chạm

		// --- Vòng 2: Áp dụng các thay đổi và reset ---

		// Áp dụng cập nhật cho những người chơi không bị reset
		for id, updatedPlayer := range playerUpdates {
			// Kiểm tra xem player này có trong danh sách reset không
			shouldReset := false
			for _, resetID := range playersToReset {
				if id == resetID {
					shouldReset = true
					break
				}
			}
			if !shouldReset {
				players[id] = updatedPlayer // Chỉ cập nhật nếu không bị reset
			}
		}

		// Reset những người chơi đã va chạm
		for _, playerID := range playersToReset {
			if player, exists := players[playerID]; exists {
				log.Printf("Resetting player %s.", playerID)
				newPlayer := initPlayer(playerID) // Tạo player mới
				newPlayer.Conn = player.Conn      // Giữ lại connection cũ
				players[playerID] = newPlayer     // Thay thế player cũ trong map
			}
		}

		// --- Vòng 3: Chuẩn bị và gửi game state ---
		// Tạo gameState với trạng thái players đã được cập nhật/reset
		gameState := GameState{
			Type:    "gameState",
			Players: players,
			Food:    foods,
		}
		stateJSON, err := json.Marshal(gameState)
		if err != nil {
			log.Println("Error marshaling game state:", err)
			// Không unlock và tiếp tục vòng lặp để tránh lỗi liên tục
		} else {
			// Mở khóa trước khi broadcast để tránh giữ lock quá lâu
			mu.Unlock()
			broadcastGameState(stateJSON) // Sử dụng hàm broadcast mới
			// Không cần Lock lại vì đã kết thúc vòng lặp tick này
		}
		// Nếu có lỗi Marshal, Mutex vẫn đang bị khóa, vòng lặp ticker tiếp theo sẽ bị block
		// Cần xử lý tốt hơn, ví dụ: unlock và log lỗi rồi tiếp tục

	} // Kết thúc vòng lặp ticker.C
}

func broadcastGameState(stateJSON []byte) {
	mu.Lock()
	// Tạo bản sao danh sách player để tránh race condition khi handle disconnect
	currentPlayers := make(map[string]*Player)
	for k, v := range players {
		currentPlayers[k] = v
	}
	mu.Unlock()

	for playerID, player := range currentPlayers {
		if player.Conn != nil { // Kiểm tra conn không nil
			err := player.Conn.WriteMessage(websocket.TextMessage, stateJSON)
			if err != nil {
				log.Printf("Failed to send game state to player %s: %v. Triggering disconnect.", playerID, err)
				// Chạy xử lý disconnect trong goroutine riêng để tránh deadlock
				go handlePlayerDisconnect(playerID)
			}
		}
	}
}

func broadcastNotification(messageJSON []byte) {
	mu.Lock()
	currentPlayers := make(map[string]*Player)
	for k, v := range players {
		currentPlayers[k] = v
	}
	mu.Unlock()

	for playerID, player := range currentPlayers {
		if player.Conn != nil {
			err := player.Conn.WriteMessage(websocket.TextMessage, messageJSON)
			if err != nil {
				log.Printf("Failed to send notification to player %s: %v. Triggering disconnect.", playerID, err)
				go handlePlayerDisconnect(playerID)
			}
		}
	}
}
