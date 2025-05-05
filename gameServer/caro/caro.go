package caro

import (
	"encoding/json" // Để mã hóa và giải mã dữ liệu JSON (giao tiếp với frontend)
	"fmt"           // Để định dạng chuỗi (ví dụ: trong log và tin nhắn)
	"log"           // Để ghi log các sự kiện và lỗi trên server
	"net/http"      // Để xử lý các yêu cầu HTTP (nâng cấp lên WebSocket)
	"sync"          // Để xử lý đồng bộ (sử dụng Mutex bảo vệ dữ liệu dùng chung)
	"time"          // Để xử lý thời gian (ví dụ: đặt deadline, ticker)

	"github.com/gorilla/websocket" // Thư viện WebSocket phổ biến cho Go
)

// --- Hằng số của Game ---
const (
	BOARD_SIZE    = 15 // Kích thước bàn cờ (15x15)
	WIN_CONDITION = 5  // Số quân liên tiếp để thắng (5 quân)
)

// --- Cấu trúc dữ liệu (Structs) ---

// Player: Đại diện cho một người chơi trong game.
type Player struct {
	ID   string          `json:"id"`   // ID duy nhất của người chơi (thường là từ frontend)
	Name string          `json:"name"` // Tên hiển thị của người chơi
	Mark string          `json:"mark"` // Quân cờ của người chơi ("X" hoặc "O")
	Conn *websocket.Conn `json:"-"`    // Kết nối WebSocket của người chơi (dấu "-" để không gửi thông tin này qua JSON cho frontend)
}

// Move: Đại diện cho một nước đi trên bàn cờ.
type Move struct {
	X int `json:"x"` // Tọa độ X của nước đi (0-based index)
	Y int `json:"y"` // Tọa độ Y của nước đi (0-based index)
}

// GameState: Đại diện cho trạng thái hiện tại của game.
// Struct này được gửi tới frontend mỗi khi có thay đổi quan trọng.
type GameState struct {
	Type        string     `json:"type"`        // Loại tin nhắn, luôn là "gameState" để frontend biết cách xử lý
	Board       [][]string `json:"board"`       // Trạng thái bàn cờ (mảng 2 chiều string, "" là ô trống, "X" hoặc "O")
	Players     []Player   `json:"players"`     // Danh sách người chơi hiện tại (chỉ gồm ID, Name, Mark)
	CurrentTurn string     `json:"currentTurn"` // ID của người chơi có lượt đi hiện tại
	Winner      string     `json:"winner"`      // ID của người chơi thắng cuộc (hoặc "" nếu chưa có ai thắng)
}

// InitMessage: Tin nhắn khởi tạo gửi từ frontend khi kết nối.
type InitMessage struct {
	Type   string `json:"type"`   // Phải là "init"
	Player Player `json:"player"` // Thông tin người chơi (ID, Name) gửi từ frontend
}

// MoveMessage: Tin nhắn chứa nước đi gửi từ frontend.
type MoveMessage struct {
	Type string `json:"type"` // Phải là "move"
	Move Move   `json:"move"` // Tọa độ nước đi (X, Y)
}

// PlayerJoinedOrLeaveMessages: Tin nhắn thông báo có người chơi vào/ra.
// Gửi tới tất cả người chơi còn lại.
type PlayerJoinedOrLeaveMessages struct {
	Type        string   `json:"type"`        // Phải là "playerJoinedOrLeave"
	Message     []string `json:"message"`     // Nội dung thông báo (ví dụ: "Player A joined")
	TotalPlayer int      `json:"totalPlayer"` // Tổng số người chơi hiện tại
}

// GenericMessage: Dùng để xác định loại tin nhắn đơn giản như "reset".
type GenericMessage struct {
	Type string `json:"type"` // Loại tin nhắn ("reset", ...)
}

