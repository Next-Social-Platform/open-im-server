package nsp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/nsp/pb"
	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/options"
	"github.com/openimsdk/open-im-server/v3/pkg/common/config"
	"github.com/openimsdk/open-im-server/v3/pkg/common/storage/cache"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultTimeout = 5 * time.Second

// Nsp delivers offline pushes through community_user's SendChatNotification gRPC,
// which resolves chat user IDs to device tokens and sends via FCM/APNs.
type Nsp struct {
	client  pb.AuthServiceClient
	cache   cache.ThirdCache
	timeout time.Duration
}

type pushData struct {
	ClientMsgID string `json:"clientMsgID,omitempty"`
	Ex          string `json:"ex,omitempty"`
	// ContentType — community_user (NotificationService.sendToChatUsers) үүгээр
	// жинхэнэ чатын мессеж (insert ХИЙХГҮЙ, зөвхөн push) ба OpenIM-ийн дотоод
	// мэдэгдэл (friend request accept/reject гэх мэт, insert ХИЙХ ёстой)-ийг
	// ялгана (2026-11-04).
	ContentType int32 `json:"contentType,omitempty"`
	// Badge — хэрэглэгчийн app icon дээр харуулах уншаагүй мессежийн тоо (iOS aps.badge).
	// Хэрэглэгч бүрт өөр тул хэрэглэгч тус бүрээр тусад нь илгээнэ.
	Badge int `json:"badge,omitempty"`
}

func NewClient(pushConf *config.Push, cache cache.ThirdCache) (*Nsp, error) {
	if pushConf.Nsp.Address == "" {
		return nil, errs.New("no nsp push address config").Wrap()
	}
	conn, err := grpc.NewClient(pushConf.Nsp.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errs.WrapMsg(err, "nsp push grpc dial failed", "address", pushConf.Nsp.Address)
	}
	timeout := defaultTimeout
	if pushConf.Nsp.Timeout > 0 {
		timeout = time.Duration(pushConf.Nsp.Timeout) * time.Second
	}
	return &Nsp{client: pb.NewAuthServiceClient(conn), cache: cache, timeout: timeout}, nil
}

func (n *Nsp) Push(ctx context.Context, userIDs []string, title, content string, opts *options.Opts) error {
	if len(userIDs) == 0 {
		return nil
	}
	data := pushData{Ex: opts.Ex, ContentType: opts.ContentType}
	if opts.Signal != nil {
		data.ClientMsgID = opts.Signal.ClientMsgID
	}
	if n.cache == nil {
		return n.send(ctx, userIDs, title, content, data)
	}
	var firstErr error
	for _, userID := range userIDs {
		userData := data
		userData.Badge = n.badge(ctx, userID, opts.IOSBadgeCount)
		if err := n.send(ctx, []string{userID}, title, content, userData); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// badge FCM pusher-тэй ижил дүрмээр badge-ийг тооцно: IOSBadgeCount үед нэмэгдүүлнэ,
// үгүй бол одоогийн утгыг (байхгүй бол 1) авна.
func (n *Nsp) badge(ctx context.Context, userID string, incr bool) int {
	if incr {
		count, err := n.cache.IncrUserBadgeUnreadCountSum(ctx, userID)
		if err != nil {
			log.ZWarn(ctx, "nsp push incr badge failed", err, "userID", userID)
			return 0
		}
		return count
	}
	count, err := n.cache.GetUserBadgeUnreadCountSum(ctx, userID)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.ZWarn(ctx, "nsp push get badge failed", err, "userID", userID)
		return 0
	}
	if count == 0 {
		return 1
	}
	return count
}

func (n *Nsp) send(ctx context.Context, userIDs []string, title, content string, data pushData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return errs.WrapMsg(err, "marshal nsp push data failed")
	}
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()
	resp, err := n.client.SendChatNotification(ctx, &pb.SendChatNotificationRequest{
		ChatUserId: userIDs,
		Title:      title,
		Content:    content,
		Data:       string(b),
	})
	if err != nil {
		return errs.WrapMsg(err, "nsp SendChatNotification failed", "userIDs", userIDs)
	}
	log.ZDebug(ctx, "nsp push", "userIDs", userIDs, "success", resp.Success, "message", resp.Message)
	return nil
}
