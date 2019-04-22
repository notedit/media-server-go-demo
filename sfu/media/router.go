package main

import (
	"github.com/gofrs/uuid"
	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/sdp"
)

type Publisher struct {
	incoming  *mediaserver.IncomingStream
	transport *mediaserver.Transport
}

type Subscriber struct {
	outgoing  *mediaserver.OutgoingStream
	transport *mediaserver.Transport
}

type MediaRouter struct {
	routerID     string
	capabilities map[string]*sdp.Capability
	endpoint     *mediaserver.Endpoint
	publisher    *Publisher
	subscribers  map[string]*Subscriber
}

func NewMediaRouter(endpoint *mediaserver.Endpoint, capabilities map[string]*sdp.Capability) *MediaRouter {
	router := &MediaRouter{}
	router.routerID = uuid.Must(uuid.NewV4()).String()
	router.endpoint = endpoint
	router.capabilities = capabilities

	router.subscribers = make(map[string]*Subscriber)
	return router
}

func (r *MediaRouter) GetID() string {
	return r.routerID
}

func (r *MediaRouter) GetPublisher() *Publisher {
	return r.publisher
}

func (r *MediaRouter) CreatePublisher(sdpStr string) (*Publisher, string) {
	offer, err := sdp.Parse(sdpStr)
	if err != nil {
		panic(err)
	}

	transport := r.endpoint.CreateTransport(offer, nil)
	transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

	answer := offer.Answer(transport.GetLocalICEInfo(),
		transport.GetLocalDTLSInfo(),
		r.endpoint.GetLocalCandidates(),
		r.capabilities)

	transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

	streamInfo := offer.GetFirstStream()
	incoming := transport.CreateIncomingStream(streamInfo)

	r.publisher = &Publisher{
		incoming:  incoming,
		transport: transport,
	}

	return r.publisher, answer.String()
}

func (r *MediaRouter) CreateSubscriber(sdpStr string, subscriberId ...string) (*Subscriber, string) {
	offer, err := sdp.Parse(sdpStr)
	if err != nil {
		panic(err)
	}

	transport := r.endpoint.CreateTransport(offer, nil)
	transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

	answer := offer.Answer(transport.GetLocalICEInfo(),
		transport.GetLocalDTLSInfo(),
		r.endpoint.GetLocalCandidates(),
		r.capabilities)

	transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

	var subId string
	if len(subscriberId) == 1 {
		subId = subscriberId[0]
	} else {
		subId = uuid.Must(uuid.NewV4()).String()
	}

	audio := len(r.publisher.incoming.GetAudioTracks()) > 0
	video := len(r.publisher.incoming.GetVideoTracks()) > 0

	outgoing := transport.CreateOutgoingStreamWithID(subId, audio, video)

	outgoing.AttachTo(r.publisher.incoming)

	subscriber := &Subscriber{
		outgoing:  outgoing,
		transport: transport,
	}

	r.subscribers[subId] = subscriber

	return subscriber, answer.String()
}

func (r *MediaRouter) StopSubscriber(subscriberId string) {
	subscriber := r.subscribers[subscriberId]
	if subscriber == nil {
		return
	}
	subscriber.outgoing.Stop()
	subscriber.transport.Stop()

	delete(r.subscribers, subscriberId)
}

func (r *MediaRouter) Stop() {

	if r.publisher != nil {
		r.publisher.incoming.Stop()
		r.publisher.transport.Stop()
	}

	for _, subscriber := range r.subscribers {
		subscriber.outgoing.Stop()
		subscriber.transport.Stop()
	}

	r.publisher = nil
	r.subscribers = nil
}
