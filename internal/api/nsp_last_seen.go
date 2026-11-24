package api

import (
	"github.com/gin-gonic/gin"
	"github.com/openimsdk/open-im-server/v3/pkg/common/storage/cache/redis"
	"github.com/openimsdk/tools/apiresp"
	"github.com/openimsdk/tools/errs"
	redisv9 "github.com/redis/go-redis/v9"
)

// NSP: POST /user/get_users_last_seen — хувь чатын толгойд "сүүлд … идэвхтэй" харуулахад.
// Хэрэглэгчийн өөрийн IM token-оор дуудагдана (GinParseToken). Онлайн төлөв (/user/get_users_status)
// аль хэдийн нэвтэрсэн бүх хэрэглэгчид нээлттэй тул энэ нь ижил хүрээтэй.
//
// Request:  {"userIDs": ["u1", "u2"]}
// Response: {"lastSeenList": [{"userID":"u1","online":false,"lastSeen":1795072263}]}  // lastSeen=unix сек, 0=мэдэгдэхгүй

const maxLastSeenUsers = 100

type lastSeenApi struct {
	rdb redisv9.UniversalClient
}

type getUsersLastSeenReq struct {
	UserIDs []string `json:"userIDs" binding:"required"`
}

type getUsersLastSeenResp struct {
	LastSeenList []redis.LastSeenInfo `json:"lastSeenList"`
}

func (l *lastSeenApi) GetUsersLastSeen(c *gin.Context) {
	var req getUsersLastSeenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.GinError(c, errs.ErrArgs.WithDetail(err.Error()).Wrap())
		return
	}
	if len(req.UserIDs) > maxLastSeenUsers {
		apiresp.GinError(c, errs.ErrArgs.WrapMsg("too many userIDs"))
		return
	}
	list, err := redis.GetUsersLastSeen(c, l.rdb, req.UserIDs)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, &getUsersLastSeenResp{LastSeenList: list})
}
