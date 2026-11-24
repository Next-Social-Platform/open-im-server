package cachekey

import "time"

// NSP: хэрэглэгч хамгийн сүүлд онлайн байсан цаг (unix секунд).
// "ONLINE:" угтвартай БҮҮ эхлүүл — GetAllOnlineUsers "ONLINE:*"-ээр scan хийдэг.
const (
	LastSeenKey    = "LAST_SEEN:"
	LastSeenExpire = time.Hour * 24 * 90
)

func GetLastSeenKey(userID string) string {
	return LastSeenKey + userID
}
