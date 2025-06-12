package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MohitSilwal16/PKr-Server/db"
	"github.com/MohitSilwal16/PKr-Server/models"

	"github.com/gorilla/websocket"
)

const (
	PONG_WAIT_TIME = 5 * time.Minute
	PING_WAIT_TIME = (PONG_WAIT_TIME * 9) / 10
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all Origins
	},
}

var NotifyToPunchResponseMap = models.NotifyToPunchResponseMap{Map: map[string]models.NotifyToPunchResponse{}}

func readJSONMessage(conn *websocket.Conn, username string) {
	defer removeUserFromConnPool(conn, username)

	conn.SetReadDeadline(time.Now().Add(PONG_WAIT_TIME))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(PONG_WAIT_TIME))
		return nil
	})

	for {
		var msg models.WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read Error:", err)
			log.Printf("Description: Cannot Read Message Received from %v\n", conn.RemoteAddr().String())

			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("WebSocket Disconnected from Client Side")
				return
			}

			log.Println("Now Passing this Read Error Message to Client")
			err := conn.WriteJSON(models.WSMessage{MessageType: "Error", Message: "Error while Reading Message: " + err.Error()})
			if err != nil {
				log.Println("Error:", err)
				log.Println("Description: Could Not Write to", conn.RemoteAddr())
				log.Println("Source: readJSONMessage()")
				return
			}
			return
		}

		// TODO: Filter out MessageType like, NotifyToPunchResponse & etc.
		log.Printf("Message: %#v", msg)

		if msg.MessageType == "NotifyToPunchResponse" {
			res_json, err := json.Marshal(msg.Message)
			if err != nil {
				log.Println("Error while marshaling:", err)
				log.Println("Source: readJSONMessage()")
				continue
			}
			var res models.NotifyToPunchResponse
			if err := json.Unmarshal(res_json, &res); err != nil {
				log.Println("Error while unmarshaling:", err)
				log.Println("Source: readJSONMessage()")
				continue
			}
			NotifyToPunchResponseMap.Lock()
			NotifyToPunchResponseMap.Map[username+res.ListenerUsername] = res
			NotifyToPunchResponseMap.Unlock()
			log.Printf("Res: %#v", res)
		}
	}
}

func pingPongWriter(conn *websocket.Conn, username string) {
	ticker := time.NewTicker(PING_WAIT_TIME)
	for {
		<-ticker.C
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Println("No response of Ping from", username)
			return
		}
	}
}

func ServerWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error:", err)
		log.Println("Description: Cannot Upgrade HTTP Connection to WebSocket")
		return
	}
	query := r.URL.Query()
	username := query.Get("username")
	password := query.Get("password")
	fmt.Println()
	log.Printf("New Incoming Connection from %s with username=%s & password=%s\n", conn.RemoteAddr().String(), username, password)

	is_user_authenticated, err := db.AuthUser(username, password)
	if err != nil {
		log.Println("Error:", err)
		log.Println("Description: Could Not Authenticate User")
		log.Println("Source: ServeWS()")

		conn.WriteJSON(models.WSMessage{MessageType: "Error", Message: "Internal Server Error"})
		removeUserFromConnPool(conn, username)
		return
	}
	if !is_user_authenticated {
		conn.WriteJSON(models.WSMessage{MessageType: "Error", Message: "User Not Authenticated"})
		removeUserFromConnPool(conn, username)
		return
	}

	addUserToConnPool(conn, username)
	go readJSONMessage(conn, username)
	go pingPongWriter(conn, username)
}
