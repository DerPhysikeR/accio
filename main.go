package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/ncruces/zenity"
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
	// Query string `json:"query"`
}

type QueryResponse struct {
	Secret string `json:"secret"`
}

func createQueryHandler(queryUser UserQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		secret := queryUser()
		resp := QueryResponse{
			Secret: secret,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func queryTarget(target string) string {
	req := QueryRequest{}
	b, _ := json.Marshal(req)
	resp, err := http.Post(target, "application/json", strings.NewReader(string(b)))
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	var qr QueryResponse
	json.NewDecoder(resp.Body).Decode(&qr)
	return qr.Secret
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
