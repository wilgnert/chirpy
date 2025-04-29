package main

import (
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	api := apiConfig{}
	api.init()

	fileserverHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	respondOkHandler := http.HandlerFunc(RespondOK)

	mux := http.NewServeMux()
	mux.Handle("/app/", api.middlewareMetricsInc(fileserverHandler))
	mux.HandleFunc("GET /admin/healthz", HandleHealth)
	mux.HandleFunc("GET /admin/metrics", api.showMetrics)
	mux.Handle("POST /admin/reset", api.resetMetricsMiddleware(respondOkHandler))
	// mux.Handle("POST /api/validate_chirp", badWordsReplacementMiddleware(http.HandlerFunc(chripyValidator)))

	mux.Handle("POST /api/users", http.HandlerFunc(api.createUser))
	mux.Handle("PUT /api/users", http.HandlerFunc(api.updateUserEmailAndPassword))


	mux.Handle("POST /api/login", http.HandlerFunc(api.login))
	mux.Handle("POST /api/refresh", http.HandlerFunc(api.refresh))
	mux.Handle("POST /api/revoke", http.HandlerFunc(api.revoke))

	mux.Handle("GET /api/chirps", http.HandlerFunc(api.getAllChirps))
	mux.Handle("GET /api/chirps/{chirpID}", http.HandlerFunc(api.getChirpByID))
	mux.Handle("DELETE /api/chirps/{chirpID}", http.HandlerFunc(api.deleteChirpByID))
	mux.Handle("POST /api/chirps", badWordsReplacementMiddleware(chripyValidatorMiddleware(http.HandlerFunc(api.createChirp))))

	mux.Handle("POST /api/polka/webhooks", http.HandlerFunc(api.handleWebhook))

	server := http.Server{}
	server.Handler = mux
	server.Addr = ":8080"
	fmt.Printf("Starting server on http://localhost%v/\n", server.Addr)
	server.ListenAndServe()

}
