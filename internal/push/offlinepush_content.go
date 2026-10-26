package push

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/options"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/sdkws"
	"github.com/openimsdk/tools/log"
	"google.golang.org/protobuf/proto"
)

// Mongolian offline push texts, used when the sender did not set offlinePushInfo.
const (
	pushNewMsgTitle   = "Шинэ мессеж"
	pushNewMsgContent = "Танд шинэ мессеж ирлээ"
	pushCallContent   = "Танд дуудлага ирлээ"
	// pushPreviewMaxRunes keeps the push body short; FCM/APNs payloads are limited to 4KB.
	pushPreviewMaxRunes = 100
)

var pushContentTypeText = map[int32]string{
	constant.Picture:  "[Зураг]",
	constant.Voice:    "[Дуут мессеж]",
	constant.Video:    "[Видео]",
	constant.File:     "[Файл]",
	constant.Merger:   "[Чатын түүх]",
	constant.Card:     "[Нэрийн хуудас]",
	constant.Location: "[Байршил]",
	constant.Custom:   "[Мессеж]",
}

// fillOfflinePushInfo sets a readable Mongolian title and body on msg for the offline push when the
// sender left them empty: the sender (single chat) or group name (group chat) as the title, and a
// preview of the message as the body. Notifications keep the texts of notification.yml.
func (c *ConsumerHandler) fillOfflinePushInfo(ctx context.Context, msg *sdkws.MsgData) {
	if msg.ContentType >= constant.NotificationBegin && msg.ContentType <= constant.NotificationEnd {
		return
	}
	info := &sdkws.OfflinePushInfo{}
	if msg.OfflinePushInfo != nil {
		info = proto.Clone(msg.OfflinePushInfo).(*sdkws.OfflinePushInfo)
	}
	if info.Title != "" && info.Desc != "" {
		return
	}
	preview := pushPreview(msg)
	title := msg.SenderNickname
	content := preview
	if msg.GroupID != "" {
		title = c.pushGroupName(ctx, msg.GroupID)
		if msg.SenderNickname != "" {
			content = msg.SenderNickname + ": " + preview
		}
	}
	if title == "" {
		title = pushNewMsgTitle
	}
	if info.Title == "" {
		info.Title = title
	}
	if info.Desc == "" {
		info.Desc = content
	}
	msg.OfflinePushInfo = info
}

func (c *ConsumerHandler) pushGroupName(ctx context.Context, groupID string) string {
	group, err := c.groupLocalCache.GetGroupInfo(ctx, groupID)
	if err != nil {
		log.ZWarn(ctx, "get group info for offline push failed", err, "groupID", groupID)
		return ""
	}
	return group.GroupName
}

// pushPreview is the message as one line of text: the text itself for text messages, a label for media.
func pushPreview(msg *sdkws.MsgData) string {
	var elem struct {
		Content     string `json:"content"`
		Text        string `json:"text"`
		FileName    string `json:"fileName"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(msg.Content, &elem)
	var text string
	switch msg.ContentType {
	case constant.Text, constant.MarkdownText:
		text = elem.Content
	case constant.AtText, constant.Quote, constant.AdvancedText:
		text = elem.Text
	case constant.File:
		if elem.FileName != "" {
			text = pushContentTypeText[constant.File] + " " + elem.FileName
		}
	case constant.Merger:
		if elem.Title != "" {
			text = pushContentTypeText[constant.Merger] + " " + elem.Title
		}
	case constant.Location:
		if elem.Description != "" {
			text = pushContentTypeText[constant.Location] + " " + elem.Description
		}
	}
	if text = strings.Join(strings.Fields(text), " "); text == "" {
		if label, ok := pushContentTypeText[msg.ContentType]; ok {
			return label
		}
		return pushNewMsgContent
	}
	if r := []rune(text); len(r) > pushPreviewMaxRunes {
		return string(r[:pushPreviewMaxRunes]) + "…"
	}
	return text
}

// getOfflinePushInfos returns the title, body and options of the offline push of msg.
func getOfflinePushInfos(msg *sdkws.MsgData) (title, content string, opts *options.Opts, err error) {
	opts = &options.Opts{Signal: &options.Signal{ClientMsgID: msg.ClientMsgID}}
	if msg.OfflinePushInfo != nil {
		opts.IOSBadgeCount = msg.OfflinePushInfo.IOSBadgeCount
		opts.IOSPushSound = msg.OfflinePushInfo.IOSPushSound
		opts.Ex = msg.OfflinePushInfo.Ex
		title = msg.OfflinePushInfo.Title
		content = msg.OfflinePushInfo.Desc
	}
	if title == "" {
		switch msg.ContentType {
		case constant.SignalingNotification:
			title = pushCallContent
		default:
			title = pushNewMsgTitle
		}
	}
	if content == "" {
		content = title
	}
	return
}
