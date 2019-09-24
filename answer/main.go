package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v2"
)

var (
	addr   *string
	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
)

func main() {
	addr = flag.String("address", ":50000", "Address to host the HTTP server on.")
	flag.Parse()

	// RTCPeerConnection作成
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// 適切な通信経路を選び、変更する(ICE)
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// data channelをopenしてメッセージング
	handleDataChan(peerConnection)

	if err := establishP2P(peerConnection); err != nil {
		panic(err)
	}

	// Block forever
	select {}
}

// シグナリングサーバとして、HTTPでSDPを交換する
func mustSignalViaHTTP(address string) (offerOut chan webrtc.SessionDescription, answerIn chan webrtc.SessionDescription) {
	offerOut = make(chan webrtc.SessionDescription)
	answerIn = make(chan webrtc.SessionDescription)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var offer webrtc.SessionDescription
		err := json.NewDecoder(r.Body).Decode(&offer)
		if err != nil {
			panic(err)
		}

		offerOut <- offer
		answer := <-answerIn

		err = json.NewEncoder(w).Encode(answer)
		if err != nil {
			panic(err)
		}
	})

	go func() {
		panic(http.ListenAndServe(address, nil))
	}()
	fmt.Println("Listening on", address)

	return
}

func handleDataChan(pc *webrtc.PeerConnection) {
	// data channelを使用してメッセージング
	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// open data channel
		// answerからメッセージを送る必要はないのでコメントアウト
		/*
			d.OnOpen(func() {
				fmt.Printf("[answer] Data channel '%s'-'%d' open.\n", d.Label(), d.ID())

				for range time.NewTicker(5 * time.Second).C {
					message := fmt.Sprintf("answer to offer: %d", rand.Intn(100000))

					// Send the message as text
					sendTextErr := d.SendText(message)
					if sendTextErr != nil {
						panic(sendTextErr)
					}
				}
			})
		*/

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("[answer] Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})
}

func establishP2P(pc *webrtc.PeerConnection) error {
	// SDP交換する
	offerChan, answerChan := mustSignalViaHTTP(*addr)

	// Wait for the remote SessionDescription
	offer := <-offerChan

	err := pc.SetRemoteDescription(offer)
	if err != nil {
		return err
	}

	// Create answer
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = pc.SetLocalDescription(answer)
	if err != nil {
		return err
	}

	// Send the answer
	answerChan <- answer

	return nil
}
