package main

import (
	"fmt"

	"github.com/pions/webrtc"
	gst "github.com/pions/webrtc/examples/util/gstreamer-src"
	"github.com/pions/webrtc/pkg/ice"

	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/sdp"
)

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

func main() {

	webrtc.RegisterDefaultCodecs()

	config := webrtc.RTCConfiguration{
		IceServers: []webrtc.RTCIceServer{
			{},
		},
	}

	peerConnection, err := webrtc.New(config)

	if err != nil {
		panic("pc create error")
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState ice.ConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	opusTrack, err := peerConnection.NewRTCSampleTrack(webrtc.DefaultPayloadTypeOpus, "audio", "pion audio")
	if err != nil {
		panic(err)
	}

	_, err = peerConnection.AddTrack(opusTrack)
	if err != nil {
		panic(err)
	}

	// Create a video track
	vp8Track, err := peerConnection.NewRTCSampleTrack(webrtc.DefaultPayloadTypeVP8, "video", "pion video")
	if err != nil {
		panic(err)
	}
	_, err = peerConnection.AddTrack(vp8Track)
	if err != nil {
		panic(err)
	}

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// mediaserver go
	endpoint := mediaserver.NewEndpoint("127.0.0.1")

	offerInfo, err := sdp.Parse(offer.Sdp)
	if err != nil {
		panic(err)
	}
	transport := endpoint.CreateTransport(offerInfo, nil)

	transport.SetRemoteProperties(offerInfo.GetMedia("audio"), offerInfo.GetMedia("video"))

	answerInfo := offerInfo.Answer(transport.GetLocalICEInfo(),
		transport.GetLocalDTLSInfo(),
		endpoint.GetLocalCandidates(),
		Capabilities)

	transport.SetLocalProperties(answerInfo.GetMedia("audio"), answerInfo.GetMedia("video"))

	// pions
	answer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeAnswer,
		Sdp:  answerInfo.String(),
	}

	err = peerConnection.SetRemoteDescription(answer)

	if err != nil {
		panic(err)
	}

	gst.CreatePipeline(webrtc.Opus, opusTrack.Samples).Start()
	gst.CreatePipeline(webrtc.VP8, vp8Track.Samples).Start()

	// Block forever
	select {}
}
