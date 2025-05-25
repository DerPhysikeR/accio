package main

import (
	"flag"
	"fmt"
	"io"
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

func createQueryHandler(queryUser UserQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := queryUser()
		fmt.Fprint(w, secret)
	}
}

func queryTarget(target string) string {
	resp, err := http.Get(target + ":1234/accio")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Reading response failed: %v", err)
	}
	return string(body)
}

func main() {
	commandTemplate := flag.String("c", "", "Command to execute, optionally containing {{}} as a placeholder")
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		fmt.Println("Query target:", args[0])
		result := queryTarget(args[0])

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
		log.Fatal(http.ListenAndServe(":1234", nil))
	}
}
