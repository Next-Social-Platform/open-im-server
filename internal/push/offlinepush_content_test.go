package push

import (
	"context"
	"strings"
	"testing"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/sdkws"
)

func TestPushPreview(t *testing.T) {
	for _, tc := range []struct {
		contentType int32
		content     string
		want        string
	}{
		{constant.Text, `{"content":"Сайн уу?\nЮу байна"}`, "Сайн уу? Юу байна"},
		{constant.AtText, `{"text":"@Бат маргааш уулзъя"}`, "@Бат маргааш уулзъя"},
		{constant.Quote, `{"text":"тийм ээ"}`, "тийм ээ"},
		{constant.Picture, `{"sourcePicture":{}}`, "[Зураг]"},
		{constant.Voice, `{}`, "[Дуут мессеж]"},
		{constant.Video, `{}`, "[Видео]"},
		{constant.File, `{"fileName":"тайлан.pdf"}`, "[Файл] тайлан.pdf"},
		{constant.File, `{}`, "[Файл]"},
		{constant.Location, `{"description":"Сүхбаатарын талбай"}`, "[Байршил] Сүхбаатарын талбай"},
		{constant.Text, `not json`, "Танд шинэ мессеж ирлээ"},
		{constant.Revoke, `{}`, "Танд шинэ мессеж ирлээ"},
	} {
		got := pushPreview(&sdkws.MsgData{ContentType: tc.contentType, Content: []byte(tc.content)})
		if got != tc.want {
			t.Errorf("contentType %d %s: got %q, want %q", tc.contentType, tc.content, got, tc.want)
		}
	}

	long := strings.Repeat("я", pushPreviewMaxRunes+10)
	got := pushPreview(&sdkws.MsgData{ContentType: constant.Text, Content: []byte(`{"content":"` + long + `"}`)})
	if want := strings.Repeat("я", pushPreviewMaxRunes) + "…"; got != want {
		t.Errorf("long text: got %d runes, want %d", len([]rune(got)), len([]rune(want)))
	}
}

func TestFillOfflinePushInfo(t *testing.T) {
	c := &ConsumerHandler{}
	ctx := context.Background()

	msg := &sdkws.MsgData{SenderNickname: "Болд", ContentType: constant.Text, Content: []byte(`{"content":"Сайн уу"}`)}
	c.fillOfflinePushInfo(ctx, msg)
	if title, content, _, _ := getOfflinePushInfos(msg); title != "Болд" || content != "Сайн уу" {
		t.Errorf("single chat: got %q / %q", title, content)
	}

	// Texts the sender set are kept; only the missing one is filled.
	original := &sdkws.OfflinePushInfo{Title: "Захиргаа", Ex: "x"}
	msg = &sdkws.MsgData{SenderNickname: "Болд", ContentType: constant.Picture, OfflinePushInfo: original}
	c.fillOfflinePushInfo(ctx, msg)
	if title, content, opts, _ := getOfflinePushInfos(msg); title != "Захиргаа" || content != "[Зураг]" || opts.Ex != "x" {
		t.Errorf("sender texts: got %q / %q / %q", title, content, opts.Ex)
	}
	if original.Desc != "" {
		t.Error("the message's original offlinePushInfo was modified")
	}

	// Notifications keep the texts from notification.yml.
	msg = &sdkws.MsgData{ContentType: constant.FriendApplicationNotification}
	c.fillOfflinePushInfo(ctx, msg)
	if msg.OfflinePushInfo != nil {
		t.Error("notification push info was filled")
	}
}
