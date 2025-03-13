package packet

import "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"

func HandleDisconnectPacket(session *database.SessionData) {
	databaseStore := database.NewDatabaseStore()
	if session.TempSession {
		session.RemoveAllSubscriptions()
		databaseStore.DeleteSession(session.ClientID)
		return
	}
	databaseStore.SaveSession(session)
}
