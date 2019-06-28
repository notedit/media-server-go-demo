package main

import "C"

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	gstrtmp "github.com/notedit/gstreamer-rtmp"
	mediaserver "github.com/notedit/media-server-go"
	rtmp "github.com/notedit/rtmp-lib"
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

var Capabilities = map[string]*sdp.Capability{
	"audio": &sdp.Capability{
		Codecs: []string{"opus"},
	},
	"video": &sdp.Capability{
		Codecs: []string{"h264"},
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

			answer := offer.Answer(transport.GetLocalICEInfo(),
				transport.GetLocalDTLSInfo(),
				endpoint.GetLocalCandidates(),
				Capabilities)

			transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

			for _, stream := range offer.GetStreams() {
				incomingStream := transport.CreateIncomingStream(stream)

				refresher := mediaserver.NewRefresher(2000)
				refresher.AddStream(incomingStream)

				outgoingStream := transport.CreateOutgoingStream(stream.Clone())
				outgoingStream.AttachTo(incomingStream)
				answer.AddStream(outgoingStream.GetStreamInfo())

				if len(incomingStream.GetVideoTracks()) > 0 {

					pipeline := gstrtmp.CreatePipeline("rtmp://127.0.0.1/live/live")
					pipeline.Start()

					videoTrack := incomingStream.GetVideoTracks()[0]

					videoTrack.OnMediaFrame(func(frame []byte, timestamp uint) {

						fmt.Println("media frame ===========")
						if len(frame) <= 4 {
							return
						}
						pipeline.Push(frame)

					})
				}
			}

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})
		}
	}
}

func startRtmp() {

	server := &rtmp.Server{}

	server.HandlePublish = func(conn *rtmp.Conn) {

		fmt.Println("got rtmp stream ")

		var err error

		if _, err = conn.Streams(); err != nil {
			fmt.Println(err)
			return
		}

		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				fmt.Println(err)
				break
			}

			fmt.Println("got rtmp packet", packet.Time)
		}
	}

	server.ListenAndServe()
}

func index(c *gin.Context) {
	fmt.Println("helloworld")
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func main() {
	godotenv.Load()
	mediaserver.EnableDebug(true)
	mediaserver.EnableLog(true)
	address := ":8000"
	if os.Getenv("port") != "" {
		address = ":" + os.Getenv("port")
	}

	go startRtmp()

	r := gin.Default()
	r.LoadHTMLFiles("./index.html")
	r.GET("/channel", channel)
	r.GET("/", index)
	r.Run(address)
}
