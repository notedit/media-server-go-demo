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
)

type Message struct {
	Cmd string `json:"cmd,omitempty"`
	Sdp string `json:"sdp,omitempty"`
}

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var audioCapability = &sdp.Capability{
	Codecs: []string{"opus"},
}

var videoCapability = &sdp.Capability{
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

		if msg.Cmd == "offer" {
			offer, err := sdp.Parse(msg.Sdp)
			if err != nil {
				panic(err)
			}
			transport = endpoint.CreateTransport(offer, nil)
			transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

			ice := transport.GetLocalICEInfo()
			dtls := transport.GetLocalDTLSInfo()
			candidates := endpoint.GetLocalCandidates()

			answer := sdp.NewSDPInfo()
			answer.SetICE(ice)
			answer.SetDTLS(dtls)
			answer.AddCandidates(candidates)

			if offer.GetMedia("audio") != nil {
				audioMedia := offer.GetMedia("audio").AnswerCapability(audioCapability)
				answer.AddMedia(audioMedia)
			}

			if offer.GetMedia("video") != nil {
				videoMedia := offer.GetMedia("video").AnswerCapability(videoCapability)
				answer.AddMedia(videoMedia)
			}

			transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

			for _, stream := range offer.GetStreams() {

				incomingStream := transport.CreateIncomingStream(stream)
				outgoingStream := transport.CreateOutgoingStream(stream.Clone())

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

func index(c *gin.Context) {
	fmt.Println("helloworld")
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func main() {
	godotenv.Load()
	address := ":8000"
	if os.Getenv("port") != "" {
		address = ":" + os.Getenv("port")
	}
	r := gin.Default()
	r.LoadHTMLFiles("./index.html")
	r.GET("/channel", channel)
	r.GET("/", index)
	r.Run(address)
}
