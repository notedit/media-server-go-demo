module github.com/notedit/media-server-go-demo

require (
	github.com/gin-contrib/static v0.0.0-20181225054800-cf5e10bbd933
	github.com/gin-gonic/gin v1.3.0
	github.com/gofrs/uuid v3.1.0+incompatible
	github.com/gorilla/websocket v1.4.0
	github.com/joho/godotenv v1.3.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/notedit/gstreamer-go v0.3.0
	github.com/notedit/gstreamer-rtmp v0.0.0-20181226050148-9295bf2f2ca8
	github.com/notedit/media-server-go v0.1.12
	github.com/notedit/media-server-go-demo/rtmp-to-webrtc/rtmpstreamer v0.0.0
	github.com/notedit/rtmp-lib v0.0.1
	github.com/notedit/sdp v0.0.0-20190418080450-702b42591eb2
	github.com/sanity-io/litter v1.1.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace (
	github.com/notedit/media-server-go v0.1.12 => ../media-server-go
	github.com/notedit/media-server-go-demo/rtmp-to-webrtc/rtmpstreamer v0.0.0 => ./rtmp-to-webrtc/rtmpstreamer
)
