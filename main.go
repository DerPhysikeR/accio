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
	"time"

	"github.com/ncruces/zenity"
	"golang.org/x/crypto/nacl/box"
)

type UserQuery func(string, string) (string, error)

func defaultQueryUser(title string, question string) (string, error) {
	secret, err := zenity.Entry(
		question,
		zenity.Title(title),
		zenity.HideText(),
	)
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	return secret, nil
}

type QueryRequest struct {
	Title           string `json:"title"`
	Question        string `json:"question"`
	ClientPublicKey string `json:"client_public_key"`
}

type QueryResponse struct {
	ServerPublicKey string `json:"server_public_key"`
	Nonce           string `json:"nonce"`
	Encrypted       string `json:"encrypted"`
}

func createQueryHandler(queryUser UserQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		log.Printf("[%s] Received request from %s", time.Now().Format(time.RFC3339), remoteAddr)

		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			log.Printf("[%s] Invalid JSON from %s", time.Now().Format(time.RFC3339), remoteAddr)
			return
		}

		log.Printf("[%s] Title: %q, Question: %q", time.Now().Format(time.RFC3339), req.Title, req.Question)

		clientPubKey, err := base64.StdEncoding.DecodeString(req.ClientPublicKey)
		if err != nil || len(clientPubKey) != 32 {
			http.Error(w, "invalid public key", http.StatusBadRequest)
			log.Printf("[%s] Invalid public key from %s", time.Now().Format(time.RFC3339), remoteAddr)
			return
		}
		var clientKey [32]byte
		copy(clientKey[:], clientPubKey)

		secret, err := queryUser(req.Title, req.Question)
		if err != nil {
			http.Error(w, "user canceled or dialog error", http.StatusBadRequest)
			log.Printf("[%s] User canceled or error for request from %s: %v", time.Now().Format(time.RFC3339), remoteAddr, err)
			return
		}

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
		log.Printf("[%s] Responded to %s successfully", time.Now().Format(time.RFC3339), remoteAddr)
	}
}

func queryTarget(target string, title string, question string) string {
	pub, priv, _ := box.GenerateKey(rand.Reader)
	req := QueryRequest{
		Title:           title,
		Question:        question,
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
	return string(msg)
}

func main() {
	commandTemplate := flag.String("c", "", "Command to execute, optionally containing {{}} as a placeholder")
	port := flag.String("p", "51800", "Port to listen on for HTTP requests")
	title := flag.String("t", "Enter Secret", "Title for the dialog box")
	question := flag.String("q", "Please enter secret:", "Question for the dialog box")
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		fmt.Println("Query target:", args[0])
		result := queryTarget(args[0]+":"+*port+"/accio", *title, *question)

		commandStr := strings.ReplaceAll(*commandTemplate, "{{}}", result)
		cmd := exec.Command("sh", "-c", commandStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Command execution failed:", err)
		}
	} else {
		http.HandleFunc("/accio", createQueryHandler(defaultQueryUser))
		log.Printf("[%s] Server listening on :%s", time.Now().Format(time.RFC3339), *port)
		for {
			if err := http.ListenAndServe(":"+*port, nil); err != nil {
				log.Printf("[%s] Server crashed: %v", time.Now().Format(time.RFC3339), err)
			}
		}
	}
}
