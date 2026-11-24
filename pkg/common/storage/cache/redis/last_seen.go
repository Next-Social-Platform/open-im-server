package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/openimsdk/open-im-server/v3/pkg/common/storage/cache/cachekey"
	"github.com/openimsdk/tools/errs"
	"github.com/redis/go-redis/v9"
)

// NSP: "сүүлд онлайн байсан" цаг.
//
// msggateway онлайн хэрэглэгчийг OnlineExpire/3 (10 мин) тутамд сунгаж, офлайн болоход тэр
// даруй SetUserOnline дууддаг. Тэр бүрт энд одоогийн цагийг бичнэ — офлайн болох мөчийн цаг
// яг зөв, gateway гэнэт унасан тохиолдолд хамгийн сүүлийн сунгалтын цаг (≤10 мин зөрүү) үлдэнэ.

func setLastSeen(ctx context.Context, rdb redis.UniversalClient, userID string, now time.Time) error {
	return errs.Wrap(rdb.Set(ctx, cachekey.GetLastSeenKey(userID), now.Unix(), cachekey.LastSeenExpire).Err())
}

// LastSeenInfo — нэг хэрэглэгчийн төлөв. LastSeen=0 бол мэдээлэл байхгүй (90 хоногоос өмнө эсвэл хэзээ ч).
type LastSeenInfo struct {
	UserID   string `json:"userID"`
	Online   bool   `json:"online"`
	LastSeen int64  `json:"lastSeen"`
}

// GetUsersLastSeen — онлайн эсэх (ONLINE:<id> zset-ийн хугацаа дуусаагүй гишүүн) болон сүүлд
// онлайн байсан цагийг нэг pipeline-аар уншина.
func GetUsersLastSeen(ctx context.Context, rdb redis.UniversalClient, userIDs []string) ([]LastSeenInfo, error) {
	if len(userIDs) == 0 {
		return []LastSeenInfo{}, nil
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	pipe := rdb.Pipeline()
	onlineCmds := make([]*redis.IntCmd, len(userIDs))
	seenCmds := make([]*redis.StringCmd, len(userIDs))
	for i, id := range userIDs {
		onlineCmds[i] = pipe.ZCount(ctx, cachekey.GetOnlineKey(id), now, "+inf")
		seenCmds[i] = pipe.Get(ctx, cachekey.GetLastSeenKey(id))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, errs.Wrap(err)
	}
	res := make([]LastSeenInfo, len(userIDs))
	for i, id := range userIDs {
		info := LastSeenInfo{UserID: id}
		if n, err := onlineCmds[i].Result(); err == nil {
			info.Online = n > 0
		}
		if v, err := seenCmds[i].Int64(); err == nil {
			info.LastSeen = v
		}
		res[i] = info
	}
	return res, nil
}