// --- Biến toàn cục ---
// Sử dụng map và slice toàn cục để lưu trạng thái game.
// Cần được bảo vệ bởi Mutex khi truy cập/thay đổi từ nhiều goroutine.
var (
	players     = make(map[string]*Player)     // Map lưu trữ người chơi, key là Player ID
	board       = make([][]string, BOARD_SIZE) // Mảng 2 chiều lưu trạng thái bàn cờ
	currentTurn string                         // ID người chơi có lượt đi hiện tại
	winner      string                         // ID người chơi thắng cuộc
	gameActive  bool                           // Cờ báo hiệu game đang diễn ra hay không
	mu          sync.Mutex                     // Mutex để bảo vệ các biến toàn cục (players, board, currentTurn, winner, gameActive)
	upgrader    = websocket.Upgrader{          // Cấu hình để nâng cấp kết nối HTTP lên WebSocket
		// CheckOrigin kiểm tra nguồn gốc yêu cầu (ở đây cho phép tất cả cho môi trường dev)
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// initBoard: Khởi tạo (hoặc reset) bàn cờ về trạng thái trống.
// Cần được gọi bên trong một khu vực đã khóa Mutex hoặc lúc khởi tạo server.
func initBoard() {
	log.Println("Initializing board...")
	for i := 0; i < BOARD_SIZE; i++ {
		// Tạo các hàng của bàn cờ
		board[i] = make([]string, BOARD_SIZE)
		// Các ô mặc định là "" (chuỗi rỗng)
	}
	currentTurn = ""   // Chưa có ai có lượt
	winner = ""        // Chưa có người thắng
	gameActive = false // Game chưa bắt đầu
}

// getPlayerList: Lấy danh sách người chơi dưới dạng slice để gửi cho frontend.
// Chỉ lấy các thông tin cần thiết (ID, Name, Mark), bỏ qua Conn.
// Cần được gọi bên trong một khu vực đã khóa Mutex.
func getPlayerList() []Player {
	playerList := make([]Player, 0, len(players)) // Tạo slice với capacity ban đầu
	for _, p := range players {
		// Tạo một bản sao Player chỉ với các trường cần thiết cho JSON
		playerList = append(playerList, Player{ID: p.ID, Name: p.Name, Mark: p.Mark})
	}
	return playerList
}

// assignMarksAndStart: Gán quân cờ (X, O) cho người chơi và bắt đầu game nếu đủ người.
// Logic đơn giản: người đầu tiên là X, người thứ hai là O.
// Cần được gọi bên trong một khu vực đã khóa Mutex.
func assignMarksAndStart() {
	log.Println("Assigning marks and checking start condition...")
	// Lấy danh sách con trỏ Player từ map
	playerList := make([]*Player, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, p)
	}

	// Reset quân cờ của tất cả người chơi trước khi gán lại
	for _, p := range playerList {
		p.Mark = ""
	}

	var playerX *Player // Lưu lại người chơi X để xác định lượt đi đầu
	// Gán X cho người chơi đầu tiên (nếu có)
	if len(playerList) >= 1 {
		playerList[0].Mark = "X"
		playerX = playerList[0]
		log.Printf("Assigned Mark 'X' to %s", playerList[0].ID)
	}
	// Gán O cho người chơi thứ hai (nếu có) và bắt đầu game
	if len(playerList) >= 2 {
		playerList[1].Mark = "O"
		log.Printf("Assigned Mark 'O' to %s", playerList[1].ID)
		if playerX != nil {
			currentTurn = playerX.ID // Người chơi X đi trước
			winner = ""              // Đảm bảo chưa có người thắng
			gameActive = true        // Đánh dấu game đã bắt đầu
			log.Printf("Game started. Turn: %s (%s)", currentTurn, playerX.Name)
		} else {
			// Trường hợp này không nên xảy ra nếu logic đúng
			currentTurn = ""
			gameActive = false
			log.Println("Error: Player X not found after assigning marks.")
		}
	} else {
		// Không đủ người chơi
		currentTurn = ""
		gameActive = false
		log.Println("Not enough players to start.")
	}
	// Frontend sẽ nhận được thông tin Mark và CurrentTurn qua tin nhắn gameState tiếp theo.
}

// resetGame: Reset lại toàn bộ trạng thái game.
// Thường được gọi khi có yêu cầu "reset" từ frontend.
// Hàm này tự quản lý việc khóa Mutex.
func resetGame() {
	mu.Lock()         // Khóa Mutex khi bắt đầu hàm
	defer mu.Unlock() // Đảm bảo Mutex được mở khóa khi hàm kết thúc
	log.Println("Resetting game...")
	initBoard()           // Reset bàn cờ, lượt đi, người thắng
	assignMarksAndStart() // Gán lại quân cờ và kiểm tra bắt đầu game
	broadcastGameState()  // Gửi trạng thái mới cho tất cả người chơi
	// Frontend nhận gameState mới và vẽ lại bàn cờ, cập nhật trạng thái.
}

// broadcastGameState: Gửi trạng thái game hiện tại (GameState) cho tất cả người chơi đang kết nối.
// Được gọi sau mỗi thay đổi trạng thái quan trọng (nước đi, reset, join, leave).
// Cần được gọi bên trong một khu vực đã khóa Mutex.
func broadcastGameState() {
	// Giả định rằng Mutex đã được khóa bởi hàm gọi nó.

	playerList := getPlayerList() // Lấy danh sách người chơi (đã được đơn giản hóa)

	// Tạo đối tượng GameState
	gameState := GameState{
		Type:        "gameState", // Loại tin nhắn để frontend nhận biết
		Board:       board,       // Trạng thái bàn cờ hiện tại
		Players:     playerList,  // Danh sách người chơi
		CurrentTurn: currentTurn, // Lượt đi của ai
		Winner:      winner,      // Ai thắng (nếu có)
	}

	// Chuyển đổi GameState thành JSON
	stateJSON, err := json.Marshal(gameState)
	if err != nil {
		log.Println("broadcastGameState JSON Marshal error:", err)
		return // Không gửi nếu có lỗi
	}

	log.Printf("Broadcasting gameState: Turn=%s, Winner=%s, Players=%d", currentTurn, winner, len(playerList))

	// Gửi JSON đến từng người chơi trong map `players`
	for id, player := range players {
		// Đặt deadline để tránh goroutine bị treo nếu client không phản hồi
		player.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		// Gửi tin nhắn dạng TextMessage chứa JSON
		err := player.Conn.WriteMessage(websocket.TextMessage, stateJSON)
		// Xóa deadline sau khi gửi xong
		player.Conn.SetWriteDeadline(time.Time{})
		if err != nil {
			log.Printf("Failed to send gameState to player %s (%s): %v", id, player.Name, err)
			// Lưu ý: Không gọi handlePlayerDisconnect trực tiếp ở đây để tránh deadlock.
			// Lỗi ghi thường sẽ được phát hiện bởi vòng lặp đọc hoặc ping.
		}
	}
	// Frontend nhận tin nhắn "gameState", parse JSON và cập nhật giao diện:
	// - Vẽ lại bàn cờ (board)
	// - Cập nhật danh sách người chơi (players)
	// - Hiển thị lượt đi của ai (currentTurn)
	// - Hiển thị người thắng cuộc (winner)
}

// notifyPlayerJoinedOrLeave: Thông báo cho tất cả người chơi khi có ai đó vào hoặc rời game.
// Cần được gọi bên trong một khu vực đã khóa Mutex.
func notifyPlayerJoinedOrLeave(playerID, playerName, action string) {
	// Giả định rằng Mutex đã được khóa bởi hàm gọi nó.

	// Tạo nội dung thông báo
	messageText := fmt.Sprintf("%s (%s) %s the game.", playerName, playerID, action) // Ví dụ: "Alice (player123) joined the game."
	log.Println("Notify:", messageText)

	// Tạo đối tượng thông báo
	notification := PlayerJoinedOrLeaveMessages{
		Type:        "playerJoinedOrLeave", // Loại tin nhắn để frontend nhận biết
		Message:     []string{messageText}, // Mảng chứa các thông báo (ở đây chỉ có 1)
		TotalPlayer: len(players),          // Số người chơi hiện tại
	}

	// Chuyển đổi thành JSON
	messageJSON, err := json.Marshal(notification)
	if err != nil {
		log.Println("notifyPlayerJoinedOrLeave JSON Marshal error:", err)
		return
	}

	// Gửi JSON đến tất cả người chơi còn lại
	for id, player := range players {
		// Không gửi thông báo "left" cho chính người vừa rời đi (nếu họ còn trong map tạm thời)
		if action == "left" && id == playerID {
			continue
		}
		// Đặt deadline và gửi tin nhắn
		player.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := player.Conn.WriteMessage(websocket.TextMessage, messageJSON)
		player.Conn.SetWriteDeadline(time.Time{})
		if err != nil {
			log.Printf("Failed to send notification to player %s (%s): %v", id, player.Name, err)
		}
	}
	// Frontend nhận tin nhắn "playerJoinedOrLeave", parse JSON và cập nhật:
	// - Hiển thị thông báo trong khu vực log/chat (message)
	// - Cập nhật số lượng người chơi hiển thị (totalPlayer)
}

// switchTurn: Chuyển lượt đi cho người chơi còn lại.
// Tìm người chơi có Mark ("X" hoặc "O") mà không phải là người có lượt hiện tại.
// Cần được gọi bên trong một khu vực đã khóa Mutex.
func switchTurn() {
	// Giả định rằng Mutex đã được khóa bởi hàm gọi nó.

	// Không chuyển lượt nếu game chưa active hoặc không đủ 2 người chơi
	if !gameActive || len(players) < 2 {
		log.Println("Cannot switch turn: Game not active or not enough players.")
		currentTurn = ""
		return
	}

	nextTurnPlayerID := ""
	// Duyệt qua map người chơi để tìm người chơi tiếp theo
	for _, p := range players {
		// Tìm người không phải lượt hiện tại VÀ có quân cờ (là người đang chơi)
		if p.ID != currentTurn && (p.Mark == "X" || p.Mark == "O") {
			nextTurnPlayerID = p.ID
			break // Tìm thấy người chơi kia rồi thì dừng
		}
	}

	if nextTurnPlayerID != "" {
		currentTurn = nextTurnPlayerID
		log.Printf("Switched turn to: %s", currentTurn)
	} else {
		// Trường hợp không tìm thấy người chơi kia (lỗi logic hoặc chỉ còn 1 người?)
		log.Printf("Could not switch turn. Current: %s. Players: %d. Resetting turn.", currentTurn, len(players))
		currentTurn = "" // Reset lượt đi để tránh lỗi
	}
	// Frontend sẽ nhận được lượt đi mới qua tin nhắn gameState tiếp theo.
}

// checkWin: Kiểm tra xem nước đi cuối cùng tại (x, y) có tạo thành dãy thắng không.
// Kiểm tra theo 4 hướng: ngang, dọc, chéo chính, chéo phụ.
// Cần được gọi bên trong một khu vực đã khóa Mutex (vì đọc `board`).
func checkWin(x, y int, mark string) bool {
	// Giả định rằng Mutex đã được khóa bởi hàm gọi nó.

	// Mảng chứa các vector hướng kiểm tra: {dx, dy}
	// {1, 0}: Ngang, {0, 1}: Dọc, {1, 1}: Chéo \, {1, -1}: Chéo /
	directions := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}

	for _, dir := range directions {
		count := 1 // Bắt đầu đếm từ quân cờ vừa đặt (là 1)
		// Kiểm tra theo hướng dương (ví dụ: sang phải, xuống dưới, ...)
		for i := 1; i < WIN_CONDITION; i++ {
			nx, ny := x+dir[0]*i, y+dir[1]*i
			// Kiểm tra nếu ra ngoài bàn cờ hoặc không phải quân cờ của người chơi hiện tại
			if nx < 0 || nx >= BOARD_SIZE || ny < 0 || ny >= BOARD_SIZE || board[ny][nx] != mark {
				break // Dừng kiểm tra hướng này
			}
			count++ // Tăng biến đếm
		}
		// Kiểm tra theo hướng âm (ví dụ: sang trái, lên trên, ...)
		for i := 1; i < WIN_CONDITION; i++ {
			nx, ny := x-dir[0]*i, y-dir[1]*i
			// Kiểm tra tương tự
			if nx < 0 || nx >= BOARD_SIZE || ny < 0 || ny >= BOARD_SIZE || board[ny][nx] != mark {
				break
			}
			count++
		}
		// Nếu đếm đủ số quân liên tiếp theo một hướng nào đó
		if count >= WIN_CONDITION {
			log.Printf("Win condition met for %s at (%d, %d) in direction {%d, %d}", mark, x, y, dir[0], dir[1])
			return true // Người chơi đã thắng
		}
	}
	return false // Chưa thắng
}

