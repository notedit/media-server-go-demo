package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/sdp"
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
				ID: "ccm",
			},
			&sdp.RtcpFeedback{
				ID: "nack",
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

			var incomingStream *mediaserver.IncomingStream

			for _, stream := range offer.GetStreams() {
				incomingStream = transport.CreateIncomingStream(stream)
				incomingStreams[incomingStream.GetID()] = incomingStream
			}

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})

			// server to server test begain
			// now lets test webrtc server to webrtc server
			time.Sleep(10 * time.Second)

			endpointA := mediaserver.NewEndpoint("127.0.0.1")
			offerA := endpointA.CreateOffer(Capabilities["video"], Capabilities["audio"])

			endpointB := mediaserver.NewEndpoint("127.0.0.1")
			transportB := endpointB.CreateTransport(offerA, nil, true)
			transportB.SetRemoteProperties(offerA.GetAudioMedia(), offerA.GetVideoMedia())

			answerB := offerA.Answer(transportB.GetLocalICEInfo(),
				transportB.GetLocalDTLSInfo(),
				transportB.GetLocalCandidates(),
				Capabilities)

			transportB.SetLocalProperties(answerB.GetAudioMedia(), answerB.GetVideoMedia())
			outgoingStreamB := transportB.CreateOutgoingStreamWithID("remote-"+incomingStream.GetID(), true, true)

			answerB.AddStream(outgoingStreamB.GetStreamInfo())

			outgoingStreamB.AttachTo(incomingStream)

			transportA := endpointA.CreateTransport(answerB, offerA, true)
			transportA.SetLocalProperties(offerA.GetAudioMedia(), offerA.GetVideoMedia())
			transportA.SetRemoteProperties(answerB.GetAudioMedia(), answerB.GetVideoMedia())

			for _, stream := range answerB.GetStreams() {
				incoming := transportA.CreateIncomingStream(stream)

				fmt.Println("create incoming stream ", incoming.GetID())

				videoTrack := incoming.GetVideoTracks()[0]

				fmt.Println("create incoming track ", videoTrack.GetID())
			}

		}

	}
}

func index(c *gin.Context) {
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
