package nsp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/nsp/pb"
	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/options"
	"github.com/openimsdk/open-im-server/v3/pkg/common/config"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultTimeout = 5 * time.Second

// Nsp delivers offline pushes through community_user's SendChatNotification gRPC,
// which resolves chat user IDs to device tokens and sends via FCM/APNs.
type Nsp struct {
	client  pb.AuthServiceClient
	timeout time.Duration
}

type pushData struct {
	ClientMsgID string `json:"clientMsgID,omitempty"`
	Ex          string `json:"ex,omitempty"`
}

func NewClient(pushConf *config.Push) (*Nsp, error) {
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
	return &Nsp{client: pb.NewAuthServiceClient(conn), timeout: timeout}, nil
}

func (n *Nsp) Push(ctx context.Context, userIDs []string, title, content string, opts *options.Opts) error {
	if len(userIDs) == 0 {
		return nil
	}
	data := pushData{Ex: opts.Ex}
	if opts.Signal != nil {
		data.ClientMsgID = opts.Signal.ClientMsgID
	}
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
