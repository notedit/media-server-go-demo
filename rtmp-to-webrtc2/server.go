package main

import (
	"context"
	"fmt"
	"github.com/nareix/joy5/format/rtmp"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	"github.com/notedit/media-server-go"
	"github.com/notedit/sdp"
)

const (
	videoPt    = 100
	audioPt    = 96
	videoCodec = "h264"
	audioCodec = "opus"
)

type Message struct {
	Cmd string `json:"cmd,omitempty"`
	Sdp string `json:"sdp,omitempty"`
}


var endpoint = mediaserver.NewEndpoint("127.0.0.1")

var rtcstream *Stream

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Capabilities = map[string]*sdp.Capability{
	"audio": &sdp.Capability{
		Codecs: []string{"opus"},
		Rtcpfbs: []*sdp.RtcpFeedback{
			&sdp.RtcpFeedback{
				ID: "nack",
			},
		},
	},
	"video": &sdp.Capability{
		Codecs: []string{"h264"},
		Rtx:    true,
		Rtcpfbs: []*sdp.RtcpFeedback{
			&sdp.RtcpFeedback{
				ID: "transport-cc",
			},
			&sdp.RtcpFeedback{
				ID:     "ccm",
				Params: []string{"fir"},
			},
			&sdp.RtcpFeedback{
				ID: "nack",
			},
		},
		Extensions: []string{
			"urn:3gpp:video-orientation",
			"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		},
	},
}

func index(c *gin.Context) {

	fmt.Println("helloworld")
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func channel(c *gin.Context) {

	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	var transport *mediaserver.Transport

	for {
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

			outgoingStream := transport.CreateOutgoingStreamWithID(uuid.Must(uuid.NewV4()).String(), true, true)

			outgoingStream.GetVideoTracks()[0].AttachTo(rtcstream.GetVideoTrack())
			outgoingStream.GetAudioTracks()[0].AttachTo(rtcstream.GetAuidoTrack())

			info := outgoingStream.GetStreamInfo()
			answer.AddStream(info)

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})
		}

	}

}

func startRtmp() {
	lis, err := net.Listen("tcp", ":1935")
	if err != nil {
		panic(err)
	}

	s := rtmp.NewServer()

	s.LogEvent = func(c *rtmp.Conn, nc net.Conn, e int) {
		es := rtmp.EventString[e]
		log.Println(nc.LocalAddr(), nc.RemoteAddr(), es)
	}

	s.HandleConn = func(c *rtmp.Conn, nc net.Conn) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Println(c.URL.Path)

		rtcstream = NewStreamer(ctx,c, Capabilities["audio"], Capabilities["video"])

		if c.Publishing {
			for {
				<-ctx.Done()
				return
			}
		}
	}

	for {
		nc, err := lis.Accept()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		go s.HandleNetConn(nc)
	}



}

func main() {

	address := ":8000"
	r := gin.Default()

	r.LoadHTMLFiles("./index.html")
	r.GET("/channel", channel)
	r.GET("/", index)


	go startRtmp()

	r.Run(address)

}