// handlePlayerDisconnect: Xử lý khi một người chơi ngắt kết nối.
// Được gọi bởi `defer` trong `HandleConnection` hoặc khi `pingPlayer` thất bại.
// Hàm này tự quản lý việc khóa Mutex.
func handlePlayerDisconnect(playerID string) {
	mu.Lock()         // Khóa Mutex khi bắt đầu xử lý
	defer mu.Unlock() // Đảm bảo Mutex được mở khóa khi kết thúc

	// Kiểm tra xem người chơi có còn trong map không (tránh xử lý trùng lặp)
	player, exists := players[playerID]
	if !exists {
		// mu.Unlock() // Mở khóa nếu không tìm thấy người chơi
		log.Printf("Player %s already disconnected or not found.", playerID)
		return
	}

	playerName := player.Name
	log.Printf("Handling disconnect for player %s (%s)...", playerID, playerName)
	player.Conn.Close()       // Đóng kết nối WebSocket
	delete(players, playerID) // Xóa người chơi khỏi map
	log.Printf("Player %s (%s) removed. Total players: %d", playerID, playerName, len(players))

	wasTurn := currentTurn == playerID // Lưu lại xem có phải lượt của người chơi này không
	wasActive := gameActive            // Lưu lại xem game có đang diễn ra không

	// --- Cập nhật trạng thái Game ---
	if wasActive { // Nếu game đang diễn ra
		if len(players) < 2 { // Nếu không đủ người chơi nữa
			log.Println("Game stopped due to insufficient players after disconnect.")
			gameActive = false // Dừng game
			currentTurn = ""   // Reset lượt
			winner = ""        // Reset người thắng
		} else if wasTurn { // Nếu là lượt của người vừa ngắt kết nối
			log.Printf("Player %s disconnected on their turn, switching.", playerID)
			switchTurn() // Chuyển lượt cho người còn lại
		}
		// Nếu không phải lượt của họ thì không cần đổi lượt
	}

	// Gán lại quân cờ nếu game đã dừng hoặc không active, hoặc chỉ còn < 2 người
	// Điều này đảm bảo người chơi còn lại (nếu có) sẽ là 'X' và sẵn sàng chờ người mới.
	if !gameActive || len(players) < 2 {
		log.Println("Re-assigning marks after disconnect.")
		assignMarksAndStart() // Gán lại X/O và kiểm tra xem có thể bắt đầu lại không
	}

	// --- Thông báo cho những người còn lại ---
	// Các hàm này cần được gọi khi Mutex đang được giữ
	broadcastGameState()                                    // Gửi trạng thái game mới nhất
	notifyPlayerJoinedOrLeave(playerID, playerName, "left") // Thông báo có người rời đi

	// Mutex sẽ được mở khóa bởi defer
	// Frontend nhận gameState và playerJoinedOrLeave, cập nhật giao diện.
}

