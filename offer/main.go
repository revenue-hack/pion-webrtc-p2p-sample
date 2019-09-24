package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/pion/webrtc/v2"
)

var (
	// stunサーバを使用する
	// localだとなくてもつながることは確認済み
	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	addr *string
)

func main() {
	addr = flag.String("address", ":50000", "Address that the HTTP server is hosted on.")
	flag.Parse()

	// RTCPeerコネクションを作成
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// 適切な通信経路を選び、変更する(ICE)
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// P2Pの接続をSDPで確立する
	if err := establishP2P(peerConnection); err != nil {
		panic(err)
	}

	// data channelでメッセージング
	if err := handleDataChan(peerConnection); err != nil {
		panic(err)
	}

	// Block forever
	select {}
}

func establishP2P(pc *webrtc.PeerConnection) error {
	// offer用のSDP作成
	offerSDP, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}

	// offer側のSDPセット
	err = pc.SetLocalDescription(offerSDP)
	if err != nil {
		return err
	}

	// httpでSDPを互いに交換する
	answer := mustSignalViaHTTP(offerSDP, *addr)

	// answerのSDPをセット
	return pc.SetRemoteDescription(answer)
}

// シグナリングサーバとして、HTTPでSDPを交換する
func mustSignalViaHTTP(offer webrtc.SessionDescription, address string) webrtc.SessionDescription {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(offer)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(fmt.Sprintf("http://%s", address), "application/json; charset=utf-8", b)
	if err != nil {
		panic(err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			panic(closeErr)
		}
	}()

	var answer webrtc.SessionDescription
	err = json.NewDecoder(resp.Body).Decode(&answer)
	if err != nil {
		panic(err)
	}

	return answer
}

// data channelをopenしてメッセージング
func handleDataChan(pc *webrtc.PeerConnection) error {
	// Create a datachannel with label 'data'
	dataChannel, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		return err
	}

	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("[offer] Data channel '%s'-'%d' open.\n", dataChannel.Label(), dataChannel.ID())

		for range time.NewTicker(5 * time.Second).C {
			message := fmt.Sprintf("offer to answer: %d", rand.Intn(100000))

			// Send the message as text
			sendTextErr := dataChannel.SendText(message)
			if sendTextErr != nil {
				panic(sendTextErr)
			}
		}
	})

	// Register text message handling
	// answerから受け取る必要は今回ないのでコメントアウト
	/*
		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("[offer] Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
		})
	*/

	return nil
}
