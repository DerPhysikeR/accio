package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
	if len(os.Args) > 1 {
		fmt.Println("Query target:", os.Args[1])
		result := queryTarget(os.Args[1])
		fmt.Println(result)
	} else {
		http.HandleFunc("/accio", createQueryHandler(defaultQueryUser))
		log.Fatal(http.ListenAndServe(":1234", nil))
	}
}
