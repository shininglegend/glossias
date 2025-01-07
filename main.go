package logosstories

import (
	"fmt"
	"net/http"
)

func main() {
	// Define routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})

	// Start server
	fmt.Println("Server running on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Read from html file
func servePage1(w http.ResponseWriter, r *http.Request) {
	
}