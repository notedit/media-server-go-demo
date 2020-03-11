package main

import "C"

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	mediaserver "github.com/notedit/media-server-go"
	rtmppusher "github.com/notedit/media-server-go-demo/webrtc-to-rtmp/rtmp"
	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
	"github.com/notedit/rtmp-lib/pubsub"
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
				

				refresher := mediaserver.NewRefresher(5000)
				refresher.AddStream(incomingStream)

				outgoingStream := transport.CreateOutgoingStream(stream.Clone())
				outgoingStream.AttachTo(incomingStream)
				answer.AddStream(outgoingStream.GetStreamInfo())

				pusher, err := rtmppusher.NewRtmpPusher("rtmp://127.0.0.1/live/live")
				if err != nil {
					panic(err)
				}

				pusher.Start()

				if len(incomingStream.GetVideoTracks()) > 0 {

					videoTrack := incomingStream.GetVideoTracks()[0]

					videoTrack.OnMediaFrame(func(frame []byte, timestamp uint) {

						if len(frame) <= 4 {
							return
						}
						pusher.Push(frame,false)

					})
				}

				if len(incomingStream.GetAudioTracks()) > 0 {

					audioTrack := incomingStream.GetAudioTracks()[0]

					audioTrack.OnMediaFrame(func(frame []byte, timestamp uint){

						if len(frame) <= 4 {
							return
						}

						pusher.Push(frame,true)
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


type Channel struct {
	que  *pubsub.Queue
}

var channels = map[string]*Channel{}

func startRtmp() {

	l := &sync.RWMutex{}

	server := &rtmp.Server{}

	server.HandlePublish = func(conn *rtmp.Conn) {


		l.Lock()
		ch := channels[conn.URL.Path]

		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.SetMaxGopCount(1)
			channels[conn.URL.Path] = ch
		}
		l.Unlock()

		var err error

		var streams []av.CodecData

		if streams, err = conn.Streams(); err != nil {
			fmt.Println(err)
			return
		}

		ch.que.WriteHeader(streams)

		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				fmt.Println(err)
				break
			}

			ch.que.WritePacket(packet)

			fmt.Println("got rtmp packet", packet.Time)
		}

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()

		ch.que.Close()
	}

	server.HandlePlay = func(conn *rtmp.Conn) {


		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {

			cursor := ch.que.Latest()

			streams, err := cursor.Streams()

			if err != nil {
				panic(err)
			}

			conn.WriteHeader(streams)

			for {
				packet, err := cursor.ReadPacket()
				if err != nil {
					break
				}
				conn.WritePacket(packet)
			}
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