// pingPlayer: Gửi tin nhắn Ping đến client định kỳ để giữ kết nối và phát hiện client chết.
// Chạy trong một goroutine riêng cho mỗi người chơi.
func pingPlayer(playerID string, conn *websocket.Conn) {
	// Tạo một ticker gửi tín hiệu sau mỗi 30 giây
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop() // Dừng ticker khi hàm kết thúc

	log.Printf("Ping routine started for %s", playerID)

	// Vòng lặp chạy theo ticker
	for range ticker.C {
		// Khóa Mutex ngắn hạn để kiểm tra sự tồn tại và gửi Ping
		mu.Lock()
		player, exists := players[playerID]
		// Kiểm tra xem người chơi có còn trong map không (có thể đã bị disconnect bởi lỗi đọc)
		if !exists {
			mu.Unlock()
			log.Printf("Stopping ping for disconnected player %s", playerID)
			return // Dừng goroutine ping nếu người chơi không còn
		}

		// Gửi tin nhắn điều khiển Ping (Control Message)
		// Đặt deadline cho việc gửi Ping
		err := player.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
		mu.Unlock() // Mở khóa ngay sau khi gửi xong

		if err != nil {
			// Nếu gửi Ping thất bại -> coi như client đã mất kết nối
			log.Printf("Ping failed for player %s: %v. Disconnecting.", playerID, err)
			// Gọi handlePlayerDisconnect để xử lý việc ngắt kết nối
			// Lưu ý: Gọi handlePlayerDisconnect từ đây có thể gây tranh chấp lock nếu handlePlayerDisconnect đang chạy.
			// Trong trường hợp phức tạp hơn, có thể cần cơ chế khác (ví dụ: gửi tín hiệu qua channel).
			// Ở đây, để đơn giản, gọi trực tiếp.
			handlePlayerDisconnect(playerID)
			return // Dừng goroutine ping
		}
		// log.Printf("Ping sent to %s", playerID) // Ghi log nếu cần debug
	}
	log.Printf("Ping routine stopped for %s", playerID)
}

