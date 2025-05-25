package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/ncruces/zenity"
	"golang.org/x/crypto/nacl/box"
)

type UserQuery func() string

func defaultQueryUser() string {
	secret, err := zenity.Entry(
		"Enter secret",
		zenity.Title("Enter Secret"),
		zenity.HideText(),
	)
	if err != nil {
		log.Fatalf("Dialog error: %v", err)
	}
	return secret
}

type QueryRequest struct {
	ClientPublicKey string `json:"client_public_key"`
}

type QueryResponse struct {
	ServerPublicKey string `json:"server_public_key"`
	Nonce           string `json:"nonce"`
	Encrypted       string `json:"encrypted"`
}

func createQueryHandler(queryUser UserQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		clientPubKey, err := base64.StdEncoding.DecodeString(req.ClientPublicKey)
		if err != nil || len(clientPubKey) != 32 {
			http.Error(w, "invalid public key", http.StatusBadRequest)
			return
		}
		var clientKey [32]byte
		copy(clientKey[:], clientPubKey)

		secret := queryUser()

		var nonce [24]byte
		rand.Read(nonce[:])

		pub, priv, _ := box.GenerateKey(rand.Reader)

		encrypted := box.Seal(nil, []byte(secret), &nonce, &clientKey, priv)

		resp := QueryResponse{
			ServerPublicKey: base64.StdEncoding.EncodeToString(pub[:]),
			Nonce:           base64.StdEncoding.EncodeToString(nonce[:]),
			Encrypted:       base64.StdEncoding.EncodeToString(encrypted),

		}
		json.NewEncoder(w).Encode(resp)
	}
}

func queryTarget(target string) string {
	pub, priv, _ := box.GenerateKey(rand.Reader)
	req := QueryRequest{
		ClientPublicKey: base64.StdEncoding.EncodeToString(pub[:]),
	}
	b, _ := json.Marshal(req)
	resp, err := http.Post(target, "application/json", strings.NewReader(string(b)))
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	var qr QueryResponse
	json.NewDecoder(resp.Body).Decode(&qr)

	srvPubBytes, _ := base64.StdEncoding.DecodeString(qr.ServerPublicKey)
	var srvPub [32]byte
	copy(srvPub[:], srvPubBytes)

	nonceBytes, _ := base64.StdEncoding.DecodeString(qr.Nonce)
	var nonce [24]byte
	copy(nonce[:], nonceBytes)

	encBytes, _ := base64.StdEncoding.DecodeString(qr.Encrypted)
	msg, ok := box.Open(nil, encBytes, &nonce, &srvPub, priv)
	if !ok {
		log.Fatal("decryption failed")
	}
	fmt.Println("Decrypted response:", string(msg))
	return string(msg)
}

func main() {
	commandTemplate := flag.String("c", "", "Command to execute, optionally containing {{}} as a placeholder")
	port := flag.String("p", "51800", "Port to listen on for HTTP requests")
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		fmt.Println("Query target:", args[0])
		result := queryTarget(args[0] + ":" + *port + "/accio")

		commandStr := strings.ReplaceAll(*commandTemplate, "{{}}", result)
		cmd := exec.Command("sh", "-c", commandStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Command execution failed:", err)
			os.Exit(1)
		}
	} else {
		http.HandleFunc("/accio", createQueryHandler(defaultQueryUser))
		log.Fatal(http.ListenAndServe(":"+*port, nil))
	}
}
