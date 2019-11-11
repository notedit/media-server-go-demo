package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/sdp"
	"github.com/sanity-io/litter"
	"github.com/unrolled/secure"
)

type Message struct {
	Cmd      string `json:"cmd,omitempty"`
	Sdp      string `json:"sdp,omitempty"`
	StreamID string `json:"stream,omitempty"`
}

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Capabilities = map[string]*sdp.Capability{
	"audio": &sdp.Capability{
		Codecs: []string{"opus"},
	},
	"video": &sdp.Capability{
		Codecs: []string{"vp8"},
		Rtx:    true,
		Rtcpfbs: []*sdp.RtcpFeedback{
			&sdp.RtcpFeedback{
				ID: "goog-remb",
			},
			&sdp.RtcpFeedback{
				ID: "transport-cc",
			},
			&sdp.RtcpFeedback{
				ID:     "ccm",
				Params: []string{"fir"},
			},
			&sdp.RtcpFeedback{
				ID:     "nack",
				Params: []string{"pli"},
			},
		},
		Extensions: []string{
			"urn:3gpp:video-orientation",
			"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			"urn:ietf:params:rtp-hdrext:toffse",
			"urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
			"urn:ietf:params:rtp-hdrext:sdes:mid",
		},
	},
}

var incomingStreams = map[string]*mediaserver.IncomingStream{}

func channel(c *gin.Context) {

	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	var transport *mediaserver.Transport
	endpoint := mediaserver.NewEndpoint("127.0.0.1")

	for {
		// read json
		var msg Message
		err = ws.ReadJSON(&msg)
		if err != nil {
			fmt.Println("error: ", err)
			break
		}
		// 推流端
		if msg.Cmd == "publish" {
			offer, err := sdp.Parse(msg.Sdp)
			if err != nil {
				panic(err)
			}

			transport = endpoint.CreateTransport(offer, nil)
			transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

			answer := offer.Answer(transport.GetLocalICEInfo(),
				transport.GetLocalDTLSInfo(),
				endpoint.GetLocalCandidates(),
				Capabilities)

			transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

			for _, stream := range offer.GetStreams() {
				incomingStream := transport.CreateIncomingStream(stream)
				incomingStreams[incomingStream.GetID()] = incomingStream
			}

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})

		}

		// 观看者
		if msg.Cmd == "watch" {

			offer, err := sdp.Parse(msg.Sdp)
			if err != nil {
				panic(err)
			}
			transport = endpoint.CreateTransport(offer, nil)
			transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

			answer := offer.Answer(transport.GetLocalICEInfo(),
				transport.GetLocalDTLSInfo(),
				endpoint.GetLocalCandidates(),
				Capabilities)

			transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

			// 通过StreamID获取对应的流
			if incomingStream, ok := incomingStreams[msg.StreamID]; ok {
				litter.Dump(incomingStream.GetStreamInfo())

				//　关联推流内容和该观看者
				outgoingStream := transport.CreateOutgoingStream(incomingStream.GetStreamInfo())
				outgoingStream.AttachTo(incomingStream)
				answer.AddStream(outgoingStream.GetStreamInfo())
			}

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})
		}

	}
}

func publish(c *gin.Context) {
	c.HTML(http.StatusOK, "publish.html", gin.H{})
}

func watch(c *gin.Context) {
	c.HTML(http.StatusOK, "watch.html", gin.H{})
}

func main() {
	godotenv.Load()
	address := ":8000"
	if os.Getenv("port") != "" {
		address = ":" + os.Getenv("port")
	}
	r := gin.Default()
	r.Use(TlsHandler())
	r.LoadHTMLFiles("./publish.html", "./watch.html")
	r.GET("/channel", channel)
	r.GET("/watch/:stream", watch)
	r.GET("/", publish)
	r.RunTLS(address, "../certs/mycert.pem", "../certs/mycert.key")
}

func TlsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		secureMiddleware := secure.New(secure.Options{
			SSLRedirect: true,
			SSLHost:     "localhost:8080",
		})
		err := secureMiddleware.Process(c.Writer, c.Request)

		// If there was an error, do not continue.
		if err != nil {
			return
		}

		c.Next()
	}
}
