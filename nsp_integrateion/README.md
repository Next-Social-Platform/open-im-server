# NSP openim-server

Энэ repo нь албан ёсны [openimsdk/open-im-server](https://github.com/openimsdk/open-im-server)-ийн
**`v3.8.3-patch.16`** tag-аас салгасан NSP-ийн хувилбар юм. NSP-д зориулсан бүх өөрчлөлт **`nsp-chat`** branch дээр байгаа.

| | |
|---|---|
| Repo (origin) | `git@github.com:Next-Social-Platform/open-im-server.git` |
| Upstream | `https://github.com/openimsdk/open-im-server.git` |
| Үндсэн хувилбар | `v3.8.3-patch.16` |
| Ажлын branch | `nsp-chat` (PR-ийг `main` руу биш, энэ branch руу хийнэ) |
| Docker image | `repo.nextsocial.mn/nsp-openim-server` |

Холбоотой төслүүд:

- **nsp-chat-server**: webhook-уудыг хүлээн авч, community_organization-оос зөвшөөрөл асууна.
- **community_user**: offline push-ийг FCM/APNs-ээр илгээнэ (`SendChatNotification` gRPC).
- **community_organization**: мессеж, найз, group-ийн зөвшөөрлийн логик.

---

## 1. Branch хэрхэн үүссэн бэ

```bash
git remote add upstream https://github.com/openimsdk/open-im-server.git
git fetch upstream --tags
git checkout v3.8.3-patch.16 -b nsp-chat
git push -u origin nsp-chat
```

`main` branch нь upstream-ийнхтэй ижил хэвээр байгаа. Тэнд NSP-ийн өөрчлөлт хийхгүй.

### Upstream-ийн шинэ хувилбар руу шилжих

Жишээ нь `v3.8.3-patch.17` гарсан бол:

```bash
git fetch upstream --tags
git checkout nsp-chat
git rebase v3.8.3-patch.17        # эсвэл: git merge v3.8.3-patch.17
```

- Зөрчил (conflict) ихэвчлэн `Dockerfile`, `config/*.yml`, `internal/push/` дээр гарна.
- Rebase хийсний дараа `go build ./...` болон `go test ./internal/push/...` ажиллуулж шалгана.
- `version/version` файл upstream-ийн хувилбарыг агуулдаг. Image-ийн tag энэ файлаас үүснэ (5-р хэсгийг үз).

---

## 2. NSP-ийн өөрчлөлтүүд

### 2.1 `nsp` offline push

Хэрэглэгч offline байх үед OpenIM push-ийг community_user-ийн `SendChatNotification` gRPC руу илгээнэ. Getui, FCM, JPush ашиглахгүй.

| Файл | Тайлбар |
|---|---|
| `internal/push/offlinepush/nsp/push.go` | gRPC client. `chatUserId` = OpenIM userID |
| `internal/push/offlinepush/nsp/pb/chat_notification.proto` | community_user-ийн `auth.proto`-ийн хэсэгчилсэн хуулбар |
| `internal/push/offlinepush/offlinepusher.go` | `enable: nsp` сонголт |
| `pkg/common/config/config.go` | `Push.Nsp.Address`, `Push.Nsp.Timeout` |
| `config/openim-push.yml` | `nsp:` хэсэг |

- Push-ийн `data` талбарт `{"clientMsgID":"...","ex":"..."}` гэсэн JSON явна.
- **Proto-г өөрчлөх бол:** community_user-ийн `auth.proto` дээрх `SendChatNotification` өөрчлөгдвөл энд байгаа хуулбарыг мөн шинэчилж, Go кодыг дахин үүсгэнэ. Package (`mn.nextsocial.user`) болон service-ийн нэр (`AuthService`) эхийнхтэй ижил байх ёстой, эс бөгөөс method олдохгүй.

  ```bash
  cd internal/push/offlinepush/nsp/pb
  protoc --go_out=. --go_opt=paths=source_relative \
         --go-grpc_out=. --go-grpc_opt=paths=source_relative chat_notification.proto
  ```

### 2.2 Push-ийн монгол текст

`internal/push/offlinepush_content.go`

- App `offlinePushInfo` өгөөгүй үед гарчиг, агуулгыг монголоор бөглөнө:
  - Ганцаарчилсан чат: гарчиг = илгээгчийн нэр, агуулга = мессеж.
  - Group чат: гарчиг = group-ийн нэр, агуулга = `Илгээгч: мессеж`.
  - Зураг, видео гэх мэт: `[Зураг]`, `[Видео]`, `[Файл] нэр.pdf` гэх мэт.
- Урт текстийг 100 тэмдэгтээр таслана.
- Notification-ий push текст (найзын хүсэлт гэх мэт) `config/notification.yml`-д монголоор байгаа.

### 2.3 Webhook-ууд (nsp-chat-server)

`config/webhooks.yml`-д дараах 6 webhook асаалттай. Бүгд nsp-chat-server-ийн admin-api (`:10009/nsp/...`) руу явна.

| Тохиргоо | nsp-chat-server endpoint |
|---|---|
| `beforeSendSingleMsg` | `/nsp/callbackBeforeSendSingleMsgCommand` |
| `beforeSendGroupMsg` | `/nsp/callbackBeforeSendGroupMsgCommand` |
| `beforeAddFriend` | `/nsp/callbackBeforeAddFriendCommand` |
| `beforeCreateGroup` | `/nsp/callbackBeforeCreateGroupCommand` |
| `beforeMemberJoinGroup` | `/nsp/callbackBeforeMembersJoinGroupCommand` |
| `beforeInviteUserToGroup` | `/nsp/callbackBeforeInviteJoinGroupCommand` |

nsp-chat-server-ийн өөрийн admin үйлдлүүд (group удирдлага, зарлал) operationID-д HMAC гарын үсэг тавьдаг. Тиймээс тэдгээр нь webhook шалгалтаас алгасагддаг. Энэ логик nsp-chat-server-ийн `internal/api/nsp/admin_operation.go`-д байгаа.

### 2.4 Build ба тохиргоо

| Файл | Тайлбар |
|---|---|
| `Dockerfile` | Go 1.25. Host-ийн Go module cache-ийг (`modcache` build context) ашигладаг тул build хийхэд сүлжээ бараг хэрэггүй |
| `deploy.sh` | Image build хийгээд Nexus руу push хийнэ |
| `config/` | **Image дотор хуулагддаг** тохиргоо. Prod үүнийг ашиглана |
| `config-docker/` | Local docker орчны тохиргоо. `docker-compose.override.yml` volume болгон холбоно |
| `docker-compose.override.yml` | Local орчинд openim-server-ийг эх кодоос build хийнэ |
| `.env` | MinIO image-ийг `quay.io`-оос татна |

> `config/` болон `config-docker/` хоёрт ижил өөрчлөлт хийх ёстой. Ялгаатай байх ёстой зүйл нь зөвхөн хаягууд: `config-docker/`-д `host.docker.internal` гэх мэт.

---

## 3. Local орчинд ажиллуулах

Шаардлагатай: Docker Desktop, Go 1.25.

```bash
go mod download                        # Dockerfile нь host-ийн module cache-ийг ашигладаг
docker compose up -d --build           # docker-compose.override.yml автоматаар уншигдана
docker compose logs -f openim-server
```

- `docker-compose.override.yml` дахь `modcache: /Users/batxac/go/pkg/mod` замыг өөрийн `go env GOMODCACHE`-ийн утгаар солино.
- nsp-chat-server (`:10009`) болон community_user (`:9081`) host дээр ажиллаж байх ёстой. Container-оос `host.docker.internal`-ээр холбогдоно.
- Кодыг өөрчилсний дараа: `docker compose up -d --build openim-server`
- Зөвхөн `config-docker/`-ийг өөрчилсөн бол: `docker compose restart openim-server`

Test:

```bash
go test ./internal/push/...
```

`pkg/common/config` дээрх зарим test (`TestLoadNotificationConfig` гэх мэт) upstream-ээс өвлөгдсөн байдлаар унадаг. Эдгээр нь NSP-ийн өөрчлөлттэй холбоогүй.

---

## 4. Build хийж Nexus руу push хийх (`deploy.sh`)

```bash
docker login repo.nextsocial.mn        # анх удаа
./deploy.sh
```

Script-ийн хийх ажил:

1. Go module cache-ийг шинэчилнэ (`go mod download`).
2. `docker buildx build --platform linux/amd64 --push` ажиллуулна. Image build хийгдмэгц шууд registry руу явна.
3. Prod-ийн `.env`-д тавих image-ийн нэрийг хэвлэнэ.

### Хувилбарын tag

Tag автоматаар `{version/version}-nsp-{git sha}` хэлбэрээр үүснэ. Жишээ нь `v3.8.3-patch.16-nsp-57551a69e`.

- Commit хийгээгүй өөрчлөлт байвал ард нь `-dirty` нэмэгдэнэ. **Prod-д `-dirty` image гаргахгүй.** Эхлээд commit, push хийнэ.
- `:latest` tag мөн давхар push хийгдэнэ.
- Гараар өгөх: `VERSION=v3.8.3-nsp.2 ./deploy.sh`
- Өөр архитектур: `PLATFORM=linux/arm64 ./deploy.sh`

Mac (arm64) дээр amd64 image build хийхэд эмуляци ашиглагддаг тул хэдэн минут удаж болно.

---

## 5. Prod дээр гаргах

Prod нь `config/` хавтсыг volume болгон холбодоггүй. Image доторх `config/`-ийг ашиглаад, орчноос хамаарах утгуудыг `IMENV_` env-ээр дарна.

### Env-ийн нэр

Нэр нь файлын нэр болон YAML key-ийн замаас бүрдэнэ:

```
config/openim-push.yml → nsp.address   ⇒  IMENV_OPENIM_PUSH_NSP_ADDRESS
config/webhooks.yml    → url           ⇒  IMENV_WEBHOOKS_URL
```

Зөвхөн image доторх YAML-д байгаа key-ийг ингэж дарж болно.

### docker-compose (openim-server)

Одоо байгаа `IMENV_MONGODB_*`, `IMENV_REDIS_*` гэх мэт мөрүүд дээр нэмнэ:

```yaml
  openim-server:
    image: ${OPENIM_SERVER_IMAGE}
    environment:
      # ...
      - IMENV_OPENIM_PUSH_ENABLE=nsp
      - IMENV_OPENIM_PUSH_NSP_ADDRESS=${NSP_PUSH_ADDRESS}
      - IMENV_OPENIM_PUSH_NSP_TIMEOUT=5
      - IMENV_WEBHOOKS_URL=${NSP_WEBHOOK_URL}
    extra_hosts:
      - "host.docker.internal:host-gateway"   # host network дээрх сервис рүү хандахад
```

`.env`:

```
OPENIM_SERVER_IMAGE=repo.nextsocial.mn/nsp-openim-server:v3.8.3-patch.16-nsp-xxxxxxx
NSP_PUSH_ADDRESS=host.docker.internal:9081          # community_user gRPC
NSP_WEBHOOK_URL=http://host.docker.internal:10009/nsp   # nsp-chat-server admin-api
```

Шинэчлэх:

```bash
docker compose pull openim-server && docker compose up -d openim-server
docker exec openim-server mage check     # бүх сервис ажиллаж байгаа эсэх
```

### Асаах дараалал

**nsp-chat-server, community_organization, community_user** гурвыг openim-server-ээс өмнө асаасан байх ёстой. Шалтгааныг 6-р хэсгээс үз.

---

## 6. Анхаарах зүйлс

- **`failedContinue` ажилладаггүй.** `webhooks.yml`-д `failedContinue: true` гэж байгаа ч энэ хувилбарын код үүнийг ашигладаггүй. Хэрэв nsp-chat-server хариу өгөхгүй бол (унасан, 5 секундээс удсан, буруу JSON буцаасан) мессеж илгээх, найз нэмэх, group үүсгэх үйлдэл **амжилтгүй** болно.
- **Webhook-ийн HTTP status шалгагддаггүй.** Зөвхөн body-г уншдаг. Зөвшөөрөх бол `actionCode: 0, nextCode: 0`, хориглох бол `actionCode: 0, nextCode: 1` буцаана.
- **community_user-ийн gRPC (9081) TLS-гүй, нэвтрэлт шалгадаггүй.** Портыг гадна талд нээж болохгүй.
- **Offline push-д мессежийн текст харагдана.** Нууцлалын шаардлага гарвал `offlinepush_content.go` дахь `fillOfflinePushInfo`-г өөрчилнө.
- **RPC портууд санамсаргүй.** `autoSetPorts: true` тул сервис бүр асах үедээ сул порт аваад etcd-д бүртгүүлдэг. `dial tcp 172.x.x.x:3xxxx: connection refused` гэсэн алдаа нь ихэвчлэн аль нэг RPC сервис унасан эсвэл etcd-д хуучин бүртгэл үлдсэн гэсэн үг:

  ```bash
  docker exec openim-server mage check
  docker exec etcd etcdctl get --prefix openim | grep -B1 <порт>
  docker logs openim-server 2>&1 | grep -iE "error|panic" | tail -50
  ```

- **`deploy.sh` дээр `ERROR: failed to build: EOF` гарвал** Docker Desktop-ийн daemon гацсан байна. Docker Desktop-ийг дахин асаагаад `docker info` хариу өгч байгаа эсэхийг шалгана.
