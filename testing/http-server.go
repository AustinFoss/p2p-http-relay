package main

import (
    "flag"
    "fmt"
    "net/http"
)

func main() {

    port := flag.String("port", "43110", "Port to run the server on (e.g., 43110)")
    message := flag.String("message", "Hi!", "Message to return from /hello endpoint")
    flag.Parse()	

    // Define the /hello endpoint
    http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(*message))
    })

    // Start the server on port 43110
    addr := fmt.Sprintf(":%s", *port)
    fmt.Printf("Starting HTTP server on %s\n", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        fmt.Println("Server error:", err)
    }
}
