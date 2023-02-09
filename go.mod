module github.com/notedit/media-server-go-demo

require (
	github.com/gin-contrib/static v0.0.0-20181225054800-cf5e10bbd933
	github.com/gin-gonic/gin v1.7.7
	github.com/gofrs/uuid v3.1.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/joho/godotenv v1.3.0
	github.com/nareix/joy5 v0.0.0-20200409150540-6c2a804a2816
	github.com/notedit/gst v0.0.4
	github.com/notedit/gstreamer-go v0.3.0
	github.com/notedit/media-server-go v0.1.18
	github.com/notedit/rtmp-lib v0.0.2
	github.com/notedit/sdp v0.0.4
	github.com/pion/rtp v1.4.0
	github.com/pion/webrtc/v2 v2.2.8
	github.com/sanity-io/litter v1.1.0
)

replace github.com/notedit/media-server-go v0.1.18 => ../media-server-go

go 1.13