// HandleConnection: Hàm chính xử lý một kết nối WebSocket mới.
// Được gọi cho mỗi client kết nối đến endpoint WebSocket.
func HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Nâng cấp kết nối HTTP thành WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return // Không xử lý nếu nâng cấp thất bại
	}
	// Log địa chỉ của client vừa kết nối
	log.Println("Client connected:", conn.RemoteAddr())

	var playerID string // Biến lưu ID người chơi, sẽ được gán sau khi nhận tin nhắn init

	// --- Defer Cleanup ---
	// Logic này sẽ được thực thi khi hàm HandleConnection kết thúc (dù là kết thúc bình thường hay do lỗi/panic)
	defer func() {
		log.Printf("Defer cleanup executing for connection %s (PlayerID: %s)", conn.RemoteAddr(), playerID)
		if playerID != "" {
			// Nếu playerID đã được gán (nghĩa là người chơi đã init thành công)
			// thì gọi handlePlayerDisconnect để dọn dẹp trạng thái game
			handlePlayerDisconnect(playerID)
		} else {
			// Nếu playerID rỗng (người chơi chưa init hoặc init lỗi)
			// thì chỉ cần đóng kết nối WebSocket là đủ
			log.Printf("Closing connection %s (player not fully initialized).", conn.RemoteAddr())
			conn.Close()
		}
	}()

	// --- Cấu hình kết nối WebSocket ---
	conn.SetReadLimit(512) // Giới hạn kích thước tin nhắn đọc được từ client
	// Đặt deadline ban đầu cho việc đọc tin nhắn (chờ tin nhắn init)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	// Thiết lập Pong Handler: khi nhận được tin nhắn Pong từ client (phản hồi Ping từ server),
	// gia hạn deadline đọc thêm 60 giây.
	conn.SetPongHandler(func(string) error {
		// log.Printf("Pong received from %s", playerID) // Ghi log nếu cần debug
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// --- Khởi tạo người chơi (Player Initialization) ---
	// Chờ đợi và đọc tin nhắn đầu tiên từ client, mong đợi đó là tin nhắn "init"
	log.Printf("Waiting for init message from %s...", conn.RemoteAddr())
	var initMsg InitMessage
	// Đọc và giải mã JSON từ kết nối vào struct InitMessage
	err = conn.ReadJSON(&initMsg)
	if err != nil {
		// Xử lý lỗi đọc hoặc giải mã JSON
		// Kiểm tra xem có phải lỗi do client đóng kết nối không
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("Init read error (client closed connection?) for %s: %v", conn.RemoteAddr(), err)
		} else {
			log.Printf("Failed to read/parse init message from %s: %v", conn.RemoteAddr(), err)
		}
		// Kết thúc hàm, defer sẽ đóng kết nối
		return
	}
	// Log thông tin nhận được từ tin nhắn init
	log.Printf("Received init message from %s: Type=%s, ID=%s, Name=%s", conn.RemoteAddr(), initMsg.Type, initMsg.Player.ID, initMsg.Player.Name)

	// Kiểm tra tính hợp lệ của tin nhắn init
	if initMsg.Type != "init" || initMsg.Player.ID == "" {
		log.Printf("Invalid init message type ('%s') or missing ID from %s.", initMsg.Type, conn.RemoteAddr())
		// Gửi lại tin nhắn lỗi cho client nếu init không hợp lệ
		conn.WriteJSON(map[string]string{"type": "error", "message": "Invalid initialization message"})
		// Kết thúc hàm, defer sẽ đóng kết nối
		return
	}

	// Gán playerID và playerName từ tin nhắn init
	playerID = initMsg.Player.ID
	playerName := initMsg.Player.Name
	if playerName == "" {
		// Đặt tên mặc định nếu client không gửi tên
		playerName = "Anon_" + playerID[:min(4, len(playerID))] // Lấy 4 ký tự đầu ID làm tên tạm
	}

	// --- Thêm người chơi vào Game (Locked section) ---
	mu.Lock() // Khóa Mutex trước khi thay đổi map `players` và trạng thái game
	// Kiểm tra xem ID người chơi này đã tồn tại chưa (tránh kết nối trùng lặp)
	if _, exists := players[playerID]; exists {
		log.Printf("Player %s (%s) attempted to connect again while already connected.", playerID, playerName)
		mu.Unlock() // Mở khóa trước khi gửi tin nhắn lỗi và return
		// Gửi lỗi cho kết nối *mới* này
		conn.WriteJSON(map[string]string{"type": "error", "message": "Player ID already connected"})
		// Quan trọng: Reset playerID về rỗng để defer không gọi handlePlayerDisconnect cho kết nối *mới* này.
		// Việc disconnect của kết nối *cũ* sẽ do goroutine của kết nối đó xử lý.
		playerID = ""
		// Kết thúc hàm cho kết nối *mới*, defer sẽ chỉ đóng conn này.
		return
	}

	// Tạo đối tượng Player mới
	newPlayer := &Player{
		ID:   playerID,
		Name: playerName,
		Conn: conn, // Lưu trữ kết nối WebSocket của người chơi
		Mark: "",   // Quân cờ sẽ được gán sau
	}
	// Thêm người chơi mới vào map `players`
	players[playerID] = newPlayer
	log.Printf("Player %s (%s) added to game. Total players: %d", playerID, playerName, len(players))

	// Khởi tạo bàn cờ nếu đây là người chơi đầu tiên (hoặc sau khi reset mà chưa có ai vào lại)
	if len(players) == 1 && !gameActive {
		log.Println("First player joined, initializing board.")
		initBoard()
	}

	// Gán quân cờ (X/O) và kiểm tra xem game có thể bắt đầu chưa
	assignMarksAndStart()

	// Gửi trạng thái game và thông báo có người mới vào cho TẤT CẢ người chơi (bao gồm cả người mới)
	broadcastGameState()
	notifyPlayerJoinedOrLeave(playerID, playerName, "joined")

	mu.Unlock() // Mở khóa Mutex sau khi hoàn tất cập nhật trạng thái và gửi thông báo ban đầu

	// --- Bắt đầu Goroutine Ping ---
	// Chạy một goroutine riêng để gửi Ping định kỳ cho client này
	go pingPlayer(playerID, conn)

	// --- Vòng lặp đọc tin nhắn (Message Read Loop) ---
	// Vòng lặp vô hạn để liên tục đọc tin nhắn từ client này
	for {
		// Gia hạn deadline đọc trước mỗi lần chờ đọc tin nhắn mới
		// Nếu không nhận được tin nhắn (hoặc Pong) trong 60s, ReadJSON sẽ báo lỗi
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var rawMsg json.RawMessage    // Đọc tin nhắn dưới dạng byte thô trước
		err := conn.ReadJSON(&rawMsg) // Đọc và giải mã JSON vào rawMsg
		if err != nil {
			// Xử lý lỗi đọc (thường là do client ngắt kết nối hoặc hết deadline)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error (client closed connection?) for player %s (%s): %v", playerID, playerName, err)
			} else {
				log.Printf("Read error for player %s (%s): %v", playerID, playerName, err)
			}
			// Thoát khỏi vòng lặp đọc, defer sẽ xử lý ngắt kết nối
			break
		}
		// log.Printf("Received raw message from %s", playerID) // Ghi log nếu cần debug

		// Xác định loại tin nhắn bằng cách giải mã một phần vào GenericMessage
		var genericMsg GenericMessage
		if err := json.Unmarshal(rawMsg, &genericMsg); err != nil {
			log.Printf("Error unmarshalling generic message from %s (%s): %v", playerID, playerName, err)
			continue // Bỏ qua tin nhắn không hợp lệ và chờ tin nhắn tiếp theo
		}
		// log.Printf("Received message type '%s' from %s", genericMsg.Type, playerID) // Ghi log nếu cần debug

		// --- Xử lý tin nhắn (Locked section) ---
		mu.Lock() // Khóa Mutex trước khi xử lý tin nhắn có thể thay đổi trạng thái game
		// log.Printf("Acquired lock for message type '%s' from %s", genericMsg.Type, playerID) // Ghi log nếu cần debug

		// Kiểm tra lại xem người chơi có còn tồn tại không
		// (Có thể đã bị disconnect bởi pingPlayer giữa lúc chờ lock)
		currentPlayer, exists := players[playerID]
		if !exists {
			log.Printf("Player %s (%s) disconnected before message '%s' could be processed.", playerID, playerName, genericMsg.Type)
			mu.Unlock() // Mở khóa và thoát vòng lặp
			break
		}

		// Xử lý dựa trên loại tin nhắn
		switch genericMsg.Type {
		case "move": // Nếu là tin nhắn nước đi
			var msg MoveMessage
			// Giải mã toàn bộ tin nhắn vào MoveMessage
			if err := json.Unmarshal(rawMsg, &msg); err != nil {
				log.Printf("Error unmarshalling move message from %s (%s): %v", playerID, playerName, err)
				// Thoát switch, unlock sẽ được gọi ở cuối
				break
			}

			// --- Xác thực nước đi ---
			log.Printf("Validating move (%d, %d) from %s (%s)...", msg.Move.X, msg.Move.Y, playerID, playerName)
			validMove := true
			if !gameActive {
				log.Printf("Move ignored from %s: Game not active.", playerID)
				validMove = false
			} else if currentTurn != playerID {
				log.Printf("Move ignored from %s: Not their turn (Current: %s).", playerID, currentTurn)
				validMove = false
			} else if winner != "" {
				log.Printf("Move ignored from %s: Game already won by %s.", playerID, winner)
				validMove = false
			} else if msg.Move.Y < 0 || msg.Move.Y >= BOARD_SIZE || msg.Move.X < 0 || msg.Move.X >= BOARD_SIZE {
				log.Printf("Move ignored from %s: Out of bounds (%d, %d).", playerID, msg.Move.X, msg.Move.Y)
				validMove = false
			} else if board[msg.Move.Y][msg.Move.X] != "" {
				log.Printf("Move ignored from %s: Cell (%d, %d) already taken by '%s'.", playerID, msg.Move.X, msg.Move.Y, board[msg.Move.Y][msg.Move.X])
				validMove = false
			}

			// --- Xử lý nước đi hợp lệ ---
			if validMove {
				playerMark := currentPlayer.Mark           // Lấy quân cờ của người chơi
				board[msg.Move.Y][msg.Move.X] = playerMark // Cập nhật bàn cờ
				log.Printf("Player %s (%s) placed '%s' at (%d, %d)", playerID, currentPlayer.Name, playerMark, msg.Move.X, msg.Move.Y)

				// Kiểm tra thắng thua sau nước đi
				if checkWin(msg.Move.X, msg.Move.Y, playerMark) {
					winner = playerID  // Gán người thắng
					gameActive = false // Dừng game
					log.Printf("Player %s (%s) won!", playerID, currentPlayer.Name)
					broadcastGameState() // Gửi trạng thái cuối cùng (có người thắng)
				} else {
					// Nếu chưa thắng, chuyển lượt
					switchTurn()
					broadcastGameState() // Gửi trạng thái mới (lượt đi mới)
				}
				// Frontend nhận gameState, cập nhật bàn cờ, lượt đi, hoặc trạng thái thắng.
			}
			// Nếu nước đi không hợp lệ, không làm gì cả, chỉ ghi log.

		case "reset": // Nếu là tin nhắn yêu cầu reset game
			log.Printf("Player %s (%s) requested reset.", playerID, playerName)
			// Mở khóa Mutex TRƯỚC KHI gọi resetGame (vì resetGame tự khóa)
			mu.Unlock()
			resetGame() // Gọi hàm reset (hàm này sẽ khóa, xử lý, mở khóa, và broadcast)
			// Quan trọng: Dùng `continue` để nhảy đến lần lặp tiếp theo của vòng `for`,
			// bỏ qua việc mở khóa `mu.Unlock()` ở cuối vòng lặp này (vì đã mở khóa ở trên).
			continue

		default: // Loại tin nhắn không xác định
			log.Printf("Received unknown message type '%s' from %s (%s)", genericMsg.Type, playerID, playerName)
		}

		// log.Printf("Releasing lock for message type '%s' from %s", genericMsg.Type, playerID) // Ghi log nếu cần debug
		mu.Unlock() // Mở khóa Mutex sau khi xử lý xong một tin nhắn (trừ trường hợp 'reset')
	}
	// Kết thúc vòng lặp đọc (do lỗi hoặc client đóng kết nối)
	log.Printf("Message read loop stopped for %s (%s)", playerID, playerName)
	// defer sẽ được thực thi để dọn dẹp
}

// Helper function (not strictly necessary but good practice)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GameLoop is not used in this WebSocket model
// func GameLoop() { ... }
