module github.com/notedit/media-server-go-demo

require (
	github.com/gin-contrib/sse v0.0.0-20170109093832-22d885f9ecc7 // indirect
	github.com/gin-gonic/gin v1.3.0
	github.com/gofrs/uuid v3.1.0+incompatible
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/joho/godotenv v1.3.0
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/notedit/gstreamer-go v0.0.0-20181227130428-6e962b0d13bc
	github.com/notedit/gstreamer-rtmp v0.0.0-20181226050148-9295bf2f2ca8
	github.com/notedit/media-server-go v0.1.2
	github.com/olahol/melody v0.0.0-20180227134253-7bd65910e5ab
	github.com/pions/webrtc v1.2.0
	github.com/sanity-io/litter v1.1.0
	github.com/ugorji/go/codec v0.0.0-20181209151446-772ced7fd4c2 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/go-playground/validator.v8 v8.18.2 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace github.com/notedit/media-server-go v0.1.2 => ../media-server-go
