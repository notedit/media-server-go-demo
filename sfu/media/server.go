package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

var routers = map[string]*MediaRouter{}
var endpoint = mediaserver.NewEndpoint("127.0.0.1")

func publish(c *gin.Context) {

	var data struct {
		Sdp string `json:"sdp"`
	}

	if err := c.ShouldBind(&data); err != nil {
		c.JSON(200, gin.H{"s": 10000, "e": err})
		return
	}

	router := NewMediaRouter(endpoint, Capabilities)

	publisher, answer := router.CreatePublisher(data.Sdp)

	routers[publisher.incoming.GetID()] = router

	c.JSON(200, gin.H{
		"s": 10000,
		"d": map[string]string{
			"sdp":      answer,
			"streamId": publisher.incoming.GetID(),
		},
		"e": "",
	})

}

func unpublish(c *gin.Context) {

	var data struct {
		StreamId string `json:"streamId"`
	}

	if err := c.ShouldBind(&data); err != nil {
		c.JSON(200, gin.H{"s": 10000, "e": err})
		return
	}

	router := routers[data.StreamId]

	router.Stop()

	c.JSON(200, gin.H{
		"s": 10000,
		"d": map[string]string{},
		"e": "",
	})
}

func play(c *gin.Context) {

	var data struct {
		Sdp        string `json:"sdp"`
		StreamId   string `json:"streamId"`
		OutgoingId string `json:"outgoingId"`
	}

	if err := c.ShouldBind(&data); err != nil {
		c.JSON(200, gin.H{"s": 10000, "e": err})
		return
	}

	router := routers[data.StreamId]

	subscriber, answer := router.CreateSubscriber(data.Sdp)

	c.JSON(200, gin.H{
		"s": 10000,
		"d": map[string]string{
			"sdp":        answer,
			"outgoingId": subscriber.outgoing.GetID(),
		},
		"e": "",
	})
}

func unplay(c *gin.Context) {

	var data struct {
		StreamId   string `json:"streamId"`
		OutgoingId string `json:"outgoingId"`
	}

	if err := c.ShouldBind(&data); err != nil {
		c.JSON(200, gin.H{"s": 10000, "e": err})
		return
	}

	router := routers[data.StreamId]

	router.StopSubscriber(data.OutgoingId)

	c.JSON(200, gin.H{
		"s": 10000,
		"d": map[string]string{},
		"e": "",
	})

}

func offer(c *gin.Context) {

	remoteOffer := endpoint.CreateOffer(Capabilities["video"], Capabilities["audio"])

	c.JSON(200, gin.H{
		"s": 10000,
		"d": map[string]string{
			"sdp": remoteOffer.String(),
		},
		"e": "",
	})

}

func main() {
	r := gin.Default()

	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello World")
	})

	r.POST("/api/publish", publish)
	r.POST("/api/unpublish", unpublish)
	r.POST("/api/play", play)
	r.POST("/api/unplay", unplay)
	r.POST("/api/offer", offer)

	r.Run(":5000")
}
