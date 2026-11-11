package options

// Opts opts.
type Opts struct {
	Signal        *Signal
	IOSPushSound  string
	IOSBadgeCount bool
	Ex            string
	// ContentType нь msg.ContentType-г (sdkws/protocol-ийн тоон код, жиш нь энгийн
	// текст мессеж 101, эсвэл friend-application-ийн мэдэгдэл 1200-1299 хооронд)
	// доош дамжуулна — community_user (nsp push)-д "энэ бол жинхэнэ чатын мессеж
	// үү, эсвэл OpenIM-ийн дотоод мэдэгдэл (friend request зэрэг) үү" гэдгийг
	// ялгахад ашиглагдана (2026-11-04).
	ContentType int32
}

// Signal message id.
type Signal struct {
	ClientMsgID string
}
