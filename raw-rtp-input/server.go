package main

import "C"

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	gstreamer "github.com/notedit/gstreamer-go"
	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/media-server-go/sdp"
	uuid "github.com/satori/go.uuid"
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
			transport = endpoint.CreateTransportWithRemote(offer, false)
			transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

			dtls := transport.GetLocalDTLSInfo()
			ice := transport.GetLocalICEInfo()

			answer := sdp.NewSDPInfo()
			answer.SetDTLS(dtls)
			answer.SetICE(ice)

			for _, candidate := range endpoint.GetLocalCandidates() {
				answer.AddCandidate(candidate)
			}

			videoInfo := sdp.NewMediaInfo("video", "video")
			videoCodecInfo := offer.GetMedia("video").GetCodec(videoCodec)
			videoInfo.AddCodec(videoCodecInfo)

			rawStreamer := mediaserver.NewRawRTPStreamer()
			videoSession := rawStreamer.CreateSession(videoInfo)

			videoInfo.SetDirection(sdp.SENDONLY)

			answer.AddMedia(videoInfo)

			transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

			outgoingStream := transport.CreateOutgoingStream(uuid.Must(uuid.NewV4()).String(), false, true)

			outgoingStream.GetVideoTracks()[0].AttachTo(videoSession.GetIncomingStreamTrack())

			go generteRawRTP(videoSession, videoCodecInfo.GetType())

			info := outgoingStream.GetStreamInfo()
			answer.AddStream(info)

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})
		}
	}
}

func generteRawRTP(session *mediaserver.RawRTPStreamerSession, payload int) {

	pipelineStr := "videotestsrc is-live=true ! video/x-raw,format=I420,framerate=15/1 ! x264enc aud=false bframes=0 speed-preset=veryfast key-int-max=15 ! video/x-h264,stream-format=byte-stream,profile=baseline ! h264parse ! rtph264pay config-interval=-1  pt=%d ! appsink name=appsink"
	pipelineStr = fmt.Sprintf(pipelineStr, payload)
	pipeline, err := gstreamer.New(pipelineStr)

	if err != nil {
		panic("can not create pipeline")
	}

	appsink := pipeline.FindElement("appsink")
	pipeline.Start()
	out := appsink.Poll()

	for {
		buffer := <-out
		fmt.Println("push buffer", len(buffer))
		session.Push(buffer)
	}
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
	r := gin.Default()
	r.LoadHTMLFiles("./index.html")
	r.GET("/channel", channel)
	r.GET("/", index)
	r.Run(address)
}
