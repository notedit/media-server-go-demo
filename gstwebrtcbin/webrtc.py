

from pyee import EventEmitter

import gi
gi.require_version('GObject', '2.0')
from gi.repository import GObject
gi.require_version('Gst', '1.0')
from gi.repository import Gst
gi.require_version('GstWebRTC', '1.0')
from gi.repository import GstWebRTC
gi.require_version('GstSdp', '1.0')
from gi.repository import GstSdp
from gi.repository import GLib


Gst.init(None)

VP8_CAPS = Gst.Caps.from_string('application/x-rtp,media=video,encoding-name=VP8,payload=97,clock-rate=90000')
H264_CAPS = Gst.Caps.from_string('application/x-rtp,media=video,encoding-name=H264,payload=98,clock-rate=90000')
OPUS_CAPS = Gst.Caps.from_string('application/x-rtp,media=audio,encoding-name=OPUS,payload=100,clock-rate=48000')


class WebRTC(EventEmitter):

    def __init__(self,stun_server=None,turn_server=None):
        super().__init__()

        self.stun_server = stun_server
        self.turn_server = turn_server

        self.streams = []

        self.pipe = Gst.Pipeline.new('webrtc')
        self.webrtc = Gst.ElementFactory.make('webrtcbin')

        self.pipe.add(self.webrtc)

        self.webrtc.connect('on-negotiation-needed', self.on_negotiation_needed)
        self.webrtc.connect('on-ice-candidate', self.on_ice_candidate)
        self.webrtc.connect('pad-added', self.on_add_stream)
        self.webrtc.connect('pad-removed', self.on_remove_stream)

        if self.stun_server:
            self.webrtc.set_property('stun-server', self.stun_server)

        if self.turn_server:
            self.webrtc.set_property('turn-server', self.turn_server)

        self.webrtc.set_property('bundle-policy','max-bundle')

        bus = self.pipe.get_bus()
        bus.add_signal_watch()
        bus.connect('message', self._bus_call, None)

        self.pipe.set_state(Gst.State.PLAYING)


    @property
    def local_description(self):
        return self.webrtc.get_property('local-description')

    @property
    def remote_description(self):
        return self.webrtc.get_property('remote-description')

    def on_negotiation_needed(self, element):
        print('on_negotiation_needed==')

    def on_ice_candidate(self, element, mlineindex, candidate):
        print('candidate==',candidate)


    def create_offer(self):
        promise = Gst.Promise.new_with_change_func(self.on_offer_created, self.webrtc, None)
        self.webrtc.emit('create-offer', None, promise)

    def on_offer_created(self, promise, element, _):
        promise.wait()
        reply = promise.get_reply()
        offer = reply.get_value('offer')
        if offer:
            print('============')
            self.emit('offer', offer)

    def add_stream(self, stream):
        self.pipe.add(stream)

        if stream.audio_pad:
            audio_sink_pad = self.webrtc.get_request_pad('sink_%u')
            stream.audio_pad.link(audio_sink_pad)

        if stream.video_pad:
            video_sink_pad = self.webrtc.get_request_pad('sink_%u')
            stream.video_pad.link(video_sink_pad)

        stream.sync_state_with_parent()
        self.streams.append(stream)


    def create_answer(self):
        promise = Gst.Promise.new_with_change_func(self.on_answer_created, self.webrtc, None)
        self.webrtc.emit('create-answer', None, promise)

    def on_answer_created(self, promise, element, _):
        ret = promise.wait()
        if ret != Gst.PromiseResult.REPLIED:
            return
        reply = promise.get_reply()
        answer = reply.get_value('answer')
        if answer:
            self.emit('answer', answer)

    def add_ice_candidate(self, ice):
        sdpMLineIndex = ice['sdpMLineIndex']
        candidate = ice['candidate']
        self.webrtc.emit('add-ice-candidate', sdpMLineIndex, candidate)

    def set_local_description(self, sdp):
        promise = Gst.Promise.new()
        self.webrtc.emit('set-local-description', sdp, promise)
        promise.interrupt()


    def set_remote_description(self, sdp):
        promise = Gst.Promise.new()
        self.webrtc.emit('set-remote-description', sdp, promise)
        promise.interrupt()
    
    def _bus_call(self, bus, message, _):
        t = message.type
        if t == Gst.MessageType.EOS:
            print('End-of-stream')
        elif t == Gst.MessageType.ERROR:
            err, debug = message.parse_error()
            print('Error: %s: %s\n' % (err, debug))
        return True

    def on_add_stream(self,element, pad):
        if pad.direction == Gst.PadDirection.SINK:
            # local stream added
            print('local stream added')
            return
        # remote stream 
        print('got one remote stream')

    def on_remove_stream(self, element, pad):
        if pad.direction == Gst.PadDirection.SINK:
            # local stream removed
            return


class Source(Gst.Bin):

    def __init__(self):
        Gst.Bin.__init__(self)

    @property
    def audio_pad(self):
        raise 'need have audio src pad'

    @property
    def video_pad(self):
        raise 'need have video src pad'


TEST_VIDEO_BIN_STR = '''
videotestsrc ! videoconvert ! queue ! vp8enc deadline=1 ! rtpvp8pay !
application/x-rtp,media=video,encoding-name=VP8,payload=97,clock-rate=90000 ! queue
'''

TEST_AUDIO_BIN_STR = '''
audiotestsrc wave=red-noise ! audioconvert ! audioresample ! queue ! opusenc ! rtpopuspay !
application/x-rtp,media=audio,encoding-name=OPUS,payload=100,clock-rate=48000 ! queue
'''


class TestSource(Source):

    def __init__(self):
        Source.__init__(self)

        audiobin = Gst.parse_bin_from_description(TEST_AUDIO_BIN_STR, True)
        videobin = Gst.parse_bin_from_description(TEST_VIDEO_BIN_STR, True)

        self.add(audiobin)
        self.add(videobin)

        self.audio_srcpad = Gst.GhostPad.new('audio_src', audiobin.get_static_pad('src'))
        self.add_pad(self.audio_srcpad)

        self.video_srcpad = Gst.GhostPad.new('video_src', videobin.get_static_pad('src'))
        self.add_pad(self.video_srcpad)


    @property
    def audio_pad(self):
        return self.audio_srcpad

    @property
    def video_pad(self):
        return self.video_srcpad

    


import json
import asyncio
import websockets



async def connect():

   async with websockets.connect(
            'ws://localhost:8000/channel') as websocket:

        pc = WebRTC()

        @pc.on('offer')
        def on_offer(offer):
            print('offer\n')
            pc.set_local_description(offer)
            loop = asyncio.new_event_loop()
            loop.run_until_complete(websocket.send(json.dumps({
                'sdp':offer.sdp.as_text(),
                'cmd':'publish'
            })))
            print('offer\n')
            print(offer.sdp.as_text())

        @pc.on('answer')
        def on_answer(answer):
            print('answer\n')
            print(answer.sdp.as_text())

        source = TestSource()
        pc.add_stream(source)
        pc.create_offer()
        
        async for message in websocket:
            print(message)
            msg = json.loads(message)
            if msg['cmd'] == 'answer':
                sdp = msg['sdp']
                _,sdpmsg = GstSdp.SDPMessage.new()
                GstSdp.sdp_message_parse_buffer(bytes(sdp.encode()), sdpmsg)
                answer = GstWebRTC.WebRTCSessionDescription.new(GstWebRTC.WebRTCSDPType.ANSWER, sdpmsg)
                pc.set_remote_description(answer)



asyncio.get_event_loop().run_until_complete(connect())


